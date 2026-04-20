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
	client, cerr := pubsub.BrokerHelper("go-server", conf.MqttHost)
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
	//detector := detection.NewObjectDetectionService(conf, deps)

	a := app.NewApp(conf, &client, db, deps)

	// Create message processor with 3 workers (adjust based on your needs)
	processor := NewMessageProcessor(a, 3)
	processor.Start(appCtx)
	defer processor.Shutdown()

	// MQTT goroutine
	go func() {
		if err := SubscribeToStartStreamTopic(appCtx, processor, &client); err != nil {
			log.Printf("Error in MQTT subscription: %v", err)
			cancel() // Cancel context to trigger shutdown
		}
	}()
	//go func() {
	//	err2 := SubscribeToStartStreamTopic(appCtx, a)
	//	if err2 != nil {
	//		log.Printf("Error subscribing to start-stream topic: %v", err2)
	//	}
	//}()
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan
}

func SubscribeToStartStreamTopic(ctx context.Context, processor *MessageProcessor, client *pubsub.MqttClient) error {
	topics := []string{"start-stream", "motion-detected"}

	// Track subscriptions for cleanup
	var subscriptionWg sync.WaitGroup

	// Subscribe to all topics
	for _, topic := range topics {
		subscriptionWg.Add(1)
		t := topic // Capture loop variable

		// Create message handler
		messageHandler := func(_ mqtt.Client, m mqtt.Message) {
			log.Printf("Received message on topic: %s", t)
			if !processor.EnqueueMessage(m) {
				log.Printf("Failed to enqueue message from topic: %s", t)
			}
		}

		// Subscribe to topic
		if err := client.Subscribe(t, messageHandler); err != nil {
			log.Printf("Error subscribing to topic %s: %v", t, err)
			subscriptionWg.Done()
			return err
		}

		log.Printf("Successfully subscribed to topic: %s", t)

		// Handle unsubscription on context cancellation
		go func(topic string) {
			defer subscriptionWg.Done()
			<-ctx.Done()

			// Note: The pubsub.MqttClient doesn't expose an Unsubscribe method
			// but it's handled in the Close() method which is called in main()
			log.Printf("Context cancelled, will unsubscribe from topic %s on client close", topic)
		}(t)
	}

	// Wait for context cancellation
	<-ctx.Done()
	log.Println("Context cancelled, waiting for subscription cleanup...")
	subscriptionWg.Wait()

	return ctx.Err()
}

//
//func SubscribeToStartStreamTopic(ctx context.Context, a *app.App) error {
//	mc := make(chan mqtt.Message)
//	defer close(mc)
//	topics := []string{"start-stream", "motion-detected"}
//	var wg sync.WaitGroup
//	for _, t := range topics {
//		err := a.MqttClient.Subscribe(t, func(_ mqtt.Client, m mqtt.Message) {
//			log.Print("devicecapture.main -> subscribe -> pushing message to channel")
//			mc <- m
//			//wg.Add(1)
//			//streamErr := ReceiveStartStreamMessage(ctx, &wg, m, a)
//			//if streamErr != nil {
//			//	log.Fatalf("fatal error from ReceiveStartStreamMessage %v", streamErr)
//			//}
//		})
//		if err != nil {
//			log.Printf("Error subscribing to topic: %v", err)
//			return err
//		}
//	}
//
//	for {
//		select {
//		case m := <-mc:
//			wg.Add(1)
//			go func() {
//				_ = ReceiveStartStreamMessage(ctx, &wg, m, a)
//			}()
//			//err := ReceiveStartStreamMessage(ctx, &wg, m, a)
//			//if err != nil {
//			//	log.Printf("Error receiving message: %v", err)
//			//}
//		case <-ctx.Done():
//			wg.Wait()
//			return nil
//		}
//	}
//}

func ReceiveStartStreamMessage(ctx context.Context, wg *sync.WaitGroup, m mqtt.Message, a *app.App) error {
	defer wg.Done()
	log.Print("devicecapture.app -> ReceiveStartStreamMessage()")
	value := m.Payload()
	var msg app.StartStreamMessage
	err := json.Unmarshal(value, &msg)
	if err != nil {
		return err
	}
	msgCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	//-- Services
	cameraService := camera.NewCameraService(a.AppDeps, detection.NewObjectDetectionService(a.Conf))
	sesh, err := cameraService.StartStream(msgCtx, msg.DeviceId)
	if err != nil {
		log.Printf("ReceiveStartStreamMessage.startStream threw an error %v", err)
		return err
	}
	log.Printf("ended session: %v", sesh.StartedAt)
	return nil
}
