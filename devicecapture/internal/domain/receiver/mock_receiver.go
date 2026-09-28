package receiver

import (
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

type MockFrameRepo struct {
	Running     bool
	mu          sync.Mutex
	lastFrame   *Frame
	lastSession *CaptureSession
	outbox      map[string][]string
}

func NewMockFrameRepo() *MockFrameRepo {
	return &MockFrameRepo{
		Running: false,
		mu:      sync.Mutex{},
		outbox:  map[string][]string{},
	}
}

func (fr *MockFrameRepo) GetFrame() *Frame {
	return fr.lastFrame
}

func (fr *MockFrameRepo) GetSession() *CaptureSession {
	return fr.lastSession
}

// StartSession MockFrameRepo implements receiver.Publisher
func (fr *MockFrameRepo) StartSession(deviceId string) (*CaptureSession, error) {
	fr.mu.Lock()
	defer fr.mu.Unlock()
	cs := &CaptureSession{
		DeviceID:  deviceId,
		StartedAt: time.Now().UnixMilli(),
	}
	fr.lastSession = cs
	fr.Running = true
	return cs, nil
}

// EndSession MockFrameRepo implements receiver.Publisher
func (fr *MockFrameRepo) EndSession(_ *CaptureSession) error {
	fr.mu.Lock()
	defer fr.mu.Unlock()
	fr.Running = false
	return nil
}

// PublishFrame MockFrameRepo implements receiver.Publisher
func (fr *MockFrameRepo) PublishFrame(frame Frame, _ string, _ string) error {
	fr.mu.Lock()
	defer fr.mu.Unlock()
	fr.lastFrame = &frame
	return nil
}

// WriteFrame writes [receiver.Frame] to the FileSystem
func (fr *MockFrameRepo) WriteFrame(frame Frame, framePath string) error {
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

// PublishDetections implements receiver.Publisher
func (fr *MockFrameRepo) PublishDetections(
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
	return fr.publish(topic, p)
}

// PublishDetectionEvent implements receiver.Publisher
func (fr *MockFrameRepo) PublishDetectionEvent(evt event.DetectionEvent) error {
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
	return fr.publish(topic, string(payload))
}

func (fr *MockFrameRepo) GetOutboxFor(topic string) []string {
	fr.mu.Lock()
	defer fr.mu.Unlock()
	value := fr.outbox[topic]
	return value
}

func (fr *MockFrameRepo) publish(topic string, payload string) error {
	fr.mu.Lock()
	defer fr.mu.Unlock()
	fr.outbox[topic] = append(fr.outbox[topic], payload)
	return nil
}
