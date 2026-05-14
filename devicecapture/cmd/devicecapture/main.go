package main

import (
	"context"
	"devicecapture/internal/app"
	"devicecapture/internal/camera"
	"devicecapture/internal/config"
	"devicecapture/internal/domain"
	"devicecapture/internal/domain/detection"
	"devicecapture/internal/domain/devices"
	"devicecapture/internal/logger"
	"devicecapture/internal/postgres"
	"devicecapture/internal/postgres/repos"
	"devicecapture/internal/pubsub"
	"github.com/google/uuid"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
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
		repos.NewPgDetectionHistoryRepo(queries, conf),
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

	// Capture loop goroutine
	go func() {
		for {
			select {
			case <-appCtx.Done():
				return
			default:
				err := loop(appCtx, a)
				if err != nil {
					return
				}
				logger.Debug().Str("fn", "main").Msg("sleeping...")
				time.Sleep(1 * time.Minute)
			}
		}
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

func loop(ctx context.Context, a *app.App) error {
	logger.Debug().Str("fn", "main.loop").Msg("begin...")
	deviceRepo := a.AppDeps.DeviceRepo
	deviceList, rErr := deviceRepo.ListDevices(ctx)
	if rErr != nil {
		return rErr
	}
	cs := camera.NewCameraService(
		a.Conf,
		a.AppDeps,
		detection.NewObjectDetectionService(a.Conf),
		a.MqttClient,
	)
	var wg sync.WaitGroup
	// Call "Snapshot" for each device
	for _, device := range deviceList {
		wg.Add(1)
		go func(d devices.Device) {
			defer wg.Done()
			logger.Info().Str("fn", "main.loop").
				Msgf("getting snapshot from device %d", device.ID)
			err := cs.Snapshot(ctx, device)
			if err != nil {
				logger.Error().Str("fn", "main.loop").
					Msgf("error %v", err)
			}
		}(device)
	}
	// Wait until we grab images and detections for all devices
	wg.Wait()
	return nil
}
