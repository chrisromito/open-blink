// http Starts an http server that listens on port 4000
// API routes:
// /api/domain - List all devices
// /api/domain/<int:id> - Device detail
// Camera/media routes
// /camera/<int:DeviceID>/stream - MJPEG stream
// /camera/<int:DeviceID>/snapshot - JPEG snapshot
package main

import (
	"context"
	"devicecapture/internal/app"
	"devicecapture/internal/config"
	"devicecapture/internal/domain"
	"devicecapture/internal/postgres"
	"devicecapture/internal/postgres/repos"
	"devicecapture/internal/pubsub"
	"devicecapture/internal/server"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	conf := config.NewConfig()
	client, cerr := pubsub.BrokerHelper("go-deviceserver", conf.MqttHost, conf.MqttUser, conf.MqttPassword)
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
	queries := db.GetQueries()
	deps := domain.NewDeps(
		repos.NewPgDeviceRepo(queries),
		repos.NewPgHeartbeatRepo(queries),
		repos.NewPgDetectionRepo(queries),
		repos.NewPgImageRepo(queries),
		pubsub.NewMqttReceiver(&client, conf.VideoPath),
	)

	//-- App
	a := app.NewApp(conf, &client, db, deps)

	// Register HTTP endpoints
	http.HandleFunc("/", server.HomePageHandler())
	http.HandleFunc("/device", server.DeviceListHandler(a))
	http.HandleFunc("/image-stream/{id}", server.StreamProxyHandler(a))
	http.HandleFunc("/heartbeat", server.HeartBeatListHandler(a))
	http.HandleFunc("/detection-stream", server.DetectionStreamHandler(a))

	//lis, nErr := net.Listen("tcp", ":")

	appServer := &http.Server{
		Addr: ":4000",
	}
	shutdownChan := make(chan bool, 1)

	go func() {
		err := appServer.ListenAndServe()
		if errors.Is(err, http.ErrServerClosed) {
			fmt.Print("server closed")
		} else if err != nil {
			fmt.Printf("error listening for server")
		}
		// simulate time to close connections
		time.Sleep(1 * time.Millisecond)

		log.Println("Stopped serving new connections.")
		shutdownChan <- true
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	ctx, release := context.WithTimeout(context.Background(), 10*time.Second)
	defer release()

	if err := appServer.Shutdown(ctx); err != nil {
		_ = appServer.Close()
		log.Fatalf("HTTP shutdown error: %v", err)
	}

	<-shutdownChan
	log.Println("Graceful shutdown complete.")
}
