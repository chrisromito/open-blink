package main

import (
	"context"
	"devicecapture/internal/config"
	"devicecapture/internal/device"
	pubsub2 "devicecapture/internal/pubsub"
	"encoding/json"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type App struct {
	conf       *config.Config
	MqttClient *pubsub2.MqttClient
	Devices    []device.CameraDevice
	DeviceMap  map[string]device.CameraDevice
}

func (a *App) SetupDevices() error {
	deviceConfigs := a.conf.Devices
	deviceMap := make(map[string]device.CameraDevice)
	devices := make([]device.CameraDevice, len(deviceConfigs))
	for _, dConf := range deviceConfigs {
		r := pubsub2.NewMqttReceiver(a.MqttClient, "/videos")
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

func main() {
	appCtx, cancel := context.WithCancel(context.Background())
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	conf := config.NewConfig()
	//brokers := []string{conf.MqttHost, "tcp://0.0.0.0:1883", "tcp://host.docker.internal:1833"}
	//client, err := pubsub.NewDeviceClient("go-server", conf.MqttHost)
	//if err != nil {
	//	log.Fatalf("Error creating MQTT client: %v", err)
	//}
	client, cerr := pubsub2.BrokerHelper("go-server", conf.MqttHost)
	if cerr != nil {
		log.Fatalf("Error creating MQTT client: %v", cerr)
	}
	if !client.Valid() {
		log.Fatalf("Failed to connect to a client")
	}
	app := App{
		conf:       conf,
		MqttClient: &client,
		Devices:    make([]device.CameraDevice, 0),
	}
	err := app.SetupDevices()
	if err != nil {
		log.Fatalf("Error setting up devices: %v", err)
	}
	go func() {
		err2 := app.SubscribeToStartStreamTopic(appCtx)
		if err2 != nil {
			log.Printf("Error subscribing to start-stream topic: %v", err)
		}
	}()
	<-sigChan
	cancel()

}
