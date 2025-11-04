package devices

import "context"

type Device struct {
	ID        int32  `json:"id"`
	Name      string `json:"name"`
	DeviceUrl string `json:"device_url"`
}

type CreateDeviceParams struct {
	Name      string `json:"name"`
	DeviceUrl string `json:"device_url"`
}

type UpdateDeviceParams struct {
	ID        int32  `json:"id"`
	Name      string `json:"name"`
	DeviceUrl string `json:"device_url"`
}

type DeviceRepo interface {
	CreateDevice(ctx context.Context, params CreateDeviceParams) (*Device, error)
	GetDevice(ctx context.Context, deviceId string) (*Device, error)
	ListDevices(ctx context.Context) ([]*Device, error)
	UpdateDevice(ctx context.Context, deviceId string, params UpdateDeviceParams) (*Device, error)
}
