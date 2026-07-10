package event

import (
	"context"
	"math/rand/v2"
	"slices"
	"time"
)

type MockEventRepo struct {
}

func NewMockEventRepo() *MockEventRepo {
	return &MockEventRepo{}
}

// StartEvent implements DetectionEventRepo
func (de *MockEventRepo) StartEvent(
	_ context.Context,
	deviceID int64,
	labels []string,
) (DetectionEvent, error) {
	ls := slices.Clone(labels)
	slices.Sort(ls)
	evt := DetectionEvent{
		ID:        int64(1),
		CreatedAt: time.Now(),
		EndedAt:   time.Time{},
		DeviceID:  deviceID,
		Labels:    ls,
		State:     2,
	}
	return evt, nil
}

// EndEvent implements DetectionEventRepo
func (de *MockEventRepo) EndEvent(
	_ context.Context,
	e DetectionEvent,
) (DetectionEvent, error) {
	// Copy all fields over to a new struct with [event.Ended] state and [EndedAt] set to [time.Now]
	evt := DetectionEvent{
		ID:        e.ID + int64(1),
		CreatedAt: e.CreatedAt,
		Labels:    e.Labels,
		// Flag it as "ended"
		EndedAt: time.Now(),
		State:   3,
	}
	return evt, nil
}

// GetDeviceEvents implements DetectionEventRepo
func (de *MockEventRepo) GetDeviceEvents(
	_ context.Context,
	p QueryParams,
) ([]DetectionEvent, error) {
	if p.DeviceID == 0 {
		return []DetectionEvent{}, nil
	}
	var temp []DetectionEvent
	state := p.State
	if state == 0 {
		state = Ended
	}
	for i := range 10 {
		temp = append(temp, DetectionEvent{
			ID:        int64(i + 1),
			DeviceID:  p.DeviceID,
			CreatedAt: time.Now().Add(-1 * time.Minute),
			EndedAt:   time.Now(),
			State:     int(state),
			Labels:    GetRandomLabels(i),
		})
	}
	return temp, nil
}

func GetRandomLabels(n int) []string {
	//var temp []string
	temp := make([]string, n)
	labelChoices := []string{"car", "person", "truck", "dog", "cat", "airplane"}

	for i := range n {
		idx := rand.N(len(labelChoices))
		temp[i] = labelChoices[idx]
	}
	// CombineLabels will sort and de-duplicate
	return CombineLabels(temp, []string{})
}
