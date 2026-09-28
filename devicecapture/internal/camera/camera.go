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
)

// CameraService provides high-level methods for interacting with camera devices
type CameraService struct {
	Config        *config.Config
	DeviceRepo    devices.DeviceRepository
	FrameRepo     receiver.Publisher
	DetectionRepo devices.DetectionRepo
	Detector      detection.ObjectDetector
	ImageRepo     devices.ImageRepo
	tracker       *DetectionTracker
	ingestor      *frameIngestor
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
		FrameRepo:     deps.Publisher,
		DetectionRepo: deps.DetectionRepo,
		ImageRepo:     deps.ImageRepo,
		connectedIds:  ids,
		Detector:      detector,
		ingestor:      newFrameIngestor(conf, deps),
		tracker:       NewDetectionTracker(deps),
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
	defer func(FrameRepo receiver.Publisher) {
		_ = FrameRepo.EndSession(session)
	}(s.FrameRepo)

	// Get a frame and pass it down the pipe
	frame, err := api.Snapshot(ctx)
	if err != nil {
		return err
	}
	return s.ingestFrame(ctx, session, d, frame)
}

// StreamSnapshots streams [receiver.Frame] from the [devices.Device] for [seconds].
// This method uses the `snapshot` endpoint, which is less taxing on the camera device than streaming MJPEG frames.
func (s *CameraService) StreamSnapshots(ctx context.Context, d devices.Device, seconds int) error {
	stringId := d.StringId()
	if s.IsStreaming(stringId) {
		return errors.New("cannot start stream for device")
	}

	s.addId(stringId)
	defer s.removeId(stringId)

	session, sErr := s.FrameRepo.StartSession(stringId)
	if sErr != nil {
		return sErr
	}
	defer func(FrameRepo receiver.Publisher) {
		_ = FrameRepo.EndSession(session)
	}(s.FrameRepo)

	api := NewApi(stringId, d.DeviceUrl)
	streamCtx, cancel := context.WithTimeout(ctx, time.Duration(seconds)*time.Second)
	defer cancel()

	// Allow backpressure of 3 frames
	imgChan := make(chan receiver.Frame, 3)
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

// runApiStreamer handles connecting to the API and passing frames to the "imgChan" channel
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

// runFrameProcessor receives frames and generates file paths before handing off to [CameraService.ingestFrame]
func (s *CameraService) runFrameProcessor(
	wg *sync.WaitGroup,
	ctx context.Context,
	device devices.Device,
	session *receiver.CaptureSession,
	imgChan <-chan receiver.Frame,
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

// ingestFrame handles the Detector calls and passes the BatchResponse to the ingestor
func (s *CameraService) ingestFrame(
	ctx context.Context,
	cs *receiver.CaptureSession,
	device devices.Device,
	frame receiver.Frame,
) error {
	br, bErr := s.batchResponseFromFrame(ctx, device.ID, frame)
	if bErr != nil {
		return bErr
	}
	err := s.receiveBatchResponse(ctx, cs, device, br)
	return err
}

// batchResponseFromFrame helper func for API for receiver.Frame -> detection.BatchResponse
func (s *CameraService) batchResponseFromFrame(
	ctx context.Context,
	deviceId int64,
	frame receiver.Frame,
) (detection.BatchResponse, error) {
	req := detection.Req{
		DeviceId: deviceId,
		Frame:    frame,
	}
	b, err := s.Detector.BatchDetections(ctx, req)
	return b, err
}

// receiveBatchResponse helper method for ingestor and updating the tracker
//
//	Guarantees an eventID for repo associativity
func (s *CameraService) receiveBatchResponse(
	ctx context.Context,
	cs *receiver.CaptureSession,
	device devices.Device,
	br detection.BatchResponse,
) error {
	evtID := s.tracker.CurrentEventID(device.ID)
	logger.Debug().Str("CameraService", "receiveBatchResponse").
		Int64("eventID", evtID).Send()
	if evtID == 0 {
		logger.Debug().Str("CameraService", "receiveBatchResponse").
			Msg("evtID was empty, starting a new event")
		var labels []string
		for _, d := range br.Detections {
			labels = append(labels, d.Label)
		}

		// Empty means we don't have an event, so we need to start one
		err := s.tracker.Start(ctx, device.ID, labels)
		if err != nil {
			return err
		}
		evtID = s.tracker.CurrentEventID(device.ID)
		logger.Debug().Str("CameraService", "receiveBatchResponse").
			Strs("labels", labels).
			Int64("EventID", evtID).
			Msg("Event Labels")
	}
	detections, err := s.ingestor.ReceiveBatch(ctx, cs, br, &evtID)
	if err != nil {
		return err
	}
	sds := detection.DeviceDetectionsToService(detections)
	return s.tracker.ReceiveDetections(ctx, device.ID, sds)
}

//// receiveDetectionResponse takes a [receiver.Frame], and the object detection results [detection.Detection]
//// for that frame, and disseminates them to the DB (via DetectionRepo), MQTT (via FrameRepo),
//// and annotates the original images with the detection results ([devices.DeviceImage.AnnotatedPath])
//// where the annotated image is then stored to the file system
//func (s *CameraService) receiveDetectionResponse(
//	ctx context.Context,
//	device devices.Device,
//	frame receiver.Frame,
//	detections []detection.Detection,
//	framePath string,
//	ap string,
//) ([]devices.Detection, error) {
//	eventID := s.tracker.CurrentEventID(device.ID)
//	imageRecord, err := s.ImageRepo.CreateImage(
//		ctx,
//		devices.CreateImageParams{
//			DeviceID:      device.ID,
//			ImagePath:     framePath,
//			AnnotatedPath: ap,
//			EventID:       &eventID,
//		},
//	)
//	if err != nil {
//		logger.Error().Str("service", "camera.trackDetections").
//			Str("threwFrom", "camera.ImageRepo.CreateImage").
//			Str("failed to save image to", framePath).
//			Err(err).Send()
//		return []devices.Detection{}, err
//	}
//	//s.tracker.AddImage(device.ID, imageRecord.ID)
//	// We have >= 1 detection, draw the bboxes
//	err = writeAnnotatedImage(frame, ap, detections)
//	if err != nil {
//		logger.Error().Str("service", "camera.trackDetections").
//			Str("threwFrom", "camera.ImageRepo.CreateImage").
//			Str("failed to save image to", ap).
//			Err(err).Send()
//		return []devices.Detection{}, err
//	}
//	// Store the detections in the DB & broadcast to MQTT
//	logger.Debug().
//		Str("annotatedImage", ap).
//		Str("what", "wrote to disk").
//		Any("detections", detections).
//		Msg("Wrote annotated image to disk")
//	var pgDetections []devices.CreateDetectionParams
//	// Set up the slice of DB params
//	partial := devices.CreateDetectionParams{
//		ImageID: &imageRecord.ID,
//		EventID: &eventID,
//	}
//	for _, d := range detections {
//		pgDetections = append(pgDetections, detectionServiceToPg(partial, d))
//	}
//	// Write to the DB
//	toPublish, err := s.DetectionRepo.CreateDetections(ctx, pgDetections)
//	if err != nil {
//		logger.Error().Str("service", "camera.DetectionRepo.CreateDetections").
//			Err(err).Send()
//		return []devices.Detection{}, err
//	}
//
//	var dids []int64
//	for _, det := range toPublish {
//		dids = append(dids, det.ID)
//	}
//
//	// Publish batch to MQTT
//	err = s.FrameRepo.PublishDetections(device.ID, imageRecord, toPublish)
//	return toPublish, err
//}

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

// writeAnnotatedImage draws [detections] on the given frame [f] and writes it to the file-system at "path"
func writeAnnotatedImage(f receiver.Frame, path string, detections []detection.Detection) error {
	// We have >= 1 detection, draw the bboxes
	anno := DrawDetections(f.Image, detections)
	file, err := os.Create(path)
	defer func(fil *os.File) {
		_ = fil.Close()
	}(file)
	if err != nil {
		logger.Error().Msgf("error writing frame to file %v @ %s", f.Timestamp, path)
		return err
	}
	err2 := jpeg.Encode(file, anno, nil)
	if err2 != nil {
		logger.Error().Msgf("error encoding frame to JPEG for %s", path)
		return err2
	}
	return nil
}

func detectionServiceToPg(
	partial devices.CreateDetectionParams,
	d detection.Detection,
) devices.CreateDetectionParams {
	Bbox := [][]float64{
		{d.Bbox.X1, d.Bbox.Y1},
		{d.Bbox.X2, d.Bbox.Y2},
	}
	return devices.CreateDetectionParams{
		Label:      d.Label,
		Confidence: d.Confidence,
		Bbox:       Bbox,
		DeviceID:   partial.DeviceID,
		ImageID:    partial.ImageID,
		EventID:    partial.EventID,
	}
}
