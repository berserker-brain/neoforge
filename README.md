# neoforge

A small Go package on top of [neo4j-go-driver/v5](https://github.com/neo4j/neo4j-go-driver-go) that runs Cypher against Neo4j, maps rows into structs via `key` tags, and exposes read/write transactions with optional callbacks. It also provides helpers to turn `neo4j.Node` and `neo4j.Relationship` values into structs using JSON field tags.

## Install

```bash
go get github.com/berserker-brain/neoforge
```

Requires Go 1.23+.

## Configuration

### Environment variables (`NewCypherRepository`)

`NewCypherRepository()` builds a [`CypherRepository`](repository.go) from the process environment. All of the following must be set or it panics:

| Variable | Purpose |
|----------|---------|
| `NEO4J_URI` | Bolt URI (e.g. `neo4j://localhost:7687`) |
| `NEO4J_USERNAME` | Neo4j user |
| `NEO4J_PASSWORD` | Neo4j password |
| `NEO4J_DATABASE` | Database name |

### Manual setup (`NewDriver`)

For tests or custom wiring, use `NewDriver(uri, username, password)`, which returns a `neo4j.DriverWithContext` and a `context.Context`, verifies connectivity, and applies sensible pool/timeouts (see [`config.go`](config.go)). Then build a `CypherRepository` yourself:

```go
driver, ctx := neoforge.NewDriver(uri, user, pass)
defer driver.Close(ctx)

repo := neoforge.CypherRepository{
    Driver:   driver,
    Ctx:      ctx,
    Database: "neo4j", // or your database name
}
```

## Running queries

### `RunQuery`

`RunQuery` executes a single statement with `neo4j.ExecuteQuery` (eager results). It fills `query.Error`, optionally parses into `query.Result`, and populates `query.Stats` from the result summary.

```go
var rows []struct {
    N neo4j.Node `key:"n"`
}

q := neoforge.CypherQuery{
    Query:   "MATCH (n:Node) RETURN n LIMIT 10",
    Result:  &rows,
    EmptyOk: true, // no error if the query returns zero rows
}
repo.RunQuery(&q)
if q.Error != nil {
    // handle
}
// rows is filled; q.Stats has counters / timings
```

### Domain structs in `Result` (e.g. `User`)

The same struct you would pass to `ParseNode`—with `json:"..."` for properties and optional `db:"labels"` on nodes—can appear as a field on the **row struct**. The field’s `key` tag must match the **RETURN** alias; the library decodes a returned node (or relationship) into that type the same way as `ParseNode` / `ParseRelationship`.

```go
type UserNode struct {
    Labels    []string `db:"labels"`
    FirstName string   `json:"first_name"`
    LastName  string   `json:"last_name"`
    Email     string   `json:"email"`
    Phone     int64    `json:"phone"`
}

var rows []struct {
    U UserNode `key:"u"` // matches "RETURN u"
}

q := neoforge.CypherQuery{
    Query:   "MATCH (u:User) RETURN u LIMIT 10",
    Result:  &rows,
    EmptyOk: true,
}
repo.RunQuery(&q)
// rows[i].U is filled per record
```

You can mix primitives, `neo4j.Node`, and custom types in the same row struct; each column needs its own `key:"<alias>"`.

- If `Result` is `nil`, nothing is unmarshalled (useful for write-only or `MATCH` checks).
- If `Result` is set, it must be a **pointer to a slice of structs**. Each field that should map from a RETURN column needs a struct tag `key:"<column_name>"` matching the name in the Cypher result.
- If you expect zero rows and that is valid, set **`EmptyOk: true`**. Otherwise an empty result sets an error.

### `CypherQuery` fields

| Field | Meaning |
|-------|---------|
| `Query` | Cypher string |
| `Params` | `map[string]any` for `$parameters` |
| `Result` | `*[]YourStruct` — optional |
| `EmptyOk` | Allow zero records when `Result` is set |
| `Error` | Set by the library on failure |
| `Stats` | Filled after a successful run (see [`stats.go`](stats.go)) |

### Nullable columns

Use `key:"name,omitempty"` on a field if the column may be missing; the field stays at its zero value when absent.

## Mapping rules (result structs)

- Top-level `Result` must be a pointer to a slice of structs (one slice element per record).
- Every mapped field needs `key:"..."` aligned with RETURN aliases.
- Values can be primitives, `neo4j.Node`, `neo4j.Relationship`, nested structs, slices, and maps; nodes/relationships with custom types use JSON tags on nested structs (see tests in [`cypher_test.go`](cypher_test.go)).
- Custom node/relationship types in query results are decoded similarly to `ParseNode` / `ParseRelationship` (JSON properties plus optional `db` tags for labels — see below).

## Transactions

### `RunReadTransaction` / `RunWriteTransaction`

Pass a [`CypherTransaction`](cypher.go) with one or more `*CypherQuery` entries. They run in order inside a managed read or write transaction. Optional `OnCommit` and `OnRollback` run after success or failure.

```go
tx := neoforge.CypherTransaction{
    Queries: []*neoforge.CypherQuery{
        {Query: "MATCH (n:Node) RETURN n", Result: &rows, EmptyOk: true},
    },
    OnCommit: func() { /* ... */ },
}
err := repo.RunReadTransaction(&tx)
```

Read transactions cannot perform writes; a write in a read transaction surfaces as an error on the relevant query and triggers `OnRollback` if set.

## `ParseNode` and `ParseRelationship`

When you already have a `neo4j.Node` or `neo4j.Relationship` (for example from a lower-level API), you can decode properties into a struct with JSON tags:

```go
type User struct {
    Labels    []string `db:"labels"` // filled from node labels
    FirstName string   `json:"first_name"`
    // ...
}

u, err := neoforge.ParseNode[User](node)
```

- Use `json:"..."` tags for property names.
- On nodes, `db:"labels"` (or a tag containing `"labels"`) fills the field with `node.Labels`.
- On relationships, `db:"label"` (or containing `"label"`) fills the field with the relationship type.

## Error handling

Neo4j client errors are surfaced on `CypherQuery.Error`. Some messages are normalized to shorter errors (syntax, constraint, missing parameters) in [`repository.go`](repository.go).

## Testing this package

Integration tests expect a live Neo4j instance. Load credentials with `.env` (see [`github.com/joho/godotenv`](https://github.com/joho/godotenv)) or export:

| Variable | Notes |
|----------|--------|
| `NEO4J_URI` | Bolt URI |
| `NEO4J_USERNAME` | User |
| `NEO4J_PASSWORD` | Password |
| `NEO4J_TEST_DATABASE` | Optional; defaults to `neo4j` |

Tests use `TestMain` in [`repository_test.go`](repository_test.go) to open a driver and share a `CypherRepository`. Run:

```bash
go test ./...
```
