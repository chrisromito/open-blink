package deviceimage

import (
	"encoding/json"
	"fmt"

	"devicecapture/internal/domain/devices"
)

// Message is the wire-format payload for DeviceImage-created messages.
// The full DeviceImage is included so receivers can process directly from the
// event payload, while still being free to reload by ID if they need fresh DB state.
type Message struct {
	Image devices.DeviceImage `json:"device_image"`
}

func NewMessage(image devices.DeviceImage) Message {
	return Message{
		Image: image,
	}
}

func Marshal(msg Message) ([]byte, error) {
	payload, err := json.Marshal(msg)
	if err != nil {
		return nil, fmt.Errorf("marshal device image message: %w", err)
	}

	return payload, nil
}

func Unmarshal(payload []byte) (Message, error) {
	var msg Message
	if err := json.Unmarshal(payload, &msg); err != nil {
		return Message{}, fmt.Errorf("unmarshal device image message: %w", err)
	}

	return msg, nil
}
