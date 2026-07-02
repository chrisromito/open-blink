package history

import (
	"context"
	"sync"
	"time"
)

type MockDetectionHistory struct {
	mu sync.Mutex
}

func NewMockDetectionHistory() *MockDetectionHistory {
	return &MockDetectionHistory{}
}

func (m *MockDetectionHistory) GetRecentLabels(_ context.Context) ([]string, error) {
	return []string{"car", "truck", "person", "bicycle", "dog"}, nil
}

func (m *MockDetectionHistory) GetDetectionImagesByLabel(
	_ context.Context,
	_ DetectionWithImageParams,
) ([]DetectionWithImage, error) {
	return []DetectionWithImage{
		{
			ID:         1,
			CreatedAt:  time.Now().Add(-1 * time.Hour),
			Label:      "car",
			Confidence: 0.95,
			Bbox:       [][]float64{{10, 10}, {100, 100}},
			DeviceID:   1,
			ImageUrl:   "http://localhost:8080/mock/car.jpg",
		},
		{
			ID:         2,
			CreatedAt:  time.Now().Add(-2 * time.Hour),
			Label:      "person",
			Confidence: 0.87,
			Bbox:       [][]float64{{20, 20}, {80, 150}},
			DeviceID:   2,
			ImageUrl:   "http://localhost:8080/mock/person.jpg",
		},
	}, nil
}

func (m *MockDetectionHistory) GetDetectionTimeline(
	_ context.Context,
	_ DetectionTimelineParams,
) ([]DetectionEvent, error) {
	return []DetectionEvent{
		{
			ID:        1,
			CreatedAt: time.Now().Add(-1 * time.Hour),
			DeviceID:  1,
			ImageUrl:  "http://localhost:8080/mock/car.jpg",
			Detections: []EventMeta{
				{
					ID:         123,
					Label:      "car",
					Confidence: 0.95,
				},
			},
		},
		{
			ID:        2,
			CreatedAt: time.Now().Add(-2 * time.Hour),
			DeviceID:  2,
			ImageUrl:  "http://localhost:8080/mock/person.jpg",
			Detections: []EventMeta{
				{
					ID:         234,
					Label:      "person",
					Confidence: 0.87,
				},
			},
		},
	}, nil
}
