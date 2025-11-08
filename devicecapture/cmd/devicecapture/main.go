package main

import (
	"context"
	"devicecapture/internal/app"
	"devicecapture/internal/config"
	"devicecapture/internal/device"
	"devicecapture/internal/postgres"
	"devicecapture/internal/postgres/repos"
	"devicecapture/internal/pubsub"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	appCtx, cancel := context.WithCancel(context.Background())
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	conf := config.NewConfig()
	client, cerr := pubsub.BrokerHelper("go-server", conf.MqttHost)
	if cerr != nil {
		log.Fatalf("Error creating MQTT client: %v", cerr)
	}
	if !client.Valid() {
		log.Fatalf("Failed to connect to a client")
	}
	db := postgres.NewAppDb()
	dberr := db.Connect(conf.DbUrl)
	if dberr != nil {
		log.Fatalf("Error connecting to database: %v", dberr)
	}

	//-- Repos
	cameraRepo := repos.NewPgDeviceRepo(db.GetQueries())
	frameRepo := pubsub.NewMqttReceiver(&client, conf.VideoPath)

	//-- Services
	svc := device.NewCameraService(cameraRepo, frameRepo)

	//-- App
	a := app.NewApp(conf, &client, db, svc)
	go func() {
		err2 := a.SubscribeToStartStreamTopic(appCtx)
		if err2 != nil {
			log.Printf("Error subscribing to start-stream topic: %v", err2)
		}
	}()
	<-sigChan
	cancel()

}
