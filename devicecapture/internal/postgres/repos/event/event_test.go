package event

import (
	"fmt"
	"testing"

	dEvent "devicecapture/internal/domain/event"
	"devicecapture/internal/postgres"
	"devicecapture/internal/postgres/repos"
	"github.com/stretchr/testify/assert"
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
		equal := dEvent.LabelsEq(test.left, test.right)
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
		record, err := repo.EndEvent(ctx, baseEvt)
		as.NoError(err)
		as.NotEmpty(record)
		as.Equal(baseLabels, record.Labels, "Labels were not updated because they did not change")
		as.Equal(int(dEvent.Ended), record.State, "State was set to ended")
		as.NotEmpty(record.EndedAt, "EndedAt was set")
	})
}

func Test_Get_Detection_Events(t *testing.T) {
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

	// Write 15 records to the DB and read them out
	toWrite := []struct {
		DeviceID int64
		Labels   []string
	}{
		{
			DeviceID: deviceID,
			Labels:   []string{"car"},
		},
		{
			DeviceID: deviceID,
			Labels:   []string{"person"},
		},
		{
			DeviceID: deviceID,
			Labels:   []string{"car", "person"},
		},
		{
			DeviceID: deviceID,
			Labels:   []string{"car", "car", "car"},
		},
		{
			DeviceID: deviceID,
			Labels:   []string{"car", "truck"},
		},
	}
	writeCount := 0
	for _, params := range toWrite {
		for i := range 3 {
			fmt.Println(i)
			de, err := repo.StartEvent(ctx, params.DeviceID, params.Labels)
			a.NoError(err)
			_, err = repo.EndEvent(ctx, de)
			a.NoError(err)
			writeCount++
		}
	}

	t.Run("read_first_page", func(t *testing.T) {
		param := dEvent.QueryParams{
			DeviceID: deviceID,
		}
		results, err := repo.GetDeviceEvents(t.Context(), param)
		a.NoError(err)
		a.NotEmpty(results)
		a.Equal(len(results), writeCount)
	})

	t.Run("read_first_page", func(t *testing.T) {
		// querying for a page # that doesn't exist returns empty instead of an error
		param := dEvent.QueryParams{
			DeviceID: deviceID,
			Page:     100,
		}
		_, err := repo.GetDeviceEvents(t.Context(), param)
		a.NoError(err)
	})

	t.Run("read_started_events", func(t *testing.T) {
		started, dErr := repo.StartEvent(ctx, deviceID, []string{"unfinished"})
		a.NoError(dErr)
		a.NotEmpty(started)
		param := dEvent.QueryParams{
			DeviceID: deviceID,
			State:    dEvent.Started,
		}
		records, err := repo.GetDeviceEvents(t.Context(), param)
		a.NoError(err)
		for _, de := range records {
			a.Equal(de.State, int(dEvent.Started), "State query param works as a filter")
		}
	})
}
