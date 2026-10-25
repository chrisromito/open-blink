package deviceimage

import (
	"devicecapture/internal/config"
	"devicecapture/internal/domain/detection"
	"testing"

	"devicecapture/internal/domain/devices"
)

func newTestReceiver() *Receiver {
	deps := newDeviceImageProcessorTestDeps()

	detector := &stubObjectDetector{
		detections: []detection.Detection{},
	}
	conf := config.Config{}
	processor := NewDeviceImageProcessor(&conf, deps, detector)
	return NewReceiver(processor)
}

func TestReceiverHandle(t *testing.T) {
	image := devices.DeviceImage{
		ID:        456,
		DeviceID:  123,
		ImagePath: "/tmp/device-123/image.jpg",
	}

	payload, err := Marshal(NewMessage(image))
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	var handled Message
	receiver := newTestReceiver()

	err = receiver.Handle(t.Context(), CreatedTopic(123), payload)
	if err != nil {
		t.Fatalf("Handle() error = %v", err)
	}

	if handled.Image.ID != image.ID {
		t.Fatalf("handled Image.ID = %d, want %d", handled.Image.ID, image.ID)
	}
}

func TestReceiverHandleRejectsTopicPayloadDeviceMismatch(t *testing.T) {
	image := devices.DeviceImage{
		ID:       456,
		DeviceID: 999,
	}

	payload, err := Marshal(NewMessage(image))
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	receiver := newTestReceiver()

	err = receiver.Handle(t.Context(), CreatedTopic(123), payload)
	if err == nil {
		t.Fatal("Handle() error = nil, want error")
	}
}

func TestReceiverHandleRejectsInvalidTopic(t *testing.T) {
	receiver := newTestReceiver()

	err := receiver.Handle(t.Context(), CreatedTopic(123), []byte("{}"))
	if err == nil {
		t.Fatal("Handle() error = nil, want error")
	}
}

func TestReceiverHandleRejectsInvalidPayload(t *testing.T) {
	receiver := newTestReceiver()

	err := receiver.Handle(t.Context(), CreatedTopic(123), []byte("{"))
	if err == nil {
		t.Fatal("Handle() error = nil, want error")
	}
}
