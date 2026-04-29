package main

import (
	"context"
	"devicecapture/internal/app"
	"devicecapture/internal/camera"
	"devicecapture/internal/config"
	"devicecapture/internal/domain"
	"devicecapture/internal/domain/detection"
	"devicecapture/internal/postgres"
	"devicecapture/internal/postgres/repos"
	"devicecapture/internal/pubsub"
	"encoding/json"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

func main() {
	conf := config.NewConfig()
	client, cerr := pubsub.BrokerHelper("go-server", conf.MqttHost, conf.MqttUser, conf.MqttPassword)
	if cerr != nil {
		log.Fatalf("Error creating MQTT client: %v", cerr)
	}
	if !client.Valid() {
		log.Fatalf("Failed to connect to a client")
	}
	defer func(client *pubsub.MqttClient) {
		_ = client.Close()
	}(&client)
	db := postgres.NewAppDb()
	dberr := db.Connect(conf.DbUrl)
	if dberr != nil {
		log.Fatalf("Error connecting to database: %v", dberr)
	}

	defer db.Db.Close()
	//-- Repos
	queries := db.GetQueries()
	deps := domain.NewDeps(
		repos.NewPgDeviceRepo(queries),
		repos.NewPgHeartbeatRepo(queries),
		repos.NewPgDetectionRepo(queries),
		repos.NewPgImageRepo(queries),
		pubsub.NewMqttReceiver(&client, conf.VideoPath),
	)

	//-- App
	appCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	a := app.NewApp(conf, &client, db, deps)

	// MQTT goroutine
	go func() {
		if err := SubscribeToStartStreamTopic(appCtx, a, &client); err != nil {
			log.Printf("Error in MQTT subscription: %v", err)
			cancel() // Cancel context to trigger shutdown
		}
	}()
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan
}

func SubscribeToStartStreamTopic(ctx context.Context, a *app.App, client *pubsub.MqttClient) error {
	//topics := []string{"start-stream", "motion-detected"}

	inChan := make(chan mqtt.Message, 3)
	queueChan := make(chan mqtt.Message, 3)
	// Track subscriptions for cleanup
	var subscriptionWg sync.WaitGroup

	// pull from inChan & push to queueChan
	subscriptionWg.Add(1)
	go func() {
		defer subscriptionWg.Done()
		for {
			select {
			case msg, ok := <-inChan:
				if !ok {
					log.Printf("devicecapture.SubscribeToStartStreamTopic() -> exiting because inChan !ok")
					return
				}
				log.Printf("pushing message to queueChan")
				queueChan <- msg
			case <-ctx.Done():
				return
			}
		}
	}()

	subscriptionWg.Add(1)
	go func() {
		defer subscriptionWg.Done()
		for {
			select {
			case m, ok := <-queueChan:
				// Handle start messages
				msgCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
				if !ok {
					log.Printf("devicecapture.SubscribeToStartStreamTopic() -> exiting because queueChan !ok")
					cancel()
					return
				}

				log.Printf("Processing message from topic: %s", m.Topic())

				value := m.Payload()
				var msg app.StartStreamMessage
				if err := json.Unmarshal(value, &msg); err != nil {
					log.Printf("failed to unmarshal message: %v", err)
					cancel()
					return
				}

				// Create camera service instance for this message
				cameraService := camera.NewCameraService(
					a.Conf,
					a.AppDeps,
					detection.NewObjectDetectionService(a.Conf),
					a.MqttClient,
				)
				log.Printf("starting stream for device %s", msg.DeviceId)
				_, err := cameraService.StartStream(msgCtx, msg.DeviceId)
				if err != nil {
					log.Printf("failed to start stream for device %s: %v", msg.DeviceId, err)
					cancel()
					return
				}
				cancel()
			case <-ctx.Done():
				return
			}
		}
	}()

	receiveMessage := func(_ mqtt.Client, m mqtt.Message) {
		subscriptionWg.Add(1)
		defer subscriptionWg.Done()
		log.Printf("received message on topic %v", m.Topic())
		inChan <- m
	}

	if err := client.Subscribe("start-stream", receiveMessage); err != nil {
		log.Printf("Error subscribing to topic %s: %v", "start-stream", err)
		return err
	}
	if err := client.Subscribe("motion-detected", receiveMessage); err != nil {
		log.Printf("Error subscribing to topic %s: %v", "motion-detected", err)
		return err
	}

	log.Println("deviceCapture -> waiting for subscription cleanup...")
	subscriptionWg.Wait()
	return nil
}
