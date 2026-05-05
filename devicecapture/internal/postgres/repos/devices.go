package repos

import (
	"context"
	"devicecapture/internal/domain/devices"
	"devicecapture/internal/postgres/db"
	"errors"
	"github.com/jackc/pgx/v5"
	"log"
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

func (dr *PgDeviceRepo) IsValidId(id string) bool {
	stringId, err := strconv.ParseInt(id, 10, 64)
	d, err2 := dr.GetDevice(context.Background(), stringId)
	return err == nil && err2 == nil && d.ID != 0
}

// GetDevice PgDeviceRepo implements devices.DeviceRepository
func (dr *PgDeviceRepo) GetDevice(ctx context.Context, deviceId int64) (devices.Device, error) {
	d, err := dr.queries.GetDeviceById(ctx, deviceId)
	if err != nil {
		return devices.Device{}, err
	}
	if d.ID == 0 {
		return devices.Device{}, ErrNotFound
	}
	return dr.dbToDomain(d), nil
}

// ListDevices PgDeviceRepo implements devices.DeviceRepository
func (dr *PgDeviceRepo) ListDevices(ctx context.Context) ([]devices.Device, error) {
	value, err := dr.queries.GetDevices(ctx)
	if err != nil {
		return nil, err
	}
	var dslice []devices.Device
	for _, d := range value {
		idevice := dr.dbToDomain(d)
		dslice = append(dslice, idevice)
	}
	return dslice, nil
}

// CreateDevice PgDeviceRepo implements devices.DeviceRepository
func (dr *PgDeviceRepo) CreateDevice(ctx context.Context, params devices.CreateDeviceParams) (devices.Device, error) {
	d, err := dr.queries.CreateDevice(ctx, db.CreateDeviceParams{Name: params.Name, DeviceUrl: params.DeviceUrl})
	if err != nil {
		return devices.Device{}, err
	}
	return devices.Device{
		ID:        d.ID,
		Name:      d.Name,
		DeviceUrl: d.DeviceUrl,
	}, nil
}

func (dr *PgDeviceRepo) UpdateDevice(ctx context.Context, params devices.UpdateDeviceParams) (devices.Device, error) {
	err := dr.queries.UpdateDevice(ctx, db.UpdateDeviceParams{
		ID:        params.ID,
		Name:      params.Name,
		DeviceUrl: params.DeviceUrl,
	})
	if err != nil {
		return devices.Device{}, err
	}
	return devices.Device{
		ID:        params.ID,
		Name:      params.Name,
		DeviceUrl: params.DeviceUrl,
	}, nil
}

func (dr *PgDeviceRepo) DeleteDevice(ctx context.Context, id int64) error {
	err := dr.queries.DeleteDevice(ctx, id)
	return err
}

func (dr *PgDeviceRepo) DeleteTestDevices(ctx context.Context) error {
	err := dr.queries.DeleteTestDevices(ctx)
	return err
}

func (dr *PgDeviceRepo) dbToDomain(d db.Device) devices.Device {
	return devices.Device{
		ID:        d.ID,
		Name:      d.Name,
		DeviceUrl: d.DeviceUrl,
	}
}

func GetOrCreateTestDevice(ctx context.Context, q *db.Queries) (db.Device, error) {
	d, err := q.GetTestDevice(ctx)
	if err == nil {
		return d, nil
	}
	if errors.Is(err, pgx.ErrNoRows) || d.ID == 0 {
		log.Printf("No test device found, creating it")
		return q.CreateTestDevice(ctx)
	}
	log.Printf("GetOrCreateTestDevice ERROR %v", err)
	return db.Device{}, err
}
