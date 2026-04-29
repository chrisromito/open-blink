package repos

import (
	"devicecapture/internal/domain/devices"
	"devicecapture/internal/postgres"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestCreateDetection(t *testing.T) {
	a := assert.New(t)
	appDb, dbErr := postgres.NewTestAppDb()
	a.NoError(dbErr)
	defer appDb.Db.Close()
	q := appDb.GetQueries()
	repo := NewPgDetectionRepo(q)
	testDevice, deviceErr := q.CreateTestDevice(t.Context())
	a.NoError(deviceErr)
	deviceId := testDevice.ID

	tests := []struct {
		params  devices.CreateDetectionParams
		wantErr bool
		message string
	}{
		{
			params:  devices.CreateDetectionParams{DeviceID: deviceId, Label: "dog", Confidence: 0.9},
			wantErr: false,
			message: "We can create detections for specified devices, labels, & confidences",
		},
		{
			params:  devices.CreateDetectionParams{DeviceID: -5, Label: "dog", Confidence: 0.9},
			wantErr: true,
			message: "An error is thrown if the device ID is not in the database",
		},
		{
			params:  devices.CreateDetectionParams{DeviceID: deviceId, Label: "cat", Confidence: 0.85},
			wantErr: false,
			message: "We can create detections for specified devices, labels, & confidences",
		},
	}

	t.Run("create_test_detection", func(t *testing.T) {
		ctx := t.Context()
		for _, test := range tests {
			value, err := repo.CreateDetection(ctx, test.params)
			if test.wantErr {
				a.Error(err, test.message)
			}
			if !test.wantErr {
				a.NoError(err, test.message)
				a.NotNil(value)
			}
		}
	})

	t.Run("create_test_detection_with_image", func(t *testing.T) {
		ctx := t.Context()
		imgRepo := NewPgImageRepo(q)
		// Create the faux image record so we can associate a detection record with a valid DB entry
		img, imgErr := imgRepo.CreateImage(ctx, devices.CreateImageParams{
			DeviceID:  deviceId,
			ImagePath: "/videos/test_detection123.jpg",
		})
		a.NoError(imgErr)
		params := devices.CreateDetectionParams{DeviceID: deviceId, Label: "dog", Confidence: 0.9, ImageID: &img.ID}
		record, err := repo.CreateDetection(ctx, params)
		a.NoError(err)
		a.NotEmpty(record)
		a.Equal(record.ImageID, &img.ID)
	})
}

func TestGetDetectionsAfter(t *testing.T) {
	a := assert.New(t)
	appDb, dbErr := postgres.NewTestAppDb()
	a.NoError(dbErr)
	defer appDb.Db.Close()
	repo := NewPgDetectionRepo(appDb.GetQueries())
	testDevice, deviceErr := repo.queries.CreateTestDevice(t.Context())
	a.NoError(deviceErr)
	deviceId := testDevice.ID

	t.Run("create_detections_for_query_test", func(t *testing.T) {
		ctx := t.Context()
		params := []devices.CreateDetectionParams{
			{DeviceID: deviceId, Label: "person", Confidence: 0.9},
			{DeviceID: deviceId, Label: "dog", Confidence: 0.8},
			{DeviceID: deviceId, Label: "cat", Confidence: 0.2},
		}
		for _, p := range params {
			_, err := repo.CreateDetection(ctx, p)
			a.NoError(err)
		}
	})

	tests := []struct {
		params  devices.QueryParams
		wantErr bool
		isEmpty bool
		message string
	}{
		{
			params:  devices.QueryParams{DeviceID: deviceId, CreatedAt: time.Now()},
			wantErr: false,
			isEmpty: true,
			message: "Empty slice because query date is too recent",
		},
		{
			params:  devices.QueryParams{DeviceID: deviceId, CreatedAt: startOfDay()},
			wantErr: false,
			isEmpty: false,
			message: "non-empty slice because results were inserted today",
		},
		{
			params:  devices.QueryParams{DeviceID: int64(-5), CreatedAt: time.Now()},
			wantErr: false,
			isEmpty: true,
			message: "empty slice because the DeviceID is invalid",
		},
	}

	t.Run("test_query_all_detections_after", func(t *testing.T) {
		ctx := t.Context()
		for _, test := range tests {
			value, err := repo.GetDetectionsAfter(ctx, test.params)
			if test.wantErr {
				a.Error(err, test.message)
			}
			if test.isEmpty {
				a.Empty(value, test.message)
			} else {
				a.NotNil(value, test.message)
			}
		}
	})

	t.Run("test_query_device_detections_after", func(t *testing.T) {
		ctx := t.Context()
		for _, test := range tests {
			value, err := repo.GetDeviceDetectionsAfter(ctx, test.params)
			if test.wantErr {
				a.Error(err, test.message)
			}
			if test.isEmpty {
				a.Empty(value, test.message)
			} else {
				a.NotEmpty(value, test.message)
			}
		}
	})
}
