package receiver

import (
	"context"
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
		DeviceId:  deviceId,
		StartedAt: time.Now().UnixMilli(),
	}
	fr.lastSession = cs
	fr.Running = true
	return cs, nil
}

// EndSession MockFrameRepo implements receiver.FrameRepository
func (fr *MockFrameRepo) EndSession() error {
	fr.mu.Lock()
	defer fr.mu.Unlock()
	fr.Running = false
	return nil
}

// ReceiveFrame MockFrameRepo implements receiver.FrameRepository
func (fr *MockFrameRepo) ReceiveFrame(frame Frame) error {
	fr.mu.Lock()
	defer fr.mu.Unlock()
	fr.lastFrame = &frame
	return nil
}

// ReceiveFrameStream MockFrameRepo implements receiver.FrameRepository
func (fr *MockFrameRepo) ReceiveFrameStream(ctx context.Context, imgChan <-chan Frame) error {
	done := make(chan error)
	go func() {
		for {
			select {
			case img, ok := <-imgChan:
				if !ok {
					done <- nil
					return
				}
				err := fr.ReceiveFrame(img)
				if err != nil {
					done <- err
					return
				}
			case <-ctx.Done():
				done <- nil
				return
			}
		}
	}()
	return <-done
}
