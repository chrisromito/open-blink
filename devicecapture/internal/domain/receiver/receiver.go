package receiver

import (
	"devicecapture/internal/domain/devices"
	"devicecapture/internal/domain/event"
)

// Publisher interface for broadcasting events
type Publisher interface {
	StartSession(deviceId string) (*CaptureSession, error)
	EndSession(session *CaptureSession) error
	// PublishFrame publishes the [Frame] so subscribers can consume it
	PublishFrame(frame Frame, framePath string, deviceId string) error
	// WriteFrame writes the [Frame] to the FileSystem
	WriteFrame(frame Frame, fp string) error
	// PublishDetections Publishes a device image and its detections as JSON
	PublishDetections(deviceID int64, img devices.DeviceImage, detections []devices.Detection) error
	// PublishDetectionEvent Publishes [event.DetectionEvent]
	PublishDetectionEvent(evt event.DetectionEvent) error
}
