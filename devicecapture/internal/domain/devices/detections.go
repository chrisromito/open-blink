package devices

import (
	"context"
	"time"
)

type Detection struct {
	ID         int64     `db:"id" json:"id"`
	DeviceID   int64     `db:"device_id" json:"device_id"`
	ImageID    *int64    `db:"image_id" json:"image_id"`
	CreatedAt  time.Time `db:"created_at" json:"created_at"`
	Label      string    `db:"label" json:"label"`
	Confidence float64   `db:"confidence" json:"confidence"`
}

type CreateDetectionParams struct {
	DeviceID   int64   `db:"device_id" json:"device_id"`
	Label      string  `db:"label" json:"label"`
	Confidence float64 `db:"confidence" json:"confidence"`
	ImageID    *int64  `db:"image_id" json:"image_id"`
}

type QueryParams struct {
	DeviceID  int64     `db:"device_id" json:"device_id"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	ImageID   *int64    `db:"image_id" json:"image_id"`
}

type DetectionRepo interface {
	GetDetectionsAfter(ctx context.Context, params QueryParams) ([]*Detection, error)
	GetDeviceDetectionsAfter(ctx context.Context, params QueryParams) ([]*Detection, error)
	CreateDetection(ctx context.Context, params CreateDetectionParams) (*Detection, error)
	CreateDetections(ctx context.Context, params []CreateDetectionParams) ([]*Detection, error)
	DeleteDetections(ctx context.Context, deviceId int64) error
}
