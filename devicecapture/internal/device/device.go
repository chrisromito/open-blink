package device

import (
	"context"
	"devicecapture/internal/device/devices"
	"devicecapture/internal/device/receiver"
	"errors"
	"log"
	"slices"
	"strconv"
	"sync"
	"time"
)

type CameraService struct {
	DeviceRepo   devices.DeviceRepository
	FrameRepo    receiver.FrameRepository
	connectedIds []string
	mu           sync.Mutex
}

func NewCameraService(deviceRepo devices.DeviceRepository, frameRepo receiver.FrameRepository) *CameraService {
	ids := make([]string, 10)
	return &CameraService{DeviceRepo: deviceRepo, FrameRepo: frameRepo, connectedIds: ids}
}

func (s *CameraService) IsStreaming(deviceId string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, id := range s.connectedIds {
		if id == deviceId {
			return true
		}
	}
	return false
}

func (s *CameraService) StartStream(ctx context.Context, deviceId string) error {
	if s.IsStreaming(deviceId) {
		return errors.New("multiplexing is not supported")
	}
	record, err := s.DeviceRepo.GetDevice(ctx, deviceId)
	if err != nil {
		return err
	}
	s.addId(deviceId)
	defer s.removeId(deviceId)
	cameraDev := NewCameraDevice(*record, s.FrameRepo)
	return cameraDev.Start(ctx)
}
func (s *CameraService) addId(deviceId string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.connectedIds = append(s.connectedIds, deviceId)
}

func (s *CameraService) removeId(deviceId string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	// Remove this ID from the list
	s.connectedIds = slices.DeleteFunc(s.connectedIds, func(id string) bool {
		return id == deviceId
	})
}

// CameraDevice juncture between a device.Device, an Api, and a FrameRepo
type CameraDevice struct {
	device     devices.Device
	api        Api
	FrameRepo  receiver.FrameRepository
	Capturing  bool
	Connecting bool
	StartedAt  int64
	StoppedAt  int64
}

func NewCameraDevice(d devices.Device, repo receiver.FrameRepository) CameraDevice {
	return CameraDevice{
		device:     d,
		api:        NewApi(strconv.Itoa(int(d.ID)), d.DeviceUrl),
		FrameRepo:  repo,
		Capturing:  false,
		Connecting: false,
	}
}

func (d *CameraDevice) stringId() string {
	return strconv.Itoa(int(d.device.ID))
}

func (d *CameraDevice) Url() string {
	return d.api.Url
}

func (d *CameraDevice) Start(ctx context.Context) error {
	defer d.Stop()
	var wg sync.WaitGroup
	imgChan := make(chan receiver.Frame, 64)
	defer close(imgChan)
	outChan := make(chan receiver.Frame, 64)
	defer close(outChan)
	_, e := d.FrameRepo.StartSession(d.stringId())
	if e != nil {
		return e
	}
	d.StartedAt = time.Now().UnixMilli()
	// Worker goroutine
	wg.Add(1)
	go func() {
		defer wg.Done()
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

	wg.Add(1)
	go func() {
		defer wg.Done()
		err := d.api.StreamFrames(ctx, imgChan)
		if err != nil {
			log.Printf("device -> Start -> api worker -> Error streaming frames: %v", err)
			return
		}
	}()

	// Receiver goroutine
	wg.Add(1)
	go func() {
		defer wg.Done()
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

	wg.Wait()
	return nil
}

func (d *CameraDevice) Stop() {
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

	log.Printf("Stopped device: %s", d.stringId())
}

func (d *CameraDevice) IsConnected() bool {
	return d.Connecting
}

func (d *CameraDevice) IsCapturing() bool {
	return d.Capturing
}
