package detection

import (
	"context"
	"encoding/json"
)

// MockDetectionService implements ObjectDetector
type MockDetectionService struct {
}

// DetectObjectsForImage MockDetectionService implements ObjectDetector
func (o MockDetectionService) DetectObjectsForImage(ctx context.Context, req Req) ([]Detection, error) {
	value := "[{\"confidence\": 0.31383, \"label\": \"train\", \"bbox\": {\"x1\": 2423.35669, \"x2\": 3278.05127, \"y1\": 1170.4054, \"y2\": 1772.38293}}]"
	var detections []Detection
	err := json.Unmarshal([]byte(value), &detections)
	if err != nil {
		return detections, err
	}
	return detections, nil
}
