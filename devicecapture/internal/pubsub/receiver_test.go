package pubsub

import (
	"devicecapture/internal/config"
	"devicecapture/internal/domain/receiver"
	"github.com/stretchr/testify/assert"
	"image"
	"image/color"
	"os"
	"strings"
	"testing"
)

func getTestConfig(videoPath string) *config.Config {
	if videoPath == "" {
		videoPath = "/tmp/videos"
	}
	return &config.Config{
		MqttHost:            "",
		DbUrl:               "postgres://postgres:postgres@postgres:5432/test_openblink",
		VideoPath:           videoPath,
		DetectionServiceUrl: "http://0.0.0.0:4000",
		ThisIp:              "http://0.0.0.0:8000",
	}
}

// createTestImage creates a simple test image
func createTestImage() image.Image {
	img := image.NewRGBA(image.Rect(0, 0, 100, 100))
	for y := 0; y < 100; y++ {
		for x := 0; x < 100; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x), G: uint8(y), B: 255, A: 255})
		}
	}
	return img
}

// createTestFrame creates a test frame with the given timestamp
func createTestFrame(timestamp int64) receiver.Frame {
	return receiver.Frame{
		Buf:       []byte("test-frame-data"),
		Image:     createTestImage(),
		Timestamp: timestamp,
	}
}

// setupTestDir creates a temporary directory for testing and returns cleanup function
func setupTestDir(t *testing.T) (string, func()) {
	tempDir, err := os.MkdirTemp("", "mqtt_receiver_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	// Change the global videos directory for testing
	originalVideosDir := "/videos"

	return tempDir, func() {
		_ = os.RemoveAll(tempDir)
		// Restore original if needed
		_ = originalVideosDir
	}
}

func TestNewMqttReceiver(t *testing.T) {
	tests := []struct {
		name   string
		client *MqttClient
	}{
		{
			name:   "creates receiver with valid client",
			client: &MqttClient{},
		},
		{
			name:   "creates receiver with nil client",
			client: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := NewMqttReceiver(tt.client, getTestConfig(""))

			if rec == nil {
				t.Fatal("NewMqttReceiver returned nil")
			}

			if rec.client != tt.client {
				t.Errorf("Expected client %v, got %v", tt.client, rec.client)
			}
		})
	}
}

func TestMqttReceiver_StartSession(t *testing.T) {
	tempDir, cleanup := setupTestDir(t)
	defer cleanup()
	a := assert.New(t)

	// Create a test client (doesn't need to be connected for this test)
	client := &MqttClient{}
	rec := NewMqttReceiver(client, getTestConfig(tempDir))

	deviceId := "test-domain-123"

	// Patch the videos directory for testing
	//originalCheckSessionDir := rec.checkSessionDir

	session, err := rec.StartSession(deviceId)
	a.NoError(err)
	a.NotNil(session)
	a.Equalf(session.DeviceID, deviceId, "Expected receiver session domain ID %s, got %s", deviceId, session.DeviceID)
	a.NotEqual(session.StartedAt, 0, "StartedAt should not be zero")
}

func TestFrameJson(t *testing.T) {
	deviceId := "test-domain"
	fileName := "/path/to/test.jpg"
	frame := createTestFrame(1234567890)

	jsonStr, err := receiver.FrameJson("", deviceId, fileName, frame)
	if err != nil {
		t.Fatalf("FrameJson failed: %v", err)
	}

	if jsonStr == "" {
		t.Error("FrameJson returned empty string")
	}

	expectedSubstrings := []string{
		"test-domain",
		"/path/to/test.jpg",
		"1234567890",
	}

	for _, expected := range expectedSubstrings {
		if !strings.Contains(jsonStr, expected) {
			t.Errorf("Expected JSON to contain %s, but got: %s", expected, jsonStr)
		}
	}
}

// Benchmark tests
func BenchmarkFrameJson(b *testing.B) {
	deviceId := "bench-domain"
	fileName := "/path/to/bench.jpg"
	frame := createTestFrame(1234567890)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := receiver.FrameJson("", deviceId, fileName, frame)
		if err != nil {
			b.Fatal(err)
		}
	}
}
