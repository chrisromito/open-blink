package event

import (
	"context"
	"slices"
	"strings"

	"devicecapture/internal/config"
	dEvent "devicecapture/internal/domain/event"
	"devicecapture/internal/logger"
	"devicecapture/internal/postgres/db"
	"devicecapture/internal/postgres/repos"
)

type PgDetectionEventRepo struct {
	queries *db.Queries
	config  *config.Config
}

func NewPgDetectionEventRepo(q *db.Queries, conf *config.Config) *PgDetectionEventRepo {
	return &PgDetectionEventRepo{
		queries: q,
		config:  conf,
	}
}

// StartEvent implements [dEvent.DetectionEventRepo]
func (de *PgDetectionEventRepo) StartEvent(
	ctx context.Context,
	deviceID int64,
	labels []string,
) (dEvent.DetectionEvent, error) {
	dbLabels := LabelsToDb(labels)
	p := db.StartEventParams{
		DeviceID: deviceID,
		Labels:   dbLabels,
		State:    int(dEvent.Started),
	}
	record, err := de.queries.StartEvent(ctx, p)
	if err != nil {
		return dEvent.DetectionEvent{}, err
	}
	return de.dbToDomain(record), nil
}

// EndEvent implements [dEvent.DetectionEventRepo]
func (de *PgDetectionEventRepo) EndEvent(
	ctx context.Context,
	e dEvent.DetectionEvent,
) (dEvent.DetectionEvent, error) {
	record, err := de.queries.EndEvent(ctx, e.ID)
	if err != nil {
		return dEvent.DetectionEvent{}, err
	}

	return de.dbToDomain(record), nil
}

var pageSize = int32(100)

// GetDetectionEvents implements [dEvent.DetectionEventRepo]
func (de *PgDetectionEventRepo) GetDetectionEvents(
	ctx context.Context,
	p dEvent.QueryParams,
) ([]dEvent.DetectionEvent, error) {
	page := int32(p.Page)
	if page < 1 {
		page = 1
	}
	params := db.GetEventsParams{
		DeviceID: p.DeviceID,
		Lim:      pageSize,
		Off:      (page - 1) * pageSize,
		Startdt:  p.Start,
		Enddt:    p.End,
	}
	if p.State != 0 {
		params.State = int32(p.State)
	}
	rows, err := de.queries.GetEvents(ctx, params)
	if err != nil {
		return []dEvent.DetectionEvent{}, err
	}
	var temp []dEvent.DetectionEvent
	for _, row := range rows {
		temp = append(temp, de.dbToDomain(row))
	}
	return temp, nil
}

// GetDetectionsForEvent implements [dEvent.DetectionEventRepo]
func (de *PgDetectionEventRepo) GetDetectionsForEvent(
	ctx context.Context,
	eventID int64,
) (dEvent.DetectionDetail, error) {
	results, err := de.queries.GetEventDetails(ctx, eventID)
	if err != nil {
		logger.Error().Err(err).Send()
		return dEvent.DetectionDetail{}, err
	}
	details := de.detailsDbToDomain(results)
	return details, nil
}

// DetectionsForImage populates the Details field for [dEvent.DetectionDetail]
func DetectionsForImage(rows []db.GetEventDetailsRow, imageID int64) []dEvent.DetectionForImage {
	var temp []dEvent.DetectionForImage
	logger.Debug().Str("event", "DetectionsForImage").
		Int("# rows", len(rows)).Send()
	for _, row := range rows {
		if row.ImageID == imageID {
			temp = append(temp, dEvent.DetectionForImage{
				ID:         row.DetectionID,
				Label:      row.Label,
				Confidence: row.Confidence,
			})
		}
	}
	return temp
}

// LabelsToDb converts a slice of strings to a sorted, comma-separated string of values
func LabelsToDb(labels []string) string {
	ls := slices.Clone(labels)
	slices.Sort(ls)
	return strings.Join(ls, ", ")
}

func DbToLabels(label string) []string {
	return strings.Split(label, ", ")
}

// dbToDomain converts a [db.DetectionEvent] to a [dEvent.DetectionEvent]
func (de *PgDetectionEventRepo) dbToDomain(evt db.DetectionEvent) dEvent.DetectionEvent {
	return dEvent.DetectionEvent{
		ID:        evt.ID,
		DeviceID:  evt.DeviceID,
		CreatedAt: evt.CreatedAt,
		EndedAt:   evt.EndedAt,
		State:     evt.State,
		Labels:    DbToLabels(evt.Labels),
	}
}

func (de *PgDetectionEventRepo) detailsDbToDomain(
	rows []db.GetEventDetailsRow,
) dEvent.DetectionDetail {
	temp := dEvent.DetectionDetail{
		Details: []dEvent.ImageDetails{},
	}
	var usedImageIDs []int64
	//var details []dEvent.ImageDetails
	for _, row := range rows {
		if temp.ID == 0 {
			temp.ID = row.ID
			temp.CreatedAt = row.CreatedAt
			temp.EndedAt = row.EndedAt
			temp.DeviceID = row.DeviceID
			temp.State = row.State
		}
		// Avoid duplicate ImageIDs
		if !slices.Contains(usedImageIDs, row.ImageID) {
			usedImageIDs = append(usedImageIDs, row.ImageID)
			detections := DetectionsForImage(rows, row.ImageID)
			temp.Details = append(temp.Details, dEvent.ImageDetails{
				ID: row.ImageID,
				ImageUrl: repos.UrlForDetection(
					de.config.ThisIp,
					row.ImagePath,
					row.AnnotatedPath,
				),
				Detections: detections,
				CreatedAt:  row.DetectedAt,
			})
		}
	}
	return temp
}
