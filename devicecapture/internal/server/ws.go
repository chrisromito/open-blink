package server

import (
	"context"
	"devicecapture/internal/app"
	"devicecapture/internal/pubsub"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/google/uuid"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

var wsOpts = websocket.AcceptOptions{InsecureSkipVerify: true}

func DetectionStreamHandler(a *app.App) http.HandlerFunc {
	// See example: https://pkg.go.dev/github.com/coder/websocket#example-package-WriteOnly
	return func(w http.ResponseWriter, r *http.Request) {

		log.Print("ws /detections")
		// Setup websocket stuff
		c, err := websocket.Accept(w, r, &wsOpts)
		if err != nil {
			log.Println(err)
			return
		}
		defer c.CloseNow()

		ctx, cancel := context.WithTimeout(r.Context(), time.Minute*10)
		defer cancel()

		ctx = c.CloseRead(ctx)

		// Subscribe to the "detection/*" topic & proxy
		// incoming messages to the WS client
		// MQTT
		qtClient := a.MqttClient.CopyWithClientId("detection-proxy" + uuid.New().String())
		qtErr := qtClient.Connect()
		if qtErr != nil {
			return
		}
		defer func(qtClient *pubsub.MqttClient) {
			_ = qtClient.Close()
		}(&qtClient)
		var wg sync.WaitGroup

		wg.Add(1)
		go func() {
			defer wg.Done()
			err = qtClient.Subscribe("detection/#", func(client mqtt.Client, msg mqtt.Message) {
				log.Printf("detection msg received, passing to websocket client")
				wsErr := wsjson.Write(ctx, c, string(msg.Payload()))
				if wsErr != nil {
					log.Printf("error writing to WS client: %v", wsErr)
					return
				}
			})
			if err != nil {
				log.Printf("mqttClient subscribe error: %v", err)
				return
			}
			<-ctx.Done()
		}()
		wg.Wait()
		closeErr := c.Close(websocket.StatusNormalClosure, "")
		if closeErr != nil {
			log.Printf("ws close error: %v", closeErr)
		}
	}
}
