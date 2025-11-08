package repos

import (
	"context"
	"devicecapture/internal/device/devices"
	"devicecapture/internal/postgres/db"
	"errors"
	"strconv"
)

// PgDeviceRepo implements devices.DeviceRepository
type PgDeviceRepo struct {
	queries *db.Queries
}

func NewPgDeviceRepo(queries *db.Queries) *PgDeviceRepo {
	return &PgDeviceRepo{
		queries: queries,
	}
}

var ErrNotFound = errors.New("record not found")

// GetDevice PgDeviceRepo implements devices.DeviceRepository
func (dr *PgDeviceRepo) GetDevice(ctx context.Context, deviceId string) (*devices.Device, error) {
	id, e := strconv.ParseInt(deviceId, 10, 32)
	if e != nil {
		return nil, e
	}
	d, err := dr.queries.GetDeviceById(ctx, int32(id))
	if err != nil {
		return nil, err
	}
	if d.ID == 0 {
		return nil, ErrNotFound
	}
	return dr.dbToDomain(d), nil
}

// ListDevices PgDeviceRepo implements devices.DeviceRepository
func (dr *PgDeviceRepo) ListDevices(ctx context.Context) ([]*devices.Device, error) {
	value, err := dr.queries.GetDevices(ctx)
	if err != nil {
		return nil, err
	}
	var dslice []*devices.Device
	for _, d := range value {
		idevice := dr.dbToDomain(d)
		dslice = append(dslice, idevice)
	}
	return dslice, nil
}

// CreateDevice PgDeviceRepo implements devices.DeviceRepository
func (dr *PgDeviceRepo) CreateDevice(ctx context.Context, params devices.CreateDeviceParams) (*devices.Device, error) {
	d, err := dr.queries.CreateDevice(ctx, db.CreateDeviceParams{Name: params.Name, DeviceUrl: params.DeviceUrl})
	if err != nil {
		return nil, err
	}
	return &devices.Device{
		ID:        d.ID,
		Name:      d.Name,
		DeviceUrl: d.DeviceUrl,
	}, nil
}

func (dr *PgDeviceRepo) UpdateDevice(ctx context.Context, deviceId string, params devices.UpdateDeviceParams) (*devices.Device, error) {
	err := dr.queries.UpdateDevice(ctx, db.UpdateDeviceParams{
		ID:        params.ID,
		Name:      params.Name,
		DeviceUrl: params.DeviceUrl,
	})
	if err != nil {
		return nil, err
	}
	return &devices.Device{
		ID:        params.ID,
		Name:      params.Name,
		DeviceUrl: params.DeviceUrl,
	}, nil
}

func (dr *PgDeviceRepo) dbToDomain(d db.Device) *devices.Device {
	return &devices.Device{
		ID:        d.ID,
		Name:      d.Name,
		DeviceUrl: d.DeviceUrl,
	}
}
