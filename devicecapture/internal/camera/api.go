package camera

import (
	"bytes"
	"context"
	"devicecapture/internal/domain/receiver"
	"errors"
	"fmt"
	"github.com/mattn/go-mjpeg"
	"image"
	"io"
	"log"
	"net/http"
	"sync"
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
			Buf:       []byte{},
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

func (a *Api) Stream(ctx context.Context, wg *sync.WaitGroup, stream *mjpeg.Stream) error {
	defer wg.Done()
	client := &http.Client{
		Timeout: 60 * time.Second,
	}
	streamUrl := a.Url + "/stream"
	log.Printf("camera.api -> stream -> Starting stream from %s", streamUrl)
	req, err := http.NewRequestWithContext(ctx, "GET", streamUrl, nil)
	if err != nil {
		return err
	}

	// Add headers that might help with streaming
	req.Header.Set("Accept", "multipart/x-mixed-replace")
	req.Header.Set("User-Agent", "GoMJPEGClient/1.0")

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("camera.api -> stream -> Error streaming: %v", err)
		return err
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)
	if resp.StatusCode != http.StatusOK {
		log.Printf("camera.api -> stream -> Not OK status: %v %d", err, resp.StatusCode)
		return fmt.Errorf("received %d StatusCode", resp.StatusCode)
	}
	dec, err2 := mjpeg.NewDecoderFromResponse(resp)
	if err2 != nil {
		return err2
	}

	for {
		b, decErr := dec.DecodeRaw()
		ctxErr := ctx.Err()
		if ctxErr != nil {
			return nil
		}
		if decErr != nil {
			log.Printf("camera.api -> stream -> Error decoding: %v", err)
			return decErr
		}
		streamErr := stream.Update(b)
		if streamErr != nil {
			log.Printf("camera.api -> stream -> error updating stream: %v", err)
			return streamErr
		}
	}
}

func (a *Api) StreamFrames(ctx context.Context, imgChan chan<- receiver.Frame) error {
	var wg sync.WaitGroup
	stream := mjpeg.NewStream()
	defer func(stream *mjpeg.Stream) {
		_ = stream.Close()
	}(stream)
	stop := make(chan error)
	done := make(chan error)
	defer close(done)

	wg.Add(1)
	go func() {
		defer wg.Done()
		ticker := time.NewTicker(100 * time.Millisecond)
		for {
			select {
			case <-ticker.C:
				ctxErr := ctx.Err()
				if ctxErr != nil {
					return
				}
				data := stream.Current()
				if len(data) > 0 {
					imgChan <- NewFrame(data)
				}
			case s := <-stop:
				done <- s
				log.Printf("camera.api.StreamFrames goroutine 1 exiting because stop chan %v", s)
				return
			}
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		wg.Add(1)
		err := a.Stream(ctx, &wg, stream)
		if errors.Is(err, context.DeadlineExceeded) {
			stop <- nil
		} else {
			stop <- err
		}
		return
	}()
	wg.Wait()
	value := <-stop
	return value
}
