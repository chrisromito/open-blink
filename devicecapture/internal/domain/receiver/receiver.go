package receiver

import (
	"context"
	"fmt"
	"image"
	"sync"
	"time"
)

type Frame struct {
	Buf       []byte
	Image     image.Image
	Timestamp int64
}

// CaptureSession A reference to when we started capturing Frames from a Camera
type CaptureSession struct {
	DeviceID  string
	StartedAt int64
	f         *Frame
	m         sync.Mutex
}

func NewCaptureSession(deviceId string) *CaptureSession {
	return &CaptureSession{
		DeviceID:  deviceId,
		StartedAt: time.Now().UnixMilli(),
		f:         nil,
		m:         sync.Mutex{},
	}
}

func (cr *CaptureSession) SetLastFrame(fr *Frame) {
	cr.m.Lock()
	defer cr.m.Unlock()
	cr.f = fr
}

func (cr *CaptureSession) GetFrame() *Frame {
	cr.m.Lock()
	defer cr.m.Unlock()
	return cr.f
}

// FrameRepository interface for storing frames
type FrameRepository interface {
	StartSession(deviceId string) (*CaptureSession, error)
	EndSession() error
	ReceiveFrame(frame Frame, framePath string) error
	ReceiveFrameStream(ctx context.Context, imgChan <-chan Frame) error
}

func FramePath(prefix string, session *CaptureSession, frame Frame) string {
	return fmt.Sprintf(
		"%s/%s-%v/output-%s-%v.jpeg",
		prefix,
		session.DeviceID,
		session.StartedAt,
		session.DeviceID,
		frame.Timestamp,
	)
}
