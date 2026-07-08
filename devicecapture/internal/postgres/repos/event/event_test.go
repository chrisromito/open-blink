package event

import (
	dEvent "devicecapture/internal/domain/event"
	"devicecapture/internal/postgres"
	"devicecapture/internal/postgres/repos"
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_Labels_To_Db(t *testing.T) {
	a := assert.New(t)
	tests := []struct {
		domain  []string
		db      string
		wantErr bool
		message string
	}{
		{
			domain:  []string{"truck"},
			db:      "truck",
			wantErr: false,
			message: "truck  == ['truck']",
		},
		{
			domain:  []string{"car", "person"},
			db:      "car, person",
			wantErr: false,
			message: "car, person == ['car', 'person']",
		},
		{
			domain:  []string{"cat", "dog", "person"},
			db:      "cat, dog, person",
			wantErr: false,
			message: "cat, dog, person == ['cat', 'dog', 'person']",
		},
	}

	for _, test := range tests {
		dbValue := LabelsToDb(test.domain)
		a.Equal(dbValue, test.db)
		domainValue := DbToLabels(test.db)
		a.Equal(domainValue, test.domain)
	}

	a.NotEqual([]string{"cat"}, DbToLabels("person"), "false positive test for db -> domain")
	a.NotEqual("person", LabelsToDb([]string{"cat"}), "false positive test for domain -> db")
}

func Test_Slice_Eq(t *testing.T) {
	tests := []struct {
		left    []string
		right   []string
		eq      bool
		message string
	}{
		{
			left:    []string{"truck"},
			right:   []string{"truck"},
			eq:      true,
			message: "basic equivalence works as expected",
		},
		{
			left:    []string{"car", "truck"},
			right:   []string{"truck", "car"},
			eq:      true,
			message: "ordering does not matter",
		},
		{
			left:    []string{"car"},
			right:   []string{"truck", "car"},
			eq:      false,
			message: "slice lengths must be equal",
		},
		{
			left:    []string{"car", "truck"},
			right:   []string{"truck"},
			eq:      false,
			message: "slice lengths must be equal",
		},
	}

	a := assert.New(t)

	for _, test := range tests {
		equal := slicesEq(test.left, test.right)
		a.Equal(equal, test.eq, test.message)
	}
}

// Test_Start_DetectionEvent Integration Tests for PgDetectionEventRepo "StartEvent" logic
func Test_Start_DetectionEvent(t *testing.T) {
	a := assert.New(t)
	appDb, dbErr := postgres.NewTestAppDb()
	a.NoError(dbErr)
	defer appDb.Db.Close()
	q := appDb.GetQueries()
	repo := NewPgDetectionEventRepo(q)
	testDevice, deviceErr := repos.GetOrCreateTestDevice(t.Context(), q)
	a.NoError(deviceErr)
	deviceID := testDevice.ID

	tests := []struct {
		deviceID int64
		labels   []string
		wantErr  bool
		message  string
	}{
		{
			deviceID: deviceID,
			labels:   []string{"person"},
			wantErr:  false,
			message:  "DetectionEvent start requires a valid deviceID and label(s)",
		},
		{
			deviceID: deviceID,
			wantErr:  false,
			message:  "labels are not required",
		},
		{
			deviceID: -1,
			labels:   []string{"person"},
			wantErr:  true,
			message:  "invalid deviceIDs throw errors",
		},
	}

	ctx := t.Context()
	for _, test := range tests {
		result, err := repo.StartEvent(ctx, test.deviceID, test.labels)
		if !test.wantErr {
			a.NoError(err, test.message)
			a.NotEmpty(result, test.message)
			a.True(result.EndedAt.IsZero(), "EndedAt is zero-valued when starting a DetectionEvent")
		}
		if test.wantErr {
			a.Error(err, test.message)
			a.Empty(result, test.message)
		}
	}
}

// Test_Start_DetectionEvent Integration Tests for PgDetectionEventRepo "EndEvent" logic
func Test_End_DetectionEvent(t *testing.T) {
	a := assert.New(t)
	appDb, dbErr := postgres.NewTestAppDb()
	a.NoError(dbErr)
	defer appDb.Db.Close()
	q := appDb.GetQueries()
	repo := NewPgDetectionEventRepo(q)
	testDevice, deviceErr := repos.GetOrCreateTestDevice(t.Context(), q)
	a.NoError(deviceErr)
	deviceID := testDevice.ID
	ctx := t.Context()

	// Set up the initial values, which we'll make updates against
	baseLabels := []string{"person"}

	t.Run("test_end_without_labels", func(t *testing.T) {
		as := assert.New(t)
		// End the event without updating the labels
		baseEvt, evtErr := repo.StartEvent(ctx, deviceID, baseLabels)
		as.NoError(evtErr)
		record, err := repo.EndEvent(ctx, baseEvt, baseLabels)
		as.NoError(err)
		as.NotEmpty(record)
		as.Equal(baseLabels, record.Labels, "Labels were not updated because they did not change")
		as.Equal(int(dEvent.Ended), record.State, "State was set to ended")
		as.NotEmpty(record.EndedAt, "EndedAt was set")
	})

	t.Run("test_end_with_labels", func(t *testing.T) {
		as := assert.New(t)
		baseEvt, evtErr := repo.StartEvent(ctx, deviceID, baseLabels)
		as.NoError(evtErr)
		// End with new labels to ensure that we update the DB accordingly
		nextLabels := []string{"car", "person"}
		record, err := repo.EndEvent(ctx, baseEvt, nextLabels)
		as.NoError(err)
		as.NotEmpty(record)
		as.Equal(nextLabels, record.Labels, "DB labels were updated")
		as.Equal(int(dEvent.Ended), record.State, "State was set to ended")
		as.NotEmpty(record.EndedAt, "EndedAt was set")
	})
}
