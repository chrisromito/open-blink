package event

import (
	"context"
	"math/rand/v2"
	"slices"
	"sync"
	"time"
)

type MockEventRepo struct {
	events    []DetectionEvent
	mu        sync.Mutex
	currentID int64
}

func NewMockEventRepo() *MockEventRepo {
	return &MockEventRepo{
		events:    []DetectionEvent{},
		mu:        sync.Mutex{},
		currentID: 1,
	}
}

// GetEvent implements [DetectionEventRepo]
func (de *MockEventRepo) GetEvent(
	_ context.Context,
	id int64,
) (DetectionEvent, error) {
	de.mu.Lock()
	defer de.mu.Unlock()
	for _, evt := range de.events {
		if evt.ID == id {
			return evt, nil
		}
	}
	return DetectionEvent{}, nil
}

// StartEvent implements DetectionEventRepo
func (de *MockEventRepo) StartEvent(
	_ context.Context,
	deviceID int64,
	labels []string,
) (DetectionEvent, error) {
	de.mu.Lock()
	defer de.mu.Unlock()
	ls := slices.Clone(labels)
	slices.Sort(ls)
	evt := DetectionEvent{
		ID:        de.getNextID(),
		CreatedAt: time.Now(),
		EndedAt:   time.Time{},
		DeviceID:  deviceID,
		Labels:    ls,
		State:     2,
	}
	de.events = append(de.events, evt)
	return evt, nil
}

// EndEvent implements DetectionEventRepo
func (de *MockEventRepo) EndEvent(
	_ context.Context,
	e DetectionEvent,
) (DetectionEvent, error) {
	de.mu.Lock()
	defer de.mu.Unlock()
	for i := range de.events {
		if de.events[i].ID == e.ID {
			de.events[i].State = int(Ended)
			de.events[i].EndedAt = time.Now()
			return de.events[i], nil
		}
	}
	return DetectionEvent{}, nil
}

// GetDetectionEvents implements DetectionEventRepo
func (de *MockEventRepo) GetDetectionEvents(
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

// GetDetectionsForEvent implements DetectionEventRepo
func (de *MockEventRepo) GetDetectionsForEvent(
	_ context.Context,
	eventID int64,
) (DetectionDetail, error) {
	return DetectionDetail{ID: eventID}, nil
}

func (de *MockEventRepo) getNextID() int64 {
	next := de.currentID + 1
	de.currentID = next
	return next
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
