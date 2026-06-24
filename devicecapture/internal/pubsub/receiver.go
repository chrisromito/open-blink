// Package pubsub receiver corresponds with domain/receiver interface
package pubsub

import (
	"fmt"
	"image/jpeg"
	"os"

	"devicecapture/internal/config"
	"devicecapture/internal/domain/receiver"
	"devicecapture/internal/logger"
)

// MqttReceiver implements receiver.FrameRepository
type MqttReceiver struct {
	client    *MqttClient
	videoPath string
	serverIp  string
}

func NewMqttReceiver(client *MqttClient, conf *config.Config) *MqttReceiver {
	return &MqttReceiver{
		client:    client,
		videoPath: conf.VideoPath,
		serverIp:  conf.ThisIp,
	}
}

// StartSession Start a receiver.CaptureSession
func (r *MqttReceiver) StartSession(deviceId string) (*receiver.CaptureSession, error) {
	s := receiver.NewCaptureSession(deviceId)
	err := r.checkSessionDir(s)
	if err != nil {
		return nil, err
	}
	return s, nil
}

func (r *MqttReceiver) EndSession(session *receiver.CaptureSession) error {
	if session == nil {
		return nil
	}
	topic := fmt.Sprintf("end-stream/%s", session.DeviceID)
	payload := fmt.Sprintf("%s/%s-%v", r.videoPath, session.DeviceID, session.StartedAt)
	err := r.client.Publish(topic, payload)
	if err != nil {
		return err
	}
	return nil
}

// PublishFrame publishes Frames (JSON) to "image/<deviceID>"
func (r *MqttReceiver) PublishFrame(frame receiver.Frame, framePath string, deviceId string) error {
	logger.Debug().Msgf("mqttreceiver.PublishFrame")
	payload, err := receiver.FrameJson(r.serverIp, deviceId, framePath, frame)
	if err != nil {
		logger.Error().Msgf("error from FrameJson for %s", framePath)
		return err
	}
	topic := fmt.Sprintf("image/%s", deviceId)
	err = r.client.Publish(topic, payload)
	logger.Debug().Msgf("Writing device %s frame to topic: %v ", deviceId, topic)
	if err != nil {
		logger.Error().Msgf("error publishing device %s frame to topic %v", deviceId, topic)
		return err
	}
	return nil
}

// WriteFrame writes [receiver.Frame] to the FileSystem
func (r *MqttReceiver) WriteFrame(frame receiver.Frame, framePath string) error {
	var fp = framePath
	logger.Debug().Msgf("Writing frame to file: %v at %s", frame.Timestamp, fp)
	f, err := os.Create(fp)
	defer func(f *os.File) {
		_ = f.Close()
	}(f)
	if err != nil {
		logger.Error().Msgf("error writing frame to file %v @ %s", frame.Timestamp, fp)
		return err
	}
	err2 := jpeg.Encode(f, frame.Image, nil)
	if err2 != nil {
		logger.Error().Msgf("error encoding frame to JPEG for %s", fp)
		return err2
	}
	return nil
}

func (r *MqttReceiver) checkSessionDir(session *receiver.CaptureSession) error {
	dir := fmt.Sprintf("%s/%s-%v", r.videoPath, session.DeviceID, session.StartedAt)
	_, err := os.Stat(dir)
	if os.IsNotExist(err) {
		err = os.MkdirAll(dir, 0755)
		if err != nil {
			return err
		}
	}
	return nil
}
