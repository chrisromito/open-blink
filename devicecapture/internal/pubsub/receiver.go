// Package pubsub receiver corresponds with domain/receiver interface
package pubsub

import (
	"encoding/json"
	"fmt"
	"image/jpeg"
	"os"
	"strconv"

	"devicecapture/internal/config"
	"devicecapture/internal/domain/devices"
	"devicecapture/internal/domain/event"
	"devicecapture/internal/domain/receiver"
	"devicecapture/internal/logger"
)

// MqttPublisher implements receiver.Publisher
type MqttPublisher struct {
	client    *MqttClient
	videoPath string
	serverIp  string
}

func NewMqttReceiver(client *MqttClient, conf *config.Config) *MqttPublisher {
	return &MqttPublisher{
		client:    client,
		videoPath: conf.VideoPath,
		serverIp:  conf.ThisIp,
	}
}

// StartSession Start a receiver.CaptureSession
func (r *MqttPublisher) StartSession(deviceId string) (*receiver.CaptureSession, error) {
	s := receiver.NewCaptureSession(deviceId)
	err := r.checkSessionDir(s)
	if err != nil {
		return nil, err
	}
	return s, nil
}

func (r *MqttPublisher) EndSession(session *receiver.CaptureSession) error {
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
func (r *MqttPublisher) PublishFrame(
	frame receiver.Frame,
	framePath string,
	deviceId string,
) error {
	logger.Debug().Str("receiver", "PublishFrame").
		Msgf("mqttreceiver.PublishFrame")
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
func (r *MqttPublisher) WriteFrame(frame receiver.Frame, framePath string) error {
	var fp = framePath
	logger.Debug().Str("receiver", "WriteFrame").
		Msgf("Writing frame to file: %v at %s", frame.Timestamp, fp)
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
		logger.Error().Str("receiver", "WriteFrame").
			Msgf("error encoding frame to JPEG for %s", fp)
		return err2
	}
	return nil
}

func (r *MqttPublisher) PublishDetections(
	deviceID int64,
	img devices.DeviceImage,
	detections []devices.Detection,
) error {
	// Marshal detections to JSON string
	p, jErr := receiver.DetectionsToMsg(r.serverIp, img, detections)
	if jErr != nil {
		logger.Error().Str("service", "receiver").
			Err(jErr).Send()
		return jErr
	}
	// Publish batch to MQTT

	topic := fmt.Sprintf("detections/%d", deviceID)
	qtErr := r.client.Publish(topic, p)
	if qtErr != nil {
		logger.Error().Str("service", "receiver").
			Str("method", "PublishDetections").
			Err(qtErr).Send()
		return qtErr
	}
	return nil
}

func (r *MqttPublisher) PublishDetectionEvent(evt event.DetectionEvent) error {
	dID := strconv.Itoa(int(evt.DeviceID))
	ds, err := event.DetectionStateFromValue(evt.State)
	if err != nil {
		return err
	}
	topic := fmt.Sprintf("detection-event/%s/%s", dID, ds.String())
	payload, jerr := json.Marshal(evt)
	if jerr != nil {
		return jerr
	}
	qtErr := r.client.Publish(topic, payload)
	if qtErr != nil {
		logger.Error().Str("service", "camera").
			Str("method", "PublishDetectionEvent").
			Err(qtErr).Send()
	}
	return qtErr
}

func (r *MqttPublisher) checkSessionDir(session *receiver.CaptureSession) error {
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
