package camera

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"sync"

	"devicecapture/internal/domain/event"
	"devicecapture/internal/logger"
	"devicecapture/internal/pubsub"
)

type DetectionTracker struct {
	mu sync.Mutex
	// labels map deviceID -> labels
	labels map[int64][]string
	// events map deviceID -> DetectionEvent
	events map[int64]event.DetectionEvent
	// EventRepo facilitates persistence
	EventRepo  event.DetectionEventRepo
	mqttClient *pubsub.MqttClient
}

func NewDetectionTracker(
	er event.DetectionEventRepo,
	qtClient *pubsub.MqttClient,
) *DetectionTracker {
	return &DetectionTracker{
		EventRepo:  er,
		mqttClient: qtClient,
		events:     map[int64]event.DetectionEvent{},
		labels:     map[int64][]string{},
		mu:         sync.Mutex{},
	}
}

func (dt *DetectionTracker) HasEvent(deviceID int64) bool {
	return dt.hasEvent(deviceID)
}

func (dt *DetectionTracker) LabelsEq(deviceID int64, labels []string) bool {
	curr := dt.getLabels(deviceID)
	value := event.LabelsEq(curr, labels)
	if !value {
		logger.Info().Str("DetectionTracker", "labelsEq").
			Any("cur", curr).
			Any("next", labels).
			Send()
	}
	return value
}

func (dt *DetectionTracker) Start(ctx context.Context, deviceID int64, labels []string) error {
	logger.Debug().Str("DetectionTracker", "Start").
		Msgf("deviceID %d", deviceID)
	if dt.HasEvent(deviceID) {
		logger.Warn().Str("DetectionTracker", "Start").
			Msgf("previous event for %d was not ended", deviceID)
		err := dt.End(ctx, deviceID)
		if err != nil {
			return err
		}
	}
	evt, err := dt.EventRepo.StartEvent(ctx, deviceID, labels)
	if err != nil {
		logger.Error().
			Str("detection_event", "End").
			Err(err).
			Send()
		return err
	}
	dt.setEvent(deviceID, evt)
	dt.setLabels(deviceID, labels)
	logger.Debug().Str("DetectionTracker", "Start").
		Msgf("returning for %d", deviceID)
	err = PublishDetectionEvent(dt.mqttClient, evt)
	return err
}

// End ensures that the EventRepo marks the [event.DetectionEvent] as [event.Ended]
func (dt *DetectionTracker) End(ctx context.Context, deviceID int64) error {
	logger.Debug().Str("DetectionTracker", "End").
		Msgf("deviceID %d", deviceID)
	// return an error if we don't have an event started
	if !dt.hasEvent(deviceID) {
		return errors.New("failed to end DetectionEvent because it has not been started")
	}
	return dt.endEvent(ctx, deviceID)
}

func (dt *DetectionTracker) endEvent(ctx context.Context, deviceID int64) error {
	dt.mu.Lock()
	defer func(d *DetectionTracker) {
		delete(d.events, deviceID)
		delete(d.labels, deviceID)
		d.mu.Unlock()
	}(dt)
	evt, pres := dt.events[deviceID]
	if !pres {
		logger.Warn().
			Str("detection_event", "endEvent").
			Msgf("expected event for deviceID %d but now I'm empty handed", deviceID)
		return nil
	}
	toPublish, err := dt.EventRepo.EndEvent(ctx, evt)
	if err != nil {
		return err
	}
	err = PublishDetectionEvent(dt.mqttClient, toPublish)
	if err != nil {
		logger.Error().
			Err(err).
			Send()
		return err
	}
	logger.Debug().Str("DetectionTracker", "endEvent").
		Str("status", "success").
		Msgf("deviceID %d", deviceID)
	return nil
}

func (dt *DetectionTracker) getLabels(deviceID int64) []string {
	dt.mu.Lock()
	defer dt.mu.Unlock()
	return dt.labels[deviceID]
}

func (dt *DetectionTracker) setLabels(deviceID int64, ls []string) bool {
	dt.mu.Lock()
	defer dt.mu.Unlock()
	curr := dt.labels[deviceID]
	dt.labels[deviceID] = ls
	return event.LabelsEq(curr, ls)
}

func (dt *DetectionTracker) hasEvent(deviceID int64) bool {
	dt.mu.Lock()
	defer dt.mu.Unlock()
	_, pres := dt.events[deviceID]
	return pres
}

func (dt *DetectionTracker) setEvent(deviceID int64, evt event.DetectionEvent) {
	dt.mu.Lock()
	defer dt.mu.Unlock()
	dt.events[deviceID] = evt
}

// PublishDetectionEvent detection event JSON to MQTT topic "detection-event/DEVICE_ID/STATE_STRING"
//
//	ex. "detection-event/1/started", "detection-event/1/ended"
func PublishDetectionEvent(qt *pubsub.MqttClient, evt event.DetectionEvent) error {
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
	logger.Debug().Str("detection_event", "PublishDetectionEvent").
		Str("payload", string(payload)).
		Str("state", ds.String()).
		Str("topic", topic).
		Send()
	return qt.Publish(topic, payload)
}
