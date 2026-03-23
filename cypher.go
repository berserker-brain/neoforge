package neoforge

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

type CypherQuery struct {
	Query  string
	Params map[string]any
	//must be a pointer to a slice of structs with key tags
	//
	//if it is left nil, it will not attempt to parse the result
	Result any
	Error  error
	//if true, it will return result as an empty slice with no error
	EmptyOk bool
	Stats   *Stats
}

type CypherTransaction struct {
	Queries    []*CypherQuery
	OnCommit   func()
	OnRollback func()
}

func (cypher *CypherQuery) ParseResult(result *neo4j.EagerResult) {
	if cypher.Result == nil {
		return
	}

	if len(result.Records) == 0 {
		if !cypher.EmptyOk {
			cypher.Error = errors.New("no records found. Set CypherQuery.EmptyOk to true if results can be empty")
			return
		}
		return
	}

	cypher.validateResultStruct()
	if cypher.Error != nil {
		return
	}

	// Get the type of the cypher.Result
	slicePtr := reflect.ValueOf(cypher.Result)
	slice := slicePtr.Elem()
	elemType := slice.Type().Elem()

	for _, record := range result.Records {
		// Get an instance of the cypher.Result
		elemPtr := reflect.New(elemType)
		elem := elemPtr.Elem()

		cypher.handleStructElements(elem, elemType, record)

		slice.Set(reflect.Append(slice, elem))
	}
}

func (cypher *CypherQuery) validateResultStruct() {
	resVal := reflect.ValueOf(cypher.Result)
	if resVal.Kind() == reflect.Ptr {
		resVal = resVal.Elem()
	}

	if resVal.Kind() != reflect.Slice {
		cypher.Error = errors.New("result must be a slice of structs. Set CypherQuery.EmptyOk to true if you don't want results")
		return
	}

	elemType := resVal.Type().Elem()
	if elemType.Kind() == reflect.Ptr {
		elemType = elemType.Elem()
	}

	if elemType.Kind() != reflect.Struct {
		cypher.Error = errors.New("slice elements must be structs")
	}
}

func (cypher *CypherQuery) handleStructElements(elem reflect.Value, elemType reflect.Type, record *neo4j.Record) {
	// Iterate over each of the fields for the instance of the cypher.Result
	for i := 0; i < elem.NumField(); i++ {
		field := elem.Field(i)
		structField := elemType.Field(i)

		key := structField.Tag.Get("key")
		if key == "" {
			cypher.Error = errors.New("no key tag found for field: " + structField.Name)
			return
		}

		cypherIsNullable := false
		if strings.Contains(key, "omitempty") {
			cypherIsNullable = true
			key = strings.Replace(key, ",omitempty", "", 1)
		}

		value, ok := record.Get(key)
		if !ok && !cypherIsNullable {
			cypher.Error = errors.New("no value from neo4j found for key: " + key)
			return
		}

		if value == nil {
			continue // if neo4j doesn't have it, just skip it
		}

		fieldType := field.Type()

		if fieldType.Kind() == reflect.Ptr {
			// Create a new value of the underlying type
			underlyingType := fieldType.Elem()
			newValue := reflect.New(underlyingType)

			// Decode the value into the underlying type
			cypher.decodeValue(value, newValue.Elem(), structField, key)

			field.Set(newValue)
			continue
		}

		// Handle non-pointer types
		cypher.decodeValue(value, field, structField, key)
	}
}

func (cypher *CypherQuery) decodeStruct(value any, field reflect.Value) {
	if reflect.TypeOf(value).AssignableTo(field.Type()) {
		field.Set(reflect.ValueOf(value))
		return // if it is assignable, assume no more parsing is needed
	}

	if nodeValue, ok := value.(neo4j.Node); ok {
		//can't use ParseNode because it requires a type at compile time
		handleCustomNodeTags(field, nodeValue)
		err := jsonDecoder(nodeValue.Props, field)
		if err != nil {
			cypher.Error = err
			return
		}
		return
	}

	if relValue, ok := value.(neo4j.Relationship); ok {
		//can't use ParseRelationship because it requires a type at compile time
		handleCustomRelationshipTags(field, relValue)
		err := jsonDecoder(relValue.Props, field)
		if err != nil {
			cypher.Error = err
			return
		}
		return
	}

	if mapValue, ok := value.(map[string]any); ok {
		cypher.decodeMap(mapValue, field)
		return
	}

	// if this error is hit, we may need more support for the type
	cypher.Error = fmt.Errorf("cannot convert %T to struct for field", value)
}

func (cypher *CypherQuery) decodeSlice(value any, field reflect.Value, structField reflect.StructField, key string) {
	convertSliceToAny := func(value any) []any {
		valueReflect := reflect.ValueOf(value)
		if valueReflect.Kind() != reflect.Slice {
			cypher.Error = fmt.Errorf("cannot convert %T to []any for field %s and key %s", value, structField.Name, key)
			return nil
		}

		valueSlice := make([]any, valueReflect.Len())
		for i := 0; i < valueReflect.Len(); i++ {
			valueSlice[i] = valueReflect.Index(i).Interface()
		}
		return valueSlice
	}

	// If the field is a slice of a struct, decode each element
	fieldType := field.Type()
	elemType := fieldType.Elem()
	elemKind := elemType.Kind()

	valueSlice := convertSliceToAny(value)
	if valueSlice == nil {
		return
	}

	// Handle slices of pointers
	if elemKind == reflect.Ptr {
		underlyingElemType := elemType.Elem()

		newSlice := reflect.MakeSlice(fieldType, 0, len(valueSlice))
		for _, v := range valueSlice {
			// Create a new pointer to the underlying type
			elemPtr := reflect.New(underlyingElemType)

			// Decode the value into the underlying type
			if underlyingElemType.Kind() == reflect.Struct {
				cypher.decodeStruct(v, elemPtr.Elem())
			} else {
				// For primitive types, try direct assignment
				cypher.attemptAssignment(v, elemPtr.Elem(), underlyingElemType.Name(), structField.Name, key)
			}

			newSlice = reflect.Append(newSlice, elemPtr)
		}
		field.Set(newSlice)
		return
	}

	if elemKind == reflect.Struct && elemType != reflect.TypeOf(neo4j.Node{}) && elemType != reflect.TypeOf(neo4j.Relationship{}) {
		newSlice := reflect.MakeSlice(fieldType, 0, len(valueSlice))
		for _, v := range valueSlice {
			elemPtr := reflect.New(fieldType.Elem())
			cypher.decodeStruct(v, elemPtr.Elem())
			newSlice = reflect.Append(newSlice, elemPtr.Elem())
		}
		field.Set(newSlice)
		return
	}

	// For slices of primitives or direct types, create a new slice of the correct type
	// and populate it with the converted values
	newSlice := reflect.MakeSlice(fieldType, 0, len(valueSlice))
	for _, v := range valueSlice {
		elemPtr := reflect.New(elemType)
		cypher.attemptAssignment(v, elemPtr.Elem(), elemType.Name(), structField.Name, key)
		newSlice = reflect.Append(newSlice, elemPtr.Elem())
	}
	field.Set(newSlice)
}

func (cypher *CypherQuery) decodeMap(mapValue map[string]any, field reflect.Value) {
	// Recursively decode each struct field from the map using key
	fieldType := field.Type()
	for i := 0; i < field.NumField(); i++ {
		structField := fieldType.Field(i)

		key := structField.Tag.Get("key")
		if key == "" {
			cypher.Error = errors.New("no key tag found for field: " + structField.Name)
			return
		}

		cypherIsNullable := false
		if strings.Contains(key, "omitempty") {
			cypherIsNullable = true
			key = strings.Replace(key, ",omitempty", "", 1)
		}

		f := field.Field(i)
		v, ok := mapValue[key]
		if !ok {
			if !cypherIsNullable {
				cypher.Error = errors.New("no value from neo4j found for key in map: " + key)
				return
			}
			continue // allow omitempty keys, leave zero value
		}

		if v == nil {
			continue // if neo4j doesn't have it, just skip it
		}

		// Handle pointer fields within structs
		if f.Kind() == reflect.Ptr {
			underlyingType := f.Type().Elem()
			newValue := reflect.New(underlyingType)

			cypher.decodeValue(v, newValue.Elem(), structField, key)

			f.Set(newValue)
			continue
		}

		cypher.decodeValue(v, f, structField, key)
	}
}

func (cypher *CypherQuery) decodeValue(value any, field reflect.Value, structField reflect.StructField, key string) {
	fieldType := field.Type()
	switch fieldType.Kind() {
	case reflect.Struct:
		cypher.decodeStruct(value, field)
	case reflect.Slice:
		cypher.decodeSlice(value, field, structField, key)
	default:
		// fallback (e.g., string, float64, etc.)
		cypher.attemptAssignment(value, field, fieldType.Name(), structField.Name, key)
	}
}

func (cypher *CypherQuery) attemptAssignment(value any, field reflect.Value, fieldName string, structFieldName string, key string) bool {
	if !reflect.TypeOf(value).AssignableTo(field.Type()) {
		cypher.Error = fmt.Errorf("cannot convert %T to %s for field %s and key %s", value, fieldName, structFieldName, key)
		return false
	}
	field.Set(reflect.ValueOf(value))
	return true
}
