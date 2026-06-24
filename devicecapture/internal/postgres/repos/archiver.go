package repos

import (
	"context"

	"devicecapture/internal/domain/archiver"
	"devicecapture/internal/domain/devices"
	"devicecapture/internal/postgres/db"
)

// PgArchiveRepo implements archiver.ArchiveRepo
type PgArchiveRepo struct {
	queries *db.Queries
}

// NewPgArchiveRepo constructor for PgArchiveRepo
func NewPgArchiveRepo(q *db.Queries) *PgArchiveRepo {
	return &PgArchiveRepo{queries: q}
}

func (a *PgArchiveRepo) GetArchiveTargets(ctx context.Context) ([]devices.DeviceImage, error) {
	is, err := a.queries.GetArchiveTargets(ctx)
	if err != nil {
		return []devices.DeviceImage{}, err
	}
	var value []devices.DeviceImage
	for _, d := range is {
		value = append(value, ImageDbToDomain(d))
	}

	return value, nil
}

func (a *PgArchiveRepo) SetArchived(ctx context.Context, params archiver.SetArchivedParams) error {
	return a.queries.SetArchived(ctx, db.SetArchivedParams{
		ID:        params.ID,
		ImagePath: params.ImagePath,
	})
}

func (a *PgArchiveRepo) PurgeOldestTargets(ctx context.Context) error {
	return a.queries.PurgeOldestTargets(ctx)
}
