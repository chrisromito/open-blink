package app

import (
	"context"
	"devicecapture/internal/config"
	"devicecapture/internal/device"
	"devicecapture/internal/postgres"
	"devicecapture/internal/pubsub"
	"encoding/json"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"log"
	"time"
)

type App struct {
	Conf          *config.Config
	MqttClient    *pubsub.MqttClient
	Db            *postgres.AppDb
	CameraService *device.CameraService
}

// NewApp create an App, under the assumption that the MqttClient & AppDb are initialized/connected
func NewApp(conf *config.Config, mqttClient *pubsub.MqttClient, db *postgres.AppDb, cs *device.CameraService) *App {
	return &App{
		Conf:          conf,
		MqttClient:    mqttClient,
		Db:            db,
		CameraService: cs,
	}
}

func (a *App) SubscribeToStartStreamTopic(ctx context.Context) error {
	mc := make(chan mqtt.Message)
	defer close(mc)
	topics := []string{"start-stream", "motion-detected"}
	for _, t := range topics {
		err := a.MqttClient.Subscribe(t, func(_ mqtt.Client, m mqtt.Message) {
			mc <- m
		})
		if err != nil {
			log.Printf("Error subscribing to topic: %v", err)
			return err
		}
	}

	for {
		select {
		case m := <-mc:
			err := a.ReceiveStartStreamMessage(m)
			if err != nil {

				log.Printf("Error receiving message: %v", err)
			}
		case <-ctx.Done():
			return nil
		}
	}
}

type StartStreamMessage struct {
	DeviceId string `json:"device_id"`
}

func (a *App) ReceiveStartStreamMessage(m mqtt.Message) error {
	value := m.Payload()
	var msg StartStreamMessage
	err := json.Unmarshal(value, &msg)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err = a.CameraService.StartStream(ctx, msg.DeviceId)
	if err != nil {
		return err
	}
	return nil
}
