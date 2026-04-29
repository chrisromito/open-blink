package devices

import (
	"context"
	"errors"
	"os"
	"slices"
	"strings"
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
		ID:        int64(2),
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

// IsValidId MockRepo implements DeviceRepository
func (mr *MockRepo) IsValidId(id string) bool {
	return !isLetter(id)
}

func isLetter(s string) bool {
	return !strings.ContainsFunc(s, func(r rune) bool {
		return (r < 'a' || r > 'z') && (r < 'A' || r > 'Z')
	})
}

// CreateDevice MockRepo implements DeviceRepository
func (mr *MockRepo) CreateDevice(_ context.Context, params CreateDeviceParams) (*Device, error) {
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
func (mr *MockRepo) GetDevice(_ context.Context, deviceId int64) (*Device, error) {
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
func (mr *MockRepo) ListDevices(_ context.Context) ([]*Device, error) {
	mr.mu.Lock()
	defer mr.mu.Unlock()
	dSlice := make([]*Device, len(mr.ds))
	for _, d := range mr.ds {
		dSlice = append(dSlice, d)
	}
	return dSlice, nil
}

// UpdateDevice MockRepo implements DeviceRepository
func (mr *MockRepo) UpdateDevice(_ context.Context, params UpdateDeviceParams) (*Device, error) {
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

func (mr *MockRepo) DeleteDevice(_ context.Context, id int64) error {
	mr.mu.Lock()
	defer mr.mu.Unlock()
	var found = false
	for idx, d := range mr.ds {
		if d.ID == id {
			mr.ds = slices.Delete(mr.ds, idx, idx+1)
			found = true
		}
	}
	if found == false {
		return errors.New("device not found")
	}
	return nil
}

func (mr *MockRepo) DeleteTestDevices(_ context.Context) error {
	mr.mu.Lock()
	defer mr.mu.Unlock()
	for idx, d := range mr.ds {
		if strings.Contains(d.Name, "mock") {
			mr.ds = slices.Delete(mr.ds, idx, idx+1)
		}
	}
	return nil
}
