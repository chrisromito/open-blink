package repos

import (
	"context"
	"devicecapture/internal/device/devices"
	"devicecapture/internal/postgres"
	"github.com/stretchr/testify/assert"
	"strconv"
	"testing"
	"time"
)

func TestCreateDetection(t *testing.T) {
	a := assert.New(t)
	appDb, dbErr := postgres.NewTestAppDb()
	a.NoError(dbErr)
	defer appDb.Db.Close()
	repo := NewPgDetectionRepo(appDb.GetQueries())
	testDevice, deviceErr := repo.queries.CreateTestDevice(context.Background())
	a.NoError(deviceErr)
	deviceId := int64(testDevice.ID)

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
		ctx := context.Background()
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
}

func TestGetDetectionsAfter(t *testing.T) {
	a := assert.New(t)
	appDb, dbErr := postgres.NewTestAppDb()
	a.NoError(dbErr)
	defer appDb.Db.Close()
	repo := NewPgDetectionRepo(appDb.GetQueries())
	testDevice, deviceErr := repo.queries.CreateTestDevice(context.Background())
	a.NoError(deviceErr)
	deviceId := int64(testDevice.ID)

	t.Run("create_detections_for_query_test", func(t *testing.T) {
		ctx := context.Background()
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
			params:  devices.QueryParams{DeviceID: strconv.FormatInt(deviceId, 10), CreatedAt: time.Now()},
			wantErr: false,
			isEmpty: true,
			message: "Empty slice because query date is too recent",
		},
		{
			params:  devices.QueryParams{DeviceID: strconv.FormatInt(deviceId, 10), CreatedAt: startOfDay()},
			wantErr: false,
			isEmpty: false,
			message: "non-empty slice because results were inserted today",
		},
		{
			params:  devices.QueryParams{DeviceID: "-5", CreatedAt: time.Now()},
			wantErr: false,
			isEmpty: true,
			message: "empty slice because the DeviceID is invalid",
		},
	}

	t.Run("test_query_all_detections_after", func(t *testing.T) {
		ctx := context.Background()
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
		ctx := context.Background()
		for _, test := range tests {
			value, err := repo.GetDeviceDetectionsAfter(ctx, test.params)
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
}
