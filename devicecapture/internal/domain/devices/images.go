package devices

import (
	"context"
	"time"
)

type DeviceImage struct {
	ID        int64     `db:"id" json:"id"`
	DeviceID  int64     `db:"device_id" json:"device_id"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	ImagePath string    `db:"image_path" json:"image_path"`
}

type CreateImageParams struct {
	DeviceID  int64  `db:"device_id" json:"device_id"`
	ImagePath string `db:"image_path" json:"image_path"`
}

type ImageRepo interface {
	CreateImage(ctx context.Context, params CreateImageParams) (*DeviceImage, error)
	GetImages(ctx context.Context, deviceId int64) ([]*DeviceImage, error)
}
