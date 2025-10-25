package devices

type Device struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	DeviceUrl string `json:"device_url"`
}

type DeviceRepo interface {
	CreateDevice(deviceId string) (*Device, error)
	GetDevice(deviceId string) (*Device, error)
	ListDevices() ([]*Device, error)
	UpdateDevice(deviceId string, device *Device) (*Device, error)
	DeleteDevice(deviceId string) error
}
