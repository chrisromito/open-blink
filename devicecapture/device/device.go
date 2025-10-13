package device

import (
	"context"
	"devicecapture/config"
	"devicecapture/device/receiver"
	"log"
	"time"
)

type Device struct {
	conf       config.DeviceConfig
	api        Api
	FrameRepo  receiver.FrameRepository
	Capturing  bool
	Connecting bool
	StartedAt  int64
	StoppedAt  int64
}

func NewDevice(conf config.DeviceConfig, repo receiver.FrameRepository) Device {
	return Device{
		conf:       conf,
		api:        NewApi(conf.DeviceId, conf.DeviceUrl),
		FrameRepo:  repo,
		Capturing:  false,
		Connecting: false,
	}
}

func (d *Device) Url() string {
	return d.api.Url
}

func (d *Device) Start(ctx context.Context) error {
	defer d.Stop()
	imgChan := make(chan receiver.Frame, 64)
	defer close(imgChan)
	outChan := make(chan receiver.Frame, 64)
	defer close(outChan)
	_, e := d.FrameRepo.StartSession(d.conf.DeviceId)
	if e != nil {
		return e
	}
	// Worker goroutine
	go func() {
		for {
			select {
			case img, ok := <-imgChan:
				if !ok {
					return
				}
				outChan <- img
			case <-ctx.Done():
				return
			}
		}
	}()

	go func() {
		err := d.api.StreamFrames(ctx, imgChan)
		if err != nil {
			log.Printf("device -> Start -> api worker -> Error streaming frames: %v", err)
			return
		}
	}()

	// Receiver goroutine
	go func() {
		for {
			select {
			case img, ok := <-outChan:
				if !ok {
					return
				}
				err := d.FrameRepo.ReceiveFrame(img)
				if err != nil {
					return
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	<-ctx.Done()
	return nil
}

func (d *Device) Stop() {
	err := d.FrameRepo.EndSession()
	if err != nil {
		log.Printf("Error ending session: %v", err)
	}
	if !d.Connecting {
		return
	}
	d.Connecting = false
	d.Capturing = false
	d.StoppedAt = time.Now().UnixMilli()

	log.Printf("Stopped device: %s", d.conf.DeviceId)
}

func (d *Device) IsConnected() bool {
	return d.Connecting
}

func (d *Device) IsCapturing() bool {
	return d.Capturing
}
