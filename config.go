package neoforge

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j/config"
)

/// NewCypherRepository is a quick constructor for setting up a neo4j connection
func NewCypherRepository() *CypherRepository {
	uri := os.Getenv("NEO4J_URI")
	username := os.Getenv("NEO4J_USERNAME")
	password := os.Getenv("NEO4J_PASSWORD")
	database := os.Getenv("NEO4J_DATABASE")
	if uri == "" || username == "" || password == "" || database == "" {
		panic("NEO4J_URI, NEO4J_USERNAME, NEO4J_PASSWORD, and NEO4J_DATABASE must be set")
	}

	driver, ctx := NewDriver(uri, username, password)

	return &CypherRepository{
		Driver:   driver,
		Ctx:      ctx,
		Database: database,
	}
}

func NewDriver(uri string, username string, password string) (neo4j.DriverWithContext, context.Context) {
	ctx := context.Background()
	config := func(conf *config.Config) {
		conf.MaxConnectionLifetime = 60 * time.Minute
		conf.MaxConnectionPoolSize = 50
		conf.ConnectionAcquisitionTimeout = 5 * time.Second
		conf.SocketConnectTimeout = 5 * time.Second
		conf.SocketKeepalive = true
	}

	driver, err := neo4j.NewDriverWithContext(uri, neo4j.BasicAuth(username, password, ""), config)
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
