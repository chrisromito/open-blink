package detectionevent

import "fmt"

// detectionevent topic is shaped like "open-blink/devices/{DEVICE_ID}/event/{updated | created}"
const (
	Prefix         = "open-blink"
	devicesSegment = "devices"
	eventSegment   = "event"
	idWildcard   = "+"
	createdSegment = "created"
	updatedSegment = "updated"
)

func CreatedTopic(deviceID int64) string {
	return fmt.Sprintf(
		"%s/%s/%d/%s/%s",
		Prefix,
		devicesSegment,
		deviceID,
		eventSegment,
		createdSegment,
	)
}

func UpdatedTopic(deviceID int64) string {
	return fmt.Sprintf(
		"%s/%s/%d/%s/%s",
		Prefix,
		devicesSegment,
		deviceID,
		eventSegment,
		updatedSegment,
	)
}


func SubscribeCreatedTopic() string {
	return fmt.Sprintf(
		"%s/%s/%s/%s/%s",
		Prefix,
		devicesSegment,
		idWildcard,
		eventSegment,
		createdSegment,
	)
}
