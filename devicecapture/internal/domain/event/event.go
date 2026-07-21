package event

import (
	"context"
	"errors"
	"time"
)

type DetectionState int

const (
	Unknown DetectionState = 0
	Down    DetectionState = 1
	Started DetectionState = 2
	Ended   DetectionState = 3
)

var stateName = map[DetectionState]string{
	Unknown: "unknown",
	Down:    "down",
	Started: "started",
	Ended:   "ended",
}

func (ds DetectionState) String() string {
	return stateName[ds]
}

func DetectionStateFromValue(value int) (DetectionState, error) {
	switch value {
	case 1:
		return Down, nil
	case 2:
		return Started, nil
	default:
		return Unknown, errors.New("invalid value for DetectionState")
	case 3:
		return Ended, nil
	}
}

type DetectionEvent struct {
	ID        int64     `db:"id"         json:"id"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	EndedAt   time.Time `db:"ended_at"   json:"ended_at"`
	DeviceID  int64     `db:"device_id"  json:"device_id"`
	State     int       `db:"state"      json:"state"`
	Labels    []string  `db:"labels"     json:"labels"`
}

type QueryParams struct {
	DeviceID int64
	Page     int
	State    DetectionState
	// Start is the minimum created_at value
	Start time.Time
	// End is the (optional) maximum created_at value
	End time.Time
}

type DetectionForImage struct {
	ID         int64   `db:"id"         json:"id"`
	Confidence float64 `db:"confidence" json:"confidence"`
	Label      string  `db:"label"      json:"label"`
}

type ImageDetails struct {
	ID         int64               `db:"id"         json:"id"`
	CreatedAt  time.Time           `db:"created_at" json:"created_at"`
	ImageUrl   string              `db:"image_url"  json:"image_url"`
	Detections []DetectionForImage `db:"detections" json:"detections"`
}

type DetectionDetail struct {
	ID        int64          `db:"id"         json:"id"`
	CreatedAt time.Time      `db:"created_at" json:"created_at"`
	EndedAt   time.Time      `db:"ended_at"   json:"ended_at"`
	DeviceID  int64          `db:"device_id"  json:"device_id"`
	State     int            `db:"state"      json:"state"`
	Details   []ImageDetails `db:"details"    json:"details"`
}

type DetectionEventRepo interface {
	// StartEvent creates a DetectionEvent with the given deviceID and labels
	StartEvent(ctx context.Context, deviceID int64, labels []string) (DetectionEvent, error)
	// EndEvent updates a DetectionEvent by flagging it as Ended
	EndEvent(ctx context.Context, e DetectionEvent) (DetectionEvent, error)
	// GetDetectionEvents returns paginated slices of DetectionEvent based on the provided QueryParams
	GetDetectionEvents(ctx context.Context, p QueryParams) ([]DetectionEvent, error)
	// GetDetectionsForEvent returns DeviceDetections that happened in the date range
	// for a given [DetectionEvent]
	GetDetectionsForEvent(ctx context.Context, eventID int64) (DetectionDetail, error)
}
