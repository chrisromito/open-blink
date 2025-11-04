package postgres

import (
	"context"
	"github.com/stretchr/testify/assert"
	"testing"
)

const testDatabaseURL = "postgres://postgres:postgres@localhost:5432/test_openblink"

func TestPingTestDb(t *testing.T) {
	appDb, err := NewTestAppDb()
	assert.NoError(t, err)
	assert.NotNil(t, appDb.Db)
	defer appDb.Db.Close()
	ctx := context.Background()
	err = appDb.Ping(ctx)
	assert.NoError(t, err)
	appDb.PrintStats()
}

func TestGetQueries(t *testing.T) {
	appDb, err := NewTestAppDb()
	assert.NoError(t, err)
	defer appDb.Db.Close()
	//assert.NotNil(t, pool)

	// Test GetQueries function
	queries := GetQueries(*appDb)
	assert.NotNil(t, queries)
}
