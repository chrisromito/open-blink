package queue

import (
	"context"

	"devicecapture/internal/domain/devices"
	"devicecapture/internal/domain/event"
)

// Publisher interface for publishing messages to the queue
type Publisher interface {
	StartSession(deviceId string) (*CaptureSession, error)
	EndSession(session *CaptureSession) error
	PublishDetections(ctx context.Context, deviceID int64, img devices.DeviceImage, detections []devices.Detection) error
	PublishDetectionEvent(ctx context.Context, evt event.DetectionEvent) error
	PublishDeviceImageCreated(ctx context.Context, img devices.DeviceImage) error
}

// Receiver interface receives messages from the queue
type Receiver interface {
	Run(ctx context.Context) error
	ReceiveDetectionEvent(ctx context.Context, topic string, payload []byte) error
	ReceiveMotion(ctx context.Context, topic string, payload []byte) error
}
