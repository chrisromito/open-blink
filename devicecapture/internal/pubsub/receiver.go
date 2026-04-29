// Package pubsub receiver corresponds with domain/receiver interface
package pubsub

import (
	"context"
	"devicecapture/internal/config"
	"devicecapture/internal/domain/receiver"
	"fmt"
	"image/jpeg"
	"log"
	"os"
)

// MqttReceiver implements receiver.FrameRepository
type MqttReceiver struct {
	client    *MqttClient
	videoPath string
	serverIp  string
	Session   *receiver.CaptureSession
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
	r.Session = s
	err := r.checkSessionDir()
	if err != nil {
		return nil, err
	}
	return s, nil
}

func (r *MqttReceiver) checkSessionDir() error {
	dir := fmt.Sprintf("%s/%s-%v", r.videoPath, r.Session.DeviceID, r.Session.StartedAt)
	_, err := os.Stat(dir)
	if os.IsNotExist(err) {
		err = os.MkdirAll(dir, 0755)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *MqttReceiver) EndSession() error {
	topic := fmt.Sprintf("end-stream/%s", r.Session.DeviceID)
	payload := fmt.Sprintf("%s/%s-%v", r.videoPath, r.Session.DeviceID, r.Session.StartedAt)
	err := r.client.Publish(topic, payload)
	if err != nil {
		return err
	}
	return nil
}

// ReceiveFrame publishes Frames (JSON) to "image/<deviceID>"
func (r *MqttReceiver) ReceiveFrame(frame receiver.Frame, framePath string) error {
	log.Printf("mqttreceiver.ReceiveFrame")
	var fp = framePath
	if framePath == "" {
		fp = receiver.FramePath(r.videoPath, r.Session, frame)
	}
	log.Printf("Writing frame to file: %v at %s", frame.Timestamp, fp)
	f, err := os.Create(fp)
	defer f.Close()
	if err != nil {
		log.Printf("error writing frame to file %v @ %s", frame.Timestamp, fp)
		return err
	}
	err2 := jpeg.Encode(f, frame.Image, nil)
	if err2 != nil {
		log.Printf("error encoding frame to JPEG for %s", fp)
		return err2
	}
	payload, err3 := r.FrameToJson(r.serverIp, frame)
	if err3 != nil {
		log.Printf("error from FrameToJson for %s", fp)
		return err3
	}
	topic := fmt.Sprintf("image/%s", r.Session.DeviceID)
	err = r.client.Publish(topic, payload)
	log.Printf("Writing device %s frame to topic: %v ", r.Session.DeviceID, topic)
	if err != nil {
		log.Fatalf("error publishing device %s frame to topic %v", r.Session.DeviceID, topic)
	}
	return nil
}

func (r *MqttReceiver) ReceiveFrameStream(ctx context.Context, imgChan <-chan receiver.Frame) error {
	outChan := make(chan receiver.Frame, 64)
	defer close(outChan)

	// Worker goroutine
	go func() {
		for {
			select {
			case img, ok := <-imgChan:
				if !ok {
					return
				}
				outChan <- img
			case <-ctx.Done():
				return
			}
		}
	}()

	// Receiver goroutine
	go func() {
		for {
			select {
			case img, ok := <-outChan:
				if !ok {
					return
				}
				err := r.ReceiveFrame(img, receiver.FramePath(r.videoPath, r.Session, img))
				if err != nil {
					return
				}
			case <-ctx.Done():
				return
			}
		}
	}()
	<-ctx.Done()
	return nil
}

func (r *MqttReceiver) FrameToJson(thisIp string, frame receiver.Frame) (string, error) {
	fp := receiver.FramePath(r.videoPath, r.Session, frame)
	payload, err := receiver.FrameJson(thisIp, r.Session.DeviceID, fp, frame)
	if err != nil {
		return "", err
	}
	return payload, nil
}
