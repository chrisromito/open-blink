package receiver

import "encoding/json"

type FrameMsg struct {
	DeviceId  string `json:"device_id"`
	FileName  string `json:"file_name"`
	Timestamp int64  `json:"timestamp"`
}

func (fm *FrameMsg) MarshalJSON() ([]byte, error) {
	return json.Marshal(fm)
}

func (fm *FrameMsg) UnmarshalJSON(b []byte) error {
	var value FrameMsg
	err := json.Unmarshal(b, &value)
	return err
}

func FrameJson(deviceId string, filePath string, fr Frame) (string, error) {
	var msg = FrameMsg{
		DeviceId:  deviceId,
		FileName:  filePath,
		Timestamp: fr.Timestamp}
	value, err := json.Marshal(msg)
	if err != nil {
		return "", err
	}
	return string(value), nil
}
