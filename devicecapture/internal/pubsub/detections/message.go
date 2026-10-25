package detections

import (
	"encoding/json"
	"fmt"

	"devicecapture/internal/domain/devices"
)

type Message struct {
	ServerIP   string              `json:"serverIp,omitempty"`
	DeviceID   int64               `json:"deviceId"`
	Image      devices.DeviceImage `json:"image"`
	Detections []devices.Detection `json:"detections"`
}

func NewMessage(
	serverIP string,
	deviceID int64,
	image devices.DeviceImage,
	detections []devices.Detection,
) Message {
	return Message{
		ServerIP:   serverIP,
		DeviceID:   deviceID,
		Image:      image,
		Detections: detections,
	}
}

func Marshal(msg Message) ([]byte, error) {
	payload, err := json.Marshal(msg)
	if err != nil {
		return nil, fmt.Errorf("marshal detections message: %w", err)
	}

	return payload, nil
}

func Unmarshal(payload []byte) (Message, error) {
	var msg Message
	if err := json.Unmarshal(payload, &msg); err != nil {
		return Message{}, fmt.Errorf("unmarshal detections message: %w", err)
	}

	return msg, nil
}
