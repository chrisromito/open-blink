package repos

import (
	"context"

	"devicecapture/internal/domain/devices"
	"devicecapture/internal/logger"
	"devicecapture/internal/postgres/db"
)

// PgDetectionRepo implements devices.DetectionRepo
type PgDetectionRepo struct {
	queries *db.Queries
}

func NewPgDetectionRepo(queries *db.Queries) *PgDetectionRepo {
	return &PgDetectionRepo{
		queries: queries,
	}
}

// GetDetectionsAfter get all domain detections after the specified point in time
func (d *PgDetectionRepo) GetDetectionsAfter(
	ctx context.Context,
	params devices.QueryParams,
) ([]devices.Detection, error) {
	value, err := d.queries.GetDetectionsAfter(ctx, params.CreatedAt)
	if err != nil {
		return nil, err
	}
	var detections []devices.Detection
	for _, detection := range value {
		detections = append(detections, d.dbToDomain(detection))
	}
	return detections, nil
}

// GetDeviceDetectionsAfter get detections for a given domain, after the specified point in time
func (d *PgDetectionRepo) GetDeviceDetectionsAfter(
	ctx context.Context,
	params devices.QueryParams,
) ([]devices.Detection, error) {
	dbParams, err := d.toDbQueryParams(params)
	if err != nil {
		return nil, err
	}
	value, err2 := d.queries.GetDeviceDetectionsAfter(ctx, dbParams)
	if err2 != nil {
		return nil, err2
	}
	var detections []devices.Detection
	logger.Debug().
		Msgf("postgres.repos.detections -> GetDeviceDetectionsAfter -> #: %d", len(detections))
	for _, detection := range value {
		detections = append(detections, d.dbToDomain(detection))
	}
	return detections, nil
}

// GetDetectionsForEvent implements [devices.DetectionRepo]
func (d *PgDetectionRepo) GetDetectionsForEvent(
	ctx context.Context,
	eventID int64,
) ([]devices.Detection, error) {
	rows, err := d.queries.GetDetectionsForEvent(ctx, eventID)
	if err != nil {
		return []devices.Detection{}, err
	}
	var detections []devices.Detection
	for _, row := range rows {
		detections = append(detections, d.dbToDomain(row))
	}
	return detections, nil
}

// CreateDetection create a new domain detection record
func (d *PgDetectionRepo) CreateDetection(
	ctx context.Context,
	params devices.CreateDetectionParams,
) (devices.Detection, error) {
	dbParams := db.CreateDetectionParams{
		DeviceID:   params.DeviceID,
		Label:      params.Label,
		Confidence: params.Confidence,
		ImageID:    params.ImageID,
		Bbox:       params.Bbox,
	}
	detect, err := d.queries.CreateDetection(ctx, dbParams)
	if err != nil {
		return devices.Detection{}, err
	}
	return d.dbToDomain(detect), nil
}

func (d *PgDetectionRepo) CreateDetections(
	ctx context.Context,
	params []devices.CreateDetectionParams,
) ([]devices.Detection, error) {
	var value []devices.Detection
	for _, p := range params {
		record, err := d.queries.CreateDetection(ctx, db.CreateDetectionParams{
			DeviceID:   p.DeviceID,
			Label:      p.Label,
			Confidence: p.Confidence,
			ImageID:    p.ImageID,
			Bbox:       p.Bbox,
		})
		if err != nil {
			return value, err
		}
		value = append(value, d.dbToDomain(record))
	}
	return value, nil
}

func (d *PgDetectionRepo) SetEvent(
	ctx context.Context,
	ids []int64,
	eventID int64,
) error {
	return d.queries.SetEventForDetections(ctx, db.SetEventForDetectionsParams{
		EventID: eventID,
		Ids:     ids,
	})
}

// DeleteDetections deletes detection records for a given deviceId
func (d *PgDetectionRepo) DeleteDetections(ctx context.Context, deviceID int64) error {
	err := d.queries.DeleteDetections(ctx, deviceID)
	return err
}

// dbToDomain convert a db.Detection to a devices.Detection
func (d *PgDetectionRepo) dbToDomain(value db.Detection) devices.Detection {
	return devices.Detection{
		ID:         value.ID,
		DeviceID:   value.DeviceID,
		ImageID:    value.ImageID,
		EventID:    value.EventID,
		CreatedAt:  value.CreatedAt,
		Label:      value.Label,
		Confidence: value.Confidence,
		Bbox:       value.Bbox,
	}
}

// toDbQueryParams convert devices.QueryParams -> db.GetDeviceDetectionsAfterParams
func (d *PgDetectionRepo) toDbQueryParams(
	params devices.QueryParams,
) (db.GetDeviceDetectionsAfterParams, error) {
	return db.GetDeviceDetectionsAfterParams{
		DeviceID:  params.DeviceID,
		CreatedAt: params.CreatedAt,
	}, nil
}
