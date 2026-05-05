package postgres

import (
	"github.com/stretchr/testify/assert"
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
}
