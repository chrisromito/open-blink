package history

import (
	"context"
	"time"
)

// DetectionWithImage Detection record w/ an image path. Ie. what users care about
type DetectionWithImage struct {
	ID         int64       `db:"id" json:"id"`
	CreatedAt  time.Time   `db:"created_at" json:"created_at"`
	Label      string      `db:"label" json:"label"`
	Confidence float64     `db:"confidence" json:"confidence"`
	Bbox       [][]float64 `db:"bbox" json:"bbox"`
	DeviceID   int64       `db:"device_id" json:"device_id"`
	ImageUrl   string      `db:"image_url" json:"image_url"`
}

type DetectionWithImageParams struct {
	Label    []string `db:"label" json:"label"`
	DeviceID int64    `db:"device_id" json:"device_id"`
	// default = 7 days ago
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}

// DetectionHistoryRepo Describes how we retrieve DetectionImages from the persistence layer
type DetectionHistoryRepo interface {
	GetRecentLabels(ctx context.Context) ([]string, error)
	// GetDetectionImagesByLabel Get DetectionImages that match the filter parameters. If "created_at" is nil, we default to 7 days ago
	GetDetectionImagesByLabel(ctx context.Context, params DetectionWithImageParams) ([]DetectionWithImage, error)
}
