package camera

import (
	"context"
	"errors"
	"image/jpeg"
	"os"
	"slices"
	"strconv"
	"sync"
	"time"

	"devicecapture/internal/config"
	"devicecapture/internal/domain"
	"devicecapture/internal/domain/detection"
	"devicecapture/internal/domain/devices"
	"devicecapture/internal/domain/receiver"
	"devicecapture/internal/logger"
	"devicecapture/internal/pubsub"
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
func NewCameraService(
	conf *config.Config,
	deps *domain.Deps,
	detector detection.ObjectDetector,
	qtClient *pubsub.MqttClient,
) *CameraService {
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
	ap := receiver.AnnotatedPath(s.Config.VideoPath, session, frame)
	return s.receiveFrame(ctx, d, fp, ap, frame)
}

// StartStream wraps the process of checking if a device is streaming, validating the device ID,
// starting the CaptureSession, and managing the streaming workflow.
func (s *CameraService) StartStream(
	ctx context.Context,
	deviceId string,
) (*receiver.CaptureSession, error) {
	device, canStart := s.canStartStream(ctx, deviceId)
	if !canStart {
		return &receiver.CaptureSession{}, errors.New("cannot start stream for device")
	}

	s.addId(deviceId)
	defer s.removeId(deviceId)

	session, err := s.FrameRepo.StartSession(deviceId)
	if err != nil {
		return &receiver.CaptureSession{}, err
	}
	if session == nil {
		return &receiver.CaptureSession{}, errors.New("unexpected nil CaptureSession")
	}
	defer func() { _ = s.FrameRepo.EndSession(session) }()

	var wg sync.WaitGroup
	imgChan := make(chan receiver.Frame, 60) // 60 frames = 4 FPS * 15 seconds
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

// runApiStreamer handles connecting to the API and piping frames to the channel
func (s *CameraService) runApiStreamer(
	wg *sync.WaitGroup,
	ctx context.Context,
	deviceId string,
	deviceUrl string,
	imgChan chan<- receiver.Frame,
) {
	defer wg.Done()
	api := NewApi(deviceId, deviceUrl)
	if err := api.StreamFrames(ctx, imgChan); err != nil {
		logger.Error().Str("service", "camera.StartStream").
			Msgf("domain -> Start -> api worker -> Error streaming frames: %v", err)
	}
}

// runFrameProcessor consumes incoming frames and processes them
func (s *CameraService) runFrameProcessor(
	wg *sync.WaitGroup,
	ctx context.Context,
	device devices.Device, // Replace with your actual Device type if different
	session *receiver.CaptureSession,
	imgChan <-chan receiver.Frame,
) {
	defer wg.Done()
	for {
		select {
		case <-ctx.Done():
			return
		case img, ok := <-imgChan:
			if !ok {
				logger.Error().Str("service", "camera.StartStream").
					Msg("CameraService -> imgChan not ok, returning")
				return
			}

			// Fixed potential bug: processed frames only if session is NOT nil
			if session == nil {
				continue
			}

			session.SetLastFrame(&img)
			fp := receiver.FramePath(s.Config.VideoPath, session, img)
			ap := receiver.AnnotatedPath(s.Config.VideoPath, session, img)

			if err := s.receiveFrame(ctx, device, fp, ap, img); err != nil {
				logger.Error().Str("service", "camera.StartStream").
					Msgf("receiveFrame threw %v", err)
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

// detectionsForFrame calls [detection.ObjectDetector] using the given frame and returns the detection results
func (s *CameraService) detectionsForFrame(
	ctx context.Context,
	deviceId int64,
	frame receiver.Frame,
) ([]detection.Detection, error) {
	req := detection.Req{
		DeviceId: deviceId,
		Frame:    frame,
	}
	detections, dErr := s.Detector.DetectObjectsForImage(ctx, req)
	if dErr != nil {
		return []detection.Detection{}, errors.New("failed to get detections for frame")
	}
	return detections, nil
}

// receiveFrame processes a single frame by saving it as an image record and optionally
// performing object detection. If detection is enabled and objects are found, it stores
// the detections in the database and publishes them to MQTT. The frame is also passed
// to the FrameRepository for storage. All operations run concurrently using goroutines.
func (s *CameraService) receiveFrame(
	ctx context.Context,
	device devices.Device,
	framePath string,
	ap string,
	frame receiver.Frame,
) error {
	var wg sync.WaitGroup
	if cErr := ctx.Err(); cErr != nil {
		return nil
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		detections, dErr := s.detectionsForFrame(ctx, device.ID, frame)
		if dErr != nil {
			logger.Error().
				Str("service", "camera.detectionsForFrame").
				Err(dErr).Send()
			return
		}
		logger.Debug().Any("detections", detections).
			Msgf("\n\nCameraService: received detections: %v", detections)
		if len(detections) < 1 {
			// Exclude the AnnotatedPath if there aren't any detections
			_, err := s.ImageRepo.CreateImage(
				ctx,
				devices.CreateImageParams{
					DeviceID:      device.ID,
					ImagePath:     framePath,
					AnnotatedPath: "",
				},
			)
			if err != nil {
				logger.Error().Str("service", "camera.receiveFrame").
					Str("threwFrom", "camera.ImageRepo.CreateImage").
					Str("failed to save image to", framePath).
					Err(err).Send()
				return
			}
			return
		}
		err := s.receiveDetectionResponse(ctx, device, frame, detections, framePath, ap)
		if err != nil {
			logger.Error().Err(err).Send()
		}
		return
	}()

	// goroutine to write frames to disk and update MQTT
	wg.Add(1)
	go func() {
		defer wg.Done()
		// Write the Frame to disk
		fsErr := s.FrameRepo.WriteFrame(frame, framePath)
		if fsErr != nil {
			logger.Error().Err(fsErr).Str("FramePath:", framePath).
				Msg("Failed to write frame to FS")
			return
		}
		// Update MQTT via FrameRepo
		repoErr := s.FrameRepo.PublishFrame(frame, ap, strconv.Itoa(int(device.ID)))
		if repoErr != nil {
			logger.Error().
				Str("CameraService", "startStream.FrameRepo.PublishFrame").
				Err(repoErr).Send()
			return
		}
		return
	}()

	wg.Wait()
	return nil
}

// receiveDetectionResponse takes a [receiver.Frame], and the object detection results [detection.Detection]
// for that frame, and disseminates them to the DB (via DetectionRepo), MQTT (via FrameRepo),
// and annotates the original images with the detection results ([devices.DeviceImage.AnnotatedPath])
// where the annotated image is then stored to the file system
func (s *CameraService) receiveDetectionResponse(
	ctx context.Context,
	device devices.Device,
	frame receiver.Frame,
	detections []detection.Detection,
	framePath string,
	ap string,
) error {
	imageRecord, err := s.ImageRepo.CreateImage(
		ctx,
		devices.CreateImageParams{DeviceID: device.ID, ImagePath: framePath, AnnotatedPath: ap},
	)
	if err != nil {
		logger.Error().Str("service", "camera.receiveFrame").
			Str("threwFrom", "camera.ImageRepo.CreateImage").
			Str("failed to save image to", framePath).
			Err(err).Send()
		return err
	}
	// We have >= 1 detection, draw the bboxes
	anno := DrawDetections(frame.Image, detections)
	f, err := os.Create(ap)
	defer func(f *os.File) {
		_ = f.Close()
	}(f)
	if err != nil {
		logger.Error().Msgf("error writing frame to file %v @ %s", frame.Timestamp, framePath)
		return err
	}
	err2 := jpeg.Encode(f, anno, nil)
	if err2 != nil {
		logger.Error().Msgf("error encoding frame to JPEG for %s", framePath)
		return err2
	}
	// Store the detections in the DB & broadcast to MQTT
	logger.Debug().
		Str("annotatedImage", ap).
		Str("what", "wrote to disk").
		Any("detections", detections).
		Msg("Wrote annotated image to disk")
	// Loop, transpose items, and write to the repo
	topic := "detections/" + strconv.Itoa(int(device.ID))
	var pgDetections []devices.CreateDetectionParams
	// Set up the slice of DB params
	for _, d := range detections {
		pgDetections = append(pgDetections, detectionServiceToPg(device.ID, &imageRecord.ID, d))
	}
	// Write to the DB
	toPublish, err := s.DetectionRepo.CreateDetections(ctx, pgDetections)
	if err != nil {
		logger.Error().Str("service", "camera.DetectionRepo.CreateDetections").
			Err(err).Send()
		return err
	}
	// Marshal detections to JSON string
	p, jErr := receiver.DetectionsToMsg(s.Config.ThisIp, imageRecord, toPublish)
	if jErr != nil {
		logger.Error().Str("service", "camera").
			Err(jErr).Send()
		return jErr
	}
	// Publish batch to MQTT
	logger.Debug().Str("service", "camera").
		Str("detections", p).
		Msg("sent to detections topic")
	qtErr := s.mqttClient.Publish(topic, p)
	if qtErr != nil {
		logger.Error().Str("service", "camera").
			Str("thing", "mqttPublish").
			Err(qtErr).Send()
		return qtErr
	}
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

func detectionServiceToPg(
	deviceId int64,
	imageID *int64,
	d detection.Detection,
) devices.CreateDetectionParams {
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
