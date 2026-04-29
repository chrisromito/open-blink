package devices

import (
	"context"
	"errors"
	"sync"
	"time"
)

type MockImage struct {
	ds []*DeviceImage
	mu sync.Mutex
}

func NewMockImageRepo() *MockImage {
	return &MockImage{
		ds: []*DeviceImage{},
		mu: sync.Mutex{},
	}
}

func (ir *MockImage) CreateImage(_ context.Context, params CreateImageParams) (*DeviceImage, error) {
	ir.mu.Lock()
	defer ir.mu.Unlock()
	if params.DeviceID < 0 {
		return &DeviceImage{}, errors.New("invalid device ID")
	}
	if params.ImagePath == "" || len(params.ImagePath) > 255 {
		return &DeviceImage{}, errors.New("invalid image path")
	}

	img := &DeviceImage{
		ID:        int64(len(ir.ds) + 1),
		DeviceID:  params.DeviceID,
		CreatedAt: time.Now(),
		ImagePath: params.ImagePath,
	}
	ir.ds = append(ir.ds, img)
	return img, nil
}

func (ir *MockImage) GetImages(_ context.Context, deviceId int64) ([]*DeviceImage, error) {
	ir.mu.Lock()
	defer ir.mu.Unlock()
	var imgs []*DeviceImage
	for _, img := range ir.ds {
		if img.DeviceID == deviceId {
			imgs = append(imgs, img)
		}
	}
	return imgs, nil
}
