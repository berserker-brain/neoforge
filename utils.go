package neoforge

import (
	"encoding/json"
	"errors"
	"reflect"
	"strings"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

func RunQuickQuery[T any](cr *CypherRepository, query string, params map[string]any) ([]T, error) {
	var out []T
	q := CypherQuery{
		Query:   query,
		Params:  params,
		Result:  &out,
		EmptyOk: true,
	}
	cr.RunQuery(&q)
	return out, q.Error
}

func ParseNode[T any](node neo4j.Node) (T, error) {
	var t T
	field := reflect.ValueOf(&t).Elem()
	if field.Kind() != reflect.Struct {
		return t, errors.New("t must be a struct")
	}

	handleCustomNodeTags(field, node)
	
	err := jsonDecoder(node.Props, field)
	if err != nil {
		return t, err
	}

	return t, nil
}

//custom tags should be in this format: db:"custom_tag"
func handleCustomNodeTags(field reflect.Value, node neo4j.Node) {
	for i := 0; i < field.NumField(); i++ {
		dbTag := field.Type().Field(i).Tag.Get("db")
		if dbTag == "" {
			continue
		}
		if strings.Contains(dbTag, "labels") {
			field.Field(i).Set(reflect.ValueOf(node.Labels))
		}
	}
}

func ParseRelationship[T any](rel neo4j.Relationship) (T, error) {
	var t T
	field := reflect.ValueOf(&t).Elem()
	if field.Kind() != reflect.Struct {
		return t, errors.New("t must be a struct")
	}

	handleCustomRelationshipTags(field, rel)
	
	err := jsonDecoder(rel.Props, field)
	if err != nil {
		return t, err
	}
	return t, nil
}

//custom tags should be in this format: db:"custom_tag"
func handleCustomRelationshipTags(field reflect.Value, rel neo4j.Relationship) {
	for i := 0; i < field.NumField(); i++ {
		dbTag := field.Type().Field(i).Tag.Get("db")
		if dbTag == "" {
			continue
		}
		if strings.Contains(dbTag, "label") {
			field.Field(i).Set(reflect.ValueOf(rel.Type))
		}
	}
}

func jsonDecoder(value any, field reflect.Value) error{
	jsonBytes, err := json.Marshal(value)
	if err != nil {
		return err
	}

	err = json.Unmarshal(jsonBytes, field.Addr().Interface())
	if err != nil {
		return err
	}
	return nil
}