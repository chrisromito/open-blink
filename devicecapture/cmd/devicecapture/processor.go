package main

import (
	"context"
	"devicecapture/internal/app"
	"devicecapture/internal/camera"
	"devicecapture/internal/domain/detection"
	"encoding/json"
	"fmt"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"log"
	"sync"
	"time"
)

// MessageProcessor handles the processing of MQTT messages with proper synchronization
type MessageProcessor struct {
	app          *app.App
	workerCount  int
	messageQueue chan mqtt.Message
	workerWg     sync.WaitGroup
	shutdownOnce sync.Once
}

func NewMessageProcessor(app *app.App, workerCount int) *MessageProcessor {
	return &MessageProcessor{
		app:          app,
		workerCount:  workerCount,
		messageQueue: make(chan mqtt.Message, 100), // Buffered to prevent blocking
	}
}

func (mp *MessageProcessor) Start(ctx context.Context) {
	// Start worker goroutines
	for i := 0; i < mp.workerCount; i++ {
		mp.workerWg.Add(1)
		go mp.worker(ctx, i)
	}
}

func (mp *MessageProcessor) worker(ctx context.Context, workerID int) {
	defer mp.workerWg.Done()
	log.Printf("Worker %d started", workerID)

	for {
		select {
		case msg, ok := <-mp.messageQueue:
			if !ok {
				log.Printf("Worker %d: message queue closed", workerID)
				return
			}

			// Process message with timeout
			if err := mp.processMessage(ctx, msg); err != nil {
				log.Printf("Worker %d: error processing message: %v", workerID, err)
			}

		case <-ctx.Done():
			log.Printf("Worker %d: context cancelled", workerID)
			return
		}
	}
}

func (mp *MessageProcessor) processMessage(ctx context.Context, m mqtt.Message) error {
	// Create a timeout context for this specific message
	msgCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	log.Printf("Processing message from topic: %s", m.Topic())

	value := m.Payload()
	var msg app.StartStreamMessage
	if err := json.Unmarshal(value, &msg); err != nil {
		return fmt.Errorf("failed to unmarshal message: %w", err)
	}

	// Create camera service instance for this message
	cameraService := camera.NewCameraService(mp.app.Conf, mp.app.AppDeps, detection.NewObjectDetectionService(mp.app.Conf))

	sesh, err := cameraService.StartStream(msgCtx, msg.DeviceId)
	if err != nil {
		return fmt.Errorf("failed to start stream for device %s: %w", msg.DeviceId, err)
	}

	log.Printf("Successfully processed message for device %s, session ended at: %v", msg.DeviceId, sesh.StartedAt)
	return nil
}

func (mp *MessageProcessor) EnqueueMessage(msg mqtt.Message) bool {
	select {
	case mp.messageQueue <- msg:
		return true
	default:
		log.Printf("Message queue full, dropping message from topic: %s", msg.Topic())
		return false
	}
}

func (mp *MessageProcessor) Shutdown() {
	mp.shutdownOnce.Do(func() {
		log.Println("Shutting down message processor...")
		close(mp.messageQueue)
		mp.workerWg.Wait()
		log.Println("Message processor shutdown complete")
	})
}

//
//func main() {
//	conf := config.NewConfig()
//	client, cerr := pubsub.BrokerHelper("go-server", conf.MqttHost)
//	if cerr != nil {
//		log.Fatalf("Error creating MQTT client: %v", cerr)
//	}
//	if !client.Valid() {
//		log.Fatalf("Failed to connect to a client")
//	}
//	defer func(client *pubsub.MqttClient) {
//		_ = client.Close()
//	}(&client)
//
//	db := postgres.NewAppDb()
//	dberr := db.Connect(conf.DbUrl)
//	if dberr != nil {
//		log.Fatalf("Error connecting to database: %v", dberr)
//	}
//	defer db.Db.Close()
//
//	// Initialize repos
//	queries := db.GetQueries()
//	deviceRepo := repos.NewPgDeviceRepo(queries)
//	hbRepo := repos.NewPgHeartbeatRepo(queries)
//	detectionRepo := repos.NewPgDetectionRepo(queries)
//	frameRepo := pubsub.NewMqttReceiver(&client, conf.VideoPath)
//	deps := domain.NewDeps(deviceRepo, hbRepo, detectionRepo, frameRepo)
//
//	// Create app and message processor
//	appCtx, cancel := context.WithCancel(context.Background())
//	defer cancel()
//
//	a := app.NewApp(conf, &client, db, deps)
//
//	// Create message processor with 3 workers (adjust based on your needs)
//	processor := NewMessageProcessor(a, 3)
//	processor.Start(appCtx)
//	defer processor.Shutdown()
//
//	// Start MQTT subscription in a separate goroutine
//	go func() {
//		if err := SubscribeToStartStreamTopic(appCtx, processor, &client); err != nil {
//			log.Printf("Error in MQTT subscription: %v", err)
//			cancel() // Cancel context to trigger shutdown
//		}
//	}()
//
//	// Wait for shutdown signal
//	sigChan := make(chan os.Signal, 1)
//	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
//	<-sigChan
//
//	log.Println("Shutdown signal received, initiating graceful shutdown...")
//}
//
//func SubscribeToStartStreamTopic(ctx context.Context, processor *MessageProcessor, client *pubsub.MqttClient) error {
//	topics := []string{"start-stream", "motion-detected"}
//
//	// Track subscriptions for cleanup
//	var subscriptionWg sync.WaitGroup
//
//	// Subscribe to all topics
//	for _, topic := range topics {
//		subscriptionWg.Add(1)
//		t := topic // Capture loop variable
//
//		// Create message handler
//		messageHandler := func(_ mqtt.Client, m mqtt.Message) {
//			log.Printf("Received message on topic: %s", t)
//			if !processor.EnqueueMessage(m) {
//				log.Printf("Failed to enqueue message from topic: %s", t)
//			}
//		}
//
//		// Subscribe to topic
//		if err := client.Subscribe(t, messageHandler); err != nil {
//			log.Printf("Error subscribing to topic %s: %v", t, err)
//			subscriptionWg.Done()
//			return err
//		}
//
//		log.Printf("Successfully subscribed to topic: %s", t)
//
//		// Handle unsubscription on context cancellation
//		go func(topic string) {
//			defer subscriptionWg.Done()
//			<-ctx.Done()
//
//			// Note: The pubsub.MqttClient doesn't expose an Unsubscribe method
//			// but it's handled in the Close() method which is called in main()
//			log.Printf("Context cancelled, will unsubscribe from topic %s on client close", topic)
//		}(t)
//	}
//
//	// Wait for context cancellation
//	<-ctx.Done()
//	log.Println("Context cancelled, waiting for subscription cleanup...")
//	subscriptionWg.Wait()
//
//	return ctx.Err()
//}
