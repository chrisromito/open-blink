package domain

import (
	"devicecapture/internal/domain/devices"
	"devicecapture/internal/domain/receiver"
)

type Deps struct {
	DeviceRepo    devices.DeviceRepository
	HeartbeatRepo devices.HeartbeatRepo
	ImageRepo     devices.ImageRepo
	DetectionRepo devices.DetectionRepo
	FrameRepo     receiver.FrameRepository
}

func NewDeps(dev devices.DeviceRepository, hb devices.HeartbeatRepo, detRepo devices.DetectionRepo, img devices.ImageRepo, fr receiver.FrameRepository) *Deps {
	return &Deps{
		DeviceRepo:    dev,
		HeartbeatRepo: hb,
		ImageRepo:     img,
		DetectionRepo: detRepo,
		FrameRepo:     fr,
	}
}

func NewMockDeps() *Deps {
	return &Deps{
		DeviceRepo:    devices.NewMockRepo(),
		HeartbeatRepo: devices.NewMockHeartbeat(),
		ImageRepo:     devices.NewMockImageRepo(),
		DetectionRepo: devices.NewMockDetection(),
		FrameRepo:     receiver.NewMockFrameRepo(),
	}
}
