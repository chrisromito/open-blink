package deviceimage

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/color"
	"image/jpeg"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"devicecapture/internal/config"
	"devicecapture/internal/domain"
	"devicecapture/internal/domain/detection"
	"devicecapture/internal/domain/devices"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newDeviceImageProcessorTestDeps() *domain.Deps {
	deps := domain.NewMockDeps()
	deps.DetectionRepo = newRecordingDetectionRepo()
	return deps
}

type recordingDetectionRepo struct {
	mu         sync.Mutex
	nextID     int64
	detections []devices.Detection
}

func newRecordingDetectionRepo() *recordingDetectionRepo {
	return &recordingDetectionRepo{
		nextID:     1,
		detections: []devices.Detection{},
	}
}

func (r *recordingDetectionRepo) GetDetectionsAfter(
	_ context.Context,
	_ devices.QueryParams,
) ([]devices.Detection, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	return append([]devices.Detection{}, r.detections...), nil
}

func (r *recordingDetectionRepo) GetDeviceDetectionsAfter(
	_ context.Context,
	params devices.QueryParams,
) ([]devices.Detection, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	var results []devices.Detection
	for _, d := range r.detections {
		if d.DeviceID == params.DeviceID {
			results = append(results, d)
		}
	}

	return results, nil
}

func (r *recordingDetectionRepo) GetDetectionsForEvent(
	_ context.Context,
	eventID int64,
) ([]devices.Detection, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	var results []devices.Detection
	for _, d := range r.detections {
		if d.EventID != nil && *d.EventID == eventID {
			results = append(results, d)
		}
	}

	return results, nil
}

func (r *recordingDetectionRepo) SetEvent(
	_ context.Context,
	ids []int64,
	eventID int64,
) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	idSet := map[int64]bool{}
	for _, id := range ids {
		idSet[id] = true
	}

	for i := range r.detections {
		if idSet[r.detections[i].ID] {
			r.detections[i].EventID = &eventID
		}
	}

	return nil
}

func (r *recordingDetectionRepo) CreateDetection(
	_ context.Context,
	params devices.CreateDetectionParams,
) (devices.Detection, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.createDetectionLocked(params), nil
}

func (r *recordingDetectionRepo) CreateDetections(
	_ context.Context,
	params []devices.CreateDetectionParams,
) ([]devices.Detection, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	results := make([]devices.Detection, 0, len(params))
	for _, p := range params {
		results = append(results, r.createDetectionLocked(p))
	}

	return results, nil
}

func (r *recordingDetectionRepo) DeleteDetections(
	_ context.Context,
	deviceID int64,
) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	var kept []devices.Detection
	for _, d := range r.detections {
		if d.DeviceID != deviceID {
			kept = append(kept, d)
		}
	}

	r.detections = kept
	return nil
}

func (r *recordingDetectionRepo) createDetectionLocked(
	params devices.CreateDetectionParams,
) devices.Detection {
	d := devices.Detection{
		ID:         r.nextID,
		DeviceID:   params.DeviceID,
		ImageID:    params.ImageID,
		EventID:    params.EventID,
		CreatedAt:  time.Now(),
		Label:      params.Label,
		Confidence: params.Confidence,
		Bbox:       params.Bbox,
	}

	r.nextID++
	r.detections = append(r.detections, d)

	return d
}
func (r *recordingDetectionRepo) recordedDetections() []devices.Detection {
	r.mu.Lock()
	defer r.mu.Unlock()

	return append([]devices.Detection{}, r.detections...)
}

func Test_AnnotatedPathFromImagePath(t *testing.T) {
	tests := []struct {
		name      string
		imagePath string
		want      string
	}{
		{
			name:      "adds annotated suffix and replaces underscores",
			imagePath: filepath.Join("my_directory", "my_image.jpeg"),
			want:      filepath.Join("my-directory", "my-image-annotated.jpeg"),
		},
		{
			name:      "preserves extension",
			imagePath: filepath.Join("images", "front_door.jpg"),
			want:      filepath.Join("images", "front-door-annotated.jpg"),
		},
		{
			name:      "handles files without extension",
			imagePath: filepath.Join("images", "front_door"),
			want:      filepath.Join("images", "front-door-annotated"),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := annotatedPathFromImagePath(test.imagePath)

			assert.Equal(t, test.want, got)
		})
	}
}

func Test_FrameFromDeviceImage_LoadsImage(t *testing.T) {
	imagePath := writeTestJPEG(t, t.TempDir(), "frame.jpeg")
	createdAt := time.Now().Add(-1 * time.Minute)

	frame, err := frameFromDeviceImage(devices.DeviceImage{
		ImagePath: imagePath,
		CreatedAt: createdAt,
	})

	require.NoError(t, err)
	assert.NotNil(t, frame.Image)
	assert.NotEmpty(t, frame.Buf)
	assert.Equal(t, createdAt.UnixMilli(), frame.Timestamp)
}

func Test_FrameFromDeviceImage_ReturnsErrorWhenFileDoesNotExist(t *testing.T) {
	frame, err := frameFromDeviceImage(devices.DeviceImage{
		ImagePath: filepath.Join(t.TempDir(), "missing.jpeg"),
	})

	assert.Error(t, err)
	assert.Nil(t, frame.Image)
	assert.Empty(t, frame.Buf)
}

func Test_DeviceImageProcessor_HandleDeviceImageCreated_ValidationErrors(t *testing.T) {
	deps := domain.NewMockDeps()
	detector := &stubObjectDetector{}
	conf := config.Config{}

	tests := []struct {
		name    string
		process *DeviceImageProcessor
		msg     Message
		wantErr string
	}{
		{
			name:    "nil processor",
			process: nil,
			msg:     Message{},
			wantErr: "processor is nil",
		},
		{
			name: "missing deps",
			process: &DeviceImageProcessor{
				Config:   &conf,
				Detector: detector,
			},
			msg:     Message{},
			wantErr: "deps are required",
		},
		{
			name: "missing detector",
			process: &DeviceImageProcessor{
				Config: &conf,
				Deps:   deps,
			},
			msg:     Message{},
			wantErr: "detector is required",
		},
		{
			name:    "missing image id",
			process: NewDeviceImageProcessor(&conf, deps, detector),
			msg: Message{
				Image: devices.DeviceImage{
					DeviceID:  1,
					ImagePath: "image.jpeg",
				},
			},
			wantErr: "image id is required",
		},
		{
			name:    "missing device id",
			process: NewDeviceImageProcessor(&conf, deps, detector),
			msg: Message{
				Image: devices.DeviceImage{
					ID:        1,
					ImagePath: "image.jpeg",
				},
			},
			wantErr: "device id is required",
		},
		{
			name:    "missing image path",
			process: NewDeviceImageProcessor(&conf, deps, detector),
			msg: Message{
				Image: devices.DeviceImage{
					ID:       1,
					DeviceID: 1,
				},
			},
			wantErr: "image path is required",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.process.HandleDeviceImageCreated(t.Context(), test.msg)

			require.Error(t, err)
			assert.Contains(t, err.Error(), test.wantErr)
		})
	}
}

func Test_DeviceImageProcessor_HandleDeviceImageCreated_CreatesAnnotatedImageAndDetections(t *testing.T) {
	ctx := t.Context()
	deps := newDeviceImageProcessorTestDeps()

	detectionRepo, ok := deps.DetectionRepo.(*recordingDetectionRepo)
	require.True(t, ok)

	detector := &stubObjectDetector{
		detections: []detection.Detection{
			{
				Label:      "car",
				Confidence: 0.95,
				Bbox: detection.BBox{
					X1: 2,
					Y1: 3,
					X2: 20,
					Y2: 25,
				},
			},
		},
	}
	conf := config.Config{}
	processor := NewDeviceImageProcessor(&conf, deps, detector)

	deviceID := int64(77)
	imagePath := writeTestJPEG(t, t.TempDir(), "camera_frame.jpeg")
	img := createTestDeviceImage(t, ctx, deps, deviceID, imagePath)

	err := processor.HandleDeviceImageCreated(ctx, NewMessage(img))

	require.NoError(t, err)
	assert.Equal(t, 1, detector.calls)
	assert.Equal(t, deviceID, detector.lastReq.DeviceId)
	assert.NotNil(t, detector.lastReq.Frame.Image)

	annotatedPath := annotatedPathFromImagePath(imagePath)
	assert.FileExists(t, annotatedPath)

	updatedImage := requireImageByID(t, ctx, deps, deviceID, img.ID)
	assert.Equal(t, annotatedPath, updatedImage.AnnotatedPath)
	assert.Nil(t, updatedImage.EventID)

	createdDetections := detectionRepo.recordedDetections()
	require.Len(t, createdDetections, 1)

	gotDetection := createdDetections[0]
	assert.Equal(t, deviceID, gotDetection.DeviceID)
	assert.Equal(t, "car", gotDetection.Label)
	assert.Equal(t, 0.95, gotDetection.Confidence)
	require.NotNil(t, gotDetection.ImageID)
	assert.Equal(t, img.ID, *gotDetection.ImageID)
	assert.Nil(t, gotDetection.EventID)
}

func Test_DeviceImageProcessor_HandleDeviceImageCreated_DetectorError(t *testing.T) {
	ctx := t.Context()
	deps := domain.NewMockDeps()
	detectorErr := errors.New("detector unavailable")
	detector := &stubObjectDetector{
		err: detectorErr,
	}
	conf := config.Config{}
	processor := NewDeviceImageProcessor(&conf, deps, detector)

	deviceID := int64(78)
	imagePath := writeTestJPEG(t, t.TempDir(), "camera_frame.jpeg")
	img := createTestDeviceImage(t, ctx, deps, deviceID, imagePath)

	err := processor.HandleDeviceImageCreated(ctx, NewMessage(img))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "detect objects for image")
	assert.ErrorIs(t, err, detectorErr)

	annotatedPath := annotatedPathFromImagePath(imagePath)
	_, statErr := os.Stat(annotatedPath)
	assert.True(t, os.IsNotExist(statErr))
}

func Test_DeviceImageProcessor_HandleDeviceImageCreated_NoDetectionsCreatesAnnotatedImageOnly(t *testing.T) {
	ctx := t.Context()
	deps := newDeviceImageProcessorTestDeps()

	detectionRepo, ok := deps.DetectionRepo.(*recordingDetectionRepo)
	require.True(t, ok)

	detector := &stubObjectDetector{
		detections: []detection.Detection{},
	}
	conf := config.Config{}
	processor := NewDeviceImageProcessor(&conf, deps, detector)

	deviceID := int64(79)
	imagePath := writeTestJPEG(t, t.TempDir(), "camera_frame.jpeg")
	img := createTestDeviceImage(t, ctx, deps, deviceID, imagePath)

	err := processor.HandleDeviceImageCreated(ctx, NewMessage(img))

	require.NoError(t, err)
	assert.Equal(t, 1, detector.calls)

	annotatedPath := annotatedPathFromImagePath(imagePath)
	assert.FileExists(t, annotatedPath)

	updatedImage := requireImageByID(t, ctx, deps, deviceID, img.ID)
	assert.Equal(t, annotatedPath, updatedImage.AnnotatedPath)

	assert.Empty(t, detectionRepo.recordedDetections())
}

type stubObjectDetector struct {
	detections []detection.Detection
	err        error
	calls      int
	lastReq    detection.Req
}

func (s *stubObjectDetector) DetectObjectsForImage(
	_ context.Context,
	req detection.Req,
) ([]detection.Detection, error) {
	s.calls++
	s.lastReq = req

	if s.err != nil {
		return nil, s.err
	}

	return s.detections, nil
}

func (s *stubObjectDetector) BatchDetections(
	_ context.Context,
	req detection.Req,
) (detection.BatchResponse, error) {
	s.calls++
	s.lastReq = req

	if s.err != nil {
		return detection.BatchResponse{}, s.err
	}

	return detection.BatchResponse{
		DeviceId:   req.DeviceId,
		Detections: s.detections,
		Frame:      req.Frame,
	}, nil
}

func createTestDeviceImage(
	t *testing.T,
	ctx context.Context,
	deps *domain.Deps,
	deviceID int64,
	imagePath string,
) devices.DeviceImage {
	t.Helper()

	img, err := deps.ImageRepo.CreateImage(ctx, devices.CreateImageParams{
		DeviceID:  deviceID,
		ImagePath: imagePath,
	})
	require.NoError(t, err)
	require.NotZero(t, img.ID)

	return img
}

func requireImageByID(
	t *testing.T,
	ctx context.Context,
	deps *domain.Deps,
	deviceID int64,
	imageID int64,
) devices.DeviceImage {
	t.Helper()

	images, err := deps.ImageRepo.GetImages(ctx, deviceID)
	require.NoError(t, err)

	for _, img := range images {
		if img.ID == imageID {
			return img
		}
	}

	require.Failf(t, "image not found", "imageID=%d deviceID=%d", imageID, deviceID)
	return devices.DeviceImage{}
}

func writeTestJPEG(t *testing.T, dir string, name string) string {
	t.Helper()

	require.NoError(t, os.MkdirAll(dir, 0o755))

	img := image.NewRGBA(image.Rect(0, 0, 64, 64))
	for y := 0; y < 64; y++ {
		for x := 0; x < 64; x++ {
			img.Set(x, y, color.RGBA{
				R: uint8(x * 3),
				G: uint8(y * 3),
				B: 100,
				A: 255,
			})
		}
	}

	var buf bytes.Buffer
	require.NoError(t, jpeg.Encode(&buf, img, nil))

	path := filepath.Join(dir, name)
	require.NoError(t, os.WriteFile(path, buf.Bytes(), 0o644))

	return path
}
