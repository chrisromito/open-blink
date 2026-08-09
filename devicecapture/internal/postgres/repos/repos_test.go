package repos

import (
	"fmt"
	"os"
	"testing"

	"devicecapture/internal/logger"
	"devicecapture/internal/postgres"
)

func TestMain(m *testing.M) {
	logger.Info().Msg("repos.TestMain()")
	appDb, err := postgres.NewTestAppDb()
	if err != nil {
		fmt.Printf("preRun failed %s", err)
		os.Exit(1)
		return
	}
	defer appDb.Db.Close()

	logger.Info().Msg("repos_test preRun complete, running remaining tests...")
	code := m.Run()
	//ctx := context.Background()
	//_, err = appDb.Db.Exec(ctx, "DELETE FROM detections")
	//if err != nil {
	//	os.Exit(1)
	//	return
	//}
	//_, err = appDb.Db.Exec(ctx, "DELETE FROM devices")
	//if err != nil {
	//	os.Exit(1)
	//	return
	//}
	logger.Info().Msg("Test run complete, returning")
	os.Exit(code)
}
