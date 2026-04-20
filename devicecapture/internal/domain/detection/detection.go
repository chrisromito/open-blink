package detection

import (
	"context"
	"devicecapture/internal/domain/receiver"
)

type Req struct {
	DeviceId int64
	Frame    receiver.Frame
}

type Detection struct {
	Confidence float64 `json:"confidence"`
	Label      string  `json:"label"`
	BBox       BBox    `json:"bbox"`
}

type BBox struct {
	X1 float64 `json:"x1"`
	X2 float64 `json:"x2"`
	Y1 float64 `json:"y1"`
	Y2 float64 `json:"y2"`
}

type ObjectDetector interface {
	DetectObjectsForImage(ctx context.Context, req Req) ([]Detection, error)
}
