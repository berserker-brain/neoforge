package neoforge

import (
	"errors"
	"log"
	"context"
	"strings"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

type CypherRepository struct {
	Driver   neo4j.DriverWithContext
	Ctx      context.Context
	Database string
}

func (cr *CypherRepository) RunQuery(cypher *CypherQuery) {
	result, err := neo4j.ExecuteQuery(
		cr.Ctx, cr.Driver,
		cypher.Query,
		cypher.Params,
		neo4j.EagerResultTransformer,
		neo4j.ExecuteQueryWithDatabase(cr.Database),
	)

	cr.parseNeo4jResults(result, err, cypher)
}

func (cr *CypherRepository) RunReadTransaction(readTransaction *CypherTransaction) error {
	session := cr.Driver.NewSession(cr.Ctx, neo4j.SessionConfig{
		DatabaseName: cr.Database,
	})
	defer session.Close(cr.Ctx)

	_, err := session.ExecuteRead(cr.Ctx, cr.basicTransaction(readTransaction))

	if err != nil && readTransaction.OnRollback != nil {
		readTransaction.OnRollback()
		return err
	}

	if readTransaction.OnCommit != nil {
		readTransaction.OnCommit()
	}
	return session.Close(cr.Ctx)
}

func (cr *CypherRepository) RunWriteTransaction(writeTransaction *CypherTransaction) error {
	session := cr.Driver.NewSession(cr.Ctx, neo4j.SessionConfig{
		DatabaseName: cr.Database,
	})
	defer session.Close(cr.Ctx)

	_, err := session.ExecuteWrite(cr.Ctx, cr.basicTransaction(writeTransaction))

	if err != nil && writeTransaction.OnRollback != nil {
		writeTransaction.OnRollback()
		return err
	}

	if writeTransaction.OnCommit != nil {
		writeTransaction.OnCommit()
	}
	return session.Close(cr.Ctx)
}

func (cr *CypherRepository) basicTransaction(transaction *CypherTransaction) func(tx neo4j.ManagedTransaction) (any, error) {
	return func(tx neo4j.ManagedTransaction) (any, error) {
		for _, cypher := range transaction.Queries {
			result, err := tx.Run(cr.Ctx, cypher.Query, cypher.Params)
			if err != nil {
				cypher.Error = cr.parseNeo4jError(err)
				return nil, err
			}

			keys, err := result.Keys()
			if err != nil {
				cypher.Error = cr.parseNeo4jError(err)
				return nil, err
			}

			records, err := result.Collect(cr.Ctx)
			if err != nil {
				cypher.Error = cr.parseNeo4jError(err)
				return nil, err
			}

			summary, err := result.Consume(cr.Ctx)
			if err != nil {
				cypher.Error = cr.parseNeo4jError(err)
				return nil, err
			}

			cr.parseNeo4jResults(
				&neo4j.EagerResult{
					Keys:    keys,
					Records: records,
					Summary: summary,
				}, err, cypher)

			if cypher.Error != nil {
				return nil, cypher.Error
			}
		}
		return nil, nil
	}
}

func (cr *CypherRepository) parseNeo4jError(err error) error {
	if err == nil {
		return nil
	}

	if strings.Contains(err.Error(), "Neo.ClientError.Statement.SyntaxError") {
		log.Println(err.Error())
		return errors.New("syntax error: " + err.Error())
	}

	if strings.Contains(err.Error(), "Neo.ClientError.Schema.ConstraintValidationFailed") {
		log.Println(err.Error())
		return errors.New("constraints failed: " + err.Error())
	}

	if strings.Contains(err.Error(), "Neo.ClientError.Statement.ParameterMissing") {
		log.Println(err.Error())
		return errors.New("missing parameters: " + err.Error())
	}

	return err
}

func (cr *CypherRepository) parseNeo4jResults(result *neo4j.EagerResult, err error, cypher *CypherQuery) {
	cypher.Error = cr.parseNeo4jError(err)
	if cypher.Error != nil {
		return
	}

	cypher.ParseResult(result)

	cypher.Stats = &Stats{}
	cypher.Stats.FromResultSummary(result.Summary)
}
