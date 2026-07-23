package camera

import (
	"testing"

	"devicecapture/internal/domain/event"
	"devicecapture/internal/logger"
	"devicecapture/internal/pubsub"
	"github.com/stretchr/testify/assert"
)

func Test_Detection_Event_Start(t *testing.T) {
	a := assert.New(t)
	tracker := getTestTracker()
	deviceID := int64(5)
	err := tracker.Start(t.Context(), deviceID, []string{"car"})
	a.NoError(err)
	a.Equal(tracker.HasEvent(deviceID), true)
	a.Equal(tracker.LabelsEq(deviceID, []string{"car"}), true)
	a.Equal(tracker.LabelsEq(deviceID, []string{"car", "person"}), false)
}

func Test_Detection_Event_End(t *testing.T) {
	a := assert.New(t)
	tracker := getTestTracker()
	deviceID := int64(4)
	// Start the DetectionEvent
	err := tracker.Start(t.Context(), deviceID, []string{"car"})
	a.NoError(err)
	// End the event for this device
	a.Equal(true, tracker.HasEvent(deviceID))
	err = tracker.End(t.Context(), deviceID)
	a.NoError(err)
	a.Equal(false, tracker.HasEvent(deviceID))
	a.Equal(false, tracker.LabelsEq(deviceID, []string{"person"}))
}

func Test_Detection_Event_End_Fail(t *testing.T) {
	a := assert.New(t)
	tracker := getTestTracker()
	deviceID := int64(3)
	// Make sure DetectionTracker.End returns an error if DetectionTracker.Start has not been called
	a.Error(tracker.End(t.Context(), deviceID))
}

func getTestTracker() *DetectionTracker {
	er := event.NewMockEventRepo()
	client, err := pubsub.NewMockMqtt()
	if err != nil {
		logger.Error().Err(err).
			Str("expect", "subsequent tests to fail").
			Msg("getTestTracker threw an error")
	}
	return NewDetectionTracker(er, &client)
}
