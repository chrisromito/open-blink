/*
publisher - Publishes 'start-stream' messages to the MQTT broker for all configured devices
*/
package main

import (
	"context"
	"devicecapture/internal/config"
	"devicecapture/internal/postgres"
	"devicecapture/internal/postgres/repos"
	"devicecapture/internal/pubsub"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"time"
)

const (
	clientID = "go-mqtt-publisher"
	topic    = "start-stream"
)

type StartStreamMessage struct {
	DeviceId string `json:"device_id"`
}

func main() {
	conf := config.NewConfig()
	client, cerr := pubsub.BrokerHelper(clientID, conf.MqttHost)
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
	defer db.Db.Close()
	if dberr != nil {
		log.Fatalf("Error connecting to database: %v", dberr)
	}

	//-- Repos
	cameraRepo := repos.NewPgDeviceRepo(db.GetQueries())
	ctx := context.Background()
	devices, camErr := cameraRepo.ListDevices(ctx)
	if camErr != nil {
		log.Fatalf("error listing devices %v", camErr)
	}
	for _, d := range devices {
		payload := StartStreamMessage{
			DeviceId: d.StringId(),
		}
		message, err := json.Marshal(payload)
		if err != nil {
			log.Fatalf("error marhsalling message: %v", err)
		}
		err = client.Publish(topic, message)
		if err != nil {
			log.Fatalf("Error publishing message: %v", err)
		}
		fmt.Printf("Published message: %s\n", message)

	}
}

func init() {
	rand.New(rand.NewSource(time.Now().UnixNano()))
}
