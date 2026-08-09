package devices

import (
	"context"
	"time"
)

type DeviceImage struct {
	ID            int64     `db:"id"             json:"id"`
	DeviceID      int64     `db:"device_id"      json:"device_id"`
	EventID       *int64    `db:"event_id"       json:"event_id"`
	CreatedAt     time.Time `db:"created_at"     json:"created_at"`
	ImagePath     string    `db:"image_path"     json:"image_path"`
	AnnotatedPath string    `db:"annotated_path" json:"annotated_path"`
}

type CreateImageParams struct {
	DeviceID      int64  `db:"device_id"      json:"device_id"`
	EventID       *int64 `db:"event_id"       json:"event_id"`
	ImagePath     string `db:"image_path"     json:"image_path"`
	AnnotatedPath string `db:"annotated_path" json:"annotated_path"`
}

type ImageRepo interface {
	CreateImage(ctx context.Context, params CreateImageParams) (DeviceImage, error)
	GetImages(ctx context.Context, deviceID int64) ([]DeviceImage, error)
	GetImagesForEvent(ctx context.Context, eventID int64) ([]DeviceImage, error)
	SetEvent(ctx context.Context, ids []int64, eventID int64) error
}
