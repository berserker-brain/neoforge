package neoforge_test

import (
	"errors"
	"testing"

	"github.com/berserker-brain/neoforge"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/stretchr/testify/assert"
)

//omitempty is not tested yet

func TestParseResult_resultIsNotSlice(t *testing.T) {
	query := neoforge.CypherQuery{
		Result: struct {
			Key1 string `key:"key1"`
			Key2 string `key:"key2"`
		}{},
	}
	query.ParseResult(mockNeo4jResult([]string{"key1", "key2"}, []any{1, "value2"}, 1))
	assert.Equal(t, query.Error, errors.New("result must be a slice of structs. Set CypherQuery.EmptyOk to true if you don't want results"))
	assert.Error(t, query.Error)
}

func TestParseResult_resultIsMissingCypherKeys(t *testing.T) {
	query := neoforge.CypherQuery{
		Result: &[]struct {
			Key1 int
			Key2 string
		}{},
	}

	query.ParseResult(mockNeo4jResult([]string{"key1", "key2"}, []any{1, "value2"}, 1))
	assert.Error(t, query.Error)
	assert.Equal(t, query.Error, errors.New("no key tag found for field: Key1"))

	query = neoforge.CypherQuery{
		Result: &[]struct {
			Key1 int `key:"key1"`
			Key2 string
		}{},
	}

	query.ParseResult(mockNeo4jResult([]string{"key1", "key2"}, []any{1, "value2"}, 1))
	assert.Error(t, query.Error)
	assert.Equal(t, query.Error, errors.New("no key tag found for field: Key2"))
}

func TestParseResult_catchesInvalidCypherKeys(t *testing.T) {
	query := neoforge.CypherQuery{
		Result: &[]struct {
			Key1 int    `key:"key1"`
			Key2 string `key:"not_in_result"`
		}{},
	}

	query.ParseResult(mockNeo4jResult([]string{"key1", "some_other_key"}, []any{1, 2}, 1))
	assert.Error(t, query.Error)
	assert.Equal(t, query.Error, errors.New("no value from neo4j found for key: not_in_result"))
}

func TestParseResult_catchesTypeMismatch(t *testing.T) {
	res := []struct {
		Key1 string `key:"key1"`
		Key2 string `key:"key2"`
	}{}
	query := neoforge.CypherQuery{
		Result: &res,
	}

	query.ParseResult(mockNeo4jResult([]string{"key1", "key2"}, []any{1, "value2"}, 1))
	assert.Error(t, query.Error)
	assert.Equal(t, query.Error, errors.New("cannot convert int to string for field Key1 and key key1"))
}

func TestParseResult_resultIsNilSoDoesntTryToParse(t *testing.T) {
	query := neoforge.CypherQuery{}
	query.ParseResult(mockNeo4jResult([]string{"key1", "key2"}, []any{1, "value2"}, 1))
	assert.NoError(t, query.Error)
	assert.Nil(t, query.Result)
}

func TestParseResult_EmptyOkDoesntErrorIfNoRecords(t *testing.T) {
	res := []struct {
		Key1 int    `key:"key1"`
		Key2 string `key:"key2"`
	}{}
	query := neoforge.CypherQuery{
		Result: &res,
		EmptyOk: true,
	}

	query.ParseResult(mockNeo4jResult([]string{}, []any{}, 0))
	assert.NoError(t, query.Error)
	assert.Equal(t, 0, len(res))
}

func TestParseResult_EmptyOkStillReturnsDataWhenAvailable(t *testing.T) {
	res := []struct {
		Key1 int    `key:"key1"`
		Key2 string `key:"key2"`
	}{}
	query := neoforge.CypherQuery{
		Result: &res,
		EmptyOk: true,
	}

	query.ParseResult(mockNeo4jResult([]string{"key1", "key2"}, []any{1, "value2"}, 1))
	assert.NoError(t, query.Error)
	assert.Equal(t, res[0].Key1, 1)
	assert.Equal(t, res[0].Key2, "value2")
}

func TestParseResult_EmptyOkErrorsIfNoResultsAndEmptyOkIsFalse(t *testing.T) {
	res := []struct {
		Key1 int    `key:"key1"`
		Key2 string `key:"key2"`
	}{}

	query := neoforge.CypherQuery{
		Result: &res,
	}

	query.ParseResult(mockNeo4jResult([]string{}, []any{}, 0))
	assert.Error(t, query.Error)
	assert.Equal(t, query.Error, errors.New("no records found. Set CypherQuery.EmptyOk to true if results can be empty"))
}

func TestParseResult_AllowNoResultStructIfEmptyOkIsTrue(t *testing.T) {
	query := neoforge.CypherQuery{
		EmptyOk: true,
	}

	query.ParseResult(mockNeo4jResult([]string{"key1"}, []any{1}, 1))
	assert.NoError(t, query.Error)
}

func TestParseResult_parsesBasicResultCorrectly(t *testing.T) {
	res := []struct {
		Key1 int    `key:"key1"`
		Key2 string `key:"key2"`
	}{}
	query := neoforge.CypherQuery{
		Result: &res,
	}

	query.ParseResult(mockNeo4jResult([]string{"key1", "key2"}, []any{1, "value2"}, 2))
	assert.NoError(t, query.Error)
	assert.Equal(t, res[0].Key1, 1)
	assert.Equal(t, res[0].Key2, "value2")
	assert.Equal(t, res[1].Key1, 1)
	assert.Equal(t, res[1].Key2, "value2")
}

func TestParseResult_parsesNodesCorrectly(t *testing.T) {
	res := []struct {
		Key1 neo4j.Node `key:"key1"`
	}{}
	query := neoforge.CypherQuery{
		Result: &res,
	}

	query.ParseResult(mockNeo4jResult(
		[]string{"key1"},
		[]any{
			neo4j.Node{
				Props: map[string]any{"key1": "value1"},
			},
		},
		1,
	))
	assert.NoError(t, query.Error)
	assert.Equal(t, res[0].Key1.Props["key1"], "value1")
}

func TestParseResult_parsesRelationshipsCorrectly(t *testing.T) {
	res := []struct {
		Key1 neo4j.Relationship `key:"key1"`
	}{}
	query := neoforge.CypherQuery{
		Result: &res,
	}

	query.ParseResult(mockNeo4jResult(
		[]string{"key1"},
		[]any{
			neo4j.Relationship{
				Props: map[string]any{"key1": "value1"},
			},
		},
		1,
	))
	assert.NoError(t, query.Error)
	assert.Equal(t, res[0].Key1.Props["key1"], "value1")
}

func TestParseResult_parsesJsonStructCorrectly(t *testing.T) {
	res := []struct {
		Key1 SomeUser `key:"key1"`
	}{}
	query := neoforge.CypherQuery{
		Result: &res,
	}

	query.ParseResult(mockNeo4jResult(
		[]string{"key1"},
		[]any{neo4j.Node{
			Props: map[string]any{
				"first_name": "John",
				"last_name":  "Doe",
				"email":      "john.doe@example.com",
				"phone":      1234567890,
			},
		}},
		1,
	))
	assert.NoError(t, query.Error)
	assert.Equal(t, "John", res[0].Key1.FirstName)
	assert.Equal(t, "Doe", res[0].Key1.LastName)
	assert.Equal(t, "john.doe@example.com", res[0].Key1.Email)
	assert.Equal(t, int64(1234567890), res[0].Key1.Phone)
}

func TestParseResult_parsesSliceOfNodesCorrectly(t *testing.T) {
	res := []struct {
		Key1 []neo4j.Node `key:"key1"`
	}{}
	query := neoforge.CypherQuery{
		Result: &res,
	}

	query.ParseResult(mockNeo4jResult(
		[]string{"key1"},
		[]any{[]neo4j.Node{
			{
				Props: map[string]any{"key1": "value1"},
			},
			{
				Props: map[string]any{"key1": "value2"},
			},
		}},
		1,
	))
	assert.NoError(t, query.Error)
	assert.Equal(t, res[0].Key1[0].Props["key1"], "value1")
	assert.Equal(t, res[0].Key1[1].Props["key1"], "value2")
}

func TestParseResult_parsesSliceOfRelationshipsCorrectly(t *testing.T) {
	res := []struct {
		Key1 []neo4j.Relationship `key:"key1"`
	}{}
	query := neoforge.CypherQuery{
		Result: &res,
	}

	query.ParseResult(mockNeo4jResult(
		[]string{"key1"},
		[]any{[]neo4j.Relationship{
			{
				Props: map[string]any{"key1": "value1"},
			},
			{
				Props: map[string]any{"key1": "value2"},
			},
		}},
		1,
	))
	assert.NoError(t, query.Error)
	assert.Equal(t, res[0].Key1[0].Props["key1"], "value1")
	assert.Equal(t, res[0].Key1[1].Props["key1"], "value2")
}

func TestParseResult_parsesSliceOfJsonStructsCorrectly(t *testing.T) {
	res := []struct {
		Key1 []SomeUser `key:"key1"`
	}{}
	query := neoforge.CypherQuery{
		Result: &res,
	}

	query.ParseResult(mockNeo4jResult(
		[]string{"key1"},
		[]any{[]neo4j.Node{
			{
				Props: map[string]any{
					"first_name": "John",
					"last_name":  "Doe",
					"email":      "john.doe@example.com",
					"phone":      1234567890,
				},
			},
			{
				Props: map[string]any{
					"first_name": "Jane",
					"last_name":  "Smith",
					"email":      "jane.smith@example.com",
					"phone":      9876543210,
				},
			},
		}},
		1,
	))
	assert.NoError(t, query.Error)
	assert.Equal(t, "John", res[0].Key1[0].FirstName)
	assert.Equal(t, "Doe", res[0].Key1[0].LastName)
	assert.Equal(t, "john.doe@example.com", res[0].Key1[0].Email)
	assert.Equal(t, int64(1234567890), res[0].Key1[0].Phone)
	assert.Equal(t, "Jane", res[0].Key1[1].FirstName)
	assert.Equal(t, "Smith", res[0].Key1[1].LastName)
	assert.Equal(t, "jane.smith@example.com", res[0].Key1[1].Email)
	assert.Equal(t, int64(9876543210), res[0].Key1[1].Phone)
}

func TestParseResult_parsesSliceOfPrimativeDataCorrectly(t *testing.T) {
	res := []struct {
		Key1 []int `key:"key1"`
	}{}
	query := neoforge.CypherQuery{
		Result: &res,
	}

	query.ParseResult(mockNeo4jResult(
		[]string{"key1"},
		[]any{[]int{
			1,
			2,
		}},
		1,
	))
	assert.NoError(t, query.Error)
	assert.Equal(t, 1, res[0].Key1[0])
	assert.Equal(t, 2, res[0].Key1[1])
}

func TestParseResult_parsesMapOfPrimativeDataCorrectly(t *testing.T) {
	res := []struct {
		Str     string `key:"str"`
		Numbers struct {
			Int   int     `key:"int"`
			Float float64 `key:"float"`
			Bool  bool    `key:"bool"`
		} `key:"numbers"`
	}{}
	query := neoforge.CypherQuery{
		Result: &res,
	}

	query.ParseResult(mockNeo4jResult(
		[]string{"str", "numbers"},
		[]any{"value1", map[string]any{"int": 1, "float": 2.0, "bool": true}},
		1,
	))
	assert.NoError(t, query.Error)
	assert.Equal(t, "value1", res[0].Str)
	assert.Equal(t, 1, res[0].Numbers.Int)
	assert.Equal(t, 2.0, res[0].Numbers.Float)
	assert.Equal(t, true, res[0].Numbers.Bool)
}

func TestParseResult_parsesSliceOfMapsOfPrimativeDataCorrectly(t *testing.T) {
	res := []struct {
		Numbers []struct {
			Int   int     `key:"int"`
			Float float64 `key:"float"`
			Bool  bool    `key:"bool"`
		} `key:"numbers"`
	}{}
	query := neoforge.CypherQuery{
		Result: &res,
	}

	query.ParseResult(mockNeo4jResult(
		[]string{"numbers"},
		[]any{[]map[string]any{
			{"int": 1, "float": 2.0, "bool": true},
			{"int": 3, "float": 4.0, "bool": false},
		}},
		1,
	))
	assert.NoError(t, query.Error)
	assert.Equal(t, 1, res[0].Numbers[0].Int)
	assert.Equal(t, 2.0, res[0].Numbers[0].Float)
	assert.Equal(t, true, res[0].Numbers[0].Bool)
	assert.Equal(t, 3, res[0].Numbers[1].Int)
	assert.Equal(t, 4.0, res[0].Numbers[1].Float)
	assert.Equal(t, false, res[0].Numbers[1].Bool)
}

func TestParseResult_parsesSliceOfMapsOfCustomStructsCorrectly(t *testing.T) {
	var res = []struct {
		Devices []struct {
			Identity neo4j.Node           `key:"d"`
			User     SomeUser             `key:"u"`
			Used     []neo4j.Relationship `key:"r"`
		} `key:"devices"`
	}{}
	query := neoforge.CypherQuery{
		Result: &res,
	}

	query.ParseResult(mockNeo4jResult(
		[]string{"devices"},
		[]any{[]map[string]any{
			{
				"d": neo4j.Node{
					Props: map[string]any{"key1": "value1"},
				},
				"u": neo4j.Node{
					Props: map[string]any{
						"first_name": "John",
						"last_name":  "Doe",
						"email":      "john.doe@example.com",
						"phone":      1234567890,
					},
				},
				"r": []neo4j.Relationship{
					{
						Props: map[string]any{"key1": "value1"},
					},
					{
						Props: map[string]any{"key1": "value2"},
					},
				},
			},
		}},
		1,
	))

	assert.NoError(t, query.Error)
	assert.Equal(t, 1, len(res[0].Devices))
	assert.Equal(t, 2, len(res[0].Devices[0].Used))
	assert.Equal(t, "value1", res[0].Devices[0].Identity.Props["key1"])
	assert.Equal(t, "John", res[0].Devices[0].User.FirstName)
	assert.Equal(t, "Doe", res[0].Devices[0].User.LastName)
	assert.Equal(t, "john.doe@example.com", res[0].Devices[0].User.Email)
	assert.Equal(t, int64(1234567890), res[0].Devices[0].User.Phone)
	assert.Equal(t, "value1", res[0].Devices[0].Used[0].Props["key1"])
	assert.Equal(t, "value2", res[0].Devices[0].Used[1].Props["key1"])
}

// for the poor soul that has been brought here. I am truly sorry.
func TestParseResult_theBehemoth(t *testing.T) {
	theLeviathan := []struct {
		MyString            string               `key:"my_string"`
		MyInt               int                  `key:"my_int"`
		MyFloat             float64              `key:"my_float"`
		MyBool              bool                 `key:"my_bool"`
		MyNode              neo4j.Node           `key:"my_node"`
		MyRelationship      neo4j.Relationship   `key:"my_relationship"`
		MyCustomStruct      SomeUser             `key:"my_custom_struct"`
		MyMap               map[string]any       `key:"my_map"`
		MyStringSlice       []string             `key:"my_string_slice"`
		MyIntSlice          []int                `key:"my_int_slice"`
		MyFloatSlice        []float64            `key:"my_float_slice"`
		MyBoolSlice         []bool               `key:"my_bool_slice"`
		MyNodeSlice         []neo4j.Node         `key:"my_node_slice"`
		MyRelationshipSlice []neo4j.Relationship `key:"my_relationship_slice"`
		MyCustomStructSlice []SomeUser           `key:"my_custom_struct_slice"`
		MyMapSlice          []map[string]any     `key:"my_map_slice"`
		MyMapOfStructs      struct {
			MyString            string               `key:"nested_string"`
			MyInt               int                  `key:"nested_int"`
			MyFloat             float64              `key:"nested_float"`
			MyBool              bool                 `key:"nested_bool"`
			MyNode              neo4j.Node           `key:"nested_node"`
			MyRelationship      neo4j.Relationship   `key:"nested_relationship"`
			MyMap               map[string]any       `key:"nested_map"`
			MyCustomStruct      SomeUser             `key:"nested_custom_struct"`
			MyStringSlice       []string             `key:"nested_string_slice"`
			MyIntSlice          []int                `key:"nested_int_slice"`
			MyFloatSlice        []float64            `key:"nested_float_slice"`
			MyBoolSlice         []bool               `key:"nested_bool_slice"`
			MyNodeSlice         []neo4j.Node         `key:"nested_node_slice"`
			MyRelationshipSlice []neo4j.Relationship `key:"nested_relationship_slice"`
			MyCustomStructSlice []SomeUser           `key:"nested_custom_struct_slice"`
			MyMapSlice          []map[string]any     `key:"nested_map_slice"`
			MyMapOfStructs      struct {
				MyString            string               `key:"double_nested_nested_string"`
				MyInt               int                  `key:"double_nested_nested_int"`
				MyFloat             float64              `key:"double_nested_nested_float"`
				MyBool              bool                 `key:"double_nested_nested_bool"`
				MyNode              neo4j.Node           `key:"double_nested_nested_node"`
				MyRelationship      neo4j.Relationship   `key:"double_nested_nested_relationship"`
				MyMap               map[string]any       `key:"double_nested_nested_map"`
				MyCustomStruct      SomeUser             `key:"double_nested_nested_custom_struct"`
				MyStringSlice       []string             `key:"double_nested_nested_string_slice"`
				MyIntSlice          []int                `key:"double_nested_nested_int_slice"`
				MyFloatSlice        []float64            `key:"double_nested_nested_float_slice"`
				MyBoolSlice         []bool               `key:"double_nested_nested_bool_slice"`
				MyNodeSlice         []neo4j.Node         `key:"double_nested_nested_node_slice"`
				MyRelationshipSlice []neo4j.Relationship `key:"double_nested_nested_relationship_slice"`
				MyCustomStructSlice []SomeUser           `key:"double_nested_nested_custom_struct_slice"`
				MyMapSlice          []map[string]any     `key:"double_nested_nested_map_slice"`
			} `key:"nested_map_of_structs"`
		} `key:"my_map_of_structs"`
		MyMapOfStructsSlice []struct {
			MyString            string               `key:"nested_string"`
			MyInt               int                  `key:"nested_int"`
			MyFloat             float64              `key:"nested_float"`
			MyBool              bool                 `key:"nested_bool"`
			MyNode              neo4j.Node           `key:"nested_node"`
			MyRelationship      neo4j.Relationship   `key:"nested_relationship"`
			MyCustomStruct      SomeUser             `key:"nested_custom_struct"`
			MyStringSlice       []string             `key:"nested_string_slice"`
			MyIntSlice          []int                `key:"nested_int_slice"`
			MyFloatSlice        []float64            `key:"nested_float_slice"`
			MyBoolSlice         []bool               `key:"nested_bool_slice"`
			MyNodeSlice         []neo4j.Node         `key:"nested_node_slice"`
			MyRelationshipSlice []neo4j.Relationship `key:"nested_relationship_slice"`
			MyCustomStructSlice []SomeUser           `key:"nested_custom_struct_slice"`
			MyMapSlice          []map[string]any     `key:"nested_map_slice"`
			MyMapOfStructs      struct {
				MyString            string               `key:"double_nested_nested_string"`
				MyInt               int                  `key:"double_nested_nested_int"`
				MyFloat             float64              `key:"double_nested_nested_float"`
				MyBool              bool                 `key:"double_nested_nested_bool"`
				MyNode              neo4j.Node           `key:"double_nested_nested_node"`
				MyRelationship      neo4j.Relationship   `key:"double_nested_nested_relationship"`
				MyCustomStruct      SomeUser             `key:"double_nested_nested_custom_struct"`
				MyStringSlice       []string             `key:"double_nested_nested_string_slice"`
				MyIntSlice          []int                `key:"double_nested_nested_int_slice"`
				MyFloatSlice        []float64            `key:"double_nested_nested_float_slice"`
				MyBoolSlice         []bool               `key:"double_nested_nested_bool_slice"`
				MyNodeSlice         []neo4j.Node         `key:"double_nested_nested_node_slice"`
				MyRelationshipSlice []neo4j.Relationship `key:"double_nested_nested_relationship_slice"`
				MyCustomStructSlice []SomeUser           `key:"double_nested_nested_custom_struct_slice"`
				MyMapSlice          []map[string]any     `key:"double_nested_nested_map_slice"`
			} `key:"nested_map_of_structs"`
		} `key:"my_map_of_structs_slice"`
	}{}

	berserker := neoforge.CypherQuery{
		Result: &theLeviathan,
	}

	berserker.ParseResult(getBehemothNeo4jResults())

	assert.NoError(t, berserker.Error)
	assert.Equal(t, "value1", theLeviathan[0].MyString)
	assert.Equal(t, 1, theLeviathan[0].MyInt)
	assert.Equal(t, 1.0, theLeviathan[0].MyFloat)
	assert.Equal(t, true, theLeviathan[0].MyBool)
	assert.Equal(t, "value1", theLeviathan[0].MyNode.Props["key1"])
	assert.Equal(t, "value1", theLeviathan[0].MyRelationship.Props["key1"])
	assert.Equal(t, "John", theLeviathan[0].MyCustomStruct.FirstName)
	assert.Equal(t, "Doe", theLeviathan[0].MyCustomStruct.LastName)
	assert.Equal(t, "john.doe@example.com", theLeviathan[0].MyCustomStruct.Email)
	assert.Equal(t, int64(1234567890), theLeviathan[0].MyCustomStruct.Phone)
	assert.Equal(t, map[string]any{"key1": "value1", "key2": "value2"}, theLeviathan[0].MyMap)
	assert.Equal(t, []string{"value1", "value2"}, theLeviathan[0].MyStringSlice)
	assert.Equal(t, []int{1, 2}, theLeviathan[0].MyIntSlice)
	assert.Equal(t, []float64{1.0, 2.0}, theLeviathan[0].MyFloatSlice)
	assert.Equal(t, []bool{true, false}, theLeviathan[0].MyBoolSlice)
	assert.Equal(t, []neo4j.Node{
		{Props: map[string]any{"key1": "value1"}},
		{Props: map[string]any{"key1": "value2"}}},
		theLeviathan[0].MyNodeSlice)
	assert.Equal(t, []neo4j.Relationship{{Props: map[string]any{"key1": "value1"}},
		{Props: map[string]any{"key1": "value2"}}},
		theLeviathan[0].MyRelationshipSlice)
	assert.Equal(t, []SomeUser{
		{FirstName: "John", LastName: "Doe", Email: "john.doe@example.com", Phone: 1234567890},
		{FirstName: "Jane", LastName: "Smith", Email: "jane.smith@example.com", Phone: 9876543210}},
		theLeviathan[0].MyCustomStructSlice)
	assert.Equal(t, map[string]any{"key1": "value1"}, theLeviathan[0].MyMapSlice[0])
	assert.Equal(t, map[string]any{"key1": "value2"}, theLeviathan[0].MyMapSlice[1])
	assert.Equal(t, "value1", theLeviathan[0].MyMapOfStructs.MyString)
	assert.Equal(t, 1, theLeviathan[0].MyMapOfStructs.MyInt)
	assert.Equal(t, 1.0, theLeviathan[0].MyMapOfStructs.MyFloat)
	assert.Equal(t, true, theLeviathan[0].MyMapOfStructs.MyBool)
	assert.Equal(t, "value1", theLeviathan[0].MyMapOfStructs.MyNode.Props["key1"])
	assert.Equal(t, "value1", theLeviathan[0].MyMapOfStructs.MyRelationship.Props["key1"])
	assert.Equal(t, "John", theLeviathan[0].MyMapOfStructs.MyCustomStruct.FirstName)
	assert.Equal(t, "Doe", theLeviathan[0].MyMapOfStructs.MyCustomStruct.LastName)
	assert.Equal(t, "john.doe@example.com", theLeviathan[0].MyMapOfStructs.MyCustomStruct.Email)
	assert.Equal(t, int64(1234567890), theLeviathan[0].MyMapOfStructs.MyCustomStruct.Phone)
	assert.Equal(t, 2, len(theLeviathan[0].MyMapOfStructs.MyStringSlice))
	assert.Equal(t, 2, len(theLeviathan[0].MyMapOfStructs.MyIntSlice))
	assert.Equal(t, 2, len(theLeviathan[0].MyMapOfStructs.MyFloatSlice))
	assert.Equal(t, 2, len(theLeviathan[0].MyMapOfStructs.MyBoolSlice))
	assert.Equal(t, 2, len(theLeviathan[0].MyMapOfStructs.MyNodeSlice))
	assert.Equal(t, 2, len(theLeviathan[0].MyMapOfStructs.MyRelationshipSlice))
	assert.Equal(t, 2, len(theLeviathan[0].MyMapOfStructs.MyCustomStructSlice))
	assert.Equal(t, 2, len(theLeviathan[0].MyMapOfStructs.MyMapSlice))
	assert.Equal(t, "value1", theLeviathan[0].MyMapOfStructs.MyMapOfStructs.MyString)
	assert.Equal(t, 1, theLeviathan[0].MyMapOfStructs.MyMapOfStructs.MyInt)
	assert.Equal(t, 1.0, theLeviathan[0].MyMapOfStructs.MyMapOfStructs.MyFloat)
	assert.Equal(t, true, theLeviathan[0].MyMapOfStructs.MyMapOfStructs.MyBool)
	assert.Equal(t, "value1", theLeviathan[0].MyMapOfStructs.MyMapOfStructs.MyNode.Props["key1"])
	assert.Equal(t, "value1", theLeviathan[0].MyMapOfStructs.MyMapOfStructs.MyRelationship.Props["key1"])
	assert.Equal(t, "John", theLeviathan[0].MyMapOfStructs.MyMapOfStructs.MyCustomStruct.FirstName)
	assert.Equal(t, "Doe", theLeviathan[0].MyMapOfStructs.MyMapOfStructs.MyCustomStruct.LastName)
	assert.Equal(t, "john.doe@example.com", theLeviathan[0].MyMapOfStructs.MyMapOfStructs.MyCustomStruct.Email)
	assert.Equal(t, int64(1234567890), theLeviathan[0].MyMapOfStructs.MyMapOfStructs.MyCustomStruct.Phone)
	assert.Equal(t, 2, len(theLeviathan[0].MyMapOfStructs.MyMapOfStructs.MyStringSlice))
	assert.Equal(t, 2, len(theLeviathan[0].MyMapOfStructs.MyMapOfStructs.MyIntSlice))
	assert.Equal(t, 2, len(theLeviathan[0].MyMapOfStructs.MyMapOfStructs.MyFloatSlice))
	assert.Equal(t, 2, len(theLeviathan[0].MyMapOfStructs.MyMapOfStructs.MyBoolSlice))
	assert.Equal(t, 2, len(theLeviathan[0].MyMapOfStructs.MyMapOfStructs.MyNodeSlice))
	assert.Equal(t, 2, len(theLeviathan[0].MyMapOfStructs.MyMapOfStructs.MyRelationshipSlice))
	assert.Equal(t, 2, len(theLeviathan[0].MyMapOfStructs.MyMapOfStructs.MyCustomStructSlice))
	assert.Equal(t, 2, len(theLeviathan[0].MyMapOfStructs.MyMapOfStructs.MyMapSlice))
	assert.Equal(t, len(theLeviathan[0].MyMapOfStructsSlice), 2)
}

func TestParseResult_theBehemothAsAPointer(t *testing.T) {
	jormungand := []BehemothPointerResult{}

	berserker := neoforge.CypherQuery{
		Result: &jormungand,
	}

	berserker.ParseResult(getBehemothNeo4jResults())

	assert.NoError(t, berserker.Error)
	assert.Equal(t, "value1", *jormungand[0].MyString)
	assert.Equal(t, 1, *jormungand[0].MyInt)
	assert.Equal(t, 1.0, *jormungand[0].MyFloat)
	assert.Equal(t, true, *jormungand[0].MyBool)
	assert.Equal(t, "value1", jormungand[0].MyNode.Props["key1"])
	assert.Equal(t, "value1", jormungand[0].MyRelationship.Props["key1"])
	assert.Equal(t, "John", jormungand[0].MyCustomStruct.FirstName)
	assert.Equal(t, "Doe", jormungand[0].MyCustomStruct.LastName)
	assert.Equal(t, "john.doe@example.com", jormungand[0].MyCustomStruct.Email)
	assert.Equal(t, int64(1234567890), jormungand[0].MyCustomStruct.Phone)
	assert.Equal(t, map[string]any{"key1": "value1", "key2": "value2"}, jormungand[0].MyMap)
	assert.Equal(t, []string{"value1", "value2"}, dereferenceSlice(jormungand[0].MyStringSlice))
	assert.Equal(t, []int{1, 2}, dereferenceSlice(jormungand[0].MyIntSlice))
	assert.Equal(t, []float64{1.0, 2.0}, dereferenceSlice(jormungand[0].MyFloatSlice))
	assert.Equal(t, []bool{true, false}, dereferenceSlice(jormungand[0].MyBoolSlice))
	assert.Equal(t, []neo4j.Node{
		{Props: map[string]any{"key1": "value1"}},
		{Props: map[string]any{"key1": "value2"}}},
		dereferenceSlice(jormungand[0].MyNodeSlice))
	assert.Equal(t, []neo4j.Relationship{{Props: map[string]any{"key1": "value1"}},
		{Props: map[string]any{"key1": "value2"}}},
		dereferenceSlice(jormungand[0].MyRelationshipSlice))
	assert.Equal(t, []SomeUser{
		{FirstName: "John", LastName: "Doe", Email: "john.doe@example.com", Phone: 1234567890},
		{FirstName: "Jane", LastName: "Smith", Email: "jane.smith@example.com", Phone: 9876543210}},
		dereferenceSlice(jormungand[0].MyCustomStructSlice))
	assert.Equal(t, map[string]any{"key1": "value1"}, jormungand[0].MyMapSlice[0])
	assert.Equal(t, map[string]any{"key1": "value2"}, jormungand[0].MyMapSlice[1])
	assert.Equal(t, "value1", *jormungand[0].MyMapOfStructs.MyString)
	assert.Equal(t, 1, *jormungand[0].MyMapOfStructs.MyInt)
	assert.Equal(t, 1.0, *jormungand[0].MyMapOfStructs.MyFloat)
	assert.Equal(t, true, *jormungand[0].MyMapOfStructs.MyBool)
	assert.Equal(t, "value1", jormungand[0].MyMapOfStructs.MyNode.Props["key1"])
	assert.Equal(t, "value1", jormungand[0].MyMapOfStructs.MyRelationship.Props["key1"])
	assert.Equal(t, "John", jormungand[0].MyMapOfStructs.MyCustomStruct.FirstName)
	assert.Equal(t, "Doe", jormungand[0].MyMapOfStructs.MyCustomStruct.LastName)
	assert.Equal(t, "john.doe@example.com", jormungand[0].MyMapOfStructs.MyCustomStruct.Email)
	assert.Equal(t, int64(1234567890), jormungand[0].MyMapOfStructs.MyCustomStruct.Phone)
	assert.Equal(t, 2, len(jormungand[0].MyMapOfStructs.MyStringSlice))
	assert.Equal(t, 2, len(jormungand[0].MyMapOfStructs.MyIntSlice))
	assert.Equal(t, 2, len(jormungand[0].MyMapOfStructs.MyFloatSlice))
	assert.Equal(t, 2, len(jormungand[0].MyMapOfStructs.MyBoolSlice))
	assert.Equal(t, 2, len(jormungand[0].MyMapOfStructs.MyNodeSlice))
	assert.Equal(t, 2, len(jormungand[0].MyMapOfStructs.MyRelationshipSlice))
	assert.Equal(t, 2, len(jormungand[0].MyMapOfStructs.MyCustomStructSlice))
	assert.Equal(t, 2, len(jormungand[0].MyMapOfStructs.MyMapSlice))
	assert.Equal(t, "value1", *jormungand[0].MyMapOfStructs.MyMapOfStructs.MyString)
	assert.Equal(t, 1, *jormungand[0].MyMapOfStructs.MyMapOfStructs.MyInt)
	assert.Equal(t, 1.0, *jormungand[0].MyMapOfStructs.MyMapOfStructs.MyFloat)
	assert.Equal(t, true, *jormungand[0].MyMapOfStructs.MyMapOfStructs.MyBool)
	assert.Equal(t, "value1", jormungand[0].MyMapOfStructs.MyMapOfStructs.MyNode.Props["key1"])
	assert.Equal(t, "value1", jormungand[0].MyMapOfStructs.MyMapOfStructs.MyRelationship.Props["key1"])
	assert.Equal(t, "John", jormungand[0].MyMapOfStructs.MyMapOfStructs.MyCustomStruct.FirstName)
	assert.Equal(t, "Doe", jormungand[0].MyMapOfStructs.MyMapOfStructs.MyCustomStruct.LastName)
	assert.Equal(t, "john.doe@example.com", jormungand[0].MyMapOfStructs.MyMapOfStructs.MyCustomStruct.Email)
	assert.Equal(t, int64(1234567890), jormungand[0].MyMapOfStructs.MyMapOfStructs.MyCustomStruct.Phone)
	assert.Equal(t, 2, len(jormungand[0].MyMapOfStructs.MyMapOfStructs.MyStringSlice))
	assert.Equal(t, 2, len(jormungand[0].MyMapOfStructs.MyMapOfStructs.MyIntSlice))
	assert.Equal(t, 2, len(jormungand[0].MyMapOfStructs.MyMapOfStructs.MyFloatSlice))
	assert.Equal(t, 2, len(jormungand[0].MyMapOfStructs.MyMapOfStructs.MyBoolSlice))
	assert.Equal(t, 2, len(jormungand[0].MyMapOfStructs.MyMapOfStructs.MyNodeSlice))
	assert.Equal(t, 2, len(jormungand[0].MyMapOfStructs.MyMapOfStructs.MyRelationshipSlice))
	assert.Equal(t, 2, len(jormungand[0].MyMapOfStructs.MyMapOfStructs.MyCustomStructSlice))
	assert.Equal(t, 2, len(jormungand[0].MyMapOfStructs.MyMapOfStructs.MyMapSlice))
	assert.Equal(t, len(jormungand[0].MyMapOfStructsSlice), 2)
}

func TestParseResult_theBehemothAsAPointerNilResult(t *testing.T) {
	dormammu := []BehemothPointerResult{}

	berserker := neoforge.CypherQuery{
		Result: &dormammu,
	}

	berserker.ParseResult(mockNeo4jResult(
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
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			[]any{},
			[]any{},
			[]any{},
			[]any{},
			[]any{},
			[]any{},
			[]any{},
			[]any{},
			nil,
			[]any{},
		},
		1,
	))

	assert.NoError(t, berserker.Error)
	assert.Nil(t, dormammu[0].MyString)
	assert.Nil(t, dormammu[0].MyInt)
	assert.Nil(t, dormammu[0].MyFloat)
	assert.Nil(t, dormammu[0].MyBool)
	assert.Nil(t, dormammu[0].MyNode)
	assert.Nil(t, dormammu[0].MyRelationship)
	assert.Nil(t, dormammu[0].MyCustomStruct)
	assert.Nil(t, dormammu[0].MyMap)
	assert.Equal(t, 0, len(dormammu[0].MyStringSlice))
	assert.Equal(t, 0, len(dormammu[0].MyIntSlice))
	assert.Equal(t, 0, len(dormammu[0].MyFloatSlice))
	assert.Equal(t, 0, len(dormammu[0].MyBoolSlice))
	assert.Equal(t, 0, len(dormammu[0].MyNodeSlice))
	assert.Equal(t, 0, len(dormammu[0].MyRelationshipSlice))
	assert.Equal(t, 0, len(dormammu[0].MyCustomStructSlice))
	assert.Equal(t, 0, len(dormammu[0].MyMapSlice))
	assert.Nil(t, dormammu[0].MyMapOfStructs)
	assert.Equal(t, 0, len(dormammu[0].MyMapOfStructsSlice))
}
