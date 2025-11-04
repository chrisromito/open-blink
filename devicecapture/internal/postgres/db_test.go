package postgres

import (
	"context"
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
		testDevice, err2 := queries.CreateTestDevice(context.Background())
		a.Nil(err2)
		a.NotNil(testDevice)
		a.Equal(testDevice.Name, "mockdevice")
		fmt.Printf("testDevice: %v", testDevice)
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
		appDb, err := NewTestAppDb()
		defer appDb.Db.Close()

		assert.Nil(t, err)
		ctx := context.Background()
		queries := GetQueries(*appDb)
		devices, err2 := queries.GetDevices(ctx)
		assert.Nil(t, err2)
		assert.NotNil(t, devices)
		var testDevice db.Device
		for _, device := range devices {
			if device.Name == "mockdevice" {
				testDevice = device
			}
		}
		assert.NotNil(t, testDevice)
		assert.Equal(t, testDevice.Name, "mockdevice")
		fmt.Printf("devices: %v", devices)
	})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			appDb := NewAppDb()

			assert.NotNil(t, appDb)
			assert.Nil(t, appDb.Db) // Should be nil until Connect is called
		})
	}
}
