package main

import (
	"context"
	"devicecapture/internal/app"
	"devicecapture/internal/camera"
	"devicecapture/internal/config"
	"devicecapture/internal/domain"
	"devicecapture/internal/domain/detection"
	"devicecapture/internal/logger"
	"devicecapture/internal/postgres"
	"devicecapture/internal/postgres/repos"
	"devicecapture/internal/pubsub"
	"encoding/json"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/google/uuid"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	conf := config.NewConfig()
	client, cerr := pubsub.BrokerHelper("go-server-"+uuid.New().String(), conf.MqttHost, conf.MqttUser, conf.MqttPassword)
	if cerr != nil {
		logger.Fatal().Err(cerr).Msgf("Error creating MQTT client: %v", cerr)
	}
	if !client.Valid() {
		logger.Fatal().Msgf("Failed to connect to a client")
	}
	defer func(client *pubsub.MqttClient) {
		_ = client.Close()
	}(&client)
	db := postgres.NewAppDb()
	dberr := db.Connect(conf.DbUrl)
	if dberr != nil {
		logger.Fatal().Msgf("Error connecting to database: %v", dberr)
	}

	defer db.Db.Close()
	//-- Repos
	queries := db.GetQueries()
	deps := domain.NewDeps(
		repos.NewPgDeviceRepo(queries),
		repos.NewPgHeartbeatRepo(queries),
		repos.NewPgDetectionRepo(queries),
		repos.NewPgImageRepo(queries),
		pubsub.NewMqttReceiver(&client, conf),
	)

	//-- App
	appCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	a := app.NewApp(conf, &client, db, deps)
	sigChan := make(chan os.Signal, 1)
	defer close(sigChan)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		cancel()
	}()

	// MQTT goroutine
	go func() {
		if err := SubscribeToStartStreamTopic(appCtx, a, &client); err != nil {
			logger.Warn().Msgf("Error in MQTT subscription: %v", err)
			cancel() // Cancel context to trigger shutdown
		}
		logger.Error().Msgf("devicecapture exiting because the start stream topic exited")
	}()

	select {
	case <-appCtx.Done():
		logger.Error().Msgf("devicecapture exiting because appCtx.Done()")
		return
	case <-sigChan:
		logger.Error().Msgf("devicecapture exiting because sigChan")
		return
	}
}

func SubscribeToStartStreamTopic(ctx context.Context, a *app.App, client *pubsub.MqttClient) error {
	// Shared camera service instance prevents duplicate camera feeds
	cameraService := camera.NewCameraService(
		a.Conf,
		a.AppDeps,
		detection.NewObjectDetectionService(a.Conf),
		a.MqttClient,
	)

	receiveMessage := func(_ mqtt.Client, m mqtt.Message) {
		logger.Debug().Str("topic", m.Topic()).
			Msg("devicecapture.SubscribeToStartStreamTopic.queueChan - Processing message")
		value := m.Payload()
		var msg app.StartStreamMessage
		if err := json.Unmarshal(value, &msg); err != nil {
			logger.Error().Str("devicecapture", "SubscribeToStartStreamTopic.queueChan").
				Msgf("failed to unmarshal message: %v", err)
			return
		}
		logger.Debug().Str("start-stream", msg.DeviceId).Msg("Starting stream")
		_, err := cameraService.StartStream(ctx, msg.DeviceId)
		if err != nil {
			logger.Error().Str("stream-fail", msg.DeviceId).
				Msgf("%v", err)
			return
		}
	}

	if err := client.Subscribe("start-stream", receiveMessage); err != nil {
		logger.Error().Msgf("Error subscribing to topic %s: %v", "start-stream", err)
		return err
	}
	if err := client.Subscribe("motion-detected", receiveMessage); err != nil {
		logger.Error().Msgf("Error subscribing to topic %s: %v", "motion-detected", err)
		return err
	}

	logger.Debug().Msg("deviceCapture -> waiting for appCtx.Done...")
	select {
	case <-ctx.Done():
		return nil
	}
}
