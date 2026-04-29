package postgres

import (
	"context"
	"github.com/stretchr/testify/assert"
	"log"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	appDb, err := NewTestAppDb()
	if err != nil {
		os.Exit(1)
		return
	}
	defer appDb.Db.Close()
	ctx := context.Background()
	err = CleanupTestData(ctx, appDb.GetQueries())
	if err != nil {
		log.Fatalf("error while cleanup test data, before running tests: %v", err)
	}
	code := m.Run()
	os.Exit(code)
}

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
