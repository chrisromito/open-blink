package server

import (
	"devicecapture/internal/app"
	"devicecapture/internal/camera"
	"devicecapture/internal/domain/devices"
	"devicecapture/internal/logger"
	"devicecapture/internal/pubsub"
	"encoding/json"
	"github.com/google/uuid"
	"github.com/mattn/go-mjpeg"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"
)

func HomePageHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger.Debug().Msgf("HomePageHandler - Request: %s %s", r.Method, r.URL.Path)

		filePath := "/usr/src/app/static/index.html"

		// Check if file exists
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			logger.Debug().Msgf("File does not exist: %s", filePath)
			http.Error(w, "File not found", http.StatusNotFound)
			return
		}

		logger.Debug().Msgf("Serving file: %s", filePath)

		logger.Debug().Msgf("HomePageHandler")
		http.ServeFile(w, r, filePath)
	}
}

func DeviceListHandler(a *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger.Debug().Msgf("DeviceListHandler -> start")
		w.Header().Set("Content-Type", "application/json")
		ctx := r.Context()
		ds, deviceErr := a.AppDeps.DeviceRepo.ListDevices(ctx)
		if deviceErr != nil {
			logger.Debug().Msgf("DeviceListHandler -> deviceErr %v", deviceErr)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		var deviceList []devices.Device
		for _, d := range ds {
			deviceList = append(deviceList, d)
		}
		if err := json.NewEncoder(w).Encode(ds); err != nil {
			logger.Error().Msgf("DeviceListHandler -> internal server error")
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		logger.Debug().Msgf("DeviceListHandler -> returning")
		return
	}
}

func HeartBeatListHandler(a *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		ctx := r.Context()
		latestBeats, dbErr := a.AppDeps.HeartbeatRepo.LatestBeats(ctx)
		if dbErr != nil {
			logger.Error().Msgf("HeartBeatListHandler -> dbErr %v", dbErr)
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
		deviceId := r.PathValue("id")
		logger.Debug().Msgf("GET /image-stream/%s", deviceId)
		// validate the domain ID
		valid := a.AppDeps.DeviceRepo.IsValidId(deviceId)
		if !valid {
			logger.Error().Msgf("invalid domain id %s", deviceId)
			http.Error(w, "Invalid domain ID", http.StatusBadRequest)
			return
		}

		// MQTT
		qtClient := a.MqttClient.CopyWithClientId("stream-proxy" + uuid.New().String())
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
			logger.Fatal().Msgf("error marhsalling message: %v", jsonErr)
		}
		publishErr := qtClient.Publish("start-stream", message)
		if publishErr != nil {
			logger.Error().Msgf("failed to publish to %s", "start-stream/"+deviceId)
			return
		}

		// API setup
		var wg sync.WaitGroup
		ctx := r.Context()
		intId, intErr := strconv.ParseInt(deviceId, 10, 64)
		if intErr != nil {
			return
		}
		device, deviceErr := a.AppDeps.DeviceRepo.GetDevice(ctx, intId)
		if deviceErr != nil || device.DeviceUrl == "" {
			return
		}

		stream := mjpeg.NewStreamWithInterval(100 * time.Millisecond)
		defer func(stream *mjpeg.Stream) {
			_ = stream.Close()
		}(stream)

		// Camera proxy
		api := camera.NewApi(deviceId, device.DeviceUrl)
		wg.Add(1)
		go func() {
			streamErr := api.Stream(ctx, &wg, stream)
			if streamErr != nil {
				logger.Fatal().Msgf("server.StreamProxyHandler -> stream error %v", streamErr)
			}
		}()
		stream.ServeHTTP(w, r)
		wg.Wait()
	}
}
