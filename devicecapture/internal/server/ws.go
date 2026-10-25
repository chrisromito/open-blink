package server

import (
	"context"
	"net/http"
	"sync"
	"time"

	"devicecapture/internal/app"
	"devicecapture/internal/logger"
	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/google/uuid"
)

var wsOpts = websocket.AcceptOptions{InsecureSkipVerify: true}

func DetectionStreamHandler(a *app.App) http.HandlerFunc {
	// See example: https://pkg.go.dev/github.com/coder/websocket#example-package-WriteOnly
	return func(w http.ResponseWriter, r *http.Request) {
		logger.Debug().Msg("ws /detections")
		// Setup websocket stuff
		c, err := websocket.Accept(w, r, &wsOpts)
		if err != nil {
			logger.Error().Err(err).Send()
			return
		}
		defer func(c *websocket.Conn) {
			_ = c.CloseNow()
		}(c)

		ctx, cancel := context.WithTimeout(r.Context(), time.Minute*10)
		defer cancel()

		ctx = c.CloseRead(ctx)

		// Subscribe to the "detection/*" topic and proxy
		// incoming messages to the WS client
		// MQTT
		qtClient, qtErr := a.MqttClient.Copy("detection-proxy" + uuid.New().String())
		if qtErr != nil {
			logger.Error().
				Str("DetectionStreamHandler", "failed to copy MqttClient").
				Err(qtErr).
				Send()
			return
		}
		defer qtClient.Disconnect(250)

		var wg sync.WaitGroup

		wg.Add(1)

		go func() {
			defer wg.Done()
			err = qtClient.Subscribe("detections/#", func(topic string, payload []byte) error {
				logger.Info().
					Msg("detection msg received, passing to websocket client")
				wsErr := wsjson.Write(ctx, c, string(payload))
				if wsErr != nil {
					logger.Error().
						Str("DetectionStreamHandler", "websocket send failed").
						Err(wsErr).
						Send()
					return wsErr
				}
				return nil
			})
			if err != nil {
				logger.Error().
					Str("DetectionStreamHandler", "MqttClient.subscribe failed").
					Err(err).
					Send()
				return
			}
			<-ctx.Done()
		}()
		wg.Wait()
		closeErr := c.Close(websocket.StatusNormalClosure, "")
		if closeErr != nil {
			logger.Error().
				Str("DetectionStreamHandler", "websocket.Close() failed").
				Err(closeErr).
				Send()
		}
	}
}
