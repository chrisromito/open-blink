package main

import (
	"bytes"
	"devicecapture/internal/app"
	"fmt"
	"image"
	"image/jpeg"
	"io"
	"net/http"
	"sync"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

func DeviceListHandler(a *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, e := fmt.Fprintf(w, "Hello, deviceListHandler")
		if e != nil {
			panic(e)
		}
		return
	}
}

// StreamProxyHandler /image-stream/{id}
func StreamProxyHandler(a *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		deviceId := r.PathValue("id")
		// validate the device ID
		valid := a.CameraService.IsValidId(deviceId)
		if !valid {
			http.Error(w, "Invalid device ID", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "multipart/x-mixed-replace; boundary=frame")

		boundary := "\r\n--frame\r\nContent-Type: image/jpeg\r\n\r\n"

		done := make(chan error)
		jpegChan := make(chan []byte)
		defer close(jpegChan)
		var wg sync.WaitGroup

		// mqtt worker goroutine
		go func() {
			wg.Add(1)
			defer wg.Done()
			buf := new(bytes.Buffer)
			err := a.MqttClient.Subscribe("image-stream/"+deviceId, func(_ mqtt.Client, m mqtt.Message) {
				b := m.Payload()
				img, _, dErr := image.Decode(bytes.NewReader(b))
				if dErr != nil {
					done <- dErr
					return
				}
				jpegErr := jpeg.Encode(buf, img, nil)
				if jpegErr != nil {
					done <- jpegErr
				}
				jpegChan <- img
			})
			if err != nil {
				done <- err
			}
			<-done
		}()
		// response writer goroutine
		go func() {
			wg.Add(1)
			defer wg.Done()
			for {
				select {
				case jpeg, ok := <-jpegChan:
					if !ok {
						return
					}
					// write boundary
					n, err := io.WriteString(w, boundary)
					if err != nil || n != len(boundary) {
						done <- err
						return
					}
					// write image
					n, err = w.Write(jpeg)
					if err != nil || n != len(jpeg) {
						done <- err
						return
					}
					// write end of frame
					n, err = io.WriteString(w, "\r\n")
					if err != nil || n != 2 {
						done <- err
						return
					}
				}
			}
		}
		a.MqttClient.Subscribe("image-stream/"+deviceId, func(_ mqtt.Client, m mqtt.Message) {
			buf = m.Payload()
			w.Write(buf)
		})
		return
	}
}

func HeartBeatListHandler(a *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, e := fmt.Fprintf(w, "Hello, HeartBeatListHandler")
		if e != nil {
			panic(e)
		}
		return
	}
}
