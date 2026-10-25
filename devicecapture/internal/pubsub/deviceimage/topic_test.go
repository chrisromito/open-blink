package deviceimage

import "testing"

func TestTopic(t *testing.T) {
	got := CreatedTopic(123)
	want := "open-blink/devices/123/images/created"

	if got != want {
		t.Fatalf("CreatedTopic() = %q, want %q", got, want)
	}
}

func TestSubscribeTopic(t *testing.T) {
	got := SubscribeCreatedTopic()
	want := "open-blink/devices/+/images/created"

	if got != want {
		t.Fatalf("SubscribeCreatedTopic() = %q, want %q", got, want)
	}
}

func TestParseTopic(t *testing.T) {
	got, err := ParseTopic("open-blink/devices/123/images/created")
	if err != nil {
		t.Fatalf("ParseTopic() error = %v", err)
	}

	if got.DeviceID != 123 {
		t.Fatalf("DeviceID = %d, want 123", got.DeviceID)
	}
}

func TestParseTopicRejectsInvalidTopic(t *testing.T) {
	_, err := ParseTopic("open-blink/device-image/123")
	if err == nil {
		t.Fatal("ParseTopic() error = nil, want error")
	}
}

func TestParseTopicRejectsInvalidDeviceID(t *testing.T) {
	_, err := ParseTopic("open-blink/devices/not-an-id/images/created")
	if err == nil {
		t.Fatal("ParseTopic() error = nil, want error")
	}
}
