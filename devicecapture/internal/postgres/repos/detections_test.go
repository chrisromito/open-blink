package repos

import (
	"devicecapture/internal/domain/devices"
	"devicecapture/internal/postgres"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

var validBbox = [][]float64{
	{0.1, 0.2},
	{0.3, 0.4},
}

func TestCreateDetection(t *testing.T) {
	appDb, dbErr := postgres.NewTestAppDb()
	assert.NoError(t, dbErr)
	defer appDb.Db.Close()
	q := appDb.GetQueries()
	repo := NewPgDetectionRepo(q)
	testDevice, deviceErr := GetOrCreateTestDevice(t.Context(), q)
	assert.NoError(t, deviceErr)
	deviceId := testDevice.ID

	tests := []struct {
		params  devices.CreateDetectionParams
		wantErr bool
		message string
	}{
		{
			params:  devices.CreateDetectionParams{DeviceID: deviceId, Label: "dog", Confidence: 0.9, Bbox: [][]float64{}},
			wantErr: false,
			message: "bbox is not required",
		},
		{
			params:  devices.CreateDetectionParams{DeviceID: deviceId, Label: "dog", Confidence: 0.9, Bbox: nil},
			wantErr: false,
			message: "bbox does not need to be specified at all",
		},
		{
			params:  devices.CreateDetectionParams{DeviceID: deviceId, Label: "cat", Confidence: 0.85, Bbox: validBbox},
			wantErr: false,
			message: "We can create detections for specified devices, labels, & confidences",
		},
		{
			params:  devices.CreateDetectionParams{DeviceID: -5, Label: "dog", Confidence: 0.9, Bbox: validBbox},
			wantErr: true,
			message: "An error is thrown if the device ID is not in the database",
		},
	}

	t.Run("create_test_detection", func(t *testing.T) {
		a := assert.New(t)
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
		a := assert.New(t)
		ctx := t.Context()
		imgRepo := NewPgImageRepo(q)
		td, de := GetOrCreateTestDevice(t.Context(), q)
		a.NoError(de)
		suffix := generateRandomString(10)
		// Create the faux image record so we can associate a detection record with a valid DB entry
		img, imgErr := imgRepo.CreateImage(ctx, devices.CreateImageParams{
			DeviceID:  td.ID,
			ImagePath: "/videos/test_detection123" + suffix + ".jpg",
		})
		a.NotNil(img)
		a.NoError(imgErr)
		params := devices.CreateDetectionParams{DeviceID: td.ID, Label: "dog", Confidence: 0.9, ImageID: &img.ID, Bbox: validBbox}
		record, err := repo.CreateDetection(ctx, params)
		a.NoError(err)
		a.NotNil(record)
		a.NotNil(record.ImageID)
		//a.Equal(record.ImageID, &img.ID)
	})
}

func TestGetDetectionsAfter(t *testing.T) {
	appDb, dbErr := postgres.NewTestAppDb()
	a := assert.New(t)
	a.NoError(dbErr)
	defer appDb.Db.Close()
	q := appDb.GetQueries()
	repo := NewPgDetectionRepo(q)
	testDevice, deviceErr := GetOrCreateTestDevice(t.Context(), q)
	a.NoError(deviceErr)
	deviceId := testDevice.ID

	params := []devices.CreateDetectionParams{
		{DeviceID: deviceId, Label: "person", Confidence: 0.9, Bbox: validBbox},
		{DeviceID: deviceId, Label: "dog", Confidence: 0.8, Bbox: validBbox},
		{DeviceID: deviceId, Label: "cat", Confidence: 0.2, Bbox: validBbox},
	}
	for _, p := range params {
		_, err := repo.CreateDetection(t.Context(), p)
		a.NoError(err)
	}

	failDate := time.Now().Add(24 * time.Hour)
	successDate := time.Now().Add(-48 * time.Hour)

	t.Run("test_future_date_gives_empty_results", func(t *testing.T) {
		value, err := repo.GetDetectionsAfter(t.Context(), devices.QueryParams{DeviceID: int64(-5), CreatedAt: failDate, ImageID: nil})
		assert.NoError(t, err)
		assert.Empty(t, value)
	})

	t.Run("test_past_date_gives_non_empty_results", func(t *testing.T) {
		value, err := repo.GetDetectionsAfter(t.Context(), devices.QueryParams{DeviceID: int64(-5), CreatedAt: successDate, ImageID: nil})
		assert.NotEmpty(t, value, "GetDetectionsAfter yields results")
		assert.NoError(t, err, "GetDetectionsAfter does not require valid DeviceIDs")
	})

	tests := []struct {
		params  devices.QueryParams
		wantErr bool
		isEmpty bool
		message string
	}{
		{
			params:  devices.QueryParams{DeviceID: deviceId, CreatedAt: failDate, ImageID: nil},
			isEmpty: true,
			message: "Empty slice because query date is too recent",
		},
		{
			params:  devices.QueryParams{DeviceID: deviceId, CreatedAt: successDate, ImageID: nil},
			isEmpty: false,
			message: "non-empty slice because results were inserted today",
		},
		{
			params:  devices.QueryParams{DeviceID: int64(-5), CreatedAt: successDate, ImageID: nil},
			isEmpty: true,
			message: "empty slice because the DeviceID is invalid",
		},
	}

	t.Run("test_query_device_detections_after", func(t *testing.T) {
		ctx := t.Context()
		for _, test := range tests {
			value, err := repo.GetDeviceDetectionsAfter(ctx, test.params)
			assert.NoError(t, err)
			if test.isEmpty {
				assert.Empty(t, value, test.message)
			} else {
				assert.NotEmpty(t, value, test.message)
			}
		}
	})
}
