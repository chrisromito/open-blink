package server

import (
	"devicecapture/internal/app"
	"devicecapture/internal/device/devices"
	"devicecapture/internal/pubsub"
	"encoding/json"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/google/uuid"
	"github.com/mattn/go-mjpeg"
	"log"
	"net/http"
	"sync"
	"time"
)

func DeviceListHandler(a *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Printf("DeviceListHandler -> start")
		w.Header().Set("Content-Type", "application/json")
		ctx := r.Context()
		ds, deviceErr := a.CameraService.DeviceRepo.ListDevices(ctx)
		if deviceErr != nil {
			log.Printf("DeviceListHandler -> deviceErr %v", deviceErr)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		var deviceList []devices.Device
		for _, d := range ds {
			deviceList = append(deviceList, *d)
		}
		if err := json.NewEncoder(w).Encode(ds); err != nil {
			log.Printf("DeviceListHandler -> internal server error")
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		log.Printf("DeviceListHandler -> returning")
		return
	}
}

func HeartBeatListHandler(a *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		ctx := r.Context()
		latestBeats, dbErr := a.HeartbeatRepo.LatestBeats(ctx)
		if dbErr != nil {
			log.Printf("HeartBeatListHandler -> dbErr %v", dbErr)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if err := json.NewEncoder(w).Encode(latestBeats); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		return
	}
}

// StreamProxyHandler /image-stream/{id}
func StreamProxyHandler(a *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Print("GET /image-stream/{id}")
		deviceId := r.PathValue("id")
		log.Printf("GET /image-stream/%s", deviceId)
		// validate the device ID
		valid := a.CameraService.IsValidId(deviceId)
		if !valid {
			log.Printf("invalid device id %s", deviceId)
			http.Error(w, "Invalid device ID", http.StatusBadRequest)
			return
		}
		stream := mjpeg.NewStreamWithInterval(200 * time.Millisecond)
		defer func(stream *mjpeg.Stream) {
			_ = stream.Close()
		}(stream)
		clientId := "stream-proxy" + uuid.New().String()
		qtClient := a.MqttClient.CopyWithClientId(clientId)
		qtErr := qtClient.Connect()
		if qtErr != nil {
			return
		}
		defer func(qtClient *pubsub.MqttClient) {
			_ = qtClient.Close()
		}(&qtClient)

		payload := app.StartStreamMessage{
			DeviceId: deviceId,
		}
		message, jsonErr := json.Marshal(payload)
		if jsonErr != nil {
			log.Fatalf("error marhsalling message: %v", jsonErr)
		}
		publishErr := qtClient.Publish("start-stream", message)
		if publishErr != nil {
			log.Printf("failed to publish to %s", "start-stream/"+deviceId)
			return
		}

		done := make(chan error)
		var wg sync.WaitGroup

		// mqtt worker goroutine
		go func() {
			wg.Add(1)
			defer wg.Done()
			err := qtClient.Subscribe("image-stream/"+deviceId, func(_ mqtt.Client, m mqtt.Message) {
				b := m.Payload()
				err2 := stream.Update(b)
				if err2 != nil {
					log.Print("stream update err")
					log.Print(err2)
					done <- err2
				}
			})
			if err != nil {
				done <- err
				return
			}
			<-done
		}()

		// interval to keep the stream rolling
		ctx := r.Context()
		go func() {
			wg.Add(1)
			defer wg.Done()
			ticker := time.NewTicker(10 * time.Second)
			defer ticker.Stop()
			for {
				select {
				case <-ctx.Done():
					done <- nil
					return
				case <-ticker.C:
					streamErr := qtClient.Publish("start-stream", message)
					if streamErr != nil {
						done <- streamErr
						return
					}

				case <-done:
					return
				}
			}
		}()

		stream.ServeHTTP(w, r)
		wg.Wait()
	}
}
