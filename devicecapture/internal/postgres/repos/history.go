package repos

import (
	"context"
	"slices"

	"devicecapture/internal/config"
	"devicecapture/internal/domain/history"
	"devicecapture/internal/postgres/db"
)

// PgDetectionHistoryRepo implements history.DetectionHistoryRepo
type PgDetectionHistoryRepo struct {
	queries *db.Queries
	config  *config.Config
}

func NewPgDetectionHistoryRepo(queries *db.Queries, conf *config.Config) *PgDetectionHistoryRepo {
	return &PgDetectionHistoryRepo{
		queries: queries,
		config:  conf,
	}
}

func (h *PgDetectionHistoryRepo) GetRecentLabels(ctx context.Context) ([]string, error) {
	labels, err := h.queries.GetRecentLabels(ctx)
	return labels, err
}

func (h *PgDetectionHistoryRepo) GetDetectionImagesByLabel(
	ctx context.Context,
	params history.DetectionWithImageParams,
) ([]history.DetectionWithImage, error) {
	rows, err := h.queries.GetDetectionImagesByLabel(ctx, db.GetDetectionImagesByLabelParams{
		Label:     params.Label,
		DeviceID:  params.DeviceID,
		CreatedAt: params.CreatedAt,
	})
	var result []history.DetectionWithImage
	if err != nil {
		return result, err
	}
	for _, row := range rows {
		result = append(result, h.dbToDomain(row))
	}
	return result, nil
}

var pageSize = int32(100)

func (h *PgDetectionHistoryRepo) GetDetectionTimeline(
	ctx context.Context,
	params history.DetectionTimelineParams,
) ([]history.DetectionEvent, error) {
	page := int32(params.Page)
	if page < 1 {
		page = 1
	}
	dbParam := db.GetDetectionTimelineParams{
		Column1: params.DeviceID,
		Limit:   pageSize,
		Offset:  (page - 1) * pageSize,
	}
	results, dbErr := h.queries.GetDetectionTimeline(ctx, dbParam)
	if dbErr != nil {
		return []history.DetectionEvent{}, dbErr
	}
	// Keep track of unique, ordered IDs
	var ids []int64
	// Mapping of imageID -> []history.EventMeta, pseudo group-by
	idMetaMap := make(map[int64][]history.EventMeta)
	// idEventMap maps imageID -> rows
	idEventMap := make(map[int64]db.GetDetectionTimelineRow)

	for _, row := range results {
		if !slices.Contains(ids, row.ID) {
			// Keep IDs unique and ordered
			ids = append(ids, row.ID)
		}
		idMetaMap[row.ID] = append(idMetaMap[row.ID], history.EventMeta{
			ID:         row.DetectionID,
			Label:      row.Label,
			Confidence: row.Confidence,
		})
		idEventMap[row.ID] = row
	}

	var events []history.DetectionEvent
	for _, id := range ids {
		row := idEventMap[id]
		metas := idMetaMap[id]
		events = append(events, h.dbToEventDomain(row, metas))
	}
	return events, nil
}

func UrlForDetection(prefix string, imagePath string, annotatedPath *string) string {
	if annotatedPath != nil {
		return prefix + *annotatedPath
	}
	return prefix + imagePath
}

func (h *PgDetectionHistoryRepo) dbToEventDomain(
	row db.GetDetectionTimelineRow,
	metas []history.EventMeta,
) history.DetectionEvent {
	return history.DetectionEvent{
		ID:         row.ID,
		CreatedAt:  row.CreatedAt,
		DeviceID:   row.DeviceID,
		ImageUrl:   UrlForDetection(h.config.ThisIp, row.ImagePath, row.AnnotatedPath),
		Detections: metas,
	}
}

func (h *PgDetectionHistoryRepo) dbToDomain(
	dwi db.GetDetectionImagesByLabelRow,
) history.DetectionWithImage {
	return history.DetectionWithImage{
		ID:         dwi.ID,
		CreatedAt:  dwi.CreatedAt,
		Label:      dwi.Label,
		Confidence: dwi.Confidence,
		Bbox:       dwi.Bbox,
		DeviceID:   dwi.DeviceID,
		ImageUrl:   UrlForDetection(h.config.ThisIp, dwi.ImagePath, dwi.AnnotatedPath),
	}
}
