package detectionevent

import (
	"encoding/json"
	"fmt"

	"devicecapture/internal/domain/event"
)

type Message struct {
	DeviceID int64                `json:"deviceId"`
	Event    event.DetectionEvent `json:"event"`
}

func NewMessage(deviceID int64, event event.DetectionEvent) Message {
	return Message{DeviceID: deviceID, Event: event}
}

func Marshal(msg Message) ([]byte, error) {
	payload, err := json.Marshal(msg)
	if err != nil {
		return nil, fmt.Errorf("marshal detectionevent message: %w", err)
	}

	return payload, nil
}

func Unmarshal(payload []byte) (Message, error) {
	var msg Message
	if err := json.Unmarshal(payload, &msg); err != nil {
		return Message{}, fmt.Errorf("unmarshal detectionevent message: %w", err)
	}

	return msg, nil
}
