package deviceimage

import (
	"fmt"
	"strconv"
	"strings"
)

const (
	Prefix           = "open-blink"
	devicesSegment   = "devices"
	imagesSegment    = "images"
	createdSegment   = "created"
	updatedSegment   = "updated"
	deviceIDWildcard = "+"
)
//
//type SubscribeClient interface {
//	Subscribe(topic string, handler func(topic string, payload []byte) error) error
//}

//func Subscribe(ctx context.Context, client SubscribeClient, receiver *Receiver) error {
//	if err := ctx.Err(); err != nil {
//		return err
//	}
//
//	return client.Subscribe(SubscribeCreatedTopic(), func(topic string, payload []byte) error {
//		return receiver.Handle(ctx, topic, payload)
//	})
//}

// CreatedTopic returns the MQTT topic for DeviceImage-created events.
//
// Shape:
//
//	open-blink/devices/<DeviceID>/images/created
func CreatedTopic(deviceID int64) string {
	return fmt.Sprintf(
		"%s/%s/%d/%s/%s",
		Prefix,
		devicesSegment,
		deviceID,
		imagesSegment,
		createdSegment,
	)
}

// UpdatedTopic returns the MQTT topic for DeviceImage-updated events.
// Shape:
//
//	open-blink/devices/<DeviceID>/images/updated
func UpdatedTopic(deviceID int64) string {
	return fmt.Sprintf(
		"%s/%s/%d/%s/%s",
		Prefix,
		devicesSegment,
		deviceID,
		imagesSegment,
		updatedSegment,
	)
}

// SubscribeCreatedTopic returns the wildcard MQTT topic for all DeviceImage-created events.
// Shape:
//
//	open-blink/devices/+/images/created
func SubscribeCreatedTopic() string {
	return fmt.Sprintf(
		"%s/%s/%s/%s/%s",
		Prefix,
		devicesSegment,
		deviceIDWildcard,
		imagesSegment,
		createdSegment,
	)
}

type TopicParts struct {
	DeviceID int64
	Created  bool
	Updated  bool
}

// ParseTopic parses a concrete DeviceImage-created topic.
// Expected shape:
//
//	open-blink/devices/<DeviceID>/images/created
func ParseTopic(topic string) (TopicParts, error) {
	parts := strings.Split(topic, "/")
	if len(parts) != 5 {
		return TopicParts{}, fmt.Errorf("invalid device image topic %q", topic)
	}

	if parts[0] != Prefix {
		return TopicParts{}, fmt.Errorf("invalid device image topic prefix %q", parts[0])
	}

	if parts[1] != devicesSegment {
		return TopicParts{}, fmt.Errorf("invalid device image topic segment %q", parts[1])
	}

	if parts[3] != imagesSegment {
		return TopicParts{}, fmt.Errorf("invalid device image topic segment %q", parts[3])
	}

	if parts[4] != createdSegment && parts[4] != updatedSegment {
		return TopicParts{}, fmt.Errorf("invalid device image topic event %q", parts[4])
	}

	deviceID, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil {
		return TopicParts{}, fmt.Errorf("invalid device image device id %q: %w", parts[2], err)
	}

	return TopicParts{
		DeviceID: deviceID,
		Created:  parts[4] == createdSegment,
		Updated:  parts[4] == updatedSegment,
	}, nil
}
