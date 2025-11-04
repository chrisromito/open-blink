package repos

import (
	"context"
	"devicecapture/internal/device/devices"
	"devicecapture/internal/postgres/db"
	"strconv"
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

// CreateDetection create a new device detection record
func (d *PgDetectionRepo) CreateDetection(ctx context.Context, params devices.CreateDetectionParams) (*devices.Detection, error) {
	dbParams := db.CreateDetectionParams{
		DeviceID:   params.DeviceID,
		Label:      params.Label,
		Confidence: params.Confidence,
	}
	detect, err := d.queries.CreateDetection(ctx, dbParams)
	if err != nil {
		return nil, err
	}
	return d.dbToDomain(detect), nil
}

// GetDetectionsAfter get all device detections after the specified point in time
func (d *PgDetectionRepo) GetDetectionsAfter(ctx context.Context, params devices.QueryParams) ([]devices.Detection, error) {
	value, err := d.queries.GetDetectionsAfter(ctx, params.CreatedAt)
	if err != nil {
		return nil, err
	}
	var detections []devices.Detection
	for _, detection := range value {
		detections = append(detections, *d.dbToDomain(detection))
	}
	return detections, nil
}

// GetDeviceDetectionsAfter get detections for a given device, after the specified point in time
func (d *PgDetectionRepo) GetDeviceDetectionsAfter(ctx context.Context, params devices.QueryParams) ([]devices.Detection, error) {
	dbParams, err := d.toDbQueryParams(params)
	if err != nil {
		return nil, err
	}
	value, err2 := d.queries.GetDeviceDetectionsAfter(ctx, *dbParams)
	if err2 != nil {
		return nil, err2
	}
	var detections []devices.Detection
	for _, detection := range value {
		detections = append(detections, *d.dbToDomain(detection))
	}
	return detections, nil
}

// dbToDomain convert a db.Detection to a devices.Detection
func (d *PgDetectionRepo) dbToDomain(value db.Detection) *devices.Detection {
	return &devices.Detection{
		ID:         value.ID,
		DeviceID:   value.DeviceID,
		CreatedAt:  value.CreatedAt,
		Label:      value.Label,
		Confidence: value.Confidence,
	}
}

// toDbQueryParams convert devices.QueryParams -> db.GetDeviceDetectionsAfterParams
func (d *PgDetectionRepo) toDbQueryParams(params devices.QueryParams) (*db.GetDeviceDetectionsAfterParams, error) {
	id, err := strconv.ParseInt(params.DeviceID, 10, 64)
	if err != nil {
		return nil, err
	}
	return &db.GetDeviceDetectionsAfterParams{
		DeviceID:  id,
		CreatedAt: params.CreatedAt,
	}, nil
}
