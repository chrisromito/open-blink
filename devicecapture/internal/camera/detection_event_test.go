package camera

import (
	"testing"

	"devicecapture/internal/domain"
	"devicecapture/internal/domain/devices"
	"devicecapture/internal/domain/event"
	"devicecapture/internal/logger"
	"devicecapture/internal/pubsub"
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
}

func Test_DetectionTracker_End_ReturnsErrorWhenEventWasNotStarted(t *testing.T) {
	a := assert.New(t)
	tracker := getTestTracker()
	deviceID := int64(3)

	err := tracker.End(t.Context(), deviceID)

	a.Error(err)
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

func Test_DetectionTracker_ReceiveImage_NoopsWhenDeviceHasNoEvent(t *testing.T) {
	a := assert.New(t)
	tracker := getTestTracker()
	deviceID := int64(20)

	err := tracker.ReceiveImage(deviceID, devices.DeviceImage{ID: 100})

	a.NoError(err)
	a.False(tracker.HasEvent(deviceID))
}

func Test_DetectionTracker_ReceiveImage_AddsImageWhenDeviceHasEvent(t *testing.T) {
	a := assert.New(t)
	tracker := getTestTracker()
	deviceID := int64(21)

	a.NoError(tracker.Start(t.Context(), deviceID, []string{"car"}))

	err := tracker.ReceiveImage(deviceID, devices.DeviceImage{ID: 101})

	a.NoError(err)
	a.True(tracker.HasEvent(deviceID))
}

func Test_DetectionTracker_ReceiveDetections_ReturnsChangedWhenDeviceHasNoEvent(t *testing.T) {
	a := assert.New(t)
	tracker := getTestTracker()
	deviceID := int64(30)

	changed := tracker.ReceiveDetections(deviceID, []devices.Detection{
		{ID: 200, Label: "car"},
	})

	a.True(changed)
	a.False(tracker.HasEvent(deviceID))
}

func Test_DetectionTracker_ReceiveDetections_ReturnsFalseAndAppendsWhenLabelsMatch(t *testing.T) {
	a := assert.New(t)
	tracker := getTestTracker()
	deviceID := int64(31)

	a.NoError(tracker.Start(t.Context(), deviceID, []string{"car"}))

	state := requireDeviceState(t, tracker, deviceID)
	state.AddDetection(devices.Detection{ID: 201, Label: "car"})

	changed := tracker.ReceiveDetections(deviceID, []devices.Detection{
		{ID: 202, Label: "car"},
	})

	a.False(changed)
	a.True(state.Eq([]devices.Detection{
		{ID: 203, Label: "car"},
		{ID: 204, Label: "car"},
	}))
}

func Test_DetectionTracker_ReceiveDetections_ReturnsTrueAndDoesNotAppendWhenLabelsDiffer(
	t *testing.T,
) {
	a := assert.New(t)
	tracker := getTestTracker()
	deviceID := int64(32)

	a.NoError(tracker.Start(t.Context(), deviceID, []string{"car"}))

	state := requireDeviceState(t, tracker, deviceID)
	state.AddDetection(devices.Detection{ID: 203, Label: "car"})

	changed := tracker.ReceiveDetections(deviceID, []devices.Detection{
		{ID: 204, Label: "person"},
	})

	a.True(changed)
	a.True(state.Eq([]devices.Detection{
		{ID: 205, Label: "car"},
	}))
	a.False(state.Eq([]devices.Detection{
		{ID: 206, Label: "car"},
		{ID: 207, Label: "person"},
	}))
}

func Test_DetectionTracker_SetEvent_CreatesStateWhenMissing(t *testing.T) {
	a := assert.New(t)
	tracker := getTestTracker()
	deviceID := int64(40)

	tracker.setEvent(deviceID, event.DetectionEvent{ID: 300, DeviceID: deviceID})

	state := requireDeviceState(t, tracker, deviceID)

	a.True(state.HasEvent())
}

func Test_DetectionTracker_SetEvent_DoesNotOverwriteExistingEvent(t *testing.T) {
	a := assert.New(t)
	tracker := getTestTracker()
	deviceID := int64(41)

	tracker.setEvent(deviceID, event.DetectionEvent{ID: 301, DeviceID: deviceID})
	tracker.setEvent(deviceID, event.DetectionEvent{ID: 302, DeviceID: deviceID})

	state := requireDeviceState(t, tracker, deviceID)

	eventID, err := state.End(
		t.Context(),
		tracker.EventRepo,
		tracker.DetectionRepo,
		tracker.ImageRepo,
	)

	a.NoError(err)
	a.Equal(int64(301), eventID)
}

func Test_DeviceEventState_NewDeviceEventState(t *testing.T) {
	a := assert.New(t)

	state := NewDeviceEventState(50)

	a.NotNil(state)
	a.False(state.HasEvent())
	a.True(state.Eq(nil))
}

func Test_DeviceEventState_SetEvent(t *testing.T) {
	a := assert.New(t)
	deps := domain.NewMockDeps()
	state := NewDeviceEventState(51)

	state.SetEvent(400)

	eventID, err := state.End(
		t.Context(),
		deps.EventRepo,
		deps.DetectionRepo,
		deps.ImageRepo,
	)

	a.NoError(err)
	a.Equal(int64(400), eventID)
}

func Test_DeviceEventState_AddImage(t *testing.T) {
	a := assert.New(t)
	state := NewDeviceEventState(52)
	deps := domain.NewMockDeps()

	evt, err := deps.EventRepo.StartEvent(t.Context(), 52, []string{"car"})
	require.NoError(t, err)

	state.SetEvent(evt.ID)
	state.AddImage(devices.DeviceImage{ID: 500})
	state.AddImage(devices.DeviceImage{ID: 501})

	eventID, err := state.End(
		t.Context(),
		deps.EventRepo,
		deps.DetectionRepo,
		deps.ImageRepo,
	)

	a.NoError(err)
	a.Equal(evt.ID, eventID)
}

func Test_DeviceEventState_AddDetection(t *testing.T) {
	a := assert.New(t)
	state := NewDeviceEventState(53)

	state.AddDetection(devices.Detection{ID: 600, Label: "car"})

	a.True(state.Eq([]devices.Detection{{ID: 601, Label: "car"}}))
	a.False(state.Eq([]devices.Detection{{ID: 602, Label: "person"}}))
}

func Test_DeviceEventState_AddDetections(t *testing.T) {
	a := assert.New(t)
	state := NewDeviceEventState(54)

	state.AddDetections([]devices.Detection{
		{ID: 700, Label: "car"},
		{ID: 701, Label: "person"},
	})

	a.True(state.Eq([]devices.Detection{
		{ID: 702, Label: "car"},
		{ID: 703, Label: "person"},
	}))
	a.False(state.Eq([]devices.Detection{
		{ID: 704, Label: "car"},
	}))
}

func getTestTracker() *DetectionTracker {
	client, err := pubsub.NewMockMqtt()
	if err != nil {
		logger.Error().Err(err).
			Str("expect", "subsequent tests to fail").
			Msg("getTestTracker threw an error")
	}

	return NewDetectionTracker(domain.NewMockDeps(), &client)
}

func requireDeviceState(t *testing.T, tracker *DetectionTracker, deviceID int64) *DeviceEventState {
	t.Helper()

	state, ok := tracker.deviceStates[deviceID]
	require.True(t, ok)
	require.NotNil(t, state)

	return state
}

func deviceStateEventID(t *testing.T, state *DeviceEventState) int64 {
	t.Helper()

	state.mu.Lock()
	defer state.mu.Unlock()

	require.NotNil(t, state.eventID)

	return *state.eventID
}

func deviceStateCounts(state *DeviceEventState) (imagesLen int, detectionsLen int) {
	state.mu.Lock()
	defer state.mu.Unlock()

	return len(state.images), len(state.detections)
}