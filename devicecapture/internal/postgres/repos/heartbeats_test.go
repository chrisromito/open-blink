package repos

import (
	"devicecapture/internal/postgres"
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_Create_Heartbeats(t *testing.T) {
	a := assert.New(t)
	appDb, dbErr := postgres.NewTestAppDb()
	a.NoError(dbErr)
	defer appDb.Db.Close()
	repo := NewPgHeartbeatRepo(appDb.GetQueries())
	testDevice, deviceErr := repo.queries.CreateTestDevice(t.Context())
	a.NoError(deviceErr)

	for range 10 {
		t.Run("create_heartbeat", func(t *testing.T) {
			hb, err := repo.RecordBeat(t.Context(), testDevice.ID)
			a.NoError(err, "We can record domain heartbeats")
			a.NotNil(hb)
			a.Equal(hb.DeviceID, testDevice.ID)
		})
	}

	getTests := []struct {
		deviceId  int64
		wantEmpty bool
		name      string
	}{
		{
			deviceId:  testDevice.ID,
			wantEmpty: false,
			name:      "valid domain",
		},
		{
			deviceId:  int64(-1),
			wantEmpty: true,
			name:      "invalid domain",
		},
	}
	for _, test := range getTests {
		t.Run(test.name, func(t *testing.T) {
			value, err := repo.GetDeviceHeartBeats(t.Context(), test.deviceId)
			a.NoError(err)
			if test.wantEmpty {
				a.Empty(value)
			} else {
				a.NotEmpty(value)
			}
		})
	}

	t.Run("delete_beats", func(t *testing.T) {
		err := repo.DeleteBeats(t.Context(), testDevice.ID)
		a.NoError(err, "heartbeats can be deleted from the DB")
	})
}
