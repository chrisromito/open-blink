package devices

import (
	"context"
	"strconv"
)

type Device struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	DeviceUrl string `json:"device_url"`
}

func (d *Device) StringId() string {
	return strconv.Itoa(int(d.ID))
}

type CreateDeviceParams struct {
	Name      string `json:"name"`
	DeviceUrl string `json:"device_url"`
}

type UpdateDeviceParams struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	DeviceUrl string `json:"device_url"`
}

type DeviceRepository interface {
	CreateDevice(ctx context.Context, params CreateDeviceParams) (*Device, error)
	GetDevice(ctx context.Context, deviceId int64) (*Device, error)
	ListDevices(ctx context.Context) ([]*Device, error)
	UpdateDevice(ctx context.Context, params UpdateDeviceParams) (*Device, error)
	DeleteDevice(ctx context.Context, id int64) error
	DeleteTestDevices(ctx context.Context) error
	IsValidId(id string) bool
}
