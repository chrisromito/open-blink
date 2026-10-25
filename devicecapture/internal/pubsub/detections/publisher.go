package detections

import (
	"context"
	"fmt"

	"devicecapture/internal/domain/devices"
)

type PublishClient interface {
	Publish(topic string, payload any) error
}

type Publisher struct {
	client   PublishClient
	serverIP string
}

func NewPublisher(client PublishClient, serverIP string) *Publisher {
	return &Publisher{
		client:   client,
		serverIP: serverIP,
	}
}

func (p *Publisher) Publish(
	ctx context.Context,
	deviceID int64,
	image devices.DeviceImage,
	detections []devices.Detection,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	msg := NewMessage(p.serverIP, deviceID, image, detections)
	payload, err := Marshal(msg)
	if err != nil {
		return err
	}

	topic := Topic(deviceID)
	if err := p.client.Publish(topic, payload); err != nil {
		return fmt.Errorf("publish detections to %q: %w", topic, err)
	}

	return nil
}
