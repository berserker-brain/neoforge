package neoforge_test

import (
	"testing"
	
	"github.com/berserker-brain/neoforge"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/stretchr/testify/assert"
)

func TestParseNode_parsesJsonStructCorrectly(t *testing.T) {
	user, err := neoforge.ParseNode[SomeUser](neo4j.Node{
		Labels: []string{"User"},
		Props: map[string]any{
			"first_name": "John",
			"last_name":  "Doe",
			"email":      "john.doe@example.com",
			"phone":      1234567890,
		},
	})

	assert.NoError(t, err)
	assert.Equal(t, "John", user.FirstName)
	assert.Equal(t, "Doe", user.LastName)
	assert.Equal(t, "john.doe@example.com", user.Email)
	assert.Equal(t, int64(1234567890), user.Phone)
	assert.Equal(t, []string{"User"}, user.Labels)
}

func TestParseRelationship_parsesJsonStructCorrectly(t *testing.T) {
	relationship, err := neoforge.ParseRelationship[SomeRelationship](neo4j.Relationship{
		Type: "User",
		Props: map[string]any{
			"first_name": "John",
		},
	})

	assert.NoError(t, err)
	assert.Equal(t, "John", relationship.FirstName)
	assert.Equal(t, "User", relationship.Label)
}