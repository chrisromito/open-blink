package repos

import (
	"context"

	"devicecapture/internal/domain/devices"
	"devicecapture/internal/postgres/db"
)

type PgImageRepo struct {
	queries *db.Queries
}

func NewPgImageRepo(q *db.Queries) *PgImageRepo {
	return &PgImageRepo{queries: q}
}

func (ir *PgImageRepo) CreateImage(
	ctx context.Context,
	params devices.CreateImageParams,
) (devices.DeviceImage, error) {
	dbImg, err := ir.queries.CreateImage(ctx, db.CreateImageParams{
		DeviceID:      params.DeviceID,
		ImagePath:     params.ImagePath,
		AnnotatedPath: &params.AnnotatedPath,
	})
	if err != nil {
		return devices.DeviceImage{}, err
	}
	return ImageDbToDomain(dbImg), nil
}

func (ir *PgImageRepo) GetImages(
	ctx context.Context,
	deviceID int64,
) ([]devices.DeviceImage, error) {
	imgs, err := ir.queries.GetDeviceImages(ctx, deviceID)
	if err != nil {
		var empty []devices.DeviceImage
		return empty, err
	}
	var list []devices.DeviceImage
	for _, img := range imgs {
		list = append(list, ImageDbToDomain(img))
	}
	return list, nil
}

func (ir *PgImageRepo) GetImagesForEvent(
	ctx context.Context,
	eventID int64,
) ([]devices.DeviceImage, error) {
	imgs, err := ir.queries.GetImagesForEvent(ctx, eventID)
	if err != nil {
		var empty []devices.DeviceImage
		return empty, err
	}
	var list []devices.DeviceImage
	for _, img := range imgs {
		list = append(list, ImageDbToDomain(img))
	}
	return list, nil
}

func (ir *PgImageRepo) SetEvent(
	ctx context.Context,
	ids []int64,
	eventID int64,
) error {
	return ir.queries.SetEventForImages(ctx, db.SetEventForImagesParams{
		EventID: eventID,
		Ids:     ids,
	})
}

func ImageDbToDomain(dbImg db.DeviceImage) devices.DeviceImage {
	return devices.DeviceImage{
		ID:            dbImg.ID,
		DeviceID:      dbImg.DeviceID,
		EventID:       dbImg.EventID,
		CreatedAt:     dbImg.CreatedAt,
		ImagePath:     dbImg.ImagePath,
		AnnotatedPath: *dbImg.AnnotatedPath,
	}
}
