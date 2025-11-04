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
	Conf       *config.Config
	MqttClient *pubsub.MqttClient
	Db         *postgres.AppDb

	// FIXME: Make this dynamic (DeviceRepo here?)
	Devices   []device.CameraDevice
	DeviceMap map[string]device.CameraDevice
}

// NewApp create an App, under the assumption that the MqttClient & AppDb are initialized/connected
func NewApp(conf *config.Config, mqttClient *pubsub.MqttClient, db *postgres.AppDb) *App {
	return &App{
		Conf:       conf,
		MqttClient: mqttClient,
		Db:         db,
	}
}

func (a *App) SetupDevices() error {
	deviceConfigs := a.Conf.Devices
	deviceMap := make(map[string]device.CameraDevice)
	devices := make([]device.CameraDevice, len(deviceConfigs))
	for _, dConf := range deviceConfigs {
		r := pubsub.NewMqttReceiver(a.MqttClient, "/videos")
		d := device.NewCameraDevice(dConf, r)
		deviceMap[dConf.DeviceId] = d
		devices = append(devices, d)
	}
	log.Printf("Devices: %v", devices)
	a.Devices = devices
	a.DeviceMap = deviceMap
	return nil
}

func (a *App) SubscribeToStartStreamTopic(ctx context.Context) error {
	mc := make(chan mqtt.Message)
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
	d, ok := a.DeviceMap[msg.DeviceId]
	if !ok {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err = d.Start(ctx)
	if err != nil {
		return err
	}
	return nil
}
