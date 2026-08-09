package devices

import (
	"context"
	"errors"
	"slices"
	"sync"
	"time"
)

type MockImage struct {
	ds []DeviceImage
	mu sync.Mutex
}

func NewMockImageRepo() *MockImage {
	return &MockImage{
		ds: []DeviceImage{},
		mu: sync.Mutex{},
	}
}

func (ir *MockImage) CreateImage(_ context.Context, params CreateImageParams) (DeviceImage, error) {
	ir.mu.Lock()
	defer ir.mu.Unlock()
	if params.DeviceID < 0 {
		return DeviceImage{}, errors.New("invalid device ID")
	}
	if params.ImagePath == "" || len(params.ImagePath) > 255 {
		return DeviceImage{}, errors.New("invalid image path")
	}

	img := DeviceImage{
		ID:        int64(len(ir.ds) + 1),
		DeviceID:  params.DeviceID,
		CreatedAt: time.Now(),
		ImagePath: params.ImagePath,
	}
	ir.ds = append(ir.ds, img)
	return img, nil
}

func (ir *MockImage) GetImages(_ context.Context, deviceID int64) ([]DeviceImage, error) {
	ir.mu.Lock()
	defer ir.mu.Unlock()
	var imgs []DeviceImage
	for _, img := range ir.ds {
		if img.DeviceID == deviceID {
			imgs = append(imgs, img)
		}
	}
	return imgs, nil
}

func (ir *MockImage) GetImagesForEvent(_ context.Context, eventID int64) ([]DeviceImage, error) {
	ir.mu.Lock()
	defer ir.mu.Unlock()

	var imgs []DeviceImage
	for _, img := range ir.ds {
		if img.EventID != nil && *img.EventID == eventID {
			imgs = append(imgs, img)
		}
	}
	return imgs, nil
}

func (ir *MockImage) SetEvent(_ context.Context, ids []int64, eventID int64) error {
	ir.mu.Lock()
	defer ir.mu.Unlock()
	for _, img := range ir.ds {
		if slices.Contains(ids, img.ID) {
			img.EventID = &eventID
		}
	}
	return nil
}
