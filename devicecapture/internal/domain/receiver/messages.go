package receiver

import (
	"encoding/json"
	"time"

	"devicecapture/internal/domain/devices"
)

// FrameMsg shape of MQTT messages that get published to image/<device_id>
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

// DetectionsMsg shape of MQTT messages that get published to /detections/<device_id>
type DetectionsMsg struct {
	ID         int64         `db:"id"         json:"id"`
	DeviceID   int64         `db:"device_id"  json:"device_id"`
	CreatedAt  time.Time     `db:"created_at" json:"created_at"`
	Url        string        `db:"url"        json:"url"`
	Detections []QtDetection `db:"detections" json:"detections"`
}

// QtDetection shape of nested `detections` within a DetectionsMsg
type QtDetection struct {
	ID         int64       `db:"id"         json:"id"`
	Label      string      `db:"label"      json:"label"`
	Confidence float64     `db:"confidence" json:"confidence"`
	Bbox       [][]float64 `db:"bbox"       json:"bbox"`
}

// DetectionsToMsg get JSON serialized string representation of detections
func DetectionsToMsg(thisIp string, i devices.DeviceImage, ds []devices.Detection) (string, error) {
	p := i.AnnotatedPath
	if p == "" {
		p = i.ImagePath
	}
	var msg = DetectionsMsg{
		ID:         i.ID,
		DeviceID:   i.DeviceID,
		CreatedAt:  i.CreatedAt,
		Url:        thisIp + p,
		Detections: detectionsToMqtt(ds),
	}
	value, err := json.Marshal(msg)
	if err != nil {
		return "", err
	}
	return string(value), nil
}

func detectionsToMqtt(ds []devices.Detection) []QtDetection {
	var accum []QtDetection
	for _, d := range ds {
		accum = append(accum, QtDetection{
			ID:         d.ID,
			Label:      d.Label,
			Confidence: d.Confidence,
			Bbox:       d.Bbox,
		})
	}
	return accum
}
