package postgres

import (
	"devicecapture/internal/postgres/db"
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestCreateTestDevice(t *testing.T) {
	t.Run("create_test_device", func(t *testing.T) {
		a := assert.New(t)
		appDb, err := NewTestAppDb()
		defer appDb.Db.Close()
		a.Nil(err)
		queries := GetQueries(*appDb)
		testDevice, err2 := queries.CreateTestDevice(t.Context())
		a.Nil(err2)
		a.NotNil(testDevice)
		a.Equal(testDevice.Name, "mockdevice")
		fmt.Printf("testDevice: %v", testDevice)
	})
	t.Run("delete_test_devices", func(t *testing.T) {
		a := assert.New(t)
		appDb, err := NewTestAppDb()
		defer appDb.Db.Close()
		a.Nil(err)
		queries := GetQueries(*appDb)
		dErr := queries.DeleteTestDevices(t.Context())
		a.Nil(dErr)
	})
}

func TestGetDevices(t *testing.T) {
	tests := []struct {
		name string
		url  string
		want string
	}{
		{
			name: "valid_url",
			url:  testDatabaseURL,
			want: testDatabaseURL,
		},
		{
			name: "empty_url",
			url:  "",
			want: "",
		},
	}

	t.Run("get_devices", func(t *testing.T) {
		a := assert.New(t)
		appDb, err := NewTestAppDb()
		defer appDb.Db.Close()
		a.Nil(err)
		ctx := t.Context()
		queries := GetQueries(*appDb)
		cleanupErr := CleanupTestData(ctx, queries)
		a.NoError(cleanupErr)
		_, testDeviceErr := queries.CreateTestDevice(ctx)
		a.NoError(testDeviceErr)
		devices, err2 := queries.GetDevices(ctx)
		a.Nil(err2)
		a.NotNil(devices)
		var testDevice db.Device
		for _, device := range devices {
			if device.Name == "mockdevice" {
				testDevice = device
			}
		}
		a.NotNil(testDevice)
		a.Equal(testDevice.Name, "mockdevice")
	})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := assert.New(t)
			appDb := NewAppDb()
			a.NotNil(appDb)
			a.Nil(appDb.Db) // Should be nil until Connect is called
		})
	}
}
