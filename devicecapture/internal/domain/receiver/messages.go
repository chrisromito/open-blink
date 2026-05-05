package receiver

import (
	"devicecapture/internal/domain/devices"
	"encoding/json"
	"time"
)

type FrameMsg struct {
	DeviceId  string `json:"device_id"`
	FileName  string `json:"file_name"`
	Timestamp int64  `json:"timestamp"`
	Url       string `json:"url"`
}

func (fm *FrameMsg) MarshalJSON() ([]byte, error) {
	return json.Marshal(fm)
}

func (fm *FrameMsg) UnmarshalJSON(b []byte) error {
	var value FrameMsg
	err := json.Unmarshal(b, &value)
	return err
}

func FrameJson(thisIp string, deviceId string, filePath string, fr Frame) (string, error) {
	var msg = FrameMsg{
		DeviceId:  deviceId,
		FileName:  filePath,
		Timestamp: fr.Timestamp,
		Url:       thisIp + filePath,
	}
	value, err := json.Marshal(msg)
	if err != nil {
		return "", err
	}
	return string(value), nil
}

type DetectionMsg struct {
	ID         int64       `db:"id" json:"id"`
	DeviceID   int64       `db:"device_id" json:"device_id"`
	ImageID    *int64      `db:"image_id" json:"image_id"`
	CreatedAt  time.Time   `db:"created_at" json:"created_at"`
	Label      string      `db:"label" json:"label"`
	Confidence float64     `db:"confidence" json:"confidence"`
	Bbox       [][]float64 `db:"bbox" json:"bbox"`
	Url        string      `json:"url"`
}

func DetectionToMsg(thisIp string, filePath string, d devices.Detection) (string, error) {
	var msg = DetectionMsg{
		ID:         d.ID,
		DeviceID:   d.DeviceID,
		ImageID:    d.ImageID,
		CreatedAt:  d.CreatedAt,
		Label:      d.Label,
		Confidence: d.Confidence,
		Bbox:       d.Bbox,
		Url:        thisIp + filePath,
	}
	value, err := json.Marshal(msg)
	if err != nil {
		return "", err
	}
	return string(value), nil
}
