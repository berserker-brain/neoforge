package neoforge

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j/config"
)

type Neo4jConfig struct {
	URI      string
	Username string
	Password string
	Database string
}

type Neo4j struct {
	Driver   neo4j.DriverWithContext
	Ctx      context.Context
	Database string
}

func NewNeo4jConfig() *Neo4jConfig {
	return &Neo4jConfig{
		URI:      os.Getenv("NEO4J_URI"),
		Username: os.Getenv("NEO4J_USERNAME"),
		Password: os.Getenv("NEO4J_PASSWORD"),
	}
}

func NewNeo4j() *Neo4j {
	driver, ctx := NewNeo4jConfig().NewDriver()

	return &Neo4j{
		Driver:   driver,
		Ctx:      ctx,
		Database: os.Getenv("NEO4J_DATABASE"),
	}
}

func (c *Neo4jConfig) NewDriver() (neo4j.DriverWithContext, context.Context) {
	ctx := context.Background()
	config := func(conf *config.Config) {
		conf.MaxConnectionLifetime = 60 * time.Minute
		conf.MaxConnectionPoolSize = 50
		conf.ConnectionAcquisitionTimeout = 5 * time.Second
		conf.SocketConnectTimeout = 5 * time.Second
		conf.SocketKeepalive = true
	}

	driver, err := neo4j.NewDriverWithContext(c.URI, neo4j.BasicAuth(c.Username, c.Password, ""), config)
	if err != nil {
		panic(err)
	}

	err = verifyConnectivity(driver, ctx)

	if err != nil {
		fmt.Println(err)
		// panic("Unable to connect to database after 5 tries")
	}

	return driver, ctx
}

func verifyConnectivity(driver neo4j.DriverWithContext, ctx context.Context) error {
	err := driver.VerifyConnectivity(ctx)

	// if err != nil && retriesLeft > 0 {
	// 	retriesLeft--
	// 	return verifyConnectivity(driver, ctx, retriesLeft)
	// }

	return err
}
