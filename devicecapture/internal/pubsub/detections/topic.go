package detections

import (
	"fmt"
	"strconv"
	"strings"
)

const (
	Prefix            = "open-blink"
	devicesSegment    = "devices"
	detectionsSegment = "detections"
	createdSegment    = "created"
	deviceIDWildcard  = "+"
	updatedSegment    = "updated" // Added new segment
)

func Topic(deviceID int64) string {
	return fmt.Sprintf(
		"%s/%s/%d/%s/%s",
		Prefix,
		devicesSegment,
		deviceID,
		detectionsSegment,
		createdSegment,
	)
}

func SubscribeTopic() string {
	return fmt.Sprintf(
		"%s/%s/%s/%s/%s",
		Prefix,
		devicesSegment,
		deviceIDWildcard,
		detectionsSegment,
		createdSegment,
	)
}

type TopicParts struct {
	DeviceID int64
}

func ParseTopic(topic string) (TopicParts, error) {
	parts := strings.Split(topic, "/")
	if len(parts) != 5 {
		return TopicParts{}, fmt.Errorf("invalid detections topic %q", topic)
	}

	if parts[0] != Prefix {
		return TopicParts{}, fmt.Errorf("invalid detections topic prefix %q", parts[0])
	}

	if parts[1] != devicesSegment {
		return TopicParts{}, fmt.Errorf("invalid detections topic segment %q", parts[1])
	}

	if parts[3] != detectionsSegment {
		return TopicParts{}, fmt.Errorf("invalid detections topic segment %q", parts[3])
	}

	if parts[4] != createdSegment && parts[4] != updatedSegment { // Allow for updated segment
		return TopicParts{}, fmt.Errorf("invalid detections topic event %q", parts[4])
	}

	deviceID, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil {
		return TopicParts{}, fmt.Errorf("invalid detections device id %q: %w", parts[2], err)
	}

	return TopicParts{
		DeviceID: deviceID,
	}, nil
}
