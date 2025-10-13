package receiver

import (
	"context"
	"image"
)

type CaptureSession struct {
	DeviceId  string
	StartedAt int64
}

type Frame struct {
	Buf       []byte
	Image     image.Image
	Timestamp int64
}

type FrameRepository interface {
	StartSession(deviceId string) (*CaptureSession, error)
	EndSession() error
	ReceiveFrame(frame Frame) error
	ReceiveFrameStream(ctx context.Context, imgChan <-chan Frame) error
}
