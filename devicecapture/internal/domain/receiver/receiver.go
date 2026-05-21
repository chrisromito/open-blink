package receiver

import (
	"fmt"
	"image"
	"strconv"
	"sync"
	"time"
)

// FrameRepository interface for storing frames
type FrameRepository interface {
	StartSession(deviceId string) (*CaptureSession, error)
	EndSession(session *CaptureSession) error
	PublishFrame(frame Frame, framePath string, deviceId string) error
}

type Frame struct {
	Buf       []byte
	Image     image.Image
	Timestamp int64
}

// CaptureSession A reference to when we started capturing Frames from a Camera
type CaptureSession struct {
	DeviceID   string
	StartedAt  int64
	f          *Frame
	m          sync.Mutex
	frameCount int
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
	cr.frameCount = cr.frameCount + 1
}

func (cr *CaptureSession) GetFrame() *Frame {
	cr.m.Lock()
	defer cr.m.Unlock()
	return cr.f
}

func (cr *CaptureSession) GetFrameCount() int {
	cr.m.Lock()
	defer cr.m.Unlock()
	return cr.frameCount
}

func (cr *CaptureSession) IntID() (int64, error) {
	value, err := strconv.ParseInt(cr.DeviceID, 10, 64)
	if err != nil {
		return int64(0), err
	}
	return value, nil
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
