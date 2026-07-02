package history

import (
	"context"
	"time"
)

// DetectionWithImage Detection record w/ an image path (what users care about).
type DetectionWithImage struct {
	ID         int64       `db:"id"         json:"id"`
	CreatedAt  time.Time   `db:"created_at" json:"created_at"`
	Label      string      `db:"label"      json:"label"`
	Confidence float64     `db:"confidence" json:"confidence"`
	Bbox       [][]float64 `db:"bbox"       json:"bbox"`
	DeviceID   int64       `db:"device_id"  json:"device_id"`
	ImageUrl   string      `db:"image_url"  json:"image_url"`
}

type DetectionWithImageParams struct {
	Label    []string `db:"label"     json:"label"`
	DeviceID int64    `db:"device_id" json:"device_id"`
	// default = 7 days ago
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}

// EventMeta is a lightweight version of devices.Detection
type EventMeta struct {
	ID         int64   `db:"id"         json:"id"`
	Label      string  `db:"label"      json:"label"`
	Confidence float64 `db:"confidence" json:"confidence"`
}

// DetectionEvent is a DeviceImage + EventMeta slice
type DetectionEvent struct {
	ID         int64       `db:"id"         json:"id"`
	CreatedAt  time.Time   `db:"created_at" json:"created_at"`
	DeviceID   int64       `db:"device_id"  json:"device_id"`
	ImageUrl   string      `db:"image_url"  json:"image_url"`
	Detections []EventMeta `db:"detections" json:"detections"`
}

type DetectionTimelineParams struct {
	Page     int
	DeviceID int64
}

// DetectionHistoryRepo Describes how we retrieve DetectionImages from the persistence layer
type DetectionHistoryRepo interface {
	GetRecentLabels(ctx context.Context) ([]string, error)
	// GetDetectionImagesByLabel Get DetectionImages that match the filter parameters.
	// NOTE If "created_at" is nil, we default to 7 days ago
	GetDetectionImagesByLabel(
		ctx context.Context,
		params DetectionWithImageParams,
	) ([]DetectionWithImage, error)

	GetDetectionTimeline(
		ctx context.Context,
		params DetectionTimelineParams,
	) ([]DetectionEvent, error)
}
