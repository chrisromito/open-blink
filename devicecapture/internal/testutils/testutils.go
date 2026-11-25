package testutils

import (
	"devicecapture/internal/config"
	"devicecapture/internal/domain"
	"devicecapture/internal/postgres"
	"devicecapture/internal/postgres/repos"
	"devicecapture/internal/postgres/repos/event"
	"devicecapture/internal/pubsub"
)

// NewTestDeps get Deps with the testDataBaseUrl pre-injected
func NewTestDeps(videoPath string, archivePath string) (*domain.Deps, *postgres.AppDb, error) {
	conf := config.GetTestConfig(videoPath, archivePath)
	db, err := postgres.NewTestAppDb()
	if err != nil {
		return &domain.Deps{}, &postgres.AppDb{}, err
	}
	queries := db.GetQueries()
	//client := &pubsub.MqttClient{}
	fakeClient := pubsub.NewMockClient()
	client := pubsub.NewMqttClient(
		fakeClient,
		pubsub.NewMqttClientOptions(conf, "test-mqtt-client"),
	)
	deps := domain.NewDeps(
		repos.NewPgDeviceRepo(queries),
		repos.NewPgHeartbeatRepo(queries),
		repos.NewPgDetectionRepo(queries),
		repos.NewPgImageRepo(queries),
		pubsub.NewMqttPublisher(client, conf),
		repos.NewPgDetectionHistoryRepo(queries, conf),
		event.NewPgDetectionEventRepo(queries, conf),
	)
	return deps, db, nil
}
