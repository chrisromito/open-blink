package detections

import (
	"context"
	"fmt"
)

//
//type Handler interface {
//	HandleDetections(ctx context.Context, msg Message) error
//}
//
//type HandlerFunc func(ctx context.Context, msg Message) error
//
//func (f HandlerFunc) HandleDetections(ctx context.Context, msg Message) error {
//	return f(ctx, msg)
//}

type Receiver struct {
	//handler Handler
	Tracker *DetectionTracker
}

func NewReceiver(tracker *DetectionTracker) *Receiver {
	return &Receiver{
		Tracker: tracker,
	}
}

// Handle creates/updates DetectionEvents in response to Detection messages
func (r *Receiver) Handle(ctx context.Context, topic string, payload []byte) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	topicParts, err := ParseTopic(topic)
	if err != nil {
		return err
	}

	msg, jErr := Unmarshal(payload)
	if jErr != nil {
		return jErr
	}

	if msg.DeviceID != topicParts.DeviceID {
		return fmt.Errorf(
			"detections topic device id %d does not match payload device id %d",
			topicParts.DeviceID,
			msg.DeviceID,
		)
	}

	//return r.handler.HandleDetections(ctx, msg)
	return r.Tracker.ReceiveDetections(
		ctx,
		msg.DeviceID,
		msg.Detections,
	)
}

//
//func pgDetectionsToService(ds []devices.Detection) []detection.Detection {
//	var tmp []detection.Detection
//	for _, d := range ds {
//		tmp = append(tmp, DeviceDetectionToService(d))
//	}
//	return tmp
//}
//
//func DeviceDetectionToService(d devices.Detection) detection.Detection {
//	bbox := detection.BBox{
//		X1: d.Bbox[0][0],
//		Y1: d.Bbox[0][1],
//		X2: d.Bbox[1][0],
//		Y2: d.Bbox[1][1],
//	}
//	return detection.Detection{
//		Confidence: d.Confidence,
//		Label:      d.Label,
//		Bbox:       bbox,
//	}
//}
