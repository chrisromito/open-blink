package repos

import (
	"context"
	"devicecapture/internal/device/devices"
	"devicecapture/internal/postgres/db"
	"strconv"
	"time"
)

// PgHeartbeatRepo implements devices.DeviceRepository
type PgHeartbeatRepo struct {
	queries *db.Queries
}

func NewPgHeartbeatRepo(queries *db.Queries) *PgHeartbeatRepo {
	return &PgHeartbeatRepo{
		queries: queries,
	}
}

// GetDeviceHeartBeats get all heartbeats for a given device
func (hb *PgHeartbeatRepo) GetDeviceHeartBeats(ctx context.Context, deviceId string) ([]devices.Heartbeat, error) {
	id, err := strconv.ParseInt(deviceId, 10, 64)
	hbs, err := hb.queries.GetDeviceHeartBeats(ctx, db.GetDeviceHeartBeatsParams{
		DeviceID:  id,
		CreatedAt: startOfDay(),
	})
	if err != nil {
		return nil, err
	}
	var dhbs []devices.Heartbeat
	for _, d := range hbs {
		idevice := hb.dbToDomain(d)
		dhbs = append(dhbs, *idevice)
	}
	return dhbs, nil
}

// HeartBeatsAfter get all heartbeats after a given time
func (hb *PgHeartbeatRepo) HeartBeatsAfter(ctx context.Context, createdAt time.Time) ([]devices.Heartbeat, error) {
	hbs, err := hb.queries.HeartBeatsAfter(ctx, createdAt)
	if err != nil {
		return nil, err
	}
	var dhbs []devices.Heartbeat
	for _, d := range hbs {
		idevice := hb.dbToDomain(d)
		dhbs = append(dhbs, *idevice)
	}
	return dhbs, nil
}

// LatestBeats get latest heartbeats for individual devices
func (hb *PgHeartbeatRepo) LatestBeats(ctx context.Context) ([]devices.LatestBeatsRow, error) {
	rows, err := hb.queries.LatestBeats(ctx)
	if err != nil {
		return nil, err
	}
	var dslice []devices.LatestBeatsRow
	for _, d := range rows {
		idevice := hb.latestToDomain(d)
		dslice = append(dslice, *idevice)
	}
	return dslice, nil
}

// RecordBeat create a DeviceHeartBeat record for a given device, using the current timestamp
func (hb *PgHeartbeatRepo) RecordBeat(ctx context.Context, deviceId string) (*devices.Heartbeat, error) {
	id, err := strconv.ParseInt(deviceId, 10, 64)
	if err != nil {
		return nil, err
	}
	record, err := hb.queries.RecordBeat(ctx, id)
	if err != nil {
		return nil, err
	}
	return hb.dbToDomain(record), nil
}

// DeleteBeats delete heart beat records for a given device
func (hb *PgHeartbeatRepo) DeleteBeats(ctx context.Context, deviceId string) error {
	id, err := strconv.ParseInt(deviceId, 10, 64)
	if err != nil {
		return err
	}
	return hb.queries.DeleteBeats(ctx, id)
}

func (hb *PgHeartbeatRepo) dbToDomain(d db.DeviceHeartbeat) *devices.Heartbeat {
	return &devices.Heartbeat{
		ID:        d.ID,
		DeviceID:  d.DeviceID,
		CreatedAt: d.CreatedAt,
	}
}

func (hb *PgHeartbeatRepo) latestToDomain(d db.LatestBeatsRow) *devices.LatestBeatsRow {
	return &devices.LatestBeatsRow{
		DeviceID:  d.DeviceID,
		CreatedAt: d.CreatedAt,
		ID:        d.ID,
		Name:      d.Name,
		DeviceUrl: d.DeviceUrl,
	}
}

func startOfDay() time.Time {
	today := time.Now()
	return time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, today.Location())
}
