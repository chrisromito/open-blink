package camera

import (
	"context"
	"devicecapture/internal/config"
	"devicecapture/internal/domain"
	"devicecapture/internal/domain/detection"
	"devicecapture/internal/domain/devices"
	"devicecapture/internal/domain/receiver"
	"errors"
	"log"
	"slices"
	"strconv"
	"sync"
)

// CameraService provides high-level methods for interacting with camera devices
type CameraService struct {
	Config        *config.Config
	DeviceRepo    devices.DeviceRepository
	FrameRepo     receiver.FrameRepository
	DetectionRepo devices.DetectionRepo
	Detector      detection.ObjectDetector
	ImageRepo     devices.ImageRepo
	connectedIds  []string
	mu            sync.Mutex
}

func NewCameraService(deps *domain.Deps, detector detection.ObjectDetector) *CameraService {
	ids := make([]string, 10)
	cs := &CameraService{
		DeviceRepo:    deps.DeviceRepo,
		FrameRepo:     deps.FrameRepo,
		DetectionRepo: deps.DetectionRepo,
		ImageRepo:     deps.ImageRepo,
		connectedIds:  ids,
		Detector:      detector,
		mu:            sync.Mutex{},
	}
	return cs
}

//func WithDetection()

func (s *CameraService) IsValidId(deviceId string) bool {
	id, err := strconv.ParseInt(deviceId, 10, 64)
	d, err2 := s.DeviceRepo.GetDevice(context.Background(), id)
	return err == nil && err2 == nil && d != nil
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
		log.Printf("starting stream for device %s @ %s", deviceId, device.DeviceUrl)
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
		_ = FrameRepo.EndSession()
	}(s.FrameRepo)

	// wg ends when the stream is complete
	var wg sync.WaitGroup
	imgChan := make(chan receiver.Frame, 64)
	outChan := make(chan receiver.Frame, 64)
	done := make(chan error)
	defer close(imgChan)
	defer close(outChan)
	defer close(done)

	wg.Add(1)
	// api goroutine receives JPEGs from the API & passes them to imageChan
	go func() {
		defer wg.Done()
		api := NewApi(deviceId, device.DeviceUrl)
		apiErr := api.StreamFrames(ctx, imgChan)
		done <- apiErr
		if apiErr != nil {
			log.Printf("domain -> Start -> api worker -> Error streaming frames: %v", apiErr)
			return
		} else {
			return
		}
	}()

	// imgChan goroutine pulls from imgChan so they can be consumed via outChan
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-done:
				return
			case img, ok := <-imgChan:
				if !ok {
					log.Print("CamService -> imgChan not ok, returning")
					return
				}
				outChan <- img
			}
		}
	}()

	// outChan goroutine pulls from outChan & passes frames to CaptureSession & FrameRepo (respectively)
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-done:
				return
			case img, ok := <-outChan:
				if !ok {
					log.Print("CameraService -> outChan not ok, returning")
					return
				}
				fp := receiver.FramePath(s.Config.VideoPath, session, img)
				e := s.receiveFrame(ctx, id, fp, img)
				if e != nil {
					log.Fatalf("receiveFrame threw %v", e)
					return
				}
			}
		}
	}()

	wg.Wait()
	return session, nil
}

func (s *CameraService) receiveFrame(ctx context.Context, id int64, framePath string, frame receiver.Frame) error {
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		imageRecord, err := s.ImageRepo.CreateImage(ctx, devices.CreateImageParams{DeviceID: id, ImagePath: framePath})
		if err != nil {
			log.Printf("failed to save image to %s", framePath)
			return
		}
		req := detection.Req{
			DeviceId: id,
			Frame:    frame,
		}
		detections, dErr := s.Detector.DetectObjectsForImage(ctx, req)
		if dErr != nil {
			log.Printf("\n\ndetection err: %v", dErr)
		} else {
			log.Printf("\n\nCameraService: writing detections: %v", detections)
			// Loop, transpose items, and write to the repo
			imageID := &imageRecord.ID
			for _, d := range detections {
				params := detectionServiceToPg(id, imageID, d)
				_, detectionErr := s.DetectionRepo.CreateDetection(ctx, params)
				if detectionErr != nil {
					log.Printf("error writing detections to DB %v", detectionErr)
					return
				}
			}
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		// Update FrameRepo
		repoErr := s.FrameRepo.ReceiveFrame(frame, framePath)
		if repoErr != nil {
			log.Printf("CameraService.startStream.FrameRepo.ReceiveFrame threw an error %v", repoErr)
		}
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
	return devices.CreateDetectionParams{
		DeviceID:   deviceId,
		Label:      d.Label,
		Confidence: d.Confidence,
		ImageID:    imageID,
	}
}
