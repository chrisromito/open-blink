package repos

import (
	"context"
	"os"
	"testing"

	"devicecapture/internal/logger"
	"devicecapture/internal/postgres"
)

func TestMain(m *testing.M) {
	logger.Info().Msg("repos.TestMain()")
	appDb, err := postgres.NewTestAppDb()
	if err != nil {
		os.Exit(1)
		return
	}
	defer appDb.Db.Close()
	ctx := context.Background()
	_, err = appDb.Db.Exec(ctx, "DELETE FROM devices")
	if err != nil {
		os.Exit(1)
	}
	_, err = appDb.Db.Exec(ctx, "DELETE FROM detections")
	if err != nil {
		os.Exit(1)
	}
	logger.Info().Msg("preRun complete, running remaining tests...")
	code := m.Run()
	logger.Info().Msg("Test run complete, returning")
	os.Exit(code)
}
