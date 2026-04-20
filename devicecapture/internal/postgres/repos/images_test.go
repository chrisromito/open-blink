package repos

import (
	"devicecapture/internal/domain/devices"
	"devicecapture/internal/postgres"
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_Create_Images(t *testing.T) {
	a := assert.New(t)
	appDb, dbErr := postgres.NewTestAppDb()
	a.NoError(dbErr)
	defer appDb.Db.Close()
	repo := NewPgImageRepo(appDb.GetQueries())
	cleanupErr := postgres.CleanupTestData(t.Context(), repo.queries)
	a.NoError(cleanupErr)
	testDevice, deviceErr := repo.queries.CreateTestDevice(t.Context())
	a.NoError(deviceErr)
	pathPrefix := "/videos"

	tests := []struct {
		params  devices.CreateImageParams
		wantErr bool
		msg     string
	}{
		{
			params: devices.CreateImageParams{
				DeviceID:  testDevice.ID,
				ImagePath: pathPrefix + "test1.jpeg",
			},
			wantErr: false,
			msg:     "valid params create valid images",
		},
		{
			params: devices.CreateImageParams{
				DeviceID:  testDevice.ID,
				ImagePath: pathPrefix + "test2.png",
			},
			wantErr: false,
			msg:     "we do not enforce jpegs on a repo-level",
		},
		{
			params: devices.CreateImageParams{
				DeviceID:  -5,
				ImagePath: pathPrefix + "test3.jpeg",
			},
			wantErr: true,
			msg:     "cannot create images with invalid device IDs",
		},
		{
			params: devices.CreateImageParams{
				DeviceID:  testDevice.ID,
				ImagePath: "",
			},
			wantErr: false,
			msg:     "empty image paths are supported, but frowned upon",
		},
	}
	ctx := t.Context()
	for _, test := range tests {
		result, err := repo.CreateImage(ctx, test.params)
		if test.wantErr {
			a.Error(err, test.msg)
			a.Empty(result)
		} else {
			a.NoError(err, test.msg)
			a.NotEmpty(result)
		}
	}
}

func Test_Get_Images(t *testing.T) {
	a := assert.New(t)
	appDb, dbErr := postgres.NewTestAppDb()
	a.NoError(dbErr)
	defer appDb.Db.Close()
	repo := NewPgImageRepo(appDb.GetQueries())
	testDevice, deviceErr := repo.queries.CreateTestDevice(t.Context())
	a.NoError(deviceErr)

	//	Create 3 images so we can read them out
	testParams := []devices.CreateImageParams{
		{
			DeviceID:  testDevice.ID,
			ImagePath: "/videos/test-123.jpeg",
		},
		{
			DeviceID:  testDevice.ID,
			ImagePath: "/videos/test-234.jpeg",
		},
		{
			DeviceID:  testDevice.ID,
			ImagePath: "/videos/test-345.jpeg",
		},
	}
	for _, p := range testParams {
		result, err := repo.CreateImage(t.Context(), p)
		a.NotNil(result)
		a.NoError(err)
	}

	tests := []struct {
		deviceId  int64
		wantEmpty bool
		msg       string
	}{
		{
			deviceId:  testDevice.ID,
			wantEmpty: false,
			msg:       "valid device IDs return valid device images",
		},
		{
			deviceId:  -12,
			wantEmpty: true,
			msg:       "invalid device IDs return errors",
		},
	}
	for _, test := range tests {
		results, err := repo.GetImages(t.Context(), test.deviceId)
		a.NoError(err)
		if test.wantEmpty {
			a.Empty(results)
		} else {
			a.NotEmpty(results)
		}
	}
}
