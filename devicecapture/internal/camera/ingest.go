package camera

import (
	"context"

	"devicecapture/internal/config"
	"devicecapture/internal/domain"
	"devicecapture/internal/domain/detection"
	"devicecapture/internal/domain/devices"
	"devicecapture/internal/domain/receiver"
	"devicecapture/internal/logger"
)

type frameIngestor struct {
	Config        *config.Config
	DeviceRepo    devices.DeviceRepository
	FrameRepo     receiver.Publisher
	DetectionRepo devices.DetectionRepo
	ImageRepo     devices.ImageRepo
}

func newFrameIngestor(conf *config.Config, deps *domain.Deps) *frameIngestor {
	return &frameIngestor{
		Config:        conf,
		DeviceRepo:    deps.DeviceRepo,
		FrameRepo:     deps.Publisher,
		DetectionRepo: deps.DetectionRepo,
		ImageRepo:     deps.ImageRepo,
	}
}

func (fi *frameIngestor) ReceiveBatch(
	ctx context.Context,
	cs *receiver.CaptureSession,
	batch detection.BatchResponse,
	eventID *int64,
) ([]devices.Detection, error) {
	frame := batch.Frame
	cs.SetLastFrame(&frame)

	if len(batch.Detections) < 1 {
		// Exclude the AnnotatedPath if there aren't any detections
		_, err := fi.ImageRepo.CreateImage(
			ctx,
			devices.CreateImageParams{
				DeviceID:  batch.DeviceId,
				ImagePath: receiver.FramePath(fi.Config.VideoPath, cs, frame),
				EventID:   eventID,
			},
		)
		return []devices.Detection{}, err
	}

	fsErr := fi.writeFrameToFs(cs, batch)
	if fsErr != nil {
		return []devices.Detection{}, fsErr
	}
	return fi.ingestBatch(ctx, cs, eventID, batch)
}

func (fi *frameIngestor) writeFrameToFs(
	cs *receiver.CaptureSession,
	batch detection.BatchResponse,
) error {
	fp := receiver.FramePath(fi.Config.VideoPath, cs, batch.Frame)
	fsErr := fi.FrameRepo.WriteFrame(batch.Frame, fp)
	if fsErr != nil {
		logger.Error().Err(fsErr).Str("FramePath:", fp).
			Msg("Failed to write frame to FS")
	}
	return fsErr
}

func (fi *frameIngestor) ingestBatch(
	ctx context.Context,
	cs *receiver.CaptureSession,
	eventID *int64,
	batch detection.BatchResponse,
) ([]devices.Detection, error) {
	ap := receiver.AnnotatedPath(fi.Config.VideoPath, cs, batch.Frame)
	drawErr := writeAnnotatedImage(batch.Frame, ap, batch.Detections)
	if drawErr != nil {
		return []devices.Detection{}, drawErr
	}
	p := devices.CreateImageParams{
		DeviceID:      batch.DeviceId,
		ImagePath:     receiver.FramePath(fi.Config.VideoPath, cs, batch.Frame),
		AnnotatedPath: ap,
		EventID: eventID,
	}
	imageRecord, irErr := fi.ImageRepo.CreateImage(ctx, p)
	if irErr != nil {
		return []devices.Detection{}, irErr
	}
	// Store the detections in the DB & broadcast to MQTT
	logger.Debug().
		Str("frameIngestor", "ingestBatch").
		Str("annotatedImage", ap).
		Str("what", "wrote to disk").
		Any("detections", batch.Detections).
		Int64("eventID", *eventID).
		Msg("Wrote annotated image to disk")
	var pgDetections []devices.CreateDetectionParams
	// Set up the slice of DB params
	partial := devices.CreateDetectionParams{
		DeviceID: batch.DeviceId,
		ImageID: &imageRecord.ID,
		EventID: eventID,
	}
	for _, d := range batch.Detections {
		pgDetections = append(
			pgDetections,
			detectionServiceToPg(partial, d),
		)
	}
	// Write to the DB
	toPublish, err := fi.DetectionRepo.CreateDetections(ctx, pgDetections)
	if err != nil {
		logger.Error().Str("service", "camera.DetectionRepo.CreateDetections").
			Err(err).Send()
		return []devices.Detection{}, err
	}

	// Publish batch to MQTT
	err = fi.FrameRepo.PublishDetections(batch.DeviceId, imageRecord, toPublish)
	return toPublish, err
}
