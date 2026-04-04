package devices

import (
	"context"
	"errors"
	"os"
	"sync"
)

func GetMockDevice() Device {
	mockUrl := os.Getenv("MOCK_DEVICE_URL")
	if mockUrl == "" {
		mockUrl = "http://0.0.0.0:8080"
	}

	return Device{
		ID:        int64(1),
		Name:      "mockdevice",
		DeviceUrl: mockUrl,
	}
}

func GetTestFailDevice() Device {
	return Device{
		ID:        int64(-1),
		Name:      "fail",
		DeviceUrl: "http://localhost:1234",
	}
}

type MockRepo struct {
	ds []*Device
	mu sync.Mutex
}

func NewMockRepo() *MockRepo {
	d := GetMockDevice()
	df := GetTestFailDevice()
	return &MockRepo{ds: []*Device{
		&df,
		&d,
	}}
}

// CreateDevice MockRepo implements DeviceRepository
func (mr *MockRepo) CreateDevice(ctx context.Context, params CreateDeviceParams) (*Device, error) {
	mr.mu.Lock()
	defer mr.mu.Unlock()
	dev := GetMockDevice()
	d := &dev
	d.ID = int64(len(mr.ds) + 1)
	d.Name = params.Name
	d.DeviceUrl = params.DeviceUrl
	mr.ds = append(mr.ds, d)
	return d, nil
}

// GetDevice MockRepo implements DeviceRepository
func (mr *MockRepo) GetDevice(ctx context.Context, deviceId int64) (*Device, error) {
	mr.mu.Lock()
	defer mr.mu.Unlock()
	for _, d := range mr.ds {
		if d.ID == deviceId {
			return d, nil
		}
	}
	return nil, errors.New("notfound")
}

// ListDevices MockRepo implements DeviceRepository
func (mr *MockRepo) ListDevices(ctx context.Context) ([]*Device, error) {
	mr.mu.Lock()
	defer mr.mu.Unlock()
	dSlice := make([]*Device, len(mr.ds))
	for _, d := range mr.ds {
		dSlice = append(dSlice, d)
	}
	return dSlice, nil
}

// UpdateDevice MockRepo implements DeviceRepository
func (mr *MockRepo) UpdateDevice(ctx context.Context, params UpdateDeviceParams) (*Device, error) {
	mr.mu.Lock()
	defer mr.mu.Unlock()
	for _, d := range mr.ds {
		if d.ID == params.ID {
			d.Name = params.Name
			d.DeviceUrl = params.DeviceUrl
			return d, nil
		}
	}
	return nil, errors.New("notfound")
}
