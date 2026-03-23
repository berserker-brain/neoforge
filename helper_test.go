package neoforge_test

import "github.com/neo4j/neo4j-go-driver/v5/neo4j"

type SomeUser struct {
	Labels    []string `db:"labels"`
	FirstName string   `json:"first_name"`
	LastName  string   `json:"last_name"`
	Email     string   `json:"email"`
	Phone     int64    `json:"phone"`
}

type SomeRelationship struct {
	Label     string `db:"label"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Email     string `json:"email"`
	Phone     int64  `json:"phone"`
}

func mockNeo4jResult(keys []string, values []any, numRecords int) *neo4j.EagerResult {
	if len(keys) != len(values) {
		panic("keys and values must be the same length")
	}

	records := make([]*neo4j.Record, numRecords)
	for i := 0; i < numRecords; i++ {
		records[i] = &neo4j.Record{
			Values: values,
			Keys:   keys,
		}
	}

	return &neo4j.EagerResult{
		Keys:    keys,
		Records: records,
	}
}

func dereferenceSlice[T any](slice []*T) []T {
	result := make([]T, len(slice))
	for i, s := range slice {
		result[i] = *s
	}
	return result
}

type BehemothPointerResult struct {
	MyString            *string               `key:"my_string"`
	MyInt               *int                  `key:"my_int"`
	MyFloat             *float64              `key:"my_float"`
	MyBool              *bool                 `key:"my_bool"`
	MyNode              *neo4j.Node           `key:"my_node"`
	MyRelationship      *neo4j.Relationship   `key:"my_relationship"`
	MyCustomStruct      *SomeUser             `key:"my_custom_struct"`
	MyMap               map[string]any        `key:"my_map"`
	MyStringSlice       []*string             `key:"my_string_slice"`
	MyIntSlice          []*int                `key:"my_int_slice"`
	MyFloatSlice        []*float64            `key:"my_float_slice"`
	MyBoolSlice         []*bool               `key:"my_bool_slice"`
	MyNodeSlice         []*neo4j.Node         `key:"my_node_slice"`
	MyRelationshipSlice []*neo4j.Relationship `key:"my_relationship_slice"`
	MyCustomStructSlice []*SomeUser           `key:"my_custom_struct_slice"`
	MyMapSlice          []map[string]any      `key:"my_map_slice"`
	MyMapOfStructs      *struct {
		MyString            *string               `key:"nested_string"`
		MyInt               *int                  `key:"nested_int"`
		MyFloat             *float64              `key:"nested_float"`
		MyBool              *bool                 `key:"nested_bool"`
		MyNode              *neo4j.Node           `key:"nested_node"`
		MyRelationship      *neo4j.Relationship   `key:"nested_relationship"`
		MyMap               map[string]any        `key:"nested_map"`
		MyCustomStruct      *SomeUser             `key:"nested_custom_struct"`
		MyStringSlice       []*string             `key:"nested_string_slice"`
		MyIntSlice          []*int                `key:"nested_int_slice"`
		MyFloatSlice        []*float64            `key:"nested_float_slice"`
		MyBoolSlice         []*bool               `key:"nested_bool_slice"`
		MyNodeSlice         []*neo4j.Node         `key:"nested_node_slice"`
		MyRelationshipSlice []*neo4j.Relationship `key:"nested_relationship_slice"`
		MyCustomStructSlice []*SomeUser           `key:"nested_custom_struct_slice"`
		MyMapSlice          []map[string]any      `key:"nested_map_slice"`
		MyMapOfStructs      *struct {
			MyString            *string               `key:"double_nested_nested_string"`
			MyInt               *int                  `key:"double_nested_nested_int"`
			MyFloat             *float64              `key:"double_nested_nested_float"`
			MyBool              *bool                 `key:"double_nested_nested_bool"`
			MyNode              *neo4j.Node           `key:"double_nested_nested_node"`
			MyRelationship      *neo4j.Relationship   `key:"double_nested_nested_relationship"`
			MyMap               map[string]any        `key:"double_nested_nested_map"`
			MyCustomStruct      *SomeUser             `key:"double_nested_nested_custom_struct"`
			MyStringSlice       []*string             `key:"double_nested_nested_string_slice"`
			MyIntSlice          []*int                `key:"double_nested_nested_int_slice"`
			MyFloatSlice        []*float64            `key:"double_nested_nested_float_slice"`
			MyBoolSlice         []*bool               `key:"double_nested_nested_bool_slice"`
			MyNodeSlice         []*neo4j.Node         `key:"double_nested_nested_node_slice"`
			MyRelationshipSlice []*neo4j.Relationship `key:"double_nested_nested_relationship_slice"`
			MyCustomStructSlice []*SomeUser           `key:"double_nested_nested_custom_struct_slice"`
			MyMapSlice          []map[string]any      `key:"double_nested_nested_map_slice"`
		} `key:"nested_map_of_structs"`
	} `key:"my_map_of_structs"`
	MyMapOfStructsSlice []*struct {
		MyString            *string               `key:"nested_string"`
		MyInt               *int                  `key:"nested_int"`
		MyFloat             *float64              `key:"nested_float"`
		MyBool              *bool                 `key:"nested_bool"`
		MyNode              *neo4j.Node           `key:"nested_node"`
		MyRelationship      *neo4j.Relationship   `key:"nested_relationship"`
		MyCustomStruct      *SomeUser             `key:"nested_custom_struct"`
		MyStringSlice       []*string             `key:"nested_string_slice"`
		MyIntSlice          []*int                `key:"nested_int_slice"`
		MyFloatSlice        []*float64            `key:"nested_float_slice"`
		MyBoolSlice         []*bool               `key:"nested_bool_slice"`
		MyNodeSlice         []*neo4j.Node         `key:"nested_node_slice"`
		MyRelationshipSlice []*neo4j.Relationship `key:"nested_relationship_slice"`
		MyCustomStructSlice []*SomeUser           `key:"nested_custom_struct_slice"`
		MyMapSlice          []map[string]any      `key:"nested_map_slice"`
		MyMapOfStructs      struct {
			MyString            *string               `key:"double_nested_nested_string"`
			MyInt               *int                  `key:"double_nested_nested_int"`
			MyFloat             *float64              `key:"double_nested_nested_float"`
			MyBool              *bool                 `key:"double_nested_nested_bool"`
			MyNode              *neo4j.Node           `key:"double_nested_nested_node"`
			MyRelationship      *neo4j.Relationship   `key:"double_nested_nested_relationship"`
			MyCustomStruct      *SomeUser             `key:"double_nested_nested_custom_struct"`
			MyStringSlice       []*string             `key:"double_nested_nested_string_slice"`
			MyIntSlice          []*int                `key:"double_nested_nested_int_slice"`
			MyFloatSlice        []*float64            `key:"double_nested_nested_float_slice"`
			MyBoolSlice         []*bool               `key:"double_nested_nested_bool_slice"`
			MyNodeSlice         []*neo4j.Node         `key:"double_nested_nested_node_slice"`
			MyRelationshipSlice []*neo4j.Relationship `key:"double_nested_nested_relationship_slice"`
			MyCustomStructSlice []*SomeUser           `key:"double_nested_nested_custom_struct_slice"`
			MyMapSlice          []map[string]any      `key:"double_nested_nested_map_slice"`
		} `key:"nested_map_of_structs"`
	} `key:"my_map_of_structs_slice"`
}

func getBehemothNeo4jResults() *neo4j.EagerResult {
	doubleMap := map[string]any{
		"double_nested_nested_string":        "value1",
		"double_nested_nested_int":           1,
		"double_nested_nested_float":         1.0,
		"double_nested_nested_bool":          true,
		"double_nested_nested_map":           map[string]any{"key1": "value1", "key2": "value2"},
		"double_nested_nested_node":          neo4j.Node{Props: map[string]any{"key1": "value1"}},
		"double_nested_nested_relationship":  neo4j.Relationship{Props: map[string]any{"key1": "value1"}},
		"double_nested_nested_custom_struct": SomeUser{FirstName: "John", LastName: "Doe", Email: "john.doe@example.com", Phone: 1234567890},
		"double_nested_nested_string_slice":  []string{"value1", "value2"},
		"double_nested_nested_int_slice":     []int{1, 2},
		"double_nested_nested_float_slice":   []float64{1.0, 2.0},
		"double_nested_nested_bool_slice":    []bool{true, false},
		"double_nested_nested_node_slice": []neo4j.Node{
			{Props: map[string]any{"key1": "value1"}},
			{Props: map[string]any{"key1": "value2"}},
		},
		"double_nested_nested_relationship_slice": []neo4j.Relationship{
			{Props: map[string]any{"key1": "value1"}},
			{Props: map[string]any{"key1": "value2"}},
		},
		"double_nested_nested_custom_struct_slice": []SomeUser{
			{FirstName: "John", LastName: "Doe", Email: "john.doe@example.com", Phone: 1234567890},
			{FirstName: "Jane", LastName: "Smith", Email: "jane.smith@example.com", Phone: 9876543210},
		},
		"double_nested_nested_map_slice": []map[string]any{
			{"key1": "value1"},
			{"key1": "value2"},
		},
	}

	nestedMap := map[string]any{
		"nested_string":              "value1",
		"nested_int":                 1,
		"nested_float":               1.0,
		"nested_bool":                true,
		"nested_map":                 map[string]any{"key1": "value1", "key2": "value2"},
		"nested_node":                neo4j.Node{Props: map[string]any{"key1": "value1"}},
		"nested_relationship":        neo4j.Relationship{Props: map[string]any{"key1": "value1"}},
		"nested_custom_struct":       SomeUser{FirstName: "John", LastName: "Doe", Email: "john.doe@example.com", Phone: 1234567890},
		"nested_string_slice":        []string{"value1", "value2"},
		"nested_int_slice":           []int{1, 2},
		"nested_float_slice":         []float64{1.0, 2.0},
		"nested_bool_slice":          []bool{true, false},
		"nested_node_slice":          []neo4j.Node{{Props: map[string]any{"key1": "value1"}}, {Props: map[string]any{"key1": "value2"}}},
		"nested_relationship_slice":  []neo4j.Relationship{{Props: map[string]any{"key1": "value1"}}, {Props: map[string]any{"key1": "value2"}}},
		"nested_custom_struct_slice": []SomeUser{{FirstName: "John", LastName: "Doe", Email: "john.doe@example.com", Phone: 1234567890}, {FirstName: "Jane", LastName: "Smith", Email: "jane.smith@example.com", Phone: 9876543210}},
		"nested_map_slice":           []map[string]any{{"key1": "value1"}, {"key1": "value2"}},
		"nested_map_of_structs":      doubleMap,
	}

	return mockNeo4jResult(
		[]string{
			"my_string",
			"my_int",
			"my_float",
			"my_bool",
			"my_node",
			"my_relationship",
			"my_custom_struct",
			"my_map",
			"my_string_slice",
			"my_int_slice",
			"my_float_slice",
			"my_bool_slice",
			"my_node_slice",
			"my_relationship_slice",
			"my_custom_struct_slice",
			"my_map_slice",
			"my_map_of_structs",
			"my_map_of_structs_slice"},
		[]any{
			"value1", //my_string
			1,        //my_int
			1.0,      //my_float
			true,     //my_bool
			neo4j.Node{Props: map[string]any{"key1": "value1"}},                                            //my_node
			neo4j.Relationship{Props: map[string]any{"key1": "value1"}},                                    //my_relationship
			SomeUser{FirstName: "John", LastName: "Doe", Email: "john.doe@example.com", Phone: 1234567890}, //my_custom_struct
			map[string]any{"key1": "value1", "key2": "value2"},                                             //my_map
			[]string{"value1", "value2"},                                                                   //my_string_slice
			[]int{1, 2},                                                                                    //my_int_slice
			[]float64{1.0, 2.0},                                                                            //my_float_slice
			[]bool{true, false},                                                                            //my_bool_slice
			[]neo4j.Node{ //my_node_slice
				{Props: map[string]any{"key1": "value1"}},
				{Props: map[string]any{"key1": "value2"}},
			},
			[]neo4j.Relationship{ //my_relationship_slice
				{Props: map[string]any{"key1": "value1"}},
				{Props: map[string]any{"key1": "value2"}},
			},
			[]SomeUser{ //my_custom_struct_slice
				{FirstName: "John", LastName: "Doe", Email: "john.doe@example.com", Phone: 1234567890},
				{FirstName: "Jane", LastName: "Smith", Email: "jane.smith@example.com", Phone: 9876543210},
			},
			[]map[string]any{ //my_map_slice
				{"key1": "value1"},
				{"key1": "value2"},
			},
			nestedMap, //my_map_of_structs
			[]map[string]any{ //my_map_of_structs_slice
				nestedMap,
				nestedMap,
			},
		},
		1,
	)
}
