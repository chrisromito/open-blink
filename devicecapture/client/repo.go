package client

import (
	"fmt"
	"github.com/mattn/go-mjpeg"
	"image"
	"image/jpeg"
	"log"
	"net/http"
	"os"
	"time"
)

func GetImages(c *Client) error {
	// Add a longer timeout for the HTTP request
	client := &http.Client{
		Timeout: 60 * time.Second,
	}

	req, err := http.NewRequestWithContext(c.ctx, "GET", c.DeviceUrl, nil)
	if err != nil {
		return err
	}

	// Add headers that might help with streaming
	req.Header.Set("Accept", "multipart/x-mixed-replace")
	req.Header.Set("User-Agent", "GoMJPEGClient/1.0")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("status code is not OK: %v (%s)", resp.StatusCode, resp.Status)
	}

	dec, err := mjpeg.NewDecoderFromResponse(resp)
	if err != nil {
		return err
	}

	log.Printf("Starting MJPEG stream")
	frameCount := 0

	for {
		select {
		case <-c.ctx.Done():
			log.Printf("Stream stopped after %d frames", frameCount)
			return nil
		default:
			img, err := dec.Decode()
			if err != nil {
				log.Printf("Error decoding frame: %v", err)
				// Wait a bit and continue instead of returning
				time.Sleep(100 * time.Millisecond)
				continue
			}

			frameCount++
			log.Printf("Successfully decoded frame %d", frameCount)
			
			if err := WriteImage(c, img); err != nil {
				log.Printf("Error writing frame %d: %v", frameCount, err)
			}
		}
	}
}

func WriteImage(c *Client, i image.Image) error {
	fileName := fmt.Sprintf("videos/output-%s-%v.jpeg", c.DeviceId, time.Now().UnixMilli())
	log.Printf("Writing image to %s", fileName)
	f, err := os.Create(fileName)
	defer f.Close()
	if err != nil {
		return err
	}
	if err = jpeg.Encode(f, i, nil); err != nil {
		return err
	}
	return nil
}
