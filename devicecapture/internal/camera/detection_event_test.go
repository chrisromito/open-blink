package camera

import (
	"devicecapture/internal/domain"
	"devicecapture/internal/domain/detection"
	"devicecapture/internal/domain/devices"
	"devicecapture/internal/domain/event"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
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

func Test_DetectionTracker_ReceiveDetections_StartsEventWhenDeviceHasNoEvent(t *testing.T) {
	a := assert.New(t)
	tracker := getTestTracker()
	deviceID := int64(30)

	err := tracker.ReceiveDetections(t.Context(), deviceID, []detection.Detection{
		{Label: "car"},
	})

	a.NoError(err)
	a.True(tracker.HasEvent(deviceID))
}

func Test_DetectionTracker_ReceiveDetections_AppendsWhenLabelsMatch(t *testing.T) {
	a := assert.New(t)
	tracker := getTestTracker()
	deviceID := int64(31)

	a.NoError(tracker.Start(t.Context(), deviceID, []string{"car"}))

	state := requireDeviceState(t, tracker, deviceID)
	state.AddDetection(detection.Detection{Label: "car"})

	err := tracker.ReceiveDetections(t.Context(), deviceID, []detection.Detection{
		{Label: "car"},
	})

	a.NoError(err)
	a.True(state.Eq([]detection.Detection{
		{Label: "car"},
		{Label: "car"},
	}))
}

func Test_DetectionTracker_ReceiveDetections_RestartsEventWhenLabelsDiffer(t *testing.T) {
	a := assert.New(t)
	tracker := getTestTracker()
	deviceID := int64(32)

	a.NoError(tracker.Start(t.Context(), deviceID, []string{"car"}))

	oldEventID := tracker.CurrentEventID(deviceID)

	state := requireDeviceState(t, tracker, deviceID)
	state.AddDetection(detection.Detection{Label: "car"})

	err := tracker.ReceiveDetections(t.Context(), deviceID, []detection.Detection{
		{Label: "person"},
	})

	a.NoError(err)
	a.True(tracker.HasEvent(deviceID))
	a.NotEqual(oldEventID, tracker.CurrentEventID(deviceID))
}

func Test_DetectionTracker_SetEvent_CreatesStateWhenMissing(t *testing.T) {
	a := assert.New(t)
	tracker := getTestTracker()
	deviceID := int64(40)

	tracker.setEvent(deviceID, event.DetectionEvent{ID: 300, DeviceID: deviceID})

	state := requireDeviceState(t, tracker, deviceID)

	a.True(state.HasEvent())
	a.Equal(int64(300), state.EventID())
}

func Test_DetectionTracker_SetEvent_DoesNotOverwriteExistingEvent(t *testing.T) {
	a := assert.New(t)
	tracker := getTestTracker()
	deviceID := int64(41)

	tracker.setEvent(deviceID, event.DetectionEvent{ID: 301, DeviceID: deviceID})
	tracker.setEvent(deviceID, event.DetectionEvent{ID: 302, DeviceID: deviceID})

	state := requireDeviceState(t, tracker, deviceID)

	a.True(state.HasEvent())
	a.Equal(int64(301), state.EventID())
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
	state := NewDeviceEventState(51)

	state.SetEvent(400)

	a.True(state.HasEvent())
	a.Equal(int64(400), state.EventID())
}

func Test_DeviceEventState_End(t *testing.T) {
	a := assert.New(t)
	deps := domain.NewMockDeps()
	state := NewDeviceEventState(51)

	evt, err := deps.EventRepo.StartEvent(t.Context(), 51, []string{"car"})
	require.NoError(t, err)

	state.SetEvent(evt.ID)

	ended, err := state.End(t.Context(), deps.EventRepo)

	a.NoError(err)
	a.Equal(evt.ID, ended.ID)
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

	ended, err := state.End(t.Context(), deps.EventRepo)

	a.NoError(err)
	a.Equal(evt.ID, ended.ID)
}

func Test_DeviceEventState_AddDetection(t *testing.T) {
	a := assert.New(t)
	state := NewDeviceEventState(53)

	state.AddDetection(detection.Detection{Label: "car"})

	a.True(state.Eq([]detection.Detection{{Label: "car"}}))
	a.False(state.Eq([]detection.Detection{{Label: "person"}}))
}

func Test_DeviceEventState_AddDetections(t *testing.T) {
	a := assert.New(t)
	state := NewDeviceEventState(54)

	state.AddDetections([]detection.Detection{
		{Label: "car"},
		{Label: "person"},
	})

	a.True(state.Eq([]detection.Detection{
		{Label: "car"},
		{Label: "person"},
	}))
	a.False(state.Eq([]detection.Detection{
		{Label: "car"},
	}))
}

func Test_DetectionLabels(t *testing.T) {
	a := assert.New(t)

	labels := DetectionLabels([]detection.Detection{
		{Label: "car"},
		{Label: "person"},
	})

	a.Equal([]string{"car", "person"}, labels)
}

func getTestTracker() *DetectionTracker {
	return NewDetectionTracker(domain.NewMockDeps())
}

func requireDeviceState(t *testing.T, tracker *DetectionTracker, deviceID int64) *DeviceEventState {
	t.Helper()

	state, ok := tracker.deviceStates[deviceID]
	require.True(t, ok)
	require.NotNil(t, state)

	return state
}
