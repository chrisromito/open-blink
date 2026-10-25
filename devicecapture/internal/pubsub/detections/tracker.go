package detections

import (
	"context"
	"fmt"
	"slices"

	"devicecapture/internal/domain"
	"devicecapture/internal/domain/devices"
	"devicecapture/internal/domain/event"
	"devicecapture/internal/domain/queue"
	"devicecapture/internal/logger"
)

// DetectionTracker tracks detection events for individual devices.
//
// The DB is the source of truth for active events and labels.
type DetectionTracker struct {
	// EventRepo facilitates persistence.
	EventRepo event.DetectionEventRepo
	// DetectionRepo lets us carry [event.DetectionEvent] -> [devices.Detection].
	DetectionRepo devices.DetectionRepo
	// ImageRepo lets us carry [event.DetectionEvent] -> [devices.DeviceImage].
	ImageRepo devices.ImageRepo
	// Publisher broadcasts detection events.
	Publisher queue.Publisher
}

func NewDetectionTracker(deps *domain.Deps) *DetectionTracker {
	return &DetectionTracker{
		EventRepo:     deps.EventRepo,
		DetectionRepo: deps.DetectionRepo,
		ImageRepo:     deps.ImageRepo,
		Publisher:     deps.Publisher,
	}
}

func (dt *DetectionTracker) HasEvent(deviceID int64) bool {
	evt, err := dt.EventRepo.GetLatest(context.Background(), deviceID)
	if err != nil {
		return false
	}

	return evt.ID != 0 && evt.State == int(event.Started)
}

func (dt *DetectionTracker) CurrentEventID(ctx context.Context, deviceID int64) int64 {
	evt, err := dt.EventRepo.GetLatest(ctx, deviceID)
	if err != nil {
		return 0
	}

	if evt.ID == 0 || evt.State != int(event.Started) {
		return 0
	}

	return evt.ID
}

// ReceiveDetections updates detection event state using the DB as the source of truth.
func (dt *DetectionTracker) ReceiveDetections(
	ctx context.Context,
	deviceID int64,
	detections []devices.Detection,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	labels := DetectionLabels(detections)

	logger.Debug().
		Str("tracker", "ReceiveDetections").
		Int64("deviceID", deviceID).
		Interface("labels", labels).
		Send()

	latestEvent, err := dt.EventRepo.GetLatest(ctx, deviceID)
	if err != nil {
		return fmt.Errorf("get latest detection event for device %d: %w", deviceID, err)
	}
	var dids []int64
	for _, det := range detections {
		dids = append(dids, det.ID)
	}
	// Empty event or ended event means we need to start a new one, then update Detection.EventIDs
	if latestEvent.ID == 0 || latestEvent.State != int(event.Started) {
		evt, err := dt.Start(ctx, deviceID, labels)
		if err != nil {
			return err
		}
		err = dt.DetectionRepo.SetEvent(ctx, dids, evt.ID)
		if err != nil {
			return err
		}
	}

	// If the detection labels didn't change, just update Detection.EventIDs
	if event.LabelsEq(latestEvent.Labels, labels) {
		err = dt.DetectionRepo.SetEvent(ctx, dids, latestEvent.ID)
		return err
	}

	// An event exists and the labels changed, so we need to end the previous
	if err = dt.endEvent(ctx, latestEvent); err != nil {
		return err
	}

	// start a new one with the next set of labels
	evt, err := dt.Start(ctx, deviceID, labels)
	if err != nil {
		return err
	}
	// update Detection.EventIDs
	return dt.DetectionRepo.SetEvent(ctx, dids, evt.ID)
}

func (dt *DetectionTracker) Start(ctx context.Context, deviceID int64, labels []string) (event.DetectionEvent, error) {
	logger.Debug().
		Str("DetectionTracker", "Start").
		Int64("deviceID", deviceID).
		Interface("labels", labels).
		Send()

	evt, err := dt.EventRepo.StartEvent(ctx, deviceID, labels)
	if err != nil {
		logger.Error().
			Str("detection_event", "Start").
			Int64("deviceID", deviceID).
			Err(err).
			Send()

		return event.DetectionEvent{}, err
	}

	return evt, dt.Publisher.PublishDetectionEvent(ctx, evt)
}

// End marks the latest started event for a device as ended.
func (dt *DetectionTracker) End(ctx context.Context, deviceID int64) error {
	logger.Debug().
		Str("DetectionTracker", "End").
		Int64("deviceID", deviceID).
		Send()

	latestEvent, err := dt.EventRepo.GetLatest(ctx, deviceID)
	if err != nil {
		return fmt.Errorf("get latest detection event for device %d: %w", deviceID, err)
	}

	if latestEvent.ID == 0 || latestEvent.State != int(event.Started) {
		return nil
	}

	return dt.endEvent(ctx, latestEvent)
}

func (dt *DetectionTracker) endEvent(ctx context.Context, evt event.DetectionEvent) error {
	endedEvent, err := dt.EventRepo.EndEvent(ctx, event.DetectionEvent{ID: evt.ID})
	if err != nil {
		return fmt.Errorf("end detection event %d: %w", evt.ID, err)
	}

	return dt.Publisher.PublishDetectionEvent(ctx, endedEvent)
}

func DetectionLabels(detections []devices.Detection) []string {
	labels := make([]string, 0, len(detections))
	for _, d := range detections {
		labels = append(labels, d.Label)
	}
	slices.Sort(labels)
	return labels
}
