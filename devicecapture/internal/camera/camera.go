package camera

import (
	"context"
	"devicecapture/internal/config"
	"devicecapture/internal/domain"
	"devicecapture/internal/domain/detection"
	"devicecapture/internal/domain/devices"
	"devicecapture/internal/domain/receiver"
	"devicecapture/internal/logger"
	"devicecapture/internal/pubsub"
	"errors"
	"slices"
	"strconv"
	"sync"
	"time"
)

// CameraService provides high-level methods for interacting with camera devices
type CameraService struct {
	Config        *config.Config
	DeviceRepo    devices.DeviceRepository
	FrameRepo     receiver.FrameRepository
	DetectionRepo devices.DetectionRepo
	Detector      detection.ObjectDetector
	ImageRepo     devices.ImageRepo
	mqttClient    *pubsub.MqttClient
	connectedIds  []string
	mu            sync.Mutex
}

// NewCameraService creates a new CameraService instance with the provided configuration,
// dependencies, object detector, and MQTT client. It initializes the service with an
// empty slice of connected device IDs and returns the configured service.
func NewCameraService(conf *config.Config, deps *domain.Deps, detector detection.ObjectDetector, qtClient *pubsub.MqttClient) *CameraService {
	ids := make([]string, 10)
	cs := &CameraService{
		Config:        conf,
		DeviceRepo:    deps.DeviceRepo,
		FrameRepo:     deps.FrameRepo,
		DetectionRepo: deps.DetectionRepo,
		ImageRepo:     deps.ImageRepo,
		connectedIds:  ids,
		Detector:      detector,
		mqttClient:    qtClient,
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
	session, sErr := s.FrameRepo.StartSession(stringId)
	if sErr != nil {
		return sErr
	}
	defer func(FrameRepo receiver.FrameRepository) {
		_ = FrameRepo.EndSession(session)
	}(s.FrameRepo)

	// Get a frame and pass it down the pipe
	frame, err := api.Snapshot(ctx)
	if err != nil {
		return err
	}
	fp := receiver.FramePath(s.Config.VideoPath, session, frame)
	return s.receiveFrame(ctx, d.ID, fp, frame, true)
}

// StartStream wraps the process of checking if a device is streaming, validating the device ID,
// starting the CaptureSession, and managing the streaming workflow. It validates the device exists
// and has a valid URL, then starts concurrent goroutines to handle frame streaming and processing.
// The stream runs for a maximum of 15 seconds and processes frames at 4 FPS with object detection on every other frame.
func (s *CameraService) StartStream(ctx context.Context, deviceId string) (*receiver.CaptureSession, error) {
	if s.IsStreaming(deviceId) {
		return &receiver.CaptureSession{}, errors.New("multiplexing is not supported")
	}
	// cast the id and grab the device record from the repo
	id, err := strconv.ParseInt(deviceId, 10, 64)
	if err != nil {
		return &receiver.CaptureSession{}, err
	}
	// invalid IDs return
	device, deviceErr := s.DeviceRepo.GetDevice(ctx, id)
	if deviceErr != nil {
		// invalid id, exit early
		return &receiver.CaptureSession{}, deviceErr
	}
	if device.DeviceUrl == "" {
		return &receiver.CaptureSession{}, errors.New("invalid Device URL for device ID")
	} else {
		logger.Debug().Str("service", "camera.StartStream").
			Msgf("starting stream for device %s @ %s", deviceId, device.DeviceUrl)
	}
	// Add id string to our list of streaming IDs
	s.addId(deviceId)
	defer s.removeId(deviceId)
	// Tell the frame repo that we're starting a session
	session, sessErr := s.FrameRepo.StartSession(deviceId)
	if sessErr != nil || session == nil {
		return &receiver.CaptureSession{}, sessErr
	}
	defer func(FrameRepo receiver.FrameRepository) {
		// make sure we close it out
		_ = FrameRepo.EndSession(session)
	}(s.FrameRepo)

	// wg ends when the stream is complete
	var wg sync.WaitGroup
	imgChan := make(chan receiver.Frame, 60) // 60 frames = 4 FPS * 15 seconds
	defer close(imgChan)

	//streamCtx, cancel := context.WithCancel(ctx)
	streamCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	wg.Add(1)
	// api goroutine receives JPEGs from the API & passes them to imageChan
	go func() {
		defer wg.Done()
		api := NewApi(deviceId, device.DeviceUrl)
		apiErr := api.StreamFrames(streamCtx, imgChan)
		if apiErr != nil {
			logger.Error().Str("service", "camera.StartStream").
				Msgf("domain -> Start -> api worker -> Error streaming frames: %v", apiErr)
		}
		return
	}()

	// imgChan goroutine pulls from imgChan & passes frames to CaptureSession & FrameRepo (respectively)
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-streamCtx.Done():
				return
			case img, ok := <-imgChan:
				if !ok {
					logger.Error().Str("service", "camera.StartStream").
						Msg("CameraService -> imgChan not ok, returning")
					return
				}
				if session == nil {
					continue
				}

				session.SetLastFrame(&img)
				fp := receiver.FramePath(s.Config.VideoPath, session, img)
				e := s.receiveFrame(streamCtx, id, fp, img, true)
				if e != nil {
					logger.Error().Str("service", "camera.StartStream").
						Msgf("receiveFrame threw %v", e)
					return
				}
			}
		}
	}()

	wg.Wait()
	logger.Error().Str("service", "camera.StartStream").
		Msgf("camera.StartStream -> returning")
	return session, nil
}

// receiveFrame processes a single frame by saving it as an image record and optionally
// performing object detection. If detection is enabled and objects are found, it stores
// the detections in the database and publishes them to MQTT. The frame is also passed
// to the FrameRepository for storage. All operations run concurrently using goroutines.
func (s *CameraService) receiveFrame(ctx context.Context, deviceId int64, framePath string, frame receiver.Frame, detect bool) error {
	var wg sync.WaitGroup
	if cErr := ctx.Err(); cErr != nil {
		return nil
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		imageRecord, err := s.ImageRepo.CreateImage(ctx, devices.CreateImageParams{DeviceID: deviceId, ImagePath: framePath})
		if err != nil {
			logger.Error().Err(err).
				Msgf("failed to save image to %s ", framePath)
			return
		}
		if !detect {
			return
		}
		req := detection.Req{
			DeviceId: deviceId,
			Frame:    frame,
		}
		detections, dErr := s.Detector.DetectObjectsForImage(ctx, req)
		if dErr != nil {
			logger.Error().Msgf("\n\ndetection err: %v", dErr)
			return
		}
		if len(detections) < 1 {
			return
		}
		// We have >= 1 detection, store them in the DB & broadcast to MQTT
		logger.Debug().Any("detections", detections).
			Msgf("\n\nCameraService: writing detections: %v", detections)
		// Loop, transpose items, and write to the repo
		//topic := "detection/" + strconv.Itoa(int(deviceId))
		topic := "detections/" + strconv.Itoa(int(deviceId))
		var pgDetections []devices.CreateDetectionParams
		// Set up the slice of DB params
		for _, d := range detections {
			pgDetections = append(pgDetections, detectionServiceToPg(deviceId, &imageRecord.ID, d))
		}
		// Write to the DB
		toPublish, err := s.DetectionRepo.CreateDetections(ctx, pgDetections)
		if err != nil {
			logger.Error().Msgf("error writing detections to detection repo %v", err)
			return
		}
		// Publish batch to MQTT
		p, jErr := receiver.DetectionsToMsg(s.Config.ThisIp, imageRecord, toPublish)
		if jErr != nil {
			logger.Error().Str("service", "camera").
				Err(jErr).Send()
			return
		}
		logger.Debug().Str("service", "camera").Str("detections", p).
			Msg("sent to detections topic")
		qtErr := s.mqttClient.Publish(topic, p)
		if qtErr != nil {
			logger.Error().Str("service", "camera").Str("thing", "mqttPublish").
				Err(qtErr).Send()
			return
		}
		return
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		// Update MQTT via FrameRepo
		repoErr := s.FrameRepo.PublishFrame(frame, framePath, strconv.Itoa(int(deviceId)))
		if repoErr != nil {
			logger.Error().Msgf("CameraService.startStream.FrameRepo.PublishFrame threw an error %v", repoErr)
		}
		return
	}()

	wg.Wait()
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

func detectionServiceToPg(deviceId int64, imageID *int64, d detection.Detection) devices.CreateDetectionParams {
	Bbox := [][]float64{
		{d.Bbox.X1, d.Bbox.Y1},
		{d.Bbox.X2, d.Bbox.Y2},
	}
	return devices.CreateDetectionParams{
		DeviceID:   deviceId,
		Label:      d.Label,
		Confidence: d.Confidence,
		ImageID:    imageID,
		Bbox:       Bbox,
	}
}
