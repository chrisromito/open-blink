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

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			b, decErr := dec.DecodeRaw()
			ctxErr := ctx.Err()
			if ctxErr != nil {
				return nil
			}
			if decErr != nil {
				logger.Error().Msgf("camera.api.Stream() -> exiting due to decoder err: %v", err)
				return decErr
			}
			streamErr := stream.Update(b)
			if streamErr != nil {
				logger.Error().Msgf("camera.api -> stream -> error updating stream: %v", err)
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
	stop := make(chan error)
	defer close(stop)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		ticker := time.NewTicker(250 * time.Millisecond)
		for {
			select {
			case <-ticker.C:
				ctxErr := ctx.Err()
				if ctxErr != nil {
					logger.Error().Str("service", "camera.api").
						Msg("camera.api.StreamFrames exiting because ctxErr")
					return
				}
				data := stream.Current()
				if len(data) > 0 {
					imgChan <- NewFrame(data)
				}
			case <-ctx.Done():
				logger.Error().Str("service", "camera.api").
					Msgf("camera.api.StreamFrames goroutine 1 exiting because ctx.Done()")
				return
			case s := <-stop:
				logger.Error().Str("service", "camera.api").
					Msgf("camera.api.StreamFrames goroutine 1 exiting because stop chan %v", s)
				return
			}
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		logger.Info().Str("service", "camera.api").
			Msg("camera.api calling api.Stream()")
		stop <- a.Stream(ctx, stream)
		logger.Error().Str("service", "camera.api").
			Msg("camera.api stream() goroutine finished")
		return
	}()
	//value := <-stop
	select {
	case value := <-stop:
		logger.Error().Str("service", "camera.api").
			Msg("Stop chan, returning, waiting for wg")
		wg.Wait()
		logger.Error().Str("service", "camera.api").
			Msg("wg.Done returning")
		return value
	case <-ctx.Done():
		logger.Error().Str("service", "camera.api").
			Msg("ctx.Done, waiting for wg")
		wg.Wait()
		logger.Error().Str("service", "camera.api").
			Msg("wg.Done returning")
		return nil
	case <-time.After(60 * time.Second):
		logger.Error().Str("service", "camera.api").
			Msg("returning after 60 seconds")
		return nil
	}

	//return value
}
