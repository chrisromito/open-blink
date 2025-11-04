package devices

import (
	"context"
	"time"
)

type Detection struct {
	ID         int32     `db:"id" json:"id"`
	DeviceID   int64     `db:"device_id" json:"device_id"`
	CreatedAt  time.Time `db:"created_at" json:"created_at"`
	Label      string    `db:"label" json:"label"`
	Confidence float64   `db:"confidence" json:"confidence"`
}

type CreateDetectionParams struct {
	DeviceID   int64   `db:"device_id" json:"device_id"`
	Label      string  `db:"label" json:"label"`
	Confidence float64 `db:"confidence" json:"confidence"`
}

type QueryParams struct {
	DeviceID  string    `db:"device_id" json:"device_id"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}

type DetectionRepo interface {
	GetDetectionsAfter(ctx context.Context, params QueryParams) ([]Detection, error)
	GetDeviceDetectionsAfter(ctx context.Context, params QueryParams) ([]Detection, error)
	CreateDetection(ctx context.Context, params CreateDetectionParams) (Detection, error)
	DeleteDetections(ctx context.Context, deviceId string) error
}
