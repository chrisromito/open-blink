package receiver

import (
	"image/jpeg"
	"os"
	"sync"
	"time"

	"devicecapture/internal/logger"
)

type MockFrameRepo struct {
	mu          sync.Mutex
	lastFrame   *Frame
	lastSession *CaptureSession
	Running     bool
}

func NewMockFrameRepo() *MockFrameRepo {
	return &MockFrameRepo{
		Running: false,
	}
}

func (fr *MockFrameRepo) GetFrame() *Frame {
	return fr.lastFrame
}

func (fr *MockFrameRepo) GetSession() *CaptureSession {
	return fr.lastSession
}

// StartSession MockFrameRepo implements receiver.FrameRepository
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

// EndSession MockFrameRepo implements receiver.FrameRepository
func (fr *MockFrameRepo) EndSession(_ *CaptureSession) error {
	fr.mu.Lock()
	defer fr.mu.Unlock()
	fr.Running = false
	return nil
}

// PublishFrame MockFrameRepo implements receiver.FrameRepository
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
