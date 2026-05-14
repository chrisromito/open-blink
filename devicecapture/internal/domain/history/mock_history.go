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

func (m *MockDetectionHistory) GetDetectionImagesByLabel(_ context.Context, params DetectionWithImageParams) ([]DetectionWithImage, error) {
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
