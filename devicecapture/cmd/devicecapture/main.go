package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"devicecapture/internal/app"
	"devicecapture/internal/archive"
	"devicecapture/internal/camera"
	"devicecapture/internal/config"
	"devicecapture/internal/domain"
	"devicecapture/internal/domain/detection"
	"devicecapture/internal/domain/devices"
	"devicecapture/internal/logger"
	"devicecapture/internal/postgres"
	"devicecapture/internal/postgres/repos"
	"devicecapture/internal/postgres/repos/event"
	"devicecapture/internal/pubsub"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"
)

func main() {
	conf := config.NewConfig()
	client, cerr := pubsub.BrokerHelper(
		"go-server-"+uuid.New().String(),
		conf.MqttHost,
		conf.MqttUser,
		conf.MqttPassword,
	)
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
		event.NewPgDetectionEventRepo(queries, conf),
	)

	//-- App
	appCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	a := app.NewApp(conf, &client, db, deps)
	sigChan := make(chan os.Signal, 1)
	defer close(sigChan)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Monitor the sigChan, cancel the app context once we receive a shutdown signal
	go func() {
		<-sigChan
		cancel()
	}()

	// Capture loop goroutine
	go func() {
		err := run(appCtx, a)
		if err != nil {
			logger.Error().Str("devicecapture", "main").Err(err).
				Msg("run threw")
		}
		cancel()
		panic("capture loop exited")
	}()

	// Pre-run
	go func() {
		err := preRunHook(appCtx, a)
		if err != nil {
			logger.Error().
				Str("devicecapture", "preRun").
				Err(err).
				Send()
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

func preRunHook(ctx context.Context, a *app.App) error {
	queries := a.Db.GetQueries()
	return queries.EndStaleDetectionEvents(ctx)
}

func run(ctx context.Context, a *app.App) error {
	msgChan := make(chan mqtt.Message, 1)
	defer close(msgChan)

	motionHandler := func(client mqtt.Client, message mqtt.Message) {
		msgChan <- message
	}

	qtErr := a.MqttClient.Subscribe("motion-detected/#", motionHandler)
	if qtErr != nil {
		return qtErr
	}
	archiveTicker := time.NewTicker(4 * time.Hour)
	snapshotTicker := time.NewTicker(30 * time.Second)
	cs := camera.NewCameraService(
		a.Conf,
		a.AppDeps,
		detection.NewObjectDetectionService(a.Conf),
		a.MqttClient,
	)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case msg := <-msgChan:
			logger.Debug().Str("fn", "run").
				Msgf("capturing streams for devices due to topic: %v, & message %v", msg.Topic(), msg.Payload())
			err := loopDevices(ctx, a, cs, true)
			if err != nil {
				logger.Error().
					Str("fn", "main").
					Str("target", "loopDevices").
					Bool("motionDetected", true).
					Err(err).
					Send()
				return err
			}
			logger.Debug().Str("fn", "run").
				Msg("captured streams, continuing loop")
		case <-archiveTicker.C:
			// Archive old images every 4 hours
			logger.Debug().Str("fn", "run").
				Msg("devicecapture is archiving old images")
			arch := archive.NewArchivist(a.Conf, a.Db.GetQueries())
			archiveErr := arch.Run(ctx)
			if archiveErr != nil {
				logger.Error().Str("Archivist", "Run").
					Err(archiveErr).
					Send()
			}
		case <-snapshotTicker.C:
			err := loopDevices(ctx, a, cs, false)
			if err != nil {
				logger.Error().
					Str("fn", "main").
					Str("target", "loopDevices").
					Bool("motionDetected", false).
					Err(err).Send()
				return err
			}
			logger.Debug().Str("fn", "main").
				Msg("sleeping...")
			//time.Sleep(30 * time.Second)
		}
	}
}

func loopDevices(
	ctx context.Context,
	a *app.App,
	cs *camera.CameraService,
	motionDetected bool,
) error {
	logger.Debug().Str("fn", "main.loop").Msg("begin...")
	deviceRepo := a.AppDeps.DeviceRepo
	deviceList, rErr := deviceRepo.ListDevices(ctx)
	if rErr != nil {
		return rErr
	}
	if motionDetected {
		return captureStreams(ctx, deviceList, cs)
	}
	return captureSnapshots(ctx, deviceList, cs)
}

func captureSnapshots(ctx context.Context, ds []devices.Device, cs *camera.CameraService) error {
	g, c := errgroup.WithContext(ctx)
	for _, device := range ds {
		g.Go(func() error {
			err := cs.Snapshot(c, device)
			if err != nil {
				logger.Error().Str("fn", "main.captureSnapshots").
					Err(err).Send()
			}
			return err
		})
	}
	return g.Wait()
}

func captureStreams(ctx context.Context, ds []devices.Device, cs *camera.CameraService) error {
	g, c := errgroup.WithContext(ctx)
	for _, device := range ds {
		g.Go(func() error {
			err := cs.StreamSnapshots(c, device, 15)
			if err != nil {
				logger.Error().Str("fn", "main.captureSnapshots").
					Err(err).Send()
			}
			return err
		})
	}
	return g.Wait()
}
