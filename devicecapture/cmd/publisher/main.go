/*
publisher - Publishes 'start-stream' messages to the MQTT broker for all configured devices
*/
package main

import (
	"devicecapture/internal/config"
	"devicecapture/internal/pubsub"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

const (
	broker   = "tcp://0.0.0.0:1883"
	clientID = "go-mqtt-publisher"
	topic    = "start-stream"
)

var connectHandler mqtt.OnConnectHandler = func(client mqtt.Client) {
	fmt.Println("Connected to MQTT Broker")
}

var connectLostHandler mqtt.ConnectionLostHandler = func(client mqtt.Client, err error) {
	fmt.Printf("Connection lost: %v", err)
}

type StartStreamMessage struct {
	DeviceId string `json:"device_id"`
}

func main() {
	deviceConfigs, dErr := config.LoadDevices()
	if dErr != nil {
		log.Fatalf("Error loading devices: %v", dErr)
	}
	opts := mqtt.NewClientOptions()
	opts.AddBroker(broker)
	opts.SetClientID(clientID)
	opts.OnConnect = connectHandler
	opts.OnConnectionLost = connectLostHandler

	client, clientErr := pubsub.NewDeviceClient(clientID, broker)
	if clientErr != nil {
		log.Fatalf("Error creating MQTT client: %v", clientErr)
	}

	for _, deviceConfig := range deviceConfigs {
		payload := StartStreamMessage{
			DeviceId: deviceConfig.DeviceId,
		}
		message, err := json.Marshal(payload)
		if err != nil {
			log.Fatalf("Error marshalling message: %v", err)
		}
		fmt.Printf("Publishing message: %s\n", message)
		err = client.Publish(topic, message)
		if err != nil {
			log.Fatalf("Error publishing message: %v", err)
		}
		fmt.Printf("Published message: %s\n", message)
		time.Sleep(1 * time.Second)
	}
}

func init() {
	rand.New(rand.NewSource(time.Now().UnixNano()))
}
