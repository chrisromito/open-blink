package event

import (
	"context"
	"time"
)

type DetectionState int

const (
	Down    DetectionState = 1
	Started DetectionState = 2
	Ended   DetectionState = 3
)

var stateName = map[DetectionState]string{
	Down:    "down",
	Started: "started",
	Ended:   "ended",
}

func (ds DetectionState) String() string {
	return stateName[ds]
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
}

type DetectionEventRepo interface {
	// StartEvent creates a DetectionEvent with the given deviceID and labels
	StartEvent(ctx context.Context, deviceID int64, labels []string) (DetectionEvent, error)
	// EndEvent updates a DetectionEvent by flagging it as Ended
	EndEvent(ctx context.Context, e DetectionEvent, labels []string) (DetectionEvent, error)
	// GetDeviceEvents returns paginated slices of DetectionEvent based on the provided QueryParams
	GetDeviceEvents(ctx context.Context, p QueryParams) ([]DetectionEvent, error)
}
