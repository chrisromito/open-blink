package device

import (
	"bytes"
	"context"
	"devicecapture/device/receiver"
	"fmt"
	"github.com/mattn/go-mjpeg"
	"image"
	"log"
	"net/http"
	"time"
)

type Api struct {
	DeviceId string
	Url      string
}

func NewFrame(b []byte) receiver.Frame {
	img, _, err := image.Decode(bytes.NewReader(b))
	if err != nil {
		log.Printf("Error decoding image: %v", err)
		return receiver.Frame{
			Buf:       b,
			Image:     nil,
			Timestamp: time.Now().UnixMilli(),
		}
	}
	return receiver.Frame{
		Buf:       b,
		Image:     img,
		Timestamp: time.Now().UnixMilli(),
	}
}

func NewApi(deviceId string, deviceUrl string) Api {
	return Api{
		DeviceId: deviceId,
		Url:      deviceUrl,
	}
}

func (a *Api) Ping() bool {
	pingUrl := a.Url + "/ping"
	resp, err := http.Get(pingUrl)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

func (a *Api) StreamFrames(ctx context.Context, imgChan chan<- receiver.Frame) error {
	done := make(chan error)
	defer close(done)

	go func() {
		client := &http.Client{
			Timeout: 60 * time.Second,
		}

		streamUrl := a.Url + "/stream"
		log.Printf("Starting stream from %s", streamUrl)
		req, err := http.NewRequestWithContext(ctx, "GET", streamUrl, nil)
		if err != nil {
			done <- err
			return
		}

		// Add headers that might help with streaming
		req.Header.Set("Accept", "multipart/x-mixed-replace")
		req.Header.Set("User-Agent", "GoMJPEGClient/1.0")

		resp, err := client.Do(req)
		if err != nil {
			log.Printf("device.api.go -> StreamFrames -> Error streaming: %v", err)
			done <- err
			return
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			done <- fmt.Errorf("status code is not OK: %v (%s)", resp.StatusCode, resp.Status)
			return
		}
		dec, err2 := mjpeg.NewDecoderFromResponse(resp)
		if err2 != nil {
			done <- fmt.Errorf("error creating decoder: %v", err2)
			return
		}

		log.Printf("Starting MJPEG stream")
		frameCount := 0
		for {
			select {
			case <-ctx.Done():
				log.Printf("Stream stopped after %d frames", frameCount)
				done <- nil
				return
			default:
				b, err3 := dec.DecodeRaw()
				if err3 != nil {
					log.Printf("Error decoding frame: %v", err3)
					// Wait a bit and continue instead of returning
					time.Sleep(100 * time.Millisecond)
					continue
				}

				frameCount++
				log.Printf("Successfully decoded frame %d", frameCount)

				// Send image to channel instead of writing to file
				select {
				case <-ctx.Done():
					log.Printf("Stream stopped while sending frame %d", frameCount)
					done <- nil
					return
				case imgChan <- NewFrame(b):
					// Image sent successfully
				}
			}
		}
	}()
	<-done
	return nil
}
