package pubsub

import (
	"context"
	"devicecapture/device/receiver"
	"encoding/json"
	"fmt"
	"image/jpeg"
	"log"
	"os"
	"sync"
	"time"
)

type MqttReceiver struct {
	client    *MqttClient
	videoPath string
	Session   receiver.CaptureSession
}

type FrameMsg struct {
	DeviceId  string `json:"device_id"`
	FileName  string `json:"file_name"`
	Timestamp int64  `json:"timestamp"`
}

func NewMqttReceiver(client *MqttClient, videoPath string) *MqttReceiver {
	vp := videoPath
	if videoPath == "" {
		vp = "/videos"
	}
	return &MqttReceiver{
		client:    client,
		videoPath: vp,
	}
}

func (r *MqttReceiver) StartSession(deviceId string) (*receiver.CaptureSession, error) {
	ts := time.Now().UnixMilli()
	s := receiver.CaptureSession{DeviceId: deviceId, StartedAt: ts}
	r.Session = s
	err := r.checkSessionDir()
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *MqttReceiver) checkSessionDir() error {
	dir := fmt.Sprintf("%s/%s-%v", r.videoPath, r.Session.DeviceId, r.Session.StartedAt)
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
	topic := fmt.Sprintf("end-stream/%s", r.Session.DeviceId)
	payload := fmt.Sprintf("%s/%s-%v", r.videoPath, r.Session.DeviceId, r.Session.StartedAt)
	err := r.client.Publish(topic, payload)
	if err != nil {
		return err
	}
	return nil
}

func (r *MqttReceiver) ReceiveFrame(frame receiver.Frame) error {
	var wg sync.WaitGroup

	wg.Add(1)

	go func() {
		defer wg.Done()
		fp := FramePath(r.videoPath, r.Session.StartedAt, r.Session.DeviceId, frame.Timestamp)
		log.Printf("Writing frame to file: %v at %s", frame.Timestamp, fp)
		f, err := os.Create(fp)
		defer f.Close()
		if err != nil {
			return
		}
		err2 := jpeg.Encode(f, frame.Image, nil)
		if err2 != nil {
			return
		}
		payload, err3 := r.FrameToJson(frame)
		if err3 != nil {
			return
		}
		topic := fmt.Sprintf("image/%s", r.Session.DeviceId)
		err = r.client.Publish(topic, payload)
		if err != nil {
			return
		}
	}()

	wg.Wait()
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
				err := r.ReceiveFrame(img)
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

func (r *MqttReceiver) FrameToJson(frame receiver.Frame) (string, error) {
	fp := FramePath(r.videoPath, r.Session.StartedAt, r.Session.DeviceId, frame.Timestamp)
	payload, err := FrameJson(r.Session.DeviceId, fp, frame)
	if err != nil {
		return "", err
	}
	return payload, nil
}

func FrameJson(deviceId string, fileName string, fr receiver.Frame) (string, error) {
	var msg = FrameMsg{
		DeviceId:  deviceId,
		FileName:  fileName,
		Timestamp: fr.Timestamp}
	value, err := json.Marshal(msg)
	if err != nil {
		return "", err
	}
	return string(value), nil
}

func FramePath(dirPath string, startStamp int64, deviceId string, timestamp int64) string {
	return fmt.Sprintf("%s/%s-%v/output-%s-%v.jpeg", dirPath, deviceId, startStamp, deviceId, timestamp)
}
