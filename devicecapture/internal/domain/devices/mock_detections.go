package devices

import (
	"context"
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

	var result []*Detection
	for _, detection := range d.ds {
		if detection.DeviceID == params.DeviceID && detection.CreatedAt.After(params.CreatedAt) {
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

func (d *MockDetection) CreateDetections(ctx context.Context, params []CreateDetectionParams) ([]*Detection, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	var value []*Detection
	for _, param := range params {
		record, err := d.CreateDetection(ctx, param)
		if err != nil {
			return value, err
		}
		value = append(value, record)
	}
	return value, nil
}

func (d *MockDetection) DeleteDetections(ctx context.Context, deviceId int64) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	var filteredDetections []*Detection
	for _, detection := range d.ds {
		if detection.DeviceID != deviceId {
			filteredDetections = append(filteredDetections, detection)
		}
	}

	d.ds = filteredDetections
	return nil
}
