package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"image/jpeg"
	"os"
	"strconv"
	"sync"
	"time"

	"devicecapture/internal/domain/devices"
	"devicecapture/internal/domain/event"
	"devicecapture/internal/logger"
)

type MockPublisher struct {
	PublishedDeviceImages []devices.DeviceImage
	mu                    sync.Mutex
	lastFrame             *Frame
	lastSession           *CaptureSession
	outbox                map[string][]string
}

func NewMockFrameRepo() *MockPublisher {
	return &MockPublisher{
		PublishedDeviceImages: []devices.DeviceImage{},
		mu:                    sync.Mutex{},
		outbox:                map[string][]string{},
	}
}

func (mp *MockPublisher) GetFrame() *Frame {
	return mp.lastFrame
}

func (mp *MockPublisher) GetSession() *CaptureSession {
	return mp.lastSession
}

// StartSession MockPublisher implements queue.Publisher
func (mp *MockPublisher) StartSession(deviceId string) (*CaptureSession, error) {
	mp.mu.Lock()
	defer mp.mu.Unlock()
	cs := &CaptureSession{
		DeviceID:  deviceId,
		StartedAt: time.Now().UnixMilli(),
	}
	mp.lastSession = cs
	return cs, nil
}

// EndSession MockPublisher implements queue.Publisher
func (mp *MockPublisher) EndSession(_ *CaptureSession) error {
	mp.mu.Lock()
	defer mp.mu.Unlock()
	return nil
}

// PublishFrame MockPublisher implements queue.Publisher
func (mp *MockPublisher) PublishFrame(frame Frame, _ string, _ string) error {
	mp.mu.Lock()
	defer mp.mu.Unlock()
	mp.lastFrame = &frame
	return nil
}

// WriteFrame writes [queue.Frame] to the FileSystem
func (mp *MockPublisher) WriteFrame(frame Frame, framePath string) error {
	var fp = framePath
	logger.Debug().Msgf("Writing frame to file: %v at %s", frame.Timestamp, fp)
	f, err := os.Create(fp)
	defer func(f *os.File) {
		_ = f.Close()
	}(f)
	if err != nil {
		logger.Error().Msgf("error writing frame to file %v @ %s", frame.Timestamp, fp)
		return err
	}
	err2 := jpeg.Encode(f, frame.Image, nil)
	if err2 != nil {
		logger.Error().Msgf("error encoding frame to JPEG for %s", fp)
		return err2
	}
	return nil
}

// PublishDetections implements queue.Publisher
func (mp *MockPublisher) PublishDetections(
	_ context.Context,
	deviceID int64,
	img devices.DeviceImage,
	detections []devices.Detection,
) error {
	p, jErr := DetectionsToMsg("http://localhost:1234", img, detections)
	if jErr != nil {
		logger.Error().Str("service", "camera").
			Err(jErr).Send()
		return jErr
	}
	// Publish batch to MQTT
	topic := fmt.Sprintf("detections/%d", deviceID)
	return mp.publish(topic, p)
}

// PublishDetectionEvent implements queue.Publisher
func (mp *MockPublisher) PublishDetectionEvent(_ context.Context, evt event.DetectionEvent) error {
	dID := strconv.Itoa(int(evt.DeviceID))
	ds, err := event.DetectionStateFromValue(evt.State)
	if err != nil {
		return err
	}
	topic := fmt.Sprintf("detection-event/%s/%s", dID, ds.String())
	payload, jerr := json.Marshal(evt)
	if jerr != nil {
		return jerr
	}
	return mp.publish(topic, string(payload))
}

func (mp *MockPublisher) GetOutboxFor(topic string) []string {
	mp.mu.Lock()
	defer mp.mu.Unlock()
	value := mp.outbox[topic]
	return value
}

func (mp *MockPublisher) PublishDeviceImageCreated(
	_ context.Context,
	img devices.DeviceImage,
) error {
	mp.PublishedDeviceImages = append(mp.PublishedDeviceImages, img)
	return nil
}

func (mp *MockPublisher) publish(topic string, payload string) error {
	mp.mu.Lock()
	defer mp.mu.Unlock()
	mp.outbox[topic] = append(mp.outbox[topic], payload)
	return nil
}
