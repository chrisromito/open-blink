// Package pubsub queue corresponds with domain/queue interface
package pubsub

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"devicecapture/internal/camera"
	"devicecapture/internal/config"
	"devicecapture/internal/domain"
	"devicecapture/internal/domain/detection"
	"devicecapture/internal/domain/queue"
	"devicecapture/internal/logger"
	"devicecapture/internal/pubsub/detectionevent"
	"devicecapture/internal/pubsub/detections"
	"devicecapture/internal/pubsub/deviceimage"
	"golang.org/x/sync/errgroup"
)

type mqttEvent struct {
	Topic   string
	Payload []byte
}

// MqttReceiver implements queue.Receiver
type MqttReceiver struct {
	Deps             *domain.Deps
	Conf             *config.Config
	client           *MqttClient
	publisher        queue.Publisher
	detectionTracker *detections.DetectionTracker
	cameraService    *camera.CameraService
}

func NewMqttReceiver(
	deps *domain.Deps,
	conf *config.Config,
	client *MqttClient,
	publisher queue.Publisher,
) *MqttReceiver {
	var detectionTracker *detections.DetectionTracker
	if deps != nil {
		detectionTracker = detections.NewDetectionTracker(deps)
	}
	cs := camera.NewCameraService(
		conf,
		deps,
		detection.NewObjectDetectionService(conf),
	)

	return &MqttReceiver{
		Deps:             deps,
		Conf:             conf,
		client:           client,
		publisher:        publisher,
		detectionTracker: detectionTracker,
		cameraService:    cs,
	}
}

func (mr *MqttReceiver) Run(ctx context.Context) error {
	g, c := errgroup.WithContext(ctx)

	g.Go(func() error {
		return mr.SubscribeDeviceImage(c)
	})

	// Detection goroutine
	g.Go(func() error {
		return mr.SubscribeDetection(ctx)
	})

	g.Go(func() error {
		return mr.SubscribeDetectionEvent(ctx)
	})

	g.Go(func() error {
		return mr.SubscribeMotion(ctx)
	})

	return g.Wait()
}

// SubscribeDeviceImage deviceImage -> detections
func (mr *MqttReceiver) SubscribeDeviceImage(ctx context.Context) error {
	deviceImageChan := make(chan mqttEvent, 32)
	defer close(deviceImageChan)

	deviceImageMqttHandler := MessageHandler(func(topic string, payload []byte) error {
		select {
		case <-ctx.Done():
			return nil
		case deviceImageChan <- mqttEvent{
			Topic:   topic,
			Payload: payload,
		}:
		}
		return nil
	})

	// open-blink/devices/+/images/created reads images and writes detections
	deviceImageSubErr := mr.client.Subscribe(
		deviceimage.SubscribeCreatedTopic(),
		deviceImageMqttHandler,
	)
	if deviceImageSubErr != nil {
		return deviceImageSubErr
	}

	deviceImageProcessor := deviceimage.NewDeviceImageProcessor(
		mr.Conf,
		mr.Deps,
		detection.NewObjectDetectionService(mr.Conf),
	)
	receiver := deviceimage.NewReceiver(deviceImageProcessor)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case msg := <-deviceImageChan:
			err := receiver.Handle(ctx, msg.Topic, msg.Payload)
			if err != nil {
				logger.Error().
					Str("MqttReceiver", "ReceiveDeviceImage").
					Str("topic", msg.Topic).
					Err(err).
					Send()
				return err
			}
		}
	}
}

// SubscribeDetection reads [devices.Detection] and writes [event.DetectionEvent]
func (mr *MqttReceiver) SubscribeDetection(ctx context.Context) error {
	detectionChan := make(chan mqttEvent)
	defer close(detectionChan)
	detectionHandler := MessageHandler(func(topic string, payload []byte) error {
		select {
		case detectionChan <- mqttEvent{
			Topic:   topic,
			Payload: payload,
		}:
		case <-ctx.Done():
			return nil
		}
		return nil
	})
	detectionSubErr := mr.client.Subscribe(
		detections.SubscribeTopic(),
		detectionHandler,
	)
	if detectionSubErr != nil {
		return detectionSubErr
	}

	tracker := detections.NewDetectionTracker(mr.Deps)
	receiver := detections.NewReceiver(tracker)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case msg := <-detectionChan:
			logger.Debug().
				Str("MqttReceiver", "ReceiveDetection").
				Str("topic", msg.Topic).
				Msgf("receiver.Handle(ctx, %s, payload)", msg.Topic)
			err := receiver.Handle(ctx, msg.Topic, msg.Payload)
			if err != nil {
				logger.Error().
					Str("MqttReceiver", "ReceiveDetection").
					Str("topic", msg.Topic).
					Err(err).
					Send()
				return err
			}
		}
	}
}

//func (mr *MqttReceiver) ReceiveDetection(ctx context.Context, topic string, payload []byte) error {
//	if err := ctx.Err(); err != nil {
//		return err
//	}
//
//	if mr == nil {
//		return fmt.Errorf("receive detection: receiver is nil")
//	}
//	if mr.Deps == nil {
//		return fmt.Errorf("receive detection: deps are required")
//	}
//	if mr.detectionTracker == nil {
//		mr.detectionTracker = detections.NewDetectionTracker(mr.Deps)
//	}
//
//	receiver := detections.NewReceiver(detections.HandlerFunc(
//		func(ctx context.Context, msg detections.Message) error {
//			if msg.Image.DeviceID != 0 && msg.Image.DeviceID != msg.DeviceID {
//				return fmt.Errorf(
//					"detections payload image device id %d does not match payload device id %d",
//					msg.Image.DeviceID,
//					msg.DeviceID,
//				)
//			}
//
//			trackerDetections := trackerDetectionsFromDeviceDetections(msg.Detections)
//
//			logger.Debug().
//				Str("receiver", "ReceiveDetection").
//				Str("topic", topic).
//				Int64("deviceID", msg.DeviceID).
//				Int("detectionCount", len(trackerDetections)).
//				Send()
//
//			if err := mr.detectionTracker.ReceiveDetections(
//				ctx,
//				msg.DeviceID,
//				trackerDetections,
//			); err != nil {
//				return fmt.Errorf("receive detections for device %d: %w", msg.DeviceID, err)
//			}
//
//			if msg.Image.ID != 0 {
//				if err := mr.detectionTracker.ReceiveImage(ctx, msg.DeviceID, msg.Image); err != nil {
//					return fmt.Errorf(
//						"track image %d for device %d: %w",
//						msg.Image.ID,
//						msg.DeviceID,
//						err,
//					)
//				}
//			}
//
//			return nil
//		},
//	))
//
//	return receiver.Handle(ctx, topic, payload)
//}

func (mr *MqttReceiver) SubscribeDetectionEvent(ctx context.Context) error {
	// DetectionEvent subscription
	eventChan := make(chan mqttEvent)
	defer close(eventChan)
	eventHandler := MessageHandler(func(topic string, payload []byte) error {
		select {
		case eventChan <- mqttEvent{
			Topic:   topic,
			Payload: payload,
		}:
		case <-ctx.Done():
			return nil
		}
		return nil
	})

	eventSubErr := mr.client.Subscribe(
		detectionevent.SubscribeCreatedTopic(),
		eventHandler,
	)
	if eventSubErr != nil {
		return eventSubErr
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case msg := <-eventChan:
			err := mr.ReceiveDetectionEvent(ctx, msg.Topic, msg.Payload)
			if err != nil {
				logger.Error().
					Str("MqttReceiver", "ReceiveDetectionEvent").
					Str("topic", msg.Topic).
					Err(err).
					Send()
				return err
			}
		}
	}
}

// ReceiveDetectionEvent MqttReceiver implements [queue.Receiver]
func (mr *MqttReceiver) ReceiveDetectionEvent(
	ctx context.Context,
	topic string,
	payload []byte,
) error {
	// FIXME Implement this method
	logger.Debug().Str("receiver", "ReceiveDetectionEvent").
		Str("topic", topic).
		Str("payload", string(payload))
	return nil
}

// SubscribeMotion handles motion-detection subscription and delegation
func (mr *MqttReceiver) SubscribeMotion(ctx context.Context) error {
	// Motion
	motionChan := make(chan mqttEvent)
	defer close(motionChan)

	motionHandler := MessageHandler(func(topic string, payload []byte) error {
		select {
		case motionChan <- mqttEvent{
			Topic:   topic,
			Payload: payload,
		}:
		case <-ctx.Done():
			return nil
		}
		return nil
	})
	motionSubErr := mr.client.Subscribe(
		"open-blink/motion-detected/#",
		motionHandler,
	)
	if motionSubErr != nil {
		return motionSubErr
	}
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case msg := <-motionChan:
			err := mr.ReceiveMotion(ctx, msg.Topic, msg.Payload)
			if err != nil {
				logger.Error().
					Str("MqttReceiver", "ReceiveMotion").
					Str("topic", msg.Topic).
					Err(err).
					Send()
				return err
			}
		}
	}
}

// ReceiveMotion MqttReceiver implements [queue.Receiver]
func (mr *MqttReceiver) ReceiveMotion(ctx context.Context, topic string, _ []byte) error {
	logger.Debug().Str("MqttReceiver", "ReceiveMotion").
		Str("topic", topic).
		Send()

	deviceID, idErr := parseMotionDetectedDeviceID(topic)
	if idErr != nil {
		logger.Error().Err(idErr).Str("MqttReceiver", "ReceiveMotion").Send()
		return idErr
	}
	d, dErr := mr.Deps.DeviceRepo.GetDevice(ctx, deviceID)
	if dErr != nil {
		return dErr
	}
	return mr.cameraService.StreamSnapshots(ctx, d, 15)
}

func parseMotionDetectedDeviceID(topic string) (int64, error) {
	const prefix = "open-blink/motion-detected"

	deviceIDPart, ok := strings.CutPrefix(topic, prefix)
	if !ok {
		return 0, fmt.Errorf("motion topic %q does not start with %q", topic, prefix)
	}

	deviceIDPart = strings.TrimPrefix(deviceIDPart, "/")
	if deviceIDPart == "" {
		return 0, fmt.Errorf("motion topic %q is missing device id", topic)
	}

	if strings.Contains(deviceIDPart, "/") {
		return 0, fmt.Errorf("motion topic %q has invalid device id %q", topic, deviceIDPart)
	}

	deviceID, err := strconv.ParseInt(deviceIDPart, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parse device id from motion topic %q: %w", topic, err)
	}

	if deviceID <= 0 {
		return 0, fmt.Errorf("motion topic %q has invalid device id %d", topic, deviceID)
	}

	return deviceID, nil
}
