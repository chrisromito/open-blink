package deviceimage

import (
	"context"
	"fmt"

	"devicecapture/internal/domain/devices"
)

type PublishClient interface {
	Publish(topic string, payload any) error
}

type Publisher struct {
	client PublishClient
}

func NewPublisher(client PublishClient) *Publisher {
	return &Publisher{
		client: client,
	}
}

func (p *Publisher) PublishCreated(ctx context.Context, image devices.DeviceImage) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	msg := NewMessage(image)
	payload, err := Marshal(msg)
	if err != nil {
		return err
	}

	topic := CreatedTopic(image.DeviceID)
	if err := p.client.Publish(topic, payload); err != nil {
		return fmt.Errorf("publish device image to %q: %w", topic, err)
	}

	return nil
}


func (p *Publisher) PublishUpdated(ctx context.Context, image devices.DeviceImage) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	msg := NewMessage(image)
	payload, err := Marshal(msg)
	if err != nil {
		return err
	}

	topic := UpdatedTopic(image.DeviceID)
	if err := p.client.Publish(topic, payload); err != nil {
		return fmt.Errorf("publish device image to %q: %w", topic, err)
	}

	return nil
}
