package deviceimage

import (
	"context"
	"devicecapture/internal/logger"
	"fmt"
)

type Receiver struct {
	dip *DeviceImageProcessor
}

func NewReceiver(dip *DeviceImageProcessor) *Receiver {
	return &Receiver{
		dip: dip,
	}
}

// Handle goes from Mqtt layer -> application layer (DeviceImageProcessor)
func (r *Receiver) Handle(ctx context.Context, topic string, payload []byte) error {
	logger.Debug().Str("deviceimage", "Receiver.Handle").
		Str("topic", topic).
		Send()
	if err := ctx.Err(); err != nil {
		return err
	}

	if r == nil {
		return fmt.Errorf("device image queue handler is required")
	}

	topicParts, err := ParseTopic(topic)
	if err != nil {
		return err
	}

	msg, err := Unmarshal(payload)
	if err != nil {
		return err
	}

	if msg.Image.DeviceID != topicParts.DeviceID {
		return fmt.Errorf(
			"device image topic device id %d does not match payload device id %d",
			topicParts.DeviceID,
			msg.Image.DeviceID,
		)
	}

	return r.dip.HandleDeviceImageCreated(ctx, msg)
}
