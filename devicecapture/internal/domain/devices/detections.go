package devices

import (
	"context"
	"time"
)

type Detection struct {
	ID         int64       `db:"id"         json:"id"`
	DeviceID   int64       `db:"device_id"  json:"device_id"`
	ImageID    *int64      `db:"image_id"   json:"image_id"`
	EventID    *int64      `db:"event_id"   json:"event_id"`
	CreatedAt  time.Time   `db:"created_at" json:"created_at"`
	Label      string      `db:"label"      json:"label"`
	Confidence float64     `db:"confidence" json:"confidence"`
	Bbox       [][]float64 `db:"bbox"       json:"bbox"`
}

type CreateDetectionParams struct {
	DeviceID   int64       `db:"device_id"  json:"device_id"`
	Label      string      `db:"label"      json:"label"`
	Confidence float64     `db:"confidence" json:"confidence"`
	ImageID    *int64      `db:"image_id"   json:"image_id"`
	EventID    *int64      `db:"event_id"   json:"event_id"`
	Bbox       [][]float64 `db:"bbox"       json:"bbox"`
}

type QueryParams struct {
	DeviceID  int64     `db:"device_id"  json:"device_id"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	ImageID   *int64    `db:"image_id"   json:"image_id"`
}

type DetectionRepo interface {
	GetDetectionsAfter(ctx context.Context, params QueryParams) ([]Detection, error)
	GetDeviceDetectionsAfter(ctx context.Context, params QueryParams) ([]Detection, error)
	GetDetectionsForEvent(ctx context.Context, eventID int64) ([]Detection, error)
	SetEvent(ctx context.Context, ids []int64, eventID int64) error
	CreateDetection(ctx context.Context, params CreateDetectionParams) (Detection, error)
	CreateDetections(ctx context.Context, params []CreateDetectionParams) ([]Detection, error)
	DeleteDetections(ctx context.Context, deviceID int64) error
}
