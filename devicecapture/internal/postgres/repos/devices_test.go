package repos

import (
	"context"
	"devicecapture/internal/device/devices"
	"devicecapture/internal/postgres"
	"github.com/stretchr/testify/assert"
	"math/rand"
	"strconv"
	"testing"
	"time"
)

func TestCreateDevices(t *testing.T) {
	a := assert.New(t)
	appDb, dbErr := postgres.NewTestAppDb()
	a.NoError(dbErr)
	defer appDb.Db.Close()
	repo := NewPgDeviceRepo(appDb.GetQueries())

	t.Run("create_test_device", func(t *testing.T) {
		testDevice, deviceErr := repo.queries.CreateTestDevice(context.Background())
		a.NoError(deviceErr)
		a.NotNil(testDevice, "The test device was inserted into the DB")
	})

	tests := []struct {
		params  devices.CreateDeviceParams
		wantErr bool
		message string
	}{
		{
			params:  devices.CreateDeviceParams{Name: "success", DeviceUrl: "http://mockdevice:1234"},
			wantErr: false,
			message: "We can create devices with short names & urls",
		},
		{
			params:  devices.CreateDeviceParams{Name: "Super long URL", DeviceUrl: generateRandomString(500)},
			wantErr: true,
			message: "device URLs must be shorter than 250 characters",
		},
		{
			params:  devices.CreateDeviceParams{Name: generateRandomString(500), DeviceUrl: "http://longname:1234"},
			wantErr: true,
			message: "device names must be shorter than 250 characters",
		},
	}

	for _, test := range tests {
		t.Run("create_device", func(t *testing.T) {
			ctx := context.Background()
			value, err := repo.CreateDevice(ctx, test.params)

			if test.wantErr {
				a.Error(err)
				a.Nil(value)
			} else {
				a.NoError(err)
				a.NotNil(value)
			}
		})
	}
}

func TestList_And_UpdateDevices(t *testing.T) {
	a := assert.New(t)
	appDb, dbErr := postgres.NewTestAppDb()
	a.NoError(dbErr)
	defer appDb.Db.Close()
	repo := NewPgDeviceRepo(appDb.GetQueries())
	testDevice, deviceErr := repo.queries.CreateTestDevice(context.Background())
	a.NoError(deviceErr)
	a.NotNil(testDevice, "The test device was inserted into the DB")

	ctx := context.Background()
	allDevices, err := repo.ListDevices(ctx)
	a.NoError(err)
	a.NotEmpty(allDevices)
	for i, d := range allDevices {
		t.Run("list_devices", func(t *testing.T) {
			name := d.Name + strconv.Itoa(i)
			updated, uErr := repo.UpdateDevice(
				ctx,
				devices.UpdateDeviceParams{ID: d.ID, Name: name, DeviceUrl: d.DeviceUrl},
			)
			a.NoError(uErr)
			a.NotNil(updated)
			a.Equal(name, updated.Name)
		})
	}
}

// generateRandomString generates a random string of a given length using a specified character set.
func generateRandomString(length int) string {
	charset := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	// Seed the random number generator using the current time for better randomness.
	seededRand := rand.New(rand.NewSource(time.Now().UnixNano()))

	b := make([]byte, length)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(b)
}
