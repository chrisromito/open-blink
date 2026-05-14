package app

import (
	"devicecapture/internal/config"
	"devicecapture/internal/domain"
	"devicecapture/internal/postgres"
	"devicecapture/internal/pubsub"
)

type StartStreamMessage struct {
	DeviceId string `json:"device_id"`
}

type App struct {
	Conf       *config.Config
	MqttClient *pubsub.MqttClient
	Db         *postgres.AppDb
	AppDeps    *domain.Deps
}

// NewApp create an App, under the assumption that the MqttClient & AppDb are initialized/connected
func NewApp(conf *config.Config, mqttClient *pubsub.MqttClient, db *postgres.AppDb, deps *domain.Deps) *App {
	return &App{
		Conf:       conf,
		MqttClient: mqttClient,
		Db:         db,
		AppDeps:    deps,
	}
}
