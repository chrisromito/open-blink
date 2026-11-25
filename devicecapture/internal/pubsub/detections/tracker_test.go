package detections_test

import (
	"context"
	"devicecapture/internal/domain/devices"
	"devicecapture/internal/postgres/db"
	"devicecapture/internal/postgres/repos"
	"devicecapture/internal/pubsub/detections"
	"devicecapture/internal/testutils"
	"testing"

	"devicecapture/internal/domain/event"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_DetectionTracker_Start_And_End(t *testing.T) {
	a := assert.New(t)
	deps, appDb, err := testutils.NewTestDeps("/videos", "/archive")
	a.NoError(err)
	defer func() {
		if appDb != nil {
			appDb.Db.Close()
		}
	}()
	tracker := detections.NewDetectionTracker(deps)
	deviceID, dErr := getTestDeviceID(t.Context(), appDb.GetQueries())
	a.NoError(dErr)
	defer func() {
		_ = deps.DeviceRepo.DeleteDevice(t.Context(), deviceID)
	}()

	t.Run("test_detection_tracker_start", func(t *testing.T) {
		de, err := tracker.Start(t.Context(), deviceID, []string{"car"})
		a.NoError(err)
		a.NotEqual(de.ID, int64(0))
		a.True(tracker.HasEvent(deviceID))
		a.NotZero(tracker.CurrentEventID(t.Context(), deviceID))
	})

	t.Run("test_detecion_tracker_end", func(t *testing.T) {
		de, err := tracker.Start(t.Context(), deviceID, []string{"car"})
		a.NoError(err)
		a.NotEqual(de.ID, int64(0))
		a.True(tracker.HasEvent(deviceID))

		err = tracker.End(t.Context(), deviceID)

		a.NoError(err)
		a.False(tracker.HasEvent(deviceID))
		a.Zero(tracker.CurrentEventID(t.Context(), deviceID))
	})
}

func Test_DetectionTracker_Start_End_Integration(t *testing.T) {
	a := assert.New(t)
	deps, appDb, depErr := testutils.NewTestDeps("/videos", "/archive")
	a.NoError(depErr)
	defer func() {
		if appDb != nil {
			appDb.Db.Close()
		}
	}()
	tracker := detections.NewDetectionTracker(deps)
	deviceID, dErr := getTestDeviceID(t.Context(), appDb.GetQueries())
	a.NoError(dErr)
	defer func() {
		_ = deps.DeviceRepo.DeleteDevice(t.Context(), deviceID)
	}()

	t.Run("end_noops_without_start", func(t *testing.T) {
		err := tracker.End(t.Context(), deviceID)
		a.NoError(err)
		a.False(tracker.HasEvent(deviceID))
	})

	t.Run("devices_tracked_independently", func(t *testing.T) {
		secondID, secErr := getTestDeviceID(t.Context(), appDb.GetQueries())
		a.NoError(secErr)
		firstStart, fErr := tracker.Start(t.Context(), deviceID, []string{"car"})
		a.NoError(fErr)
		a.NotEqual(firstStart.ID, int64(0))
		secondStart, sErr := tracker.Start(t.Context(), secondID, []string{"person"})
		a.NoError(sErr)
		a.NotEqual(secondStart.ID, int64(0))

		a.True(tracker.HasEvent(deviceID))
		a.True(tracker.HasEvent(secondID))

		a.NoError(tracker.End(t.Context(), deviceID))

		a.False(tracker.HasEvent(deviceID))
		a.True(tracker.HasEvent(secondID))
	})

	t.Run("receive_detections_starts_event_when_device_has_no_event", func(t *testing.T) {
		err := tracker.ReceiveDetections(t.Context(), deviceID, []devices.Detection{
			{Label: "car"},
		})

		a.NoError(err)
		a.True(tracker.HasEvent(deviceID))

		latestEvent, err := tracker.EventRepo.GetLatest(t.Context(), deviceID)
		require.NoError(t, err)

		a.NotZero(latestEvent.ID)
		a.Equal(int(event.Started), latestEvent.State)
		a.Equal([]string{"car"}, latestEvent.Labels)
	})

	t.Run("event_state_with_matching_labels", func(t *testing.T) {
		de, err := tracker.Start(t.Context(), deviceID, []string{"car"})
		a.NoError(err)
		a.NotEqual(de.ID, int64(0))

		oldEventID := tracker.CurrentEventID(t.Context(), deviceID)
		require.NotZero(t, oldEventID)

		err = tracker.ReceiveDetections(t.Context(), deviceID, []devices.Detection{
			{Label: "car"},
		})

		a.NoError(err)
		a.True(tracker.HasEvent(deviceID))
		a.Equal(oldEventID, tracker.CurrentEventID(t.Context(), deviceID))

		latestEvent, err := tracker.EventRepo.GetLatest(t.Context(), deviceID)
		require.NoError(t, err)

		a.Equal(oldEventID, latestEvent.ID)
		a.Equal(int(event.Started), latestEvent.State)
		a.Equal([]string{"car"}, latestEvent.Labels)
	})

	t.Run("event_ends_new_created_when_labels_dont_match", func(t *testing.T) {
		did, secErr := getTestDeviceID(t.Context(), appDb.GetQueries())
		a.NoError(secErr)
		de, err := tracker.Start(t.Context(), did, []string{"car"})
		a.NoError(err)
		a.NotEqual(de.ID, int64(0))

		oldEventID := tracker.CurrentEventID(t.Context(), did)
		require.NotZero(t, oldEventID)

		err = tracker.ReceiveDetections(t.Context(), did, []devices.Detection{
			{Label: "person"},
		})

		a.NoError(err)
		a.True(tracker.HasEvent(did))

		newEventID := tracker.CurrentEventID(t.Context(), did)
		a.NotZero(newEventID)
		a.NotEqual(oldEventID, newEventID)

		oldEvent, err := tracker.EventRepo.GetEvent(t.Context(), oldEventID)
		require.NoError(t, err)

		a.Equal(int(event.Ended), oldEvent.State)

		latestEvent, err := tracker.EventRepo.GetLatest(t.Context(), did)
		require.NoError(t, err)

		a.Equal(newEventID, latestEvent.ID)
		a.Equal(int(event.Started), latestEvent.State)
		a.Equal([]string{"person"}, latestEvent.Labels)
	})

}

func Test_DetectionTracker_ReceiveDetections_RestartsEventWhenLabelsDiffer(t *testing.T) {
	a := assert.New(t)
	ctx := t.Context()
	deps, appDb, err := testutils.NewTestDeps("/videos", "/archive")
	a.NoError(err)
	defer func() {
		if appDb != nil {
			appDb.Db.Close()
		}
	}()
	tracker := detections.NewDetectionTracker(deps)
	deviceID, dErr := getTestDeviceID(t.Context(), appDb.GetQueries())
	a.NoError(dErr)
	defer func() {
		_ = deps.DeviceRepo.DeleteDevice(t.Context(), deviceID)
	}()
	de, err := tracker.Start(ctx, deviceID, []string{"car"})
	a.NoError(err)
	a.NotEqual(de.ID, int64(0))

	oldEventID := tracker.CurrentEventID(ctx, deviceID)
	require.NotZero(t, oldEventID)

	err = tracker.ReceiveDetections(ctx, deviceID, []devices.Detection{
		{Label: "person"},
	})

	a.NoError(err)
	a.True(tracker.HasEvent(deviceID))

	newEventID := tracker.CurrentEventID(ctx, deviceID)
	a.NotZero(newEventID)
	a.NotEqual(oldEventID, newEventID)

	oldEvent, err := tracker.EventRepo.GetEvent(ctx, oldEventID)
	require.NoError(t, err)

	a.Equal(int(event.Ended), oldEvent.State)

	latestEvent, err := tracker.EventRepo.GetLatest(ctx, deviceID)
	require.NoError(t, err)

	a.Equal(newEventID, latestEvent.ID)
	a.Equal(int(event.Started), latestEvent.State)
	a.Equal([]string{"person"}, latestEvent.Labels)
}

func Test_DetectionLabels(t *testing.T) {
	a := assert.New(t)

	labels := detections.DetectionLabels([]devices.Detection{
		{Label: "car"},
		{Label: "person"},
	})

	a.Equal([]string{"car", "person"}, labels)
}

func Test_DetectionTracker_ReceiveDetections_UsesLatestPersistedEvent(t *testing.T) {
	a := assert.New(t)
	ctx := t.Context()
	deps, appDb, err := testutils.NewTestDeps("/videos", "/archive")
	a.NoError(err)
	defer func() {
		if appDb != nil {
			appDb.Db.Close()
		}
	}()
	tracker := detections.NewDetectionTracker(deps)
	deviceID, dErr := getTestDeviceID(t.Context(), appDb.GetQueries())
	a.NoError(dErr)
	defer func() {
		_ = deps.DeviceRepo.DeleteDevice(t.Context(), deviceID)
	}()

	evt, err := deps.EventRepo.StartEvent(ctx, deviceID, []string{"car"})
	require.NoError(t, err)
	require.NotZero(t, evt.ID)

	err = tracker.ReceiveDetections(ctx, deviceID, []devices.Detection{
		{Label: "car"},
	})

	a.NoError(err)
	a.True(tracker.HasEvent(deviceID))
	a.Equal(evt.ID, tracker.CurrentEventID(ctx, deviceID))

	latestEvent, err := deps.EventRepo.GetLatest(ctx, deviceID)
	require.NoError(t, err)

	a.Equal(evt.ID, latestEvent.ID)
	a.Equal(int(event.Started), latestEvent.State)
	a.Equal([]string{"car"}, latestEvent.Labels)
}

func getTestDeviceID(ctx context.Context, queries *db.Queries) (int64, error) {
	td, err := repos.GetTestDevice(ctx, queries)
	if err != nil {
		return int64(0), err
	}
	return td.ID, err
}
