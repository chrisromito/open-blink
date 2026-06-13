package archiver

import (
	"context"
	"devicecapture/internal/domain/devices"
)

type SetArchivedParams struct {
	ID        int64  `db:"id" json:"id"`
	ImagePath string `db:"image_path" json:"image_path"`
}

// ArchiveRepo describes how archiving a deviceImage interfaces with the persistence layer
type ArchiveRepo interface {
	GetArchiveTargets(ctx context.Context) ([]devices.DeviceImage, error)
	SetArchived(ctx context.Context, params SetArchivedParams) error
	PurgeOldestTargets(ctx context.Context) error
}