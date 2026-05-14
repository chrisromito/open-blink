package domain

import (
	"devicecapture/internal/domain/devices"
	"devicecapture/internal/domain/history"
	"devicecapture/internal/domain/receiver"
)

type Deps struct {
	DeviceRepo    devices.DeviceRepository
	HeartbeatRepo devices.HeartbeatRepo
	ImageRepo     devices.ImageRepo
	DetectionRepo devices.DetectionRepo
	FrameRepo     receiver.FrameRepository
	HistoryRepo   history.DetectionHistoryRepo
}

func NewDeps(dev devices.DeviceRepository, hb devices.HeartbeatRepo, detRepo devices.DetectionRepo, img devices.ImageRepo, fr receiver.FrameRepository, hr history.DetectionHistoryRepo) *Deps {
	return &Deps{
		DeviceRepo:    dev,
		HeartbeatRepo: hb,
		ImageRepo:     img,
		DetectionRepo: detRepo,
		FrameRepo:     fr,
		HistoryRepo:   hr,
	}
}

func NewMockDeps() *Deps {
	return &Deps{
		DeviceRepo:    devices.NewMockRepo(),
		HeartbeatRepo: devices.NewMockHeartbeat(),
		ImageRepo:     devices.NewMockImageRepo(),
		DetectionRepo: devices.NewMockDetection(),
		FrameRepo:     receiver.NewMockFrameRepo(),
		HistoryRepo:   history.NewMockDetectionHistory(),
	}
}
