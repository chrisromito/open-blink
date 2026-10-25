package camera

import (
	"context"
	"errors"
	"slices"
	"strconv"
	"sync"
	"time"

	"devicecapture/internal/config"
	"devicecapture/internal/domain"
	"devicecapture/internal/domain/detection"
	"devicecapture/internal/domain/devices"
	"devicecapture/internal/domain/queue"
	"devicecapture/internal/logger"
)

// CameraService provides high-level methods for interacting with camera devices
type CameraService struct {
	Config        *config.Config
	DeviceRepo    devices.DeviceRepository
	Publisher     queue.Publisher
	DetectionRepo devices.DetectionRepo
	Detector      detection.ObjectDetector
	ImageRepo     devices.ImageRepo
	connectedIds  []string
	mu            sync.Mutex
}

// NewCameraService creates a new CameraService instance with the provided configuration,
// dependencies, object detector, and MQTT client. It initializes the service with an
// empty slice of connected device IDs and returns the configured service.
func NewCameraService(
	conf *config.Config,
	deps *domain.Deps,
	detector detection.ObjectDetector,
) *CameraService {
	ids := make([]string, 10)
	cs := &CameraService{
		Config:        conf,
		DeviceRepo:    deps.DeviceRepo,
		Publisher:     deps.Publisher,
		DetectionRepo: deps.DetectionRepo,
		ImageRepo:     deps.ImageRepo,
		connectedIds:  ids,
		Detector:      detector,
		mu:            sync.Mutex{},
	}
	return cs
}

func (s *CameraService) IsValidId(deviceId string) bool {
	id, err := strconv.ParseInt(deviceId, 10, 64)
	d, err2 := s.DeviceRepo.GetDevice(context.Background(), id)
	return err == nil && err2 == nil && d.ID != 0
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

// Snapshot captures a single frame from the specified device and saves it to disk.
// It starts a temporary capture session, retrieves one frame from the device's API,
// and processes it through the frame handling pipeline including optional object detection.
// The session is automatically closed when the method completes.
func (s *CameraService) Snapshot(ctx context.Context, d devices.Device) error {
	stringId := d.StringId()
	api := NewApi(stringId, d.DeviceUrl)
	session, sErr := s.Publisher.StartSession(stringId)
	if sErr != nil {
		return sErr
	}
	defer func(FrameRepo queue.Publisher) {
		_ = FrameRepo.EndSession(session)
	}(s.Publisher)

	// Get a frame and pass it down the pipe
	frame, err := api.Snapshot(ctx)
	if err != nil {
		return err
	}
	return s.ingestFrame(ctx, session, d, frame)
}

// StreamSnapshots streams [queue.Frame] from the [devices.Device] for [seconds].
// This method uses the `snapshot` endpoint, which is less taxing on the camera device than streaming MJPEG frames.
func (s *CameraService) StreamSnapshots(ctx context.Context, d devices.Device, seconds int) error {
	logger.Debug().Str("camera", "StreamSnapshots").
		Int64("deviceID", d.ID).Send()
	stringId := d.StringId()
	if s.IsStreaming(stringId) {
		return errors.New("cannot start stream for device")
	}

	s.addId(stringId)
	defer s.removeId(stringId)

	session, sErr := s.Publisher.StartSession(stringId)
	if sErr != nil {
		return sErr
	}
	defer func(FrameRepo queue.Publisher) {
		_ = FrameRepo.EndSession(session)
	}(s.Publisher)

	api := NewApi(stringId, d.DeviceUrl)
	streamCtx, cancel := context.WithTimeout(ctx, time.Duration(seconds)*time.Second)
	defer cancel()

	// Allow backpressure of 3 frames
	imgChan := make(chan queue.Frame, 3)
	defer close(imgChan)

	var wg sync.WaitGroup

	// Sender go routine fetches JPEGs every 250 milliseconds and pushes them into imgChan
	wg.Add(1)
	go func() {
		defer wg.Done()
		camTimer := time.NewTicker(250 * time.Millisecond)
		for {
			select {
			case <-streamCtx.Done():
				return
			case <-camTimer.C:
				frame, err := api.Snapshot(streamCtx)
				if err != nil {
					logger.Error().Err(err).
						Str("camera", "StreamSnapshots").
						Msg("err fetching JPEG, cancelling")
					cancel()
					return
				}
				imgChan <- frame
			}
		}
	}()

	// runFrameProcessor go routine pulls frames from imgChan
	wg.Add(1)
	go s.runFrameProcessor(&wg, streamCtx, d, session, imgChan)

	wg.Wait()
	return nil
}

// StartStream wraps the process of checking if a device is streaming, validating the device ID,
// starting the CaptureSession, and managing the streaming workflow.
func (s *CameraService) StartStream(
	ctx context.Context,
	deviceId string,
) (*queue.CaptureSession, error) {
	device, canStart := s.canStartStream(ctx, deviceId)
	if !canStart {
		return &queue.CaptureSession{}, errors.New("cannot start stream for device")
	}

	s.addId(deviceId)
	defer s.removeId(deviceId)

	session, err := s.Publisher.StartSession(deviceId)
	if err != nil {
		return &queue.CaptureSession{}, err
	}
	if session == nil {
		return &queue.CaptureSession{}, errors.New("unexpected nil CaptureSession")
	}
	defer func() { _ = s.Publisher.EndSession(session) }()

	var wg sync.WaitGroup
	imgChan := make(chan queue.Frame, 60) // 60 frames = 4 FPS * 15 seconds
	defer close(imgChan)

	streamCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	// Spawn workers
	wg.Add(2)
	go s.runApiStreamer(&wg, streamCtx, deviceId, device.DeviceUrl, imgChan)
	go s.runFrameProcessor(&wg, streamCtx, device, session, imgChan)

	wg.Wait()
	logger.Error().Str("service", "camera.StartStream").Msg("camera.StartStream -> returning")
	return session, nil
}

// runApiStreamer handles connecting to the API and passing frames to the "imgChan" channel
func (s *CameraService) runApiStreamer(
	wg *sync.WaitGroup,
	ctx context.Context,
	deviceId string,
	deviceUrl string,
	imgChan chan<- queue.Frame,
) {
	defer wg.Done()
	api := NewApi(deviceId, deviceUrl)
	if err := api.StreamFrames(ctx, imgChan); err != nil {
		logger.Error().Str("service", "camera.StartStream").
			Msgf("domain -> Start -> api worker -> Error streaming frames: %v", err)
	}
}

// runFrameProcessor receives frames and generates file paths before handing off to [CameraService.ingestFrame]
func (s *CameraService) runFrameProcessor(
	wg *sync.WaitGroup,
	ctx context.Context,
	device devices.Device,
	session *queue.CaptureSession,
	imgChan <-chan queue.Frame,
) {
	defer wg.Done()
	for {
		select {
		case <-ctx.Done():
			return
		case frame, ok := <-imgChan:
			if !ok {
				logger.Error().Str("service", "camera.StartStream").
					Msg("CameraService -> imgChan not ok, returning")
				return
			}

			if session == nil {
				continue
			}

			err := s.ingestFrame(ctx, session, device, frame)
			if err != nil {
				logger.Error().Str("service", "camera.StartStream").
					Msgf("ingestFrame threw %v", err)
				return
			}
		}
	}
}

func (s *CameraService) canStartStream(
	ctx context.Context,
	deviceId string,
) (devices.Device, bool) {
	if s.IsStreaming(deviceId) {
		return devices.Device{}, false
	}
	// cast the id and grab the device record from the repo
	id, err := strconv.ParseInt(deviceId, 10, 64)
	if err != nil {
		return devices.Device{}, false
	}
	// invalid IDs return
	device, deviceErr := s.DeviceRepo.GetDevice(ctx, id)
	if deviceErr != nil {
		// invalid id, exit early
		return devices.Device{}, false
	}
	if device.DeviceUrl == "" {
		return device, false
	}
	return device, true
}

// ingestFrame writes the original frame, creates a DeviceImage record, and publishes a
// DeviceImage-created event for asynchronous processing.
func (s *CameraService) ingestFrame(
	ctx context.Context,
	cs *queue.CaptureSession,
	device devices.Device,
	frame queue.Frame,
) error {
	cs.SetLastFrame(&frame)

	framePath := queue.FramePath(s.Config.VideoPath, cs, frame)
	fErr := frame.WriteFrame(framePath)
	if fErr != nil {
		logger.Error().
			Err(fErr).
			Str("framePath", framePath).
			Msg("failed to write frame")
		return fErr
	}

	imageRecord, err := s.ImageRepo.CreateImage(
		ctx,
		devices.CreateImageParams{
			DeviceID:  device.ID,
			ImagePath: framePath,
		},
	)
	if err != nil {
		logger.Error().
			Err(err).
			Int64("deviceID", device.ID).
			Str("framePath", framePath).
			Msg("failed to create device image")
		return err
	}

	if err := s.Publisher.PublishDeviceImageCreated(ctx, imageRecord); err != nil {
		logger.Error().
			Err(err).
			Int64("deviceID", device.ID).
			Int64("imageID", imageRecord.ID).
			Str("PublishDeviceImageCreated", "Err").
			Send()
		return err
	}
	logger.Debug().
		Str("Camera", "ingestFrame").
		Str("PublishDeviceImageCreated", "Success").
		Int64("deviceID", device.ID).
		Int64("imageID", imageRecord.ID).
		Send()

	return nil
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
