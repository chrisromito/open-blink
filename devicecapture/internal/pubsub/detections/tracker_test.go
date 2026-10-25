package detections

import (
	"testing"

	"devicecapture/internal/domain"
	"devicecapture/internal/domain/detection"
	"devicecapture/internal/domain/event"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_DetectionTracker_Start(t *testing.T) {
	a := assert.New(t)
	tracker := getTestTracker()
	deviceID := int64(5)

	err := tracker.Start(t.Context(), deviceID, []string{"car"})

	a.NoError(err)
	a.True(tracker.HasEvent(deviceID))
	a.NotZero(tracker.CurrentEventID(t.Context(), deviceID))
}

func Test_DetectionTracker_End(t *testing.T) {
	a := assert.New(t)
	tracker := getTestTracker()
	deviceID := int64(4)

	err := tracker.Start(t.Context(), deviceID, []string{"car"})
	a.NoError(err)
	a.True(tracker.HasEvent(deviceID))

	err = tracker.End(t.Context(), deviceID)

	a.NoError(err)
	a.False(tracker.HasEvent(deviceID))
	a.Zero(tracker.CurrentEventID(t.Context(), deviceID))
}

func Test_DetectionTracker_End_NoopsWhenEventWasNotStarted(t *testing.T) {
	a := assert.New(t)
	tracker := getTestTracker()
	deviceID := int64(3)

	err := tracker.End(t.Context(), deviceID)

	a.NoError(err)
	a.False(tracker.HasEvent(deviceID))
}

func Test_DetectionTracker_Start_MultipleDevicesAreTrackedIndependently(t *testing.T) {
	a := assert.New(t)
	tracker := getTestTracker()

	firstDeviceID := int64(10)
	secondDeviceID := int64(11)

	a.NoError(tracker.Start(t.Context(), firstDeviceID, []string{"car"}))
	a.NoError(tracker.Start(t.Context(), secondDeviceID, []string{"person"}))

	a.True(tracker.HasEvent(firstDeviceID))
	a.True(tracker.HasEvent(secondDeviceID))

	a.NoError(tracker.End(t.Context(), firstDeviceID))

	a.False(tracker.HasEvent(firstDeviceID))
	a.True(tracker.HasEvent(secondDeviceID))
}

func Test_DetectionTracker_ReceiveDetections_StartsEventWhenDeviceHasNoEvent(t *testing.T) {
	a := assert.New(t)
	ctx := t.Context()
	tracker := getTestTracker()
	deviceID := int64(30)

	err := tracker.ReceiveDetections(ctx, deviceID, []detection.Detection{
		{Label: "car"},
	})

	a.NoError(err)
	a.True(tracker.HasEvent(deviceID))

	latestEvent, err := tracker.EventRepo.GetLatest(ctx, deviceID)
	require.NoError(t, err)

	a.NotZero(latestEvent.ID)
	a.Equal(int(event.Started), latestEvent.State)
	a.Equal([]string{"car"}, latestEvent.Labels)
}

func Test_DetectionTracker_ReceiveDetections_KeepsEventWhenLabelsMatch(t *testing.T) {
	a := assert.New(t)
	ctx := t.Context()
	tracker := getTestTracker()
	deviceID := int64(31)

	a.NoError(tracker.Start(ctx, deviceID, []string{"car"}))

	oldEventID := tracker.CurrentEventID(ctx, deviceID)
	require.NotZero(t, oldEventID)

	err := tracker.ReceiveDetections(ctx, deviceID, []detection.Detection{
		{Label: "car"},
	})

	a.NoError(err)
	a.True(tracker.HasEvent(deviceID))
	a.Equal(oldEventID, tracker.CurrentEventID(ctx, deviceID))

	latestEvent, err := tracker.EventRepo.GetLatest(ctx, deviceID)
	require.NoError(t, err)

	a.Equal(oldEventID, latestEvent.ID)
	a.Equal(int(event.Started), latestEvent.State)
	a.Equal([]string{"car"}, latestEvent.Labels)
}

func Test_DetectionTracker_ReceiveDetections_RestartsEventWhenLabelsDiffer(t *testing.T) {
	a := assert.New(t)
	ctx := t.Context()
	tracker := getTestTracker()
	deviceID := int64(32)

	a.NoError(tracker.Start(ctx, deviceID, []string{"car"}))

	oldEventID := tracker.CurrentEventID(ctx, deviceID)
	require.NotZero(t, oldEventID)

	err := tracker.ReceiveDetections(ctx, deviceID, []detection.Detection{
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

	labels := DetectionLabels([]detection.Detection{
		{Label: "car"},
		{Label: "person"},
	})

	a.Equal([]string{"car", "person"}, labels)
}

func Test_DetectionTracker_ReceiveDetections_UsesLatestPersistedEvent(t *testing.T) {
	a := assert.New(t)
	ctx := t.Context()
	deps := domain.NewMockDeps()
	deviceID := int64(60)

	evt, err := deps.EventRepo.StartEvent(ctx, deviceID, []string{"car"})
	require.NoError(t, err)
	require.NotZero(t, evt.ID)

	tracker := NewDetectionTracker(deps)

	err = tracker.ReceiveDetections(ctx, deviceID, []detection.Detection{
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

func Test_DetectionTracker_ReceiveDetections_RestartsLatestPersistedEventWhenLabelsDiffer(t *testing.T) {
	a := assert.New(t)
	ctx := t.Context()
	deps := domain.NewMockDeps()
	deviceID := int64(61)

	oldEvent, err := deps.EventRepo.StartEvent(ctx, deviceID, []string{"car"})
	require.NoError(t, err)
	require.NotZero(t, oldEvent.ID)

	tracker := NewDetectionTracker(deps)

	err = tracker.ReceiveDetections(ctx, deviceID, []detection.Detection{
		{Label: "person"},
	})

	a.NoError(err)
	a.True(tracker.HasEvent(deviceID))

	newEventID := tracker.CurrentEventID(ctx, deviceID)
	a.NotZero(newEventID)
	a.NotEqual(oldEvent.ID, newEventID)

	endedEvent, err := deps.EventRepo.GetEvent(ctx, oldEvent.ID)
	require.NoError(t, err)

	a.Equal(int(event.Ended), endedEvent.State)

	latestEvent, err := deps.EventRepo.GetLatest(ctx, deviceID)
	require.NoError(t, err)

	a.Equal(newEventID, latestEvent.ID)
	a.Equal(int(event.Started), latestEvent.State)
	a.Equal([]string{"person"}, latestEvent.Labels)
}

func getTestTracker() *DetectionTracker {
	return NewDetectionTracker(domain.NewMockDeps())
}
