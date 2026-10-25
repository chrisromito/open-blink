package deviceimage

import (
	"context"
	"testing"

	"devicecapture/internal/domain/devices"
)

type fakePublishClient struct {
	topic   string
	payload any
	err     error
}

func (f *fakePublishClient) Publish(topic string, payload any) error {
	f.topic = topic
	f.payload = payload
	return f.err
}

func TestPublisherPublishCreated(t *testing.T) {
	client := &fakePublishClient{}
	publisher := NewPublisher(client)

	image := devices.DeviceImage{
		ID:        456,
		DeviceID:  123,
		ImagePath: "/tmp/device-123/image.jpg",
	}

	err := publisher.PublishCreated(t.Context(), image)
	if err != nil {
		t.Fatalf("PublishCreated() error = %v", err)
	}

	wantTopic := "open-blink/devices/123/images/created"
	if client.topic != wantTopic {
		t.Fatalf("topic = %q, want %q", client.topic, wantTopic)
	}

	payload, ok := client.payload.([]byte)
	if !ok {
		t.Fatalf("payload type = %T, want []byte", client.payload)
	}

	msg, err := Unmarshal(payload)
	if err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if msg.Image.ID != image.ID {
		t.Fatalf("Image.ID = %d, want %d", msg.Image.ID, image.ID)
	}
}

func TestPublisherPublishCreatedReturnsContextError(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	client := &fakePublishClient{}
	publisher := NewPublisher(client)

	err := publisher.PublishCreated(ctx, devices.DeviceImage{
		ID:       456,
		DeviceID: 123,
	})
	if err == nil {
		t.Fatal("PublishCreated() error = nil, want error")
	}
}
