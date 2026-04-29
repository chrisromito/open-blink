package camera

import (
	"bytes"
	"devicecapture/internal/domain/receiver"
	"github.com/mattn/go-mjpeg"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"

	"context"
	"errors"
	"image"
	"image/color"
	"image/jpeg"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestApi_Ping_Success(t *testing.T) {
	server := newTestServer(t.Context())
	defer server.Close()

	api := NewApi("test-domain", server.URL)

	if !api.Ping() {
		t.Error("Expected Ping() to return true for successful response")
	}
}

func TestApi_Ping_Failure_NetworkError(t *testing.T) {
	// Use an invalid URL to simulate network errors
	api := NewApi("test-domain", "http://invalid-url-that-does-not-exist:99999")

	if api.Ping() {
		t.Error("Expected Ping() to return false for network error")
	}
}

func TestApi_StreamFrames_Success(t *testing.T) {
	server := newTestServer(t.Context())
	defer server.Close()

	api := NewApi("test-domain", server.URL)
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
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
	api := NewApi("test-domain", "http://invalid-url-that-does-not-exist:99999")
	imgChan := make(chan receiver.Frame, 1)

	err := api.StreamFrames(t.Context(), imgChan)
	if err == nil {
		t.Error("Expected StreamFrames to return error for network error")
	}
}

func TestApi_StreamFrames_ContextCancellation(t *testing.T) {
	// Create a server that streams indefinitely
	server := newTestServer(t.Context())
	defer server.Close()

	api := NewApi("test-domain", server.URL)
	ctx, cancel := context.WithCancel(t.Context())
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
	server := newTestServer(t.Context())
	defer server.Close()

	api := NewApi("test-domain", server.URL)
	ctx, cancel := context.WithTimeout(t.Context(), 1*time.Second)
	defer cancel()

	// Create a small channel that will fill up quickly
	imgChan := make(chan receiver.Frame, 1)

	err := api.StreamFrames(ctx, imgChan)
	// Should complete without hanging, even if the channel gets full
	if err != nil && !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("Unexpected error: %v", err)
	}
}

func newTestServer(ctx context.Context) *httptest.Server {

	testProxy := func(stream *mjpeg.Stream) {
		for {
			time.Sleep(200 * time.Millisecond)
			if ctx.Err() != nil {
				return
			}
			err := stream.Update(getTestImage())
			if err != nil {
				return
			}
		}
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/ping" || r.URL.Path == "" || r.URL.Path == "/" {
			w.WriteHeader(http.StatusOK)
		}
		if r.URL.Path == "/stream" {
			stream := mjpeg.NewStreamWithInterval(200 * time.Millisecond)
			defer func(stream *mjpeg.Stream) {
				_ = stream.Close()
			}(stream)
			streamErr := stream.Update(getTestImage())
			if streamErr != nil {
				w.WriteHeader(http.StatusInternalServerError)
			} else {
				go testProxy(stream)
				stream.ServeHTTP(w, r)
			}
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))

	return server
}

func getTestImage() []byte {
	img := image.NewRGBA(image.Rect(0, 0, 250, 250))
	timestring := time.Now().Format("15:04:05")
	addLabel(img, 10, 10, timestring)
	var buf bytes.Buffer
	err := jpeg.Encode(&buf, img, nil)
	if err != nil {
		log.Printf("getTestImage -> err: %v", err)
		return nil
	}
	return buf.Bytes()
}

func addLabel(img *image.RGBA, x, y int, label string) {
	col := color.RGBA{R: 200, G: 100, A: 255}
	point := fixed.Point26_6{X: fixed.I(x), Y: fixed.I(y)}

	d := &font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(col),
		Face: basicfont.Face7x13,
		Dot:  point,
	}
	d.DrawString(label)
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

	api := NewApi("test-domain", server.URL)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		api.Ping()
	}
}
