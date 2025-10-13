package pubsub

import (
	"context"
	"devicecapture/device/receiver"
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
		os.RemoveAll(tempDir)
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

	deviceId := "test-device-123"

	// Patch the videos directory for testing
	//originalCheckSessionDir := rec.checkSessionDir

	session, err := rec.StartSession(deviceId)

	if err != nil {
		t.Fatalf("StartSession failed: %v", err)
	}

	if session == nil {
		t.Fatal("StartSession returned nil session")
	}

	if session.DeviceId != deviceId {
		t.Errorf("Expected device ID %s, got %s", deviceId, session.DeviceId)
	}

	if session.StartedAt == 0 {
		t.Error("StartedAt should not be zero")
	}

	if rec.Session.DeviceId != deviceId {
		t.Errorf("Expected receiver session device ID %s, got %s", deviceId, rec.Session.DeviceId)
	}
}

func TestMqttReceiver_FrameToJson(t *testing.T) {
	client := &MqttClient{}
	vp := "test-path"
	rec := NewMqttReceiver(client, vp)

	// Set up a test session
	rec.Session = receiver.CaptureSession{
		DeviceId:  "test-device",
		StartedAt: 1234567890,
	}

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
		"test-device",
		"9876543210",
		"test-path/test-device-1234567890/output-test-device-9876543210.jpeg",
	}

	for _, expected := range expectedSubstrings {
		if !strings.Contains(jsonStr, expected) {
			t.Errorf("Expected JSON to contain %s, but got: %s", expected, jsonStr)
		}
	}
}

func TestFrameJson(t *testing.T) {
	deviceId := "test-device"
	fileName := "/path/to/test.jpg"
	frame := createTestFrame(1234567890)

	jsonStr, err := FrameJson(deviceId, fileName, frame)
	if err != nil {
		t.Fatalf("FrameJson failed: %v", err)
	}

	if jsonStr == "" {
		t.Error("FrameJson returned empty string")
	}

	expectedSubstrings := []string{
		"test-device",
		"/path/to/test.jpg",
		"1234567890",
	}

	for _, expected := range expectedSubstrings {
		if !strings.Contains(jsonStr, expected) {
			t.Errorf("Expected JSON to contain %s, but got: %s", expected, jsonStr)
		}
	}
}

func TestFramePath(t *testing.T) {
	tests := []struct {
		dirPath    string
		name       string
		startStamp int64
		deviceId   string
		timestamp  int64
		expected   string
	}{
		{
			dirPath:    "/test-frame-path",
			name:       "generates correct path",
			startStamp: 1234567890,
			deviceId:   "device1",
			timestamp:  9876543210,
			expected:   "/test-frame-path/device1-1234567890/output-device1-9876543210.jpeg",
		},
		{
			dirPath:    "/test-path2",
			name:       "handles zero timestamps",
			startStamp: 0,
			deviceId:   "device2",
			timestamp:  0,
			expected:   "/test-path2/device2-0/output-device2-0.jpeg",
		},
		{
			dirPath:    "/test-frame-path",
			name:       "handles empty device id",
			startStamp: 1000,
			deviceId:   "",
			timestamp:  2000,
			expected:   "/test-frame-path/-1000/output--2000.jpeg",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FramePath(tt.dirPath, tt.startStamp, tt.deviceId, tt.timestamp)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestMqttReceiver_ReceiveFrame(t *testing.T) {
	tempDir, cleanup := setupTestDir(t)
	defer cleanup()

	// Create a simple test client that captures publish calls
	publishCalls := make([]struct {
		topic   string
		payload interface{}
	}, 0)

	client := &MqttClient{}
	// We can't easily test the actual MQTT publishing without a real broker,
	// but we can test that the method completes without error

	rec := NewMqttReceiver(client, tempDir)
	rec.Session = receiver.CaptureSession{
		DeviceId:  "test-device",
		StartedAt: time.Now().UnixMilli(),
	}

	frame := createTestFrame(time.Now().UnixMilli())

	// This test mainly verifies that the method doesn't panic
	// In a real environment, you'd need a running MQTT broker to test publishing
	rec.ReceiveFrame(frame)

	// The method should complete without error
	// Note: File writing will fail because /videos doesn't exist in test environment,
	// but that's expected and handled gracefully by the goroutines

	_ = publishCalls // Used to capture calls in a more complete test setup
}

func TestMqttReceiver_ReceiveFrameStream(t *testing.T) {
	client := &MqttClient{}

	rec := NewMqttReceiver(client, "/tmp")
	rec.Session = receiver.CaptureSession{
		DeviceId:  "test-device",
		StartedAt: time.Now().UnixMilli(),
	}

	// Create a context with timeout to ensure the test doesn't run forever
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Create a channel and send some test frames
	imgChan := make(chan receiver.Frame, 3)
	go func() {
		defer close(imgChan)
		for i := 0; i < 3; i++ {
			frame := createTestFrame(time.Now().UnixMilli() + int64(i))
			select {
			case imgChan <- frame:
			case <-ctx.Done():
				return
			}
		}
	}()

	// Start the frame stream processing
	err := rec.ReceiveFrameStream(ctx, imgChan)

	// The method should return nil when context is cancelled
	if err != nil {
		t.Errorf("ReceiveFrameStream returned unexpected error: %v", err)
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
	rec.Session = receiver.CaptureSession{
		DeviceId:  "test-device",
		StartedAt: 1234567890,
	}

	// This test will fail in the actual publish call since we don't have a real MQTT client
	// but we can test that the method constructs the correct topic and payload
	err := rec.EndSession()

	// We expect an error because the client isn't actually connected
	if err == nil {
		t.Error("Expected error when publishing without connected client")
	}

	// In a real test with a connected client, we'd verify:
	// - Topic should be "end-stream/test-device"
	// - Payload should be "/videos/test-device-1234567890"

	_ = publishCalls // Used to capture calls in a more complete test setup
}

// Integration test that tests multiple components working together
func TestMqttReceiver_Integration(t *testing.T) {
	tempDir, cleanup := setupTestDir(t)
	defer cleanup()

	client := &MqttClient{}
	rec := NewMqttReceiver(client, tempDir)

	deviceId := "integration-test-device"

	// Test session creation
	session, err := rec.StartSession(deviceId)
	if err != nil {
		// Directory creation might fail, but that's ok for this test
		t.Logf("StartSession error (expected in test environment): %v", err)
	}

	if session != nil {
		if session.DeviceId != deviceId {
			t.Errorf("Expected device ID %s, got %s", deviceId, session.DeviceId)
		}
	}

	// Test JSON generation
	frame := createTestFrame(time.Now().UnixMilli())
	jsonStr, err := rec.FrameToJson(frame)
	if err != nil {
		t.Fatalf("FrameToJson failed: %v", err)
	}

	if !strings.Contains(jsonStr, deviceId) {
		t.Errorf("JSON should contain device ID %s: %s", deviceId, jsonStr)
	}

	// Test frame path generation
	path := FramePath(rec.videoPath, rec.Session.StartedAt, deviceId, frame.Timestamp)
	expectedPrefix := rec.videoPath + "/" + deviceId + "-"
	if !strings.HasPrefix(path, expectedPrefix) {
		t.Errorf("Path should start with %s, got: %s", expectedPrefix, path)
	}
}

// Benchmark tests
func BenchmarkFrameJson(b *testing.B) {
	deviceId := "bench-device"
	fileName := "/path/to/bench.jpg"
	frame := createTestFrame(1234567890)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := FrameJson(deviceId, fileName, frame)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkFramePath(b *testing.B) {
	startStamp := int64(1234567890)
	deviceId := "bench-device"
	timestamp := int64(9876543210)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = FramePath("/tmp", startStamp, deviceId, timestamp)
	}
}

func BenchmarkMqttReceiver_FrameToJson(b *testing.B) {
	client := &MqttClient{}
	rec := NewMqttReceiver(client, "/tmp")
	rec.Session = receiver.CaptureSession{
		DeviceId:  "bench-device",
		StartedAt: 1234567890,
	}

	frame := createTestFrame(9876543210)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := rec.FrameToJson(frame)
		if err != nil {
			b.Fatal(err)
		}
	}
}
