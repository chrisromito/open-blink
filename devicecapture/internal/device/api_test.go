package device

import (
	"devicecapture/internal/device/receiver"

	"context"
	"errors"
	"image"
	"image/color"
	"image/jpeg"
	"io"
	"log"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestApi_Ping_Success(t *testing.T) {
	server := newTestServer()
	defer server.Close()

	api := NewApi("test-device", server.URL)

	if !api.Ping() {
		t.Error("Expected Ping() to return true for successful response")
	}
}

func TestApi_Ping_Failure_NetworkError(t *testing.T) {
	// Use an invalid URL to simulate network errors
	api := NewApi("test-device", "http://invalid-url-that-does-not-exist:99999")

	if api.Ping() {
		t.Error("Expected Ping() to return false for network error")
	}
}

func TestApi_StreamFrames_Success(t *testing.T) {
	server := newTestServer()
	defer server.Close()

	api := NewApi("test-device", server.URL)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	imgChan := make(chan receiver.Frame, 10)

	go func() {
		err := api.StreamFrames(ctx, imgChan)
		if err != nil && !errors.Is(err, context.DeadlineExceeded) {
			t.Errorf("Unexpected error from StreamFrames: %v", err)
		}
	}()

	// Wait for at least one frame or timeout
	select {
	case frame := <-imgChan:
		if frame.Image == nil {
			t.Error("Expected frame to contain an image")
		}
		if frame.Timestamp == 0 {
			t.Error("Expected frame to have a timestamp")
		}
	case <-time.After(3 * time.Second):
		t.Error("Timeout waiting for frame from stream")
	}
}

func TestApi_StreamFrames_NetworkError(t *testing.T) {
	api := NewApi("test-device", "http://invalid-url-that-does-not-exist:99999")
	ctx := context.Background()
	imgChan := make(chan receiver.Frame, 1)

	err := api.StreamFrames(ctx, imgChan)
	if err == nil {
		t.Error("Expected StreamFrames to return error for network error")
	}
}

func TestApi_StreamFrames_ContextCancellation(t *testing.T) {
	// Create a server that streams indefinitely
	server := newTestServer()
	defer server.Close()

	api := NewApi("test-device", server.URL)
	ctx, cancel := context.WithCancel(context.Background())
	imgChan := make(chan receiver.Frame, 10)

	done := make(chan error, 1)
	go func() {
		done <- api.StreamFrames(ctx, imgChan)
	}()

	// Cancel context after a short time
	time.Sleep(100 * time.Millisecond)
	cancel()

	// Wait for StreamFrames to return
	select {
	case err := <-done:
		// Should return nil when context is canceled normally
		if err != nil {
			t.Errorf("Expected StreamFrames to return nil on context cancellation, got: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Error("StreamFrames did not return after context cancellation")
	}
}

func TestApi_StreamFrames_ChannelFull(t *testing.T) {
	// This test is harder to verify precisely due to the non-blocking nature,
	// but we can at least ensure it doesn't hang
	server := newTestServer()
	defer server.Close()

	api := NewApi("test-device", server.URL)
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// Create a small channel that will fill up quickly
	imgChan := make(chan receiver.Frame, 1)

	err := api.StreamFrames(ctx, imgChan)
	// Should complete without hanging, even if the channel gets full
	if err != nil && !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("Unexpected error: %v", err)
	}
}

func newTestServer() *httptest.Server {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/ping" {
			w.WriteHeader(http.StatusOK)
		}
		if r.URL.Path == "/stream" {
			w.Header().Set("Content-Type", "multipart/x-mixed-replace; boundary=frame")

			boundary := "\r\n--frame\r\nContent-Type: image/jpeg\r\n\r\n"
			for range 10 {
				img := createTestImage()

				n, err := io.WriteString(w, boundary)
				if err != nil || n != len(boundary) {
					log.Printf("TestServer -> Error writing boundary: %v", err)
					return
				}

				err = jpeg.Encode(w, img, nil)
				if err != nil {
					log.Printf("TestServer -> Error encoding image: %v", err)
					return
				}

				n, err = io.WriteString(w, "\r\n")
				if err != nil || n != 2 {
					log.Printf("TestServer -> Error writing boundary: %v", err)
					return
				}
			}
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))

	return server
}

func createTestImage() image.Image {
	img := image.NewGray(image.Rect(0, 0, 100, 100))
	for i := 0; i < 100; i++ {
		for j := 0; j < 100; j++ {
			n := rand.Intn(256)
			gray := color.Gray{Y: uint8(n)}
			img.SetGray(i, j, gray)
		}
	}
	return img
}

// Benchmark tests
func BenchmarkNewFrame(b *testing.B) {
	img := image.NewRGBA(image.Rect(0, 0, 100, 100))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		NewFrame(img.Pix)
	}
}

func BenchmarkApi_Ping(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	api := NewApi("test-device", server.URL)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		api.Ping()
	}
}
