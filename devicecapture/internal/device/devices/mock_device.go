package devices

import (
	"context"
	"errors"
	"strconv"
	"sync"
)

func GetMockDevice() Device {
	return Device{
		ID:        1,
		Name:      "mockdevice",
		DeviceUrl: "http://localhost:8080",
	}
}

func GetTestFailDevice() Device {
	return Device{
		ID:        -1,
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
	d.ID = int32(len(mr.ds) + 1)
	d.Name = params.Name
	d.DeviceUrl = params.DeviceUrl
	mr.ds = append(mr.ds, d)
	return d, nil
}

// GetDevice MockRepo implements DeviceRepository
func (mr *MockRepo) GetDevice(ctx context.Context, deviceId string) (*Device, error) {
	mr.mu.Lock()
	defer mr.mu.Unlock()
	for _, d := range mr.ds {
		dId := strconv.Itoa(int(d.ID))
		if dId == deviceId {
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
func (mr *MockRepo) UpdateDevice(ctx context.Context, deviceId string, params UpdateDeviceParams) (*Device, error) {
	mr.mu.Lock()
	defer mr.mu.Unlock()
	for _, d := range mr.ds {
		dId := strconv.Itoa(int(d.ID))
		if dId == deviceId {
			d.Name = params.Name
			d.DeviceUrl = params.DeviceUrl
			return d, nil
		}
	}
	return nil, errors.New("notfound")
}
