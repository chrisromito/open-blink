package detections

import "testing"

func TestTopic(t *testing.T) {
	got := Topic(123)
	want := "open-blink/devices/123/detections/created"

	if got != want {
		t.Fatalf("Topic() = %q, want %q", got, want)
	}
}

func TestSubscribeTopic(t *testing.T) {
	got := SubscribeTopic()
	want := "open-blink/devices/+/detections/created"

	if got != want {
		t.Fatalf("SubscribeTopic() = %q, want %q", got, want)
	}
}

func TestParseTopic(t *testing.T) {
	got, err := ParseTopic("open-blink/devices/123/detections/created")
	if err != nil {
		t.Fatalf("ParseTopic() error = %v", err)
	}

	if got.DeviceID != 123 {
		t.Fatalf("DeviceID = %d, want 123", got.DeviceID)
	}
}

func TestParseTopicRejectsInvalidTopic(t *testing.T) {
	_, err := ParseTopic("open-blink/detections/123")
	if err == nil {
		t.Fatal("ParseTopic() error = nil, want error")
	}
}
