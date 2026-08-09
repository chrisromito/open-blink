package camera

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"sync"

	"devicecapture/internal/domain"
	"devicecapture/internal/domain/devices"
	"devicecapture/internal/domain/event"
	"devicecapture/internal/logger"
	"devicecapture/internal/pubsub"
	"golang.org/x/sync/errgroup"
)

// FIXME: Refactor this struct + its logic to use [DeviceEventState] instead of tracking
// state internally & making this functionality difficult to debug

// DetectionTracker helps track detections for a given device
type DetectionTracker struct {
	mu           sync.Mutex
	deviceStates map[int64]*DeviceEventState
	// EventRepo facilitates persistence
	EventRepo event.DetectionEventRepo
	// DetectionRepo lets us carry [event.DetectionEvent] -> [devices.Detection]
	DetectionRepo devices.DetectionRepo
	// ImageRepo lets us carry [event.DetectionEvent] -> [devices.DeviceImage]
	ImageRepo devices.ImageRepo
	// Handle updating external subscribers
	mqttClient *pubsub.MqttClient
}

func NewDetectionTracker(
	deps *domain.Deps,
	qtClient *pubsub.MqttClient,
) *DetectionTracker {
	return &DetectionTracker{
		EventRepo:     deps.EventRepo,
		DetectionRepo: deps.DetectionRepo,
		ImageRepo:     deps.ImageRepo,
		mqttClient:    qtClient,
		deviceStates:  map[int64]*DeviceEventState{},
		mu:            sync.Mutex{},
	}
}

func (dt *DetectionTracker) HasEvent(deviceID int64) bool {
	return dt.hasEvent(deviceID)
}

func (dt *DetectionTracker) ReceiveImage(deviceID int64, img devices.DeviceImage) error {
	dt.mu.Lock()
	defer dt.mu.Unlock()
	state, pres := dt.deviceStates[deviceID]
	if !pres {
		return nil
	}
	state.AddImage(img)
	return nil
}

func (dt *DetectionTracker) ReceiveDetections(deviceID int64, detections []devices.Detection) bool {
	dt.mu.Lock()
	defer dt.mu.Unlock()
	state, pres := dt.deviceStates[deviceID]
	if !pres {
		return true
	}
	if !state.Eq(detections) {
		return true
	}
	state.AddDetections(detections)
	return false
}

func (dt *DetectionTracker) Start(ctx context.Context, deviceID int64, labels []string) error {
	logger.Debug().Str("DetectionTracker", "Start").
		Msgf("deviceID %d", deviceID)
	evt, err := dt.EventRepo.StartEvent(ctx, deviceID, labels)
	if err != nil {
		logger.Error().
			Str("detection_event", "End").
			Err(err).
			Send()
		return err
	}
	dt.setEvent(deviceID, evt)
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
		delete(d.deviceStates, deviceID)
		d.mu.Unlock()
	}(dt)
	s, pres := dt.deviceStates[deviceID]
	if !pres {
		return nil
	}
	if s == nil {
		return nil
	}
	// trigger the 'End' workflow for this deviceState before we wrap up
	eventID, err := s.End(
		ctx,
		dt.EventRepo,
		dt.DetectionRepo,
		dt.ImageRepo,
	)
	if err != nil {
		return err
	}
	evt, err := dt.EventRepo.GetEvent(ctx, eventID)
	if err != nil {
		return err
	}
	return PublishDetectionEvent(dt.mqttClient, evt)
}

func (dt *DetectionTracker) hasEvent(deviceID int64) bool {
	dt.mu.Lock()
	defer dt.mu.Unlock()
	dState, pres := dt.deviceStates[deviceID]
	if !pres {
		return false
	}
	return dState.HasEvent()
}

func (dt *DetectionTracker) setEvent(deviceID int64, evt event.DetectionEvent) {
	dt.mu.Lock()
	defer dt.mu.Unlock()
	dState, pres := dt.deviceStates[deviceID]
	// Set up a new instance of DeviceState
	if !pres {
		dt.deviceStates[deviceID] = NewDeviceEventState(deviceID)
		dt.deviceStates[deviceID].SetEvent(evt.ID)
		return
	}
	// If the event ID hasn't been set, set it
	if !dState.HasEvent() {
		dState.SetEvent(evt.ID)
	}
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

type DeviceEventState struct {
	deviceID   int64
	eventID    *int64
	images     []devices.DeviceImage
	detections []devices.Detection
	mu         sync.Mutex
}

func NewDeviceEventState(deviceID int64) *DeviceEventState {
	return &DeviceEventState{
		deviceID:   deviceID,
		images:     []devices.DeviceImage{},
		detections: []devices.Detection{},
		mu:         sync.Mutex{},
	}
}

func (ds *DeviceEventState) End(
	ctx context.Context,
	eventRepo event.DetectionEventRepo,
	detectionRepo devices.DetectionRepo,
	imageRepo devices.ImageRepo,
) (int64, error) {
	if !ds.HasEvent() {
		return 0, nil
	}
	g, c := errgroup.WithContext(ctx)

	ds.mu.Lock()
	defer ds.mu.Unlock()
	eventID := *ds.eventID

	// Write to the eventRepo
	g.Go(func() error {
		_, err := eventRepo.EndEvent(c, event.DetectionEvent{ID: eventID})
		if err != nil {
			logger.Error().
				Err(err).
				Send()
			return err
		}
		logger.Debug().
			Str("detection_event", "endEvent").
			Msgf("DB updated ended_at for event %d", eventID)
		return nil
	})

	// Write to the detectionRepo
	g.Go(func() error {
		var dids []int64
		for _, det := range ds.detections {
			dids = append(dids, det.ID)
		}
		return detectionRepo.SetEvent(c, dids, eventID)
	})

	// write to the imageRepo
	g.Go(func() error {
		// skip work if there isn't any to do
		if len(ds.images) < 1 {
			return nil
		}
		var ids []int64
		for _, img := range ds.images {
			ids = append(ids, img.ID)
		}
		logger.Debug().
			Str("detection_event", "endEvent").
			Msgf("DB updated detections %d for event %d", ids, eventID)
		return imageRepo.SetEvent(c, ids, eventID)
	})

	return eventID, g.Wait()
}

func (ds *DeviceEventState) Eq(detections []devices.Detection) bool {
	return event.LabelsEq(ds.getLabels(), DetectionLabels(detections))
}

func (ds *DeviceEventState) HasEvent() bool {
	ds.mu.Lock()
	defer ds.mu.Unlock()
	return ds.eventID != nil
}

func (ds *DeviceEventState) SetEvent(id int64) {
	ds.mu.Lock()
	defer ds.mu.Unlock()
	ds.eventID = &id
}

func (ds *DeviceEventState) AddImage(img devices.DeviceImage) {
	ds.mu.Lock()
	defer ds.mu.Unlock()
	ds.images = append(ds.images, img)
}

func (ds *DeviceEventState) AddDetection(detection devices.Detection) {
	ds.mu.Lock()
	defer ds.mu.Unlock()
	ds.detections = append(ds.detections, detection)
}

func (ds *DeviceEventState) AddDetections(detections []devices.Detection) {
	ds.mu.Lock()
	defer ds.mu.Unlock()
	for _, d := range detections {
		ds.detections = append(ds.detections, d)
	}
}

func (ds *DeviceEventState) getLabels() []string {
	ds.mu.Lock()
	defer ds.mu.Unlock()
	return DetectionLabels(ds.detections)
}

func DetectionLabels(detections []devices.Detection) []string {
	var labels []string
	for _, d := range detections {
		labels = append(labels, d.Label)
	}
	return labels
}
