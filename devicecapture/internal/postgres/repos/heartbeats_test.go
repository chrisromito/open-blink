package repos

import (
	"context"
	"devicecapture/internal/postgres"
	"github.com/stretchr/testify/assert"
	"strconv"
	"testing"
)

func Test_Create_Heartbeats(t *testing.T) {
	a := assert.New(t)
	appDb, dbErr := postgres.NewTestAppDb()
	a.NoError(dbErr)
	defer appDb.Db.Close()
	repo := NewPgHeartbeatRepo(appDb.GetQueries())
	testDevice, deviceErr := repo.queries.CreateTestDevice(context.Background())
	a.NoError(deviceErr)
	deviceId := strconv.Itoa(int(testDevice.ID))

	for range 10 {
		t.Run("create_heartbeat", func(t *testing.T) {
			hb, err := repo.RecordBeat(context.Background(), deviceId)
			a.NoError(err, "We can record device heartbeats")
			a.NotNil(hb)
			a.Equal(int32(hb.DeviceID), testDevice.ID)
		})
	}

	getTests := []struct {
		deviceId  string
		wantEmpty bool
		name      string
	}{
		{
			deviceId:  deviceId,
			wantEmpty: false,
			name:      "valid device",
		},
		{
			deviceId:  "fail",
			wantEmpty: true,
			name:      "invalid device",
		},
	}
	for _, test := range getTests {
		t.Run(test.name, func(t *testing.T) {
			value, err := repo.GetDeviceHeartBeats(context.Background(), test.deviceId)
			a.NoError(err)
			if test.wantEmpty {
				a.Empty(value)
			} else {
				a.NotEmpty(value)
			}
		})
	}

	t.Run("delete_beats", func(t *testing.T) {
		err := repo.DeleteBeats(context.Background(), deviceId)
		a.NoError(err, "heartbeats can be deleted from the DB")
	})
}
