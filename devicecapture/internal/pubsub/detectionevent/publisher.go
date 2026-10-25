package detectionevent

import (
	"context"
	"fmt"

	"devicecapture/internal/config"
	"devicecapture/internal/domain/event"
)

type PublishClient interface {
	Publish(topic string, payload any) error
}

type Publisher struct {
	client PublishClient
	Conf   *config.Config
}

func NewPublisher(client PublishClient, conf *config.Config) *Publisher {
	return &Publisher{
		client: client,
		Conf:   conf,
	}
}

func (p *Publisher) PublishCreated(
	ctx context.Context,
	deviceID int64,
	event event.DetectionEvent,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	msg := NewMessage(deviceID, event)
	payload, err := Marshal(msg)
	if err != nil {
		return err
	}

	topic := CreatedTopic(deviceID)
	if err := p.client.Publish(topic, payload); err != nil {
		return fmt.Errorf("publish detectionevents to %q: %w", topic, err)
	}

	return nil
}

func (p *Publisher) PublishUpdated(
	ctx context.Context,
	deviceID int64,
	event event.DetectionEvent,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	msg := NewMessage(deviceID, event)
	payload, err := Marshal(msg)
	if err != nil {
		return err
	}

	topic := UpdatedTopic(deviceID)
	if err := p.client.Publish(topic, payload); err != nil {
		return fmt.Errorf("publish detectionevents to %q: %w", topic, err)
	}

	return nil
}
