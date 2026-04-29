package devices

import (
	"context"
	"sync"
	"time"
)

type MockHeartbeat struct {
	mu  sync.Mutex
	hbs []*Heartbeat
}

func NewMockHeartbeat() *MockHeartbeat {
	return &MockHeartbeat{
		hbs: []*Heartbeat{},
	}
}

func (h *MockHeartbeat) GetDeviceHeartBeats(_ context.Context, deviceId int64) ([]*Heartbeat, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	var result []*Heartbeat
	for _, heartbeat := range h.hbs {
		if heartbeat.DeviceID == deviceId {
			result = append(result, heartbeat)
		}
	}

	return result, nil
}

func (h *MockHeartbeat) HeartBeatsAfter(_ context.Context, createdAt time.Time) ([]*Heartbeat, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	var result []*Heartbeat
	for _, heartbeat := range h.hbs {
		if heartbeat.CreatedAt.After(createdAt) {
			result = append(result, heartbeat)
		}
	}

	return result, nil
}

func (h *MockHeartbeat) LatestBeats(_ context.Context) ([]*LatestBeatsRow, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Find the latest heartbeat for each device
	deviceLatest := make(map[int64]*Heartbeat)
	for _, heartbeat := range h.hbs {
		if latest, exists := deviceLatest[heartbeat.DeviceID]; !exists || heartbeat.CreatedAt.After(latest.CreatedAt) {
			deviceLatest[heartbeat.DeviceID] = heartbeat
		}
	}

	var result []*LatestBeatsRow
	for _, heartbeat := range deviceLatest {
		row := &LatestBeatsRow{
			DeviceID:  heartbeat.DeviceID,
			CreatedAt: heartbeat.CreatedAt,
			ID:        heartbeat.ID,
			Name:      "mock_device",         // Mock device name
			DeviceUrl: "http://0.0.0.0:8080", // Mock device URL
		}
		result = append(result, row)
	}

	return result, nil
}

// RecordBeat Record a DeviceHeartbeat
func (h *MockHeartbeat) RecordBeat(_ context.Context, deviceId int64) (*Heartbeat, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	heartbeat := &Heartbeat{
		ID:        int64(len(h.hbs) + 1),
		DeviceID:  deviceId,
		CreatedAt: time.Now(),
	}

	h.hbs = append(h.hbs, heartbeat)
	return heartbeat, nil
}

func (h *MockHeartbeat) DeleteBeats(_ context.Context, deviceId int64) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	var filteredHeartbeats []*Heartbeat
	for _, heartbeat := range h.hbs {
		if heartbeat.DeviceID != deviceId {
			filteredHeartbeats = append(filteredHeartbeats, heartbeat)
		}
	}

	h.hbs = filteredHeartbeats
	return nil
}
