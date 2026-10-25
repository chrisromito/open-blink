package deviceimage

import (
	"testing"

	"devicecapture/internal/domain/devices"
)

func TestMarshalUnmarshal(t *testing.T) {
	image := devices.DeviceImage{
		ID:        456,
		DeviceID:  123,
		ImagePath: "/tmp/device-123/image.jpg",
	}

	payload, err := Marshal(NewMessage(image))
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	got, err := Unmarshal(payload)
	if err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if got.Image.ID != image.ID {
		t.Fatalf("Image.ID = %d, want %d", got.Image.ID, image.ID)
	}

	if got.Image.DeviceID != image.DeviceID {
		t.Fatalf("Image.DeviceID = %d, want %d", got.Image.DeviceID, image.DeviceID)
	}

	if got.Image.ImagePath != image.ImagePath {
		t.Fatalf("Image.ImagePath = %q, want %q", got.Image.ImagePath, image.ImagePath)
	}
}

func TestUnmarshalRejectsInvalidJSON(t *testing.T) {
	_, err := Unmarshal([]byte("{"))
	if err == nil {
		t.Fatal("Unmarshal() error = nil, want error")
	}
}
