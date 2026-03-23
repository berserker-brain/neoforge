package neoforge_test

import (
	"log"
	"os"
	"testing"

	"github.com/berserker-brain/neoforge"
	"github.com/joho/godotenv"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/stretchr/testify/assert"
)

var db neoforge.CypherRepository

func TestMain(m *testing.M) {
	_ = godotenv.Load()

	testDbName := os.Getenv("NEO4J_TEST_DATABASE")
	if testDbName == "" {
		testDbName = "neo4j"
	}

	driver, ctx := neoforge.NewDriver(
		os.Getenv("NEO4J_URI"),
		os.Getenv("NEO4J_USERNAME"),
		os.Getenv("NEO4J_PASSWORD"),
	)
	defer func() {
		if err := driver.Close(ctx); err != nil {
			log.Printf("driver close: %v", err)
		}
	}()

	db = neoforge.CypherRepository{
		Driver:   driver,
		Ctx:      ctx,
		Database: testDbName,
	}

	os.Exit(m.Run())
}

// wipeTestDatabase removes all nodes and relationships so integration tests start from a known empty state.
func wipeTestDatabase(t *testing.T) {
	t.Helper()
	q := neoforge.CypherQuery{
		Query: "MATCH (n) DETACH DELETE n",
	}
	db.RunQuery(&q)
	assert.NoError(t, q.Error)
}

func TestRunQuery(t *testing.T) {
	wipeTestDatabase(t)

	query := neoforge.CypherQuery{
		Query: "MATCH (n) RETURN n",
	}
	db.RunQuery(&query)

	assert.NoError(t, query.Error)
	assert.Nil(t, query.Result)

	res := []struct {
		N neo4j.Node `key:"n"`
	}{}

	query = neoforge.CypherQuery{
		Query:   "MATCH (n) RETURN n",
		Result:  &res,
		EmptyOk: false,
	}
	db.RunQuery(&query)

	assert.Error(t, query.Error)

	query = neoforge.CypherQuery{
		Query: "CREATE (n:Node {id: randomUUID(), name: $name}) RETURN n",
		Params: map[string]any{
			"name": "test",
		},
		Result: &res,
	}
	db.RunQuery(&query)

	assert.NoError(t, query.Error)
	assert.NotNil(t, query.Result)
	assert.Equal(t, res[0].N.Props["name"], "test")
	assert.Equal(t, 1, query.Stats.NodesCreated)
	assert.Equal(t, 2, query.Stats.PropertiesSet)
}

func TestRunReadTransaction(t *testing.T) {
	wipeTestDatabase(t)

	commitCount := 0
	rollbackCount := 0

	transaction := neoforge.CypherTransaction{
		Queries: []*neoforge.CypherQuery{
			{
				Query:   "MATCH (n:Node) RETURN n",
				EmptyOk: true,
			},
		},
		OnCommit: func() {
			commitCount++
		},
	}
	db.RunReadTransaction(&transaction)

	assert.NoError(t, transaction.Queries[0].Error)
	assert.Equal(t, 1, commitCount)

	res := []struct {
		N neo4j.Node `key:"n"`
	}{}
	query := neoforge.CypherQuery{
		Query: "CREATE (n:Node {id: randomUUID(), name: $name}) RETURN n",
		Params: map[string]any{
			"name": "test",
		},
		Result: &res,
	}
	db.RunQuery(&query)

	assert.NoError(t, query.Error)
	assert.NotNil(t, query.Result)
	assert.Equal(t, res[0].N.Props["name"], "test")

	transaction = neoforge.CypherTransaction{
		Queries: []*neoforge.CypherQuery{
			{
				Query:   "MATCH (n:Node) RETURN n",
				EmptyOk: false,
				Result:  &res,
			},
		},
		OnCommit: func() {
			commitCount++
		},
	}
	db.RunReadTransaction(&transaction)

	assert.NoError(t, transaction.Queries[0].Error)
	assert.NotNil(t, transaction.Queries[0].Result)
	assert.Equal(t, res[0].N.Props["name"], "test")
	assert.Equal(t, 2, commitCount)

	transaction = neoforge.CypherTransaction{
		Queries: []*neoforge.CypherQuery{
			{
				Query: "CREATE (n:ShouldFail {id: randomUUID(), name: $name}) RETURN n",
				Params: map[string]any{
					"name": "test",
				},
				Result: &res,
			},
		},
		OnCommit: func() {
			commitCount++
		},
		OnRollback: func() {
			rollbackCount++
		},
	}
	db.RunReadTransaction(&transaction)

	assert.Error(t, transaction.Queries[0].Error)
	assert.Equal(t, 1, rollbackCount)
}

func TestRunWriteTransaction(t *testing.T) {
	wipeTestDatabase(t)

	commitCount := 0
	rollbackCount := 0

	res := []struct {
		N neo4j.Node `key:"n"`
	}{}
	transaction := neoforge.CypherTransaction{
		Queries: []*neoforge.CypherQuery{
			{
				Query: "CREATE (n:Node {id: randomUUID(), name: $name}) RETURN n",
				Params: map[string]any{
					"name": "test",
				},
				Result: &res,
			},
		},
		OnCommit: func() {
			commitCount++
		},
		OnRollback: func() {
			rollbackCount++
		},
	}
	db.RunWriteTransaction(&transaction)

	assert.NoError(t, transaction.Queries[0].Error)
	assert.NotNil(t, transaction.Queries[0].Result)
	assert.Equal(t, res[0].N.Props["name"], "test")
	assert.Equal(t, 1, commitCount)
	assert.Equal(t, 0, rollbackCount)

	res = []struct {
		N neo4j.Node `key:"n"`
	}{}
	transaction = neoforge.CypherTransaction{
		Queries: []*neoforge.CypherQuery{
			{
				Query:  "CREATE (n:ShouldFail {id: randomUUID(), name: $name}) RETURN n",
				Result: &res,
			},
		},
		OnCommit: func() {
			commitCount++
		},
		OnRollback: func() {
			rollbackCount++
		},
	}
	db.RunWriteTransaction(&transaction)

	assert.Error(t, transaction.Queries[0].Error) //missing params
	assert.Equal(t, 1, rollbackCount)

	res = []struct {
		N neo4j.Node `key:"n"`
	}{}
	transaction = neoforge.CypherTransaction{
		Queries: []*neoforge.CypherQuery{
			{
				Query: "CREATE (n:Node {id: randomUUID(), name: $name}) RETURN n",
				Params: map[string]any{
					"name": "test",
				},
				Result: &res,
			},
			{
				Query:  "MATCH (n:Node) RETURN n",
				Result: &res,
			},
		},
		OnCommit: func() {
			commitCount++
		},
		OnRollback: func() {
			rollbackCount++
		},
	}
	db.RunWriteTransaction(&transaction)

	assert.NoError(t, transaction.Queries[0].Error)
	assert.NotNil(t, transaction.Queries[0].Result)
	assert.Equal(t, res[0].N.Props["name"], "test")
	assert.Equal(t, res[1].N.Props["name"], "test")
	assert.Equal(t, 2, commitCount)
}
