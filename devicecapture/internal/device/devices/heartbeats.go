package devices

import (
	"context"
	"time"
)

type Heartbeat struct {
	ID        int32     `db:"id" json:"id"`
	DeviceID  int64     `db:"device_id" json:"device_id"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}

type LatestBeatsRow struct {
	DeviceID  int64     `db:"device_id" json:"device_id"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	ID        int32     `db:"id" json:"id"`
	Name      string    `db:"name" json:"name"`
	DeviceUrl string    `db:"device_url" json:"device_url"`
}

type HeartbeatRepo interface {
	GetDeviceHeartBeats(ctx context.Context, deviceId string) ([]Heartbeat, error)
	HeartBeatsAfter(ctx context.Context, createdAt time.Time) ([]*Heartbeat, error)
	LatestBeats(ctx context.Context) ([]LatestBeatsRow, error)
	// RecordBeat Record a DeviceHeartbeat
	RecordBeat(ctx context.Context, deviceId string) (*Heartbeat, error)
	DeleteBeats(ctx context.Context, deviceId string) error
}
