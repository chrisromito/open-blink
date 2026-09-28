package camera

import (
	"context"
	"errors"
	"sync"

	"devicecapture/internal/domain"
	"devicecapture/internal/domain/detection"
	"devicecapture/internal/domain/devices"
	"devicecapture/internal/domain/event"
	"devicecapture/internal/domain/receiver"
	"devicecapture/internal/logger"
)

// DetectionTracker tracks detections for individual devices.
//
//	When a new detection stream starts, we create a DetectionEvent with state = started.
//	A DetectionEvent "ends" when:
//	A. There's a change in the detected objects
//	B. All DetectionEvents that have state = "started" are transitioned to "ended"
//	when the application starts
type DetectionTracker struct {
	mu           sync.Mutex
	deviceStates map[int64]*DeviceEventState
	// EventRepo facilitates persistence
	EventRepo event.DetectionEventRepo
	// DetectionRepo lets us carry [event.DetectionEvent] -> [devices.Detection]
	DetectionRepo devices.DetectionRepo
	// ImageRepo lets us carry [event.DetectionEvent] -> [devices.DeviceImage]
	ImageRepo devices.ImageRepo
	// Publisher broadcasts detection events
	Publisher receiver.Publisher
}

func NewDetectionTracker(deps *domain.Deps) *DetectionTracker {
	return &DetectionTracker{
		EventRepo:     deps.EventRepo,
		DetectionRepo: deps.DetectionRepo,
		ImageRepo:     deps.ImageRepo,
		Publisher:     deps.Publisher,
		deviceStates:  map[int64]*DeviceEventState{},
		mu:            sync.Mutex{},
	}
}

func (dt *DetectionTracker) HasEvent(deviceID int64) bool {
	return dt.hasEvent(deviceID)
}

func (dt *DetectionTracker) CurrentEventID(deviceID int64) int64 {
	return dt.currentEventID(deviceID)
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

// ReceiveDetections updates detection state where possible, returns false if we could not update detection state
func (dt *DetectionTracker) ReceiveDetections(
	ctx context.Context,
	deviceID int64,
	detections []detection.Detection,
) error {
	logger.Debug().Str("tracker", "ReceiveDetections").
		Int64("deviceID", deviceID).Send()
	dt.mu.Lock()
	state, pres := dt.deviceStates[deviceID]
	if !pres {
		dt.mu.Unlock()

		if err := dt.Start(ctx, deviceID, DetectionLabels(detections)); err != nil {
			return err
		}

		dt.mu.Lock()
		state = dt.deviceStates[deviceID]
		if state != nil {
			state.AddDetections(detections)
		}
		dt.mu.Unlock()

		return nil
	}

	if state.Eq(detections) {
		state.AddDetections(detections)
		dt.mu.Unlock()
		return nil
	}

	dt.mu.Unlock()

	if err := dt.End(ctx, deviceID); err != nil {
		return err
	}

	if err := dt.Start(ctx, deviceID, DetectionLabels(detections)); err != nil {
		return err
	}

	dt.mu.Lock()
	state = dt.deviceStates[deviceID]
	if state != nil {
		state.AddDetections(detections)
	}
	dt.mu.Unlock()

	return nil
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
	return dt.Publisher.PublishDetectionEvent(evt)
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

func (dt *DetectionTracker) currentEventID(deviceID int64) int64 {
	dt.mu.Lock()
	defer func(d *DetectionTracker) {
		d.mu.Unlock()
	}(dt)
	dState, pres := dt.deviceStates[deviceID]
	if !pres {
		return int64(0)
	}
	return dState.EventID()
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
	evt, err := s.End(ctx, dt.EventRepo)
	if err != nil {
		return err
	}
	return dt.Publisher.PublishDetectionEvent(evt)
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

type DeviceEventState struct {
	deviceID   int64
	eventID    *int64
	images     []devices.DeviceImage
	detections []detection.Detection
	mu         sync.Mutex
}

func NewDeviceEventState(deviceID int64) *DeviceEventState {
	return &DeviceEventState{
		deviceID:   deviceID,
		images:     []devices.DeviceImage{},
		detections: []detection.Detection{},
		mu:         sync.Mutex{},
	}
}

func (ds *DeviceEventState) EventID() int64 {
	ds.mu.Lock()
	defer ds.mu.Unlock()
	return *ds.eventID
}

func (ds *DeviceEventState) End(
	ctx context.Context,
	eventRepo event.DetectionEventRepo,
) (event.DetectionEvent, error) {
	logger.Debug().Str("DeviceEventState", "End").Send()
	if !ds.HasEvent() {
		return event.DetectionEvent{}, nil
	}
	ds.mu.Lock()
	defer ds.mu.Unlock()
	eventID := *ds.eventID

	de, err := eventRepo.EndEvent(ctx, event.DetectionEvent{ID: eventID})
	if err != nil {
		return event.DetectionEvent{}, err
	}
	return de, err
}

func (ds *DeviceEventState) Eq(detections []detection.Detection) bool {
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

func (ds *DeviceEventState) AddDetection(detection detection.Detection) {
	ds.mu.Lock()
	defer ds.mu.Unlock()
	ds.detections = append(ds.detections, detection)
}

func (ds *DeviceEventState) AddDetections(detections []detection.Detection) {
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

func DetectionLabels(detections []detection.Detection) []string {
	var labels []string
	for _, d := range detections {
		labels = append(labels, d.Label)
	}
	return labels
}