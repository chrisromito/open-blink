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
	client, cerr := pubsub.BrokerHelper(clientID, conf.MqttHost, conf.MqttUser, conf.MqttPassword)
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
	log.Print("Sleeping 60 seconds before we cleanup & setup test data...")
	time.Sleep(60 * time.Second)
	log.Print("Cleaning up old test data & refreshing dataset")
	q := db.GetQueries()
	deviceRepo := repos.NewPgDeviceRepo(q)
	ctx := context.Background()
	// Clean up the test data
	err := q.DeleteTestDevices(ctx)
	if err != nil {
		log.Fatalf("failed to delete test devices due to err %v", err)
	}
	log.Println("Deleted test devices...")
	devices, camErr := deviceRepo.ListDevices(ctx)
	if camErr != nil {
		log.Fatalf("error listing devices %v", camErr)
	}
	for _, d := range devices {
		delErr := deviceRepo.DeleteDevice(ctx, d.ID)
		if delErr != nil {
			log.Fatalf("error deleting device %v", d.ID)
		}
	}
	// Create a test device record
	_, err = q.CreateTestDevice(ctx)
	if err != nil {
		log.Fatalf("failed to create test device due to err %v", err)
	}
}

func init() {
	rand.New(rand.NewSource(time.Now().UnixNano()))
}
