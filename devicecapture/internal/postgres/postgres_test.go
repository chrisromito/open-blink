package postgres

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestPingTestDb(t *testing.T) {
	a := assert.New(t)
	appDb, err := NewTestAppDb()
	a.NoError(err)
	a.NotNil(appDb.Db)
	defer appDb.Db.Close()
	ctx := t.Context()
	err = appDb.Ping(ctx)
	a.NoError(err)
	cleanupErr := CleanupTestData(ctx, appDb.GetQueries())
	a.NoError(cleanupErr)
	appDb.PrintStats()
}

func TestGetQueries(t *testing.T) {
	a := assert.New(t)
	appDb, err := NewTestAppDb()
	a.NoError(err)
	defer appDb.Db.Close()
	// Test GetQueries function
	queries := GetQueries(*appDb)
	a.NotNil(queries)
}
