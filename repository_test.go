package neoforge_test

import (
	"log"
	"os"
	"testing"

	"github.com/berserker-brain/neoforge"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/stretchr/testify/assert"
)

var db neoforge.Neo4j

func TestMain(m *testing.M) {
	testDbName := os.Getenv("NEO4J_TEST_DATABASE")
	if testDbName == "" {
		testDbName = "neo4j"
	}

	cfg := neoforge.NewNeo4jConfig()
	if cfg.URI == "" || cfg.Username == "" {
		log.Fatal("integration tests require NEO4J_URI and NEO4J_USERNAME (and usually NEO4J_PASSWORD)")
	}

	driver, ctx := cfg.NewDriver()
	defer func() {
		if err := driver.Close(ctx); err != nil {
			log.Printf("driver close: %v", err)
		}
	}()

	db = neoforge.Neo4j{
		Driver:   driver,
		Ctx:      ctx,
		Database: testDbName,
	}

	os.Exit(m.Run())
}

func TestRunQuery(t *testing.T) {
	repo := neoforge.NewCypherRepository(&db)

	query := neoforge.CypherQuery{
		Query: "MATCH (n) RETURN n",
	}
	repo.RunQuery(&query)

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
	repo.RunQuery(&query)

	assert.Error(t, query.Error)

	query = neoforge.CypherQuery{
		Query: "CREATE (n:Node {id: randomUUID(), name: $name}) RETURN n",
		Params: map[string]any{
			"name": "test",
		},
		Result: &res,
	}
	repo.RunQuery(&query)

	assert.NoError(t, query.Error)
	assert.NotNil(t, query.Result)
	assert.Equal(t, res[0].N.Props["name"], "test")
	assert.Equal(t, 1, query.Stats.NodesCreated)
	assert.Equal(t, 2, query.Stats.PropertiesSet)
}

func TestRunReadTransaction(t *testing.T) {
	repo := neoforge.NewCypherRepository(&db)
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
	repo.RunReadTransaction(&transaction)

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
	repo.RunQuery(&query)

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
	repo.RunReadTransaction(&transaction)

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
	repo.RunReadTransaction(&transaction)

	assert.Error(t, transaction.Queries[0].Error)
	assert.Equal(t, 1, rollbackCount)
}

func TestRunWriteTransaction(t *testing.T) {
	repo := neoforge.NewCypherRepository(&db)
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
	repo.RunWriteTransaction(&transaction)

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
	repo.RunWriteTransaction(&transaction)

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
	repo.RunWriteTransaction(&transaction)

	assert.NoError(t, transaction.Queries[0].Error)
	assert.NotNil(t, transaction.Queries[0].Result)
	assert.Equal(t, res[0].N.Props["name"], "test")
	assert.Equal(t, res[1].N.Props["name"], "test")
	assert.Equal(t, 2, commitCount)
}
