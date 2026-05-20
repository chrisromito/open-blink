package receiver

import (
	"sync"
	"time"
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
