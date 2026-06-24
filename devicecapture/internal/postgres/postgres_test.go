package postgres

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	fmt.Println("TestMain()")
	appDb, err := NewTestAppDb()
	if err != nil {
		os.Exit(1)
		return
	}
	defer appDb.Db.Close()
	ctx := context.Background()
	err = preRun(ctx, appDb)
	if err != nil {
		os.Exit(1)
		return
	}
	fmt.Println("preRun complete, running remaining tests...")
	code := m.Run()
	fmt.Println("Test run complete, returning")
	os.Exit(code)
}

func preRun(ctx context.Context, appDb *AppDb) error {
	db := appDb.Db
	_, err := db.Exec(ctx, "DELETE FROM devices")
	if err != nil {
		return err
	}
	_, err = db.Exec(ctx, "DELETE FROM detections")
	return err
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
