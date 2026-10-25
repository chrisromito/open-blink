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
	"devicecapture/internal/domain/queue"
	"devicecapture/internal/logger"
	"devicecapture/internal/postgres"
	"devicecapture/internal/postgres/repos"
	"devicecapture/internal/postgres/repos/event"
	"devicecapture/internal/pubsub"
	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"
)

func main() {
	conf := config.NewConfig()
	qtOpts := pubsub.NewMqttClientOptions(conf, "go-server-"+uuid.New().String())
	client, cerr := pubsub.NewConnectedMqttClient(qtOpts)
	if cerr != nil {
		logger.Fatal().Err(cerr).Msgf("Error creating MQTT client: %v", cerr)
	}
	defer client.Disconnect(250)
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
		pubsub.NewMqttPublisher(client, conf),
		repos.NewPgDetectionHistoryRepo(queries, conf),
		event.NewPgDetectionEventRepo(queries, conf),
	)

	//-- App
	appCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	a := app.NewApp(conf, client, db, deps)
	err := preRunHook(appCtx, a)
	if err != nil {
		logger.Error().
			Str("devicecapture", "preRun").
			Err(err).
			Send()
		panic(err)
	}
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

func runReceiver(ctx context.Context, a *app.App, publisher queue.Publisher) error {
	deps := a.AppDeps
	receiver := pubsub.NewMqttReceiver(deps, a.Conf, a.MqttClient, publisher)
	return receiver.Run(ctx)
}

func run(ctx context.Context, a *app.App) error {
	archiveTicker := time.NewTicker(4 * time.Hour)
	defer archiveTicker.Stop()

	snapshotTicker := time.NewTicker(15 * time.Second)
	defer snapshotTicker.Stop()

	cs := camera.NewCameraService(
		a.Conf,
		a.AppDeps,
		detection.NewObjectDetectionService(a.Conf),
	)

	publisher := pubsub.NewMqttPublisher(a.MqttClient, a.Conf)
	errChan := make(chan error)
	defer close(errChan)

	go func() {
		errChan <- runReceiver(ctx, a, publisher)
	}()

	for {
		select {
		case err := <-errChan:
			return err
		case <-ctx.Done():
			return ctx.Err()
		case <-archiveTicker.C:
			logger.Debug().Str("fn", "run").
				Msg("devicecapture is archiving old images")

			arch := archive.NewArchivist(a.Conf, a.Db.GetQueries())
			aErr := arch.Run(ctx)
			if aErr != nil {
				errChan <- aErr
			}
		case <-snapshotTicker.C:
			loopErr := loopDevices(ctx, a, cs)
			if loopErr != nil {
				errChan <- loopErr
			}
			logger.Debug().Str("fn", "main").
				Msg("Successfully loopedDevices w/o motion")
		}
	}
}

func loopDevices(
	ctx context.Context,
	a *app.App,
	cs *camera.CameraService,
) error {
	logger.Debug().Str("fn", "main.loop").Msg("begin...")
	deviceRepo := a.AppDeps.DeviceRepo
	deviceList, rErr := deviceRepo.ListDevices(ctx)
	if rErr != nil {
		return rErr
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
