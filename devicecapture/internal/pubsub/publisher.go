package pubsub

import (
	"context"
	"devicecapture/internal/config"
	"devicecapture/internal/domain/devices"
	"devicecapture/internal/domain/event"
	"devicecapture/internal/domain/queue"
	"devicecapture/internal/logger"
	"devicecapture/internal/pubsub/detectionevent"
	"devicecapture/internal/pubsub/detections"
	"devicecapture/internal/pubsub/deviceimage"
	"fmt"
	"os"
)

// MqttPublisher implements queue.Publisher
type MqttPublisher struct {
	conf   *config.Config
	client *MqttClient
}

func NewMqttPublisher(client *MqttClient, conf *config.Config) *MqttPublisher {
	return &MqttPublisher{
		conf:   conf,
		client: client,
	}
}

// StartSession Start a queue.CaptureSession
func (r *MqttPublisher) StartSession(deviceId string) (*queue.CaptureSession, error) {
	s := queue.NewCaptureSession(deviceId)
	err := r.checkSessionDir(s)
	if err != nil {
		return nil, err
	}
	return s, nil
}

func (r *MqttPublisher) EndSession(session *queue.CaptureSession) error {
	if session == nil {
		return nil
	}
	topic := fmt.Sprintf("end-stream/%s", session.DeviceID)
	payload := fmt.Sprintf("%s/%s-%v", r.conf.VideoPath, session.DeviceID, session.StartedAt)
	err := r.client.Publish(topic, payload)
	if err != nil {
		return err
	}
	return nil
}

func (r *MqttPublisher) PublishDetections(
	ctx context.Context,
	deviceID int64,
	img devices.DeviceImage,
	dets []devices.Detection,
) error {
	publisher := detections.NewPublisher(r.client, r.conf.ThisIp)
	return publisher.Publish(ctx, deviceID, img, dets)
}

func (r *MqttPublisher) PublishDetectionEvent(ctx context.Context, evt event.DetectionEvent) error {
	publisher := detectionevent.NewPublisher(r.client, r.conf)
	state, stateErr := event.DetectionStateFromValue(evt.State)
	if stateErr != nil {
		logger.Error().Str("pubsub", "publisher").
			Str("PublishDetectionEvent", "Invalid Event State encountered").
			Int("state", evt.State).Send()
		return stateErr
	}
	if state == event.Started {
		return publisher.PublishCreated(ctx, evt.DeviceID, evt)
	}
	if state == event.Ended {
		return publisher.PublishUpdated(ctx, evt.DeviceID, evt)
	}
	// FIXME: Handle unknown states & down states. Returning an error for now...
	logger.Error().Str("pubsub", "publisher").
		Str("PublishDetectionEvent", "state fell through").
		Int("state", evt.State).Send()
	return fmt.Errorf("cannot process event.state %d", evt.State)
}

func (r *MqttPublisher) PublishDeviceImageCreated(
	ctx context.Context,
	img devices.DeviceImage,
) error {
	publisher := deviceimage.NewPublisher(r.client)
	return publisher.PublishCreated(ctx, img)
}

func (r *MqttPublisher) checkSessionDir(session *queue.CaptureSession) error {
	dir := fmt.Sprintf("%s/%s-%v", r.conf.VideoPath, session.DeviceID, session.StartedAt)
	_, err := os.Stat(dir)
	if os.IsNotExist(err) {
		err = os.MkdirAll(dir, 0755)
		if err != nil {
			return err
		}
	}
	return nil
}
