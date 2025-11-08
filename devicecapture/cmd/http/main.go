// http Starts an http server that listens on port 4000
// API routes:
// /api/device - List all devices
// /api/device/<int:id> - Device detail
// Camera/media routes
// /camera/<int:DeviceId>/stream - MJPEG stream
// /camera/<int:DeviceId>/snapshot - JPEG snapshot
package main

import (
	"context"
	"devicecapture/internal/app"
	"devicecapture/internal/config"
	"devicecapture/internal/device"
	"devicecapture/internal/postgres"
	"devicecapture/internal/postgres/repos"
	"devicecapture/internal/pubsub"
	"errors"
	"fmt"
	"log"
	"net/http"
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
	defer func(client *pubsub.MqttClient) {
		_ = client.Close()
	}(&client)
	if !client.Valid() {
		log.Fatalf("Failed to connect to a client")
	}
	db := postgres.NewAppDb()
	dberr := db.Connect(conf.DbUrl)
	defer db.Db.Close()
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

	// Register HTTP endpoints
	http.HandleFunc("/device", DeviceListHandler(a))
	http.HandleFunc("/image-stream/{id}", StreamProxyHandler(a))
	http.HandleFunc("/heartbeat", HeartBeatListHandler(a))

	go func() {
		err := http.ListenAndServe(":4000", nil)
		if errors.Is(err, http.ErrServerClosed) {
			fmt.Print("server closed")
		} else if err != nil {
			fmt.Printf("error listening for server")
		}
		cancel()
	}()
	<-appCtx.Done()
	<-sigChan
	cancel()

}
