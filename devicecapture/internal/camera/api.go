package camera

import (
	"bytes"
	"context"
	"devicecapture/internal/domain/receiver"
	"devicecapture/internal/logger"
	"errors"
	"fmt"
	"github.com/mattn/go-mjpeg"
	"image"
	"image/jpeg"
	"io"
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
		logger.Error().Msgf("Error decoding image: %v", err)
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

func NewFrameFromImage(img image.Image) receiver.Frame {
	var buf bytes.Buffer
	err := jpeg.Encode(&buf, img, nil)
	if err != nil {
		logger.Error().Msgf("Error decoding image: %v", err)
		return receiver.Frame{
			Buf:       []byte{},
			Image:     nil,
			Timestamp: time.Now().UnixMilli(),
		}
	}
	return receiver.Frame{
		Buf:       buf.Bytes(),
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
	client := &http.Client{
		Timeout: 1 * time.Second,
	}
	resp, err := client.Get(pingUrl)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusNotFound
}

func (a *Api) Snapshot(ctx context.Context) (receiver.Frame, error) {
	client := &http.Client{
		Timeout: 2 * time.Second,
	}
	url := a.Url + "/snapshot"
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	empty := receiver.Frame{}
	if err != nil {
		return empty, err
	}

	req.Header.Set("Accept", "image/jpeg")
	resp, err := client.Do(req)
	if err != nil {
		return empty, err
	}

	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return empty, fmt.Errorf("received %d StatusCode", resp.StatusCode)
	}
	img, _, err := image.Decode(resp.Body)
	if err != nil {
		return empty, err
	}
	return NewFrameFromImage(img), nil
}

// Create a channel to handle decoder results
type decodeResult struct {
	data []byte
	err  error
}

func (a *Api) Stream(ctx context.Context, stream *mjpeg.Stream) error {
	client := &http.Client{}
	streamUrl := a.Url + "/stream"
	logger.Debug().Msgf("camera.api -> stream -> Starting stream from %s", streamUrl)
	req, err := http.NewRequestWithContext(ctx, "GET", streamUrl, nil)
	if err != nil {
		return err
	}

	// Add headers that might help with streaming
	req.Header.Set("Accept", "multipart/x-mixed-replace")
	req.Header.Set("User-Agent", "GoMJPEGClient/1.0")

	resp, err := client.Do(req)
	if err != nil {
		logger.Error().Msgf("camera.api -> stream -> Error streaming: %v", err)
		return err
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)
	if resp.StatusCode != http.StatusOK {
		logger.Error().Msgf("camera.api -> stream -> Not OK status: %v %d", err, resp.StatusCode)
		return fmt.Errorf("received %d StatusCode", resp.StatusCode)
	}
	dec, err2 := mjpeg.NewDecoderFromResponse(resp)
	if err2 != nil {
		return err2
	}
	decodeChan := make(chan decodeResult, 1)
	for {
		go func() {
			data, dErr := dec.DecodeRaw()
			decodeChan <- decodeResult{data: data, err: dErr}
		}()
		select {
		case <-ctx.Done():
			return ctx.Err()
		case value := <-decodeChan:
			if value.err != nil {
				logger.Error().Str("service", "camera.api").Err(value.err).
					Msgf("api.Stream threw an error in decodeChan")
				return value.err
			}
			streamErr := stream.Update(value.data)
			if streamErr != nil {
				logger.Error().Str("service", "camera.api").Err(streamErr).
					Msgf("api.Stream threw an error in decodeChan")
				return streamErr
			}
		}

	}
}

func (a *Api) StreamFrames(ctx context.Context, imgChan chan<- receiver.Frame) error {
	// Ping the API before we start streaming
	if !a.Ping() {
		return errors.New(fmt.Sprintf("could not stream from URL %s", a.Url))
	}
	stream := mjpeg.NewStream()
	defer func(stream *mjpeg.Stream) {
		_ = stream.Close()
	}(stream)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		ticker := time.NewTicker(250 * time.Millisecond)
		for {
			select {
			case <-ticker.C:
				data := stream.Current()
				if len(data) > 0 {
					imgChan <- NewFrame(data)
				}
			case <-ctx.Done():
				logger.Error().Str("service", "camera.api").
					Msgf("camera.api.StreamFrames goroutine 1 exiting because ctx.Done()")
				return
			}
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		logger.Info().Str("service", "camera.api").
			Msg("camera.api calling api.Stream()")
		streamValue := a.Stream(ctx, stream)
		logger.Info().Str("service", "camera.api").Err(streamValue).
			Msg("camera.api StartStream -> api.Stream() finished")
		return
	}()
	logger.Error().Str("service", "camera.api").
		Msg("ctx.Done, waiting for wg")
	wg.Wait()
	logger.Error().Str("service", "camera.api").
		Msg("wg.Done returning")
	return nil
}
