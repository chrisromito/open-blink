package deviceimage

import (
	"bytes"
	"context"
	"devicecapture/internal/camera"
	"devicecapture/internal/logger"
	"fmt"
	"image"
	"image/jpeg"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"path/filepath"
	"strings"

	"devicecapture/internal/config"
	"devicecapture/internal/domain"
	"devicecapture/internal/domain/detection"
	"devicecapture/internal/domain/devices"
	"devicecapture/internal/domain/queue"
)

type DeviceImageProcessor struct {
	Config   *config.Config
	Deps     *domain.Deps
	Detector detection.ObjectDetector
}

func NewDeviceImageProcessor(
	conf *config.Config,
	deps *domain.Deps,
	detector detection.ObjectDetector,
) *DeviceImageProcessor {
	return &DeviceImageProcessor{
		Config:   conf,
		Deps:     deps,
		Detector: detector,
	}
}

// HandleDeviceImageCreated receives DeviceImages & Writes DeviceDetections
func (p *DeviceImageProcessor) HandleDeviceImageCreated(
	ctx context.Context,
	msg Message,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	if p == nil {
		return fmt.Errorf("handle device image created: processor is nil")
	}
	if p.Deps == nil {
		return fmt.Errorf("handle device image created: deps are required")
	}
	if p.Detector == nil {
		return fmt.Errorf("handle device image created: detector is required")
	}

	img := msg.Image
	if img.ID == 0 {
		return fmt.Errorf("handle device image created: image id is required")
	}
	if img.DeviceID == 0 {
		return fmt.Errorf("handle device image created: device id is required")
	}
	if img.ImagePath == "" {
		return fmt.Errorf("handle device image created: image path is required")
	}

	frame, err := frameFromDeviceImage(img)
	if err != nil {
		return fmt.Errorf("load original image %q: %w", img.ImagePath, err)
	}
	logger.Debug().
		Str("DeviceImageProcessor", "HandleDeviceImageCreated").
		Int64("imageID", img.ID).
		Int64("deviceID", img.DeviceID).
		Str("imagePath", img.ImagePath).
		Int("frameBytes", len(frame.Buf)).
		Int64("frameTimestamp", frame.Timestamp).
		Msg("loaded frame from device image")

	detections, err := p.Detector.DetectObjectsForImage(ctx, detection.Req{
		DeviceId: img.DeviceID,
		Frame:    frame,
	})
	if err != nil {
		return fmt.Errorf("detect objects for image %d: %w", img.ID, err)
	}
	logger.Debug().
		Str("DeviceImageProcessor", "HandleDeviceImageCreated").
		Int64("imageID", img.ID).
		Int64("deviceID", img.DeviceID).
		Int("detectionCount", len(detections)).
		Interface("detections", detections).
		Msg("detected objects for image")

	if err := ctx.Err(); err != nil {
		return err
	}

	logger.Debug().
		Str("DeviceImageProcessor", "HandleDeviceImageCreated").
		Int64("imageID", img.ID).
		Int64("deviceID", img.DeviceID).
		Int("detectionCount", len(detections)).
		Msg("tracker received detections")

	//event, evtErr := p.Deps.EventRepo.GetLatest(ctx, img.DeviceID)
	//if evtErr != nil {
	//	return evtErr
	//}
	//eventID := event.ID
	//if eventID == 0 {
	//	return fmt.Errorf("detection event was not started for device %d", img.DeviceID)
	//}
	//logger.Debug().
	//	Str("DeviceImageProcessor", "HandleDeviceImageCreated").
	//	Int64("imageID", img.ID).
	//	Int64("deviceID", img.DeviceID).
	//	Int64("eventID", eventID).
	//	Msg("resolved current detection event")

	annotatedPath := annotatedPathFromImagePath(img.ImagePath)
	if err := os.MkdirAll(filepath.Dir(annotatedPath), 0o755); err != nil {
		return fmt.Errorf("create annotated image directory %q: %w", filepath.Dir(annotatedPath), err)
	}
	logger.Debug().
		Str("DeviceImageProcessor", "HandleDeviceImageCreated").
		Int64("imageID", img.ID).
		Int64("deviceID", img.DeviceID).
		Str("annotatedPath", annotatedPath).
		Msg("created annotated image directory")

	if err := writeAnnotatedImage(frame, annotatedPath, detections); err != nil {
		return fmt.Errorf("write annotated image %q: %w", annotatedPath, err)
	}
	logger.Debug().
		Str("DeviceImageProcessor", "HandleDeviceImageCreated").
		Int64("imageID", img.ID).
		Int64("deviceID", img.DeviceID).
		Str("annotatedPath", annotatedPath).
		Int("detectionCount", len(detections)).
		Msg("wrote annotated image")

	updatedImage, err := p.Deps.ImageRepo.UpdateImage(ctx, devices.UpdateImageParams{
		ID:            img.ID,
		AnnotatedPath: annotatedPath,
	})
	if err != nil {
		return fmt.Errorf("update device image %d: %w", img.ID, err)
	}
	logger.Debug().
		Str("DeviceImageProcessor", "HandleDeviceImageCreated").
		Int64("imageID", img.ID).
		Int64("deviceID", img.DeviceID).
		Str("annotatedPath", annotatedPath).
		Interface("updatedImage", updatedImage).
		Msg("updated device image")

	imageID := img.ID
	createParams := make([]devices.CreateDetectionParams, 0, len(detections))
	for _, d := range detections {
		createParams = append(createParams, detectionServiceToPg(devices.CreateDetectionParams{
			DeviceID: img.DeviceID,
			ImageID:  &imageID,
		}, d))
	}
	logger.Debug().
		Str("DeviceImageProcessor", "HandleDeviceImageCreated").
		Int64("imageID", imageID).
		Int64("deviceID", img.DeviceID).
		Int("createDetectionParamCount", len(createParams)).
		Interface("createDetectionParams", createParams).
		Msg("built create detection params")

	createdDetections, err := p.Deps.DetectionRepo.CreateDetections(ctx, createParams)
	if err != nil {
		return fmt.Errorf("create detections for image %d: %w", imageID, err)
	}
	logger.Debug().
		Str("DeviceImageProcessor", "HandleDeviceImageCreated").
		Int64("imageID", imageID).
		Int64("deviceID", img.DeviceID).
		Int("createdDetectionCount", len(createdDetections)).
		Interface("createdDetections", createdDetections).
		Msg("created detections")

	if err := p.Deps.Publisher.PublishDetections(
		ctx,
		img.DeviceID,
		updatedImage,
		createdDetections,
	); err != nil {
		return fmt.Errorf("publish detections for image %d: %w", imageID, err)
	}
	logger.Debug().
		Str("DeviceImageProcessor", "HandleDeviceImageCreated").
		Int64("imageID", updatedImage.ID).
		Int64("deviceID", img.DeviceID).
		Int("publishedDetectionCount", len(createdDetections)).
		Msg("published detections")

	return nil
}

func frameFromDeviceImage(img devices.DeviceImage) (queue.Frame, error) {
	buf, err := os.ReadFile(img.ImagePath)
	if err != nil {
		return queue.Frame{}, err
	}

	decoded, _, err := image.Decode(bytes.NewReader(buf))
	if err != nil {
		return queue.Frame{}, err
	}

	var timestamp int64
	if !img.CreatedAt.IsZero() {
		timestamp = img.CreatedAt.UnixMilli()
	}

	return queue.Frame{
		Buf:       buf,
		Image:     decoded,
		Timestamp: timestamp,
	}, nil
}

func annotatedPathFromImagePath(imagePath string) string {
	dir := filepath.Dir(imagePath)
	base := filepath.Base(imagePath)
	ext := filepath.Ext(base)
	name := strings.TrimSuffix(base, ext)

	annotatedPath := filepath.Join(dir, name+"-annotated"+ext)
	return strings.ReplaceAll(annotatedPath, "_", "-")
}

// writeAnnotatedImage draws [detections] on the given frame [f] and writes it to the file-system at "path"
func writeAnnotatedImage(f queue.Frame, path string, detections []detection.Detection) error {
	// We have >= 1 detection, draw the bboxes
	anno := camera.DrawDetections(f.Image, detections)
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
