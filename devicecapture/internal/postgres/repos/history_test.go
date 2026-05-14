package repos

import (
	"devicecapture/internal/config"
	"devicecapture/internal/domain/devices"
	"devicecapture/internal/domain/history"
	"devicecapture/internal/postgres"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func Test_Pg_History_Repo(t *testing.T) {
	a := assert.New(t)
	appDb, dbErr := postgres.NewTestAppDb()
	a.NoError(dbErr)
	defer appDb.Db.Close()
	q := appDb.GetQueries()
	repo := NewPgDetectionRepo(q)
	//testDevice, deviceErr := repo.queries.CreateTestDevice(t.Context())
	testDevice, deviceErr := GetOrCreateTestDevice(t.Context(), q)
	a.NoError(deviceErr)
	params := []devices.CreateDetectionParams{
		{DeviceID: testDevice.ID, Label: "person", Confidence: 0.5, Bbox: validBbox},
		{DeviceID: testDevice.ID, Label: "truck", Confidence: 0.35, Bbox: validBbox},
		{DeviceID: testDevice.ID, Label: "cat", Confidence: 0.75, Bbox: validBbox},
	}

	for _, p := range params {
		detect, cErr := repo.CreateDetection(t.Context(), p)
		a.NoError(cErr)
		a.NotEmpty(detect)
	}

	t.Run("test_get_recent_labels", func(t *testing.T) {
		a = assert.New(t)
		historyRepo := NewPgDetectionHistoryRepo(q, getTestConfig(""))

		labels, err := historyRepo.GetRecentLabels(t.Context())
		a.NoError(err)
		a.GreaterOrEqual(len(labels), 3, "all existing labels are retrieved")
	})

	t.Run("test_get_detection_images_parameters", func(t *testing.T) {
		a = assert.New(t)
		historyRepo := NewPgDetectionHistoryRepo(q, getTestConfig(""))
		now := time.Now()
		hourAgo := now.Add(-1 * time.Hour)
		tests := []struct {
			params    history.DetectionWithImageParams
			wantEmpty bool
			message   string
		}{
			{
				params:    history.DetectionWithImageParams{Label: []string{"person"}, DeviceID: 0},
				wantEmpty: false,
				message:   "query for person yields results",
			},
			{

				params:    history.DetectionWithImageParams{Label: []string{"person"}, DeviceID: testDevice.ID},
				wantEmpty: false,
				message:   "deviceID filter is additive",
			},
			{

				params:    history.DetectionWithImageParams{Label: []string{"person", "test"}, DeviceID: 0, CreatedAt: hourAgo},
				wantEmpty: true,
				message:   "label parameters are ORd",
			},
			{

				params:    history.DetectionWithImageParams{Label: []string{"person"}, DeviceID: -10},
				wantEmpty: true,
				message:   "deviceID filter is additive",
			},
			{

				params:    history.DetectionWithImageParams{Label: []string{"person"}, DeviceID: testDevice.ID, CreatedAt: now.Add(48 * time.Hour)},
				wantEmpty: true,
				message:   "created at excludes future dates",
			},
		}

		for _, test := range tests {
			result, err := historyRepo.GetDetectionImagesByLabel(t.Context(), test.params)
			a.NoError(err)
			if test.wantEmpty {
				a.Empty(result, test.message)
			} else {
				a.NotEmpty(result, test.message)
			}
		}
	})

}

func getTestConfig(videoPath string) *config.Config {
	if videoPath == "" {
		videoPath = "/tmp/videos"
	}
	return &config.Config{
		MqttHost:            "",
		DbUrl:               "postgres://postgres:postgres@postgres:5432/test_openblink",
		VideoPath:           videoPath,
		DetectionServiceUrl: "http://0.0.0.0:4000",
		ThisIp:              "http://0.0.0.0:8000",
	}
}
