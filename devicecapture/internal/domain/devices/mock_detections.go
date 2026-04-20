package devices

import (
	"context"
	"errors"
	"strconv"
	"sync"
	"time"
)

type MockDetection struct {
	ds []*Detection
	mu sync.Mutex
}

func NewMockDetection() *MockDetection {
	return &MockDetection{
		ds: []*Detection{},
	}
}

func (d *MockDetection) GetDetectionsAfter(ctx context.Context, params QueryParams) ([]*Detection, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	var result []*Detection
	for _, detection := range d.ds {
		if detection.CreatedAt.After(params.CreatedAt) {
			result = append(result, detection)
		}
	}

	return result, nil
}

func (d *MockDetection) GetDeviceDetectionsAfter(ctx context.Context, params QueryParams) ([]*Detection, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	deviceID, err := strconv.ParseInt(params.DeviceID, 10, 64)
	if err != nil {
		return nil, errors.New("invalid device ID")
	}

	var result []*Detection
	for _, detection := range d.ds {
		if detection.DeviceID == deviceID && detection.CreatedAt.After(params.CreatedAt) {
			result = append(result, detection)
		}
	}

	return result, nil
}

func (d *MockDetection) CreateDetection(ctx context.Context, params CreateDetectionParams) (*Detection, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	detection := &Detection{
		ID:         int64(len(d.ds) + 1),
		DeviceID:   params.DeviceID,
		CreatedAt:  time.Now(),
		Label:      params.Label,
		Confidence: params.Confidence,
	}

	d.ds = append(d.ds, detection)
	return detection, nil
}

func (d *MockDetection) DeleteDetections(ctx context.Context, deviceId string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	deviceID, err := strconv.ParseInt(deviceId, 10, 64)
	if err != nil {
		return errors.New("invalid device ID")
	}

	var filteredDetections []*Detection
	for _, detection := range d.ds {
		if detection.DeviceID != deviceID {
			filteredDetections = append(filteredDetections, detection)
		}
	}

	d.ds = filteredDetections
	return nil
}
