package detection

import (
	"context"

	"devicecapture/internal/domain/queue"
)

type Req struct {
	DeviceId int64
	Frame    queue.Frame
}

type Detection struct {
	Confidence float64 `json:"confidence"`
	Label      string  `json:"label"`
	Bbox       BBox    `json:"bbox"`
}

type BBox struct {
	X1 float64 `json:"x1"`
	X2 float64 `json:"x2"`
	Y1 float64 `json:"y1"`
	Y2 float64 `json:"y2"`
}

type BatchResponse struct {
	DeviceId   int64
	Detections []Detection
	Frame      queue.Frame
}

type ObjectDetector interface {
	DetectObjectsForImage(ctx context.Context, req Req) ([]Detection, error)
	BatchDetections(ctx context.Context, req Req) (BatchResponse, error)
}
