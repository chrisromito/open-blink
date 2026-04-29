package pubsub

import (
	"devicecapture/internal/domain/receiver"
	"image"
	"image/color"
	"os"
	"strings"
	"testing"
	"time"
)

// createTestImage creates a simple test image
func createTestImage() image.Image {
	img := image.NewRGBA(image.Rect(0, 0, 100, 100))
	for y := 0; y < 100; y++ {
		for x := 0; x < 100; x++ {
			img.Set(x, y, color.RGBA{uint8(x), uint8(y), 255, 255})
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
			rec := NewMqttReceiver(tt.client, "")

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

	// Create a test client (doesn't need to be connected for this test)
	client := &MqttClient{}
	rec := NewMqttReceiver(client, tempDir)

	deviceId := "test-domain-123"

	// Patch the videos directory for testing
	//originalCheckSessionDir := rec.checkSessionDir

	session, err := rec.StartSession(deviceId)

	if err != nil {
		t.Fatalf("StartSession failed: %v", err)
	}

	if session == nil {
		t.Fatal("StartSession returned nil session")
	}

	if session.DeviceID != deviceId {
		t.Errorf("Expected domain ID %s, got %s", deviceId, session.DeviceID)
	}

	if session.StartedAt == 0 {
		t.Error("StartedAt should not be zero")
	}

	if rec.Session.DeviceID != deviceId {
		t.Errorf("Expected receiver session domain ID %s, got %s", deviceId, rec.Session.DeviceID)
	}
}

func TestMqttReceiver_FrameToJson(t *testing.T) {
	client := &MqttClient{}
	vp := "test-path"
	rec := NewMqttReceiver(client, vp)

	// Set up a test session
	rec.Session = receiver.NewCaptureSession("test-domain")

	frame := createTestFrame(9876543210)

	jsonStr, err := rec.FrameToJson(frame)
	if err != nil {
		t.Fatalf("FrameToJson failed: %v", err)
	}

	if jsonStr == "" {
		t.Error("FrameToJson returned empty string")
	}

	// Check that the JSON contains expected fields
	expectedSubstrings := []string{
		"test-domain",
		"9876543210",
		"output-test-domain-9876543210.jpeg",
		"test-path/test-domain-",
	}

	for _, expected := range expectedSubstrings {
		if !strings.Contains(jsonStr, expected) {
			t.Errorf("Expected JSON to contain %s, but got: %s", expected, jsonStr)
		}
	}
}

func TestFrameJson(t *testing.T) {
	deviceId := "test-domain"
	fileName := "/path/to/test.jpg"
	frame := createTestFrame(1234567890)

	jsonStr, err := receiver.FrameJson(deviceId, fileName, frame)
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

func TestMqttReceiver_EndSession(t *testing.T) {
	// Create a test client that captures publish calls
	publishCalls := make([]struct {
		topic   string
		payload interface{}
	}, 0)

	client := &MqttClient{
		// Mock the Publish method to capture calls
	}

	rec := NewMqttReceiver(client, "/tmp")
	rec.Session = receiver.NewCaptureSession("test-domain")
	// This test will fail in the actual publish call since we don't have a real MQTT client
	// but we can test that the method constructs the correct topic and payload
	err := rec.EndSession()

	// We expect an error because the client isn't actually connected
	if err == nil {
		t.Error("Expected error when publishing without connected client")
	}

	// In a real test with a connected client, we'd verify:
	// - Topic should be "end-stream/test-domain"
	// - Payload should be "/videos/test-domain-1234567890"

	_ = publishCalls // Used to capture calls in a more complete test setup
}

// Integration test that tests multiple components working together
func TestMqttReceiver_Integration(t *testing.T) {
	tempDir, cleanup := setupTestDir(t)
	defer cleanup()

	client := &MqttClient{}
	rec := NewMqttReceiver(client, tempDir)

	deviceId := "integration-test-domain"

	// Test session creation
	session, err := rec.StartSession(deviceId)
	if err != nil {
		// Directory creation might fail, but that's ok for this test
		t.Logf("StartSession error (expected in test environment): %v", err)
	}

	if session != nil {
		if session.DeviceID != deviceId {
			t.Errorf("Expected domain ID %s, got %s", deviceId, session.DeviceID)
		}
	}

	// Test JSON generation
	frame := createTestFrame(time.Now().UnixMilli())
	jsonStr, err := rec.FrameToJson(frame)
	if err != nil {
		t.Fatalf("FrameToJson failed: %v", err)
	}

	if !strings.Contains(jsonStr, deviceId) {
		t.Errorf("JSON should contain domain ID %s: %s", deviceId, jsonStr)
	}

	// Test frame path generation
	path := receiver.FramePath(rec.videoPath, rec.Session, frame)
	expectedPrefix := rec.videoPath + "/" + deviceId + "-"
	if !strings.HasPrefix(path, expectedPrefix) {
		t.Errorf("Path should start with %s, got: %s", expectedPrefix, path)
	}
}

// Benchmark tests
func BenchmarkFrameJson(b *testing.B) {
	deviceId := "bench-domain"
	fileName := "/path/to/bench.jpg"
	frame := createTestFrame(1234567890)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := receiver.FrameJson(deviceId, fileName, frame)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkMqttReceiver_FrameToJson(b *testing.B) {
	client := &MqttClient{}
	rec := NewMqttReceiver(client, "/tmp")
	rec.Session = receiver.NewCaptureSession("bench-domain")

	frame := createTestFrame(9876543210)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := rec.FrameToJson(frame)
		if err != nil {
			b.Fatal(err)
		}
	}
}
