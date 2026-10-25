package queue

import (
	"devicecapture/internal/logger"
	"fmt"
	"image"
	"image/jpeg"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"
)

type Frame struct {
	Buf       []byte
	Image     image.Image
	Timestamp int64
}

func (f *Frame) WriteFrame(fp string) error {
	logger.Debug().Str("queue", "WriteFrame").
		Msgf("Writing frame to file: %v at %s", f.Timestamp, fp)
	ff, err := os.Create(fp)
	defer func(f *os.File) {
		_ = f.Close()
	}(ff)
	if err != nil {
		logger.Error().Msgf("error writing frame to file %v @ %s", f.Timestamp, fp)
		return err
	}
	err2 := jpeg.Encode(ff, f.Image, nil)
	if err2 != nil {
		logger.Error().Str("queue", "WriteFrame").
			Msgf("error encoding frame to JPEG for %s", fp)
		return err2
	}
	return nil
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
	csDir := fmt.Sprintf("%s-%v", session.DeviceID, session.StartedAt)
	fileName := fmt.Sprintf("output-%s-%v.jpeg", session.DeviceID, frame.Timestamp)
	return filepath.Join(prefix, csDir, fileName)
}

//func AnnotatedPath(prefix string, session *CaptureSession, frame Frame) string {
//	csDir := fmt.Sprintf("%s-%v", session.DeviceID, session.StartedAt)
//	fileName := fmt.Sprintf("annotated-%s-%v.jpeg", session.DeviceID, frame.Timestamp)
//	return filepath.Join(prefix, csDir, fileName)
//}
