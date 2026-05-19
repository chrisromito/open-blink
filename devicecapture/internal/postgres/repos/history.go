package repos

import (
	"context"
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

func (h *PgDetectionHistoryRepo) GetDetectionImagesByLabel(ctx context.Context, params history.DetectionWithImageParams) ([]history.DetectionWithImage, error) {
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

func (h *PgDetectionHistoryRepo) dbToDomain(dwi db.GetDetectionImagesByLabelRow) history.DetectionWithImage {
	return history.DetectionWithImage{
		ID:         dwi.ID,
		CreatedAt:  dwi.CreatedAt,
		Label:      dwi.Label,
		Confidence: dwi.Confidence,
		Bbox:       dwi.Bbox,
		DeviceID:   dwi.DeviceID,
		ImageUrl:   h.config.ThisIp + dwi.ImagePath,
	}
}
