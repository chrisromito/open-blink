package event

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMockEventRepo_StartEvent_ReturnsStartedEvent(t *testing.T) {
	a := assert.New(t)
	r := require.New(t)

	repo := NewMockEventRepo()
	ctx := context.Background()

	evt, err := startEventWithTimeout(t, repo, ctx, 10, []string{"person", "car"})

	r.NoError(err, "StartEvent should not return an error")
	a.NotZero(evt.ID, "StartEvent should assign a non-zero event ID")
	a.Equal(int64(10), evt.DeviceID, "StartEvent should preserve the device ID")
	a.Equal([]string{"car", "person"}, evt.Labels, "StartEvent should sort labels before storing them")
	a.Equal(int(Started), evt.State, "StartEvent should create the event in Started state")
	a.False(evt.CreatedAt.IsZero(), "StartEvent should set CreatedAt")
	a.True(evt.EndedAt.IsZero(), "StartEvent should not set EndedAt for a started event")
}

func TestMockEventRepo_StartEvent_DoesNotMutateInputLabels(t *testing.T) {
	a := assert.New(t)
	r := require.New(t)

	repo := NewMockEventRepo()
	ctx := context.Background()
	labels := []string{"person", "car"}

	evt, err := startEventWithTimeout(t, repo, ctx, 11, labels)

	r.NoError(err, "StartEvent should not return an error")
	a.Equal(
		[]string{"person", "car"},
		labels,
		"StartEvent should clone labels before sorting so the caller's input is unchanged",
	)
	a.Equal(
		[]string{"car", "person"},
		evt.Labels,
		"StartEvent should store sorted labels on the event",
	)
}

func TestMockEventRepo_StartEvent_AssignsIncreasingIDs(t *testing.T) {
	a := assert.New(t)
	r := require.New(t)

	repo := NewMockEventRepo()
	ctx := context.Background()

	first, err := startEventWithTimeout(t, repo, ctx, 12, []string{"car"})
	r.NoError(err, "first StartEvent call should not return an error")

	second, err := startEventWithTimeout(t, repo, ctx, 12, []string{"person"})
	r.NoError(err, "second StartEvent call should not return an error")

	a.Greater(
		second.ID,
		first.ID,
		"StartEvent should assign increasing IDs; first ID=%d, second ID=%d",
		first.ID,
		second.ID,
	)
}

func TestMockEventRepo_GetEvent_ReturnsStartedEvent(t *testing.T) {
	a := assert.New(t)
	r := require.New(t)

	repo := NewMockEventRepo()
	ctx := context.Background()

	started, err := startEventWithTimeout(t, repo, ctx, 13, []string{"car"})
	r.NoError(err, "StartEvent should not return an error before GetEvent test")

	found, err := repo.GetEvent(ctx, started.ID)

	r.NoError(err, "GetEvent should not return an error for an existing event")
	a.Equal(started.ID, found.ID, "GetEvent should return the event with the requested ID")
	a.Equal(started.DeviceID, found.DeviceID, "GetEvent should preserve DeviceID")
	a.Equal(started.Labels, found.Labels, "GetEvent should preserve Labels")
	a.Equal(started.State, found.State, "GetEvent should preserve State")
}

func TestMockEventRepo_GetEvent_ReturnsZeroValueWhenMissing(t *testing.T) {
	a := assert.New(t)

	repo := NewMockEventRepo()

	found, err := repo.GetEvent(context.Background(), 999)

	a.NoError(err, "GetEvent should not return an error when the event is missing")
	a.Equal(
		DetectionEvent{},
		found,
		"GetEvent should return a zero-value DetectionEvent when ID is missing",
	)
}

func TestMockEventRepo_EndEvent_ReturnsEndedEvent(t *testing.T) {
	a := assert.New(t)
	r := require.New(t)

	repo := NewMockEventRepo()
	ctx := context.Background()

	started, err := startEventWithTimeout(t, repo, ctx, 14, []string{"car"})
	r.NoError(err, "StartEvent should not return an error before EndEvent test")

	ended, err := repo.EndEvent(ctx, DetectionEvent{ID: started.ID})

	r.NoError(err, "EndEvent should not return an error for an existing event")
	a.Equal(started.ID, ended.ID, "EndEvent should return the ended event with the same ID")
	a.Equal(int(Ended), ended.State, "EndEvent should return the event in Ended state")
	a.False(ended.EndedAt.IsZero(), "EndEvent should set EndedAt")
}

func TestMockEventRepo_EndEvent_PersistsEndedEvent(t *testing.T) {
	a := assert.New(t)
	r := require.New(t)

	repo := NewMockEventRepo()
	ctx := context.Background()

	started, err := startEventWithTimeout(t, repo, ctx, 15, []string{"car"})
	r.NoError(err, "StartEvent should not return an error before persistence test")

	ended, err := repo.EndEvent(ctx, DetectionEvent{ID: started.ID})
	r.NoError(err, "EndEvent should not return an error for an existing event")
	r.NotZero(ended.ID, "EndEvent should return a non-zero event ID for an existing event")

	found, err := repo.GetEvent(ctx, started.ID)

	r.NoError(err, "GetEvent should not return an error after EndEvent")
	a.Equal(started.ID, found.ID, "GetEvent should return the same event after EndEvent")
	a.Equal(
		3,
		found.State,
		"EndEvent should persist the Ended state so later GetEvent calls see it",
	)
	a.False(
		found.EndedAt.IsZero(),
		"EndEvent should persist EndedAt so later GetEvent calls see it",
	)
}

func TestMockEventRepo_EndEvent_ReturnsZeroValueWhenMissing(t *testing.T) {
	a := assert.New(t)

	repo := NewMockEventRepo()

	ended, err := repo.EndEvent(context.Background(), DetectionEvent{ID: 999})

	a.NoError(err, "EndEvent should not return an error when the event is missing")
	a.Equal(
		DetectionEvent{},
		ended,
		"EndEvent should return a zero-value DetectionEvent when ID is missing",
	)
}

func TestMockEventRepo_GetDetectionEvents_ReturnsEmptyWhenDeviceIDIsZero(t *testing.T) {
	a := assert.New(t)

	repo := NewMockEventRepo()

	events, err := repo.GetDetectionEvents(context.Background(), QueryParams{})

	a.NoError(err, "GetDetectionEvents should not return an error for empty query params")
	a.Empty(events, "GetDetectionEvents should return no events when DeviceID is zero")
}

func TestMockEventRepo_GetDetectionEvents_ReturnsEndedEventsByDefault(t *testing.T) {
	a := assert.New(t)

	repo := NewMockEventRepo()
	deviceID := int64(16)

	events, err := repo.GetDetectionEvents(context.Background(), QueryParams{
		DeviceID: deviceID,
	})

	a.NoError(err, "GetDetectionEvents should not return an error for a valid DeviceID")
	require.Len(
		t,
		events,
		10,
		"GetDetectionEvents should return 10 mock events for a valid DeviceID",
	)

	for _, evt := range events {
		a.Equal(deviceID, evt.DeviceID, "mock event should use requested DeviceID")
		a.Equal(int(Ended), evt.State, "mock event should default to Ended state")
		a.False(evt.CreatedAt.IsZero(), "mock event should have CreatedAt set")
		a.False(evt.EndedAt.IsZero(), "mock ended event should have EndedAt set")
	}
}

func TestMockEventRepo_GetDetectionEvents_UsesRequestedState(t *testing.T) {
	a := assert.New(t)

	repo := NewMockEventRepo()
	deviceID := int64(17)

	events, err := repo.GetDetectionEvents(context.Background(), QueryParams{
		DeviceID: deviceID,
		State:    Started,
	})

	a.NoError(err, "GetDetectionEvents should not return an error for a valid query")
	require.Len(
		t,
		events,
		10,
		"GetDetectionEvents should return 10 mock events for a valid DeviceID",
	)

	for _, evt := range events {
		a.Equal(deviceID, evt.DeviceID, "mock event should use requested DeviceID")
		a.Equal(int(Started), evt.State, "mock event should use requested State")
	}
}

func TestMockEventRepo_GetDetectionsForEvent_ReturnsDetailWithEventID(t *testing.T) {
	a := assert.New(t)

	repo := NewMockEventRepo()
	eventID := int64(18)

	detail, err := repo.GetDetectionsForEvent(context.Background(), eventID)

	a.NoError(err, "GetDetectionsForEvent should not return an error")
	a.Equal(
		eventID,
		detail.ID,
		"GetDetectionsForEvent should return detail with the requested event ID",
	)
}

func TestGetRandomLabels_ReturnsCombinedLabels(t *testing.T) {
	a := assert.New(t)

	labels := GetRandomLabels(20)

	a.NotEmpty(labels, "GetRandomLabels should return labels when n is greater than zero")

	for i := 1; i < len(labels); i++ {
		a.LessOrEqual(
			labels[i-1],
			labels[i],
			"GetRandomLabels should return sorted combined labels; index %d=%q, index %d=%q",
			i-1,
			labels[i-1],
			i,
			labels[i],
		)
	}
}

func startEventWithTimeout(
	t *testing.T,
	repo *MockEventRepo,
	ctx context.Context,
	deviceID int64,
	labels []string,
) (DetectionEvent, error) {
	t.Helper()

	type result struct {
		evt DetectionEvent
		err error
	}

	done := make(chan result, 1)

	go func() {
		evt, err := repo.StartEvent(ctx, deviceID, labels)
		done <- result{evt: evt, err: err}
	}()

	select {
	case res := <-done:
		return res.evt, res.err
	case <-time.After(250 * time.Millisecond):
		t.Fatalf(
			"MockEventRepo.StartEvent timed out for deviceID=%d labels=%v; this usually means StartEvent deadlocked",
			deviceID,
			labels,
		)
		return DetectionEvent{}, nil
	}
}