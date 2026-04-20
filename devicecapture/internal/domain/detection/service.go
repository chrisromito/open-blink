package detection

import (
	"context"
	"devicecapture/internal/config"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"time"
)

// ObjectDetectionService implements ObjectDetector
type ObjectDetectionService struct {
	url string
	//Config *config.Config
	//Deps   *domain.Deps
}

func NewObjectDetectionService(c *config.Config) ObjectDetectionService {
	return ObjectDetectionService{url: c.DetectionServiceUrl}
}

// DetectObjectsForImage ObjectDetectionService implements ObjectDetector
func (o ObjectDetectionService) DetectObjectsForImage(ctx context.Context, req Req) ([]Detection, error) {
	value := make(chan []Detection, 1)
	go func() {
		response, err := o.sendImage(req.Frame.Buf)
		if err != nil {
			value <- []Detection{}
		} else {
			value <- response
		}
	}()
	select {
	case resp := <-value:
		return resp, nil
	case <-time.After(1 * time.Second):
		return []Detection{}, nil
	}
}

func (o ObjectDetectionService) sendImage(imageBytes []byte) ([]Detection, error) {
	// Connect to the server
	conn, err := net.Dial("tcp", o.url)
	if err != nil {
		return nil, fmt.Errorf("failed to connect: %w", err)
	}
	defer conn.Close()

	fmt.Printf("Sending image to: %s\n", o.url)
	// Send image length (4 bytes, big-endian)
	lengthBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(lengthBytes, uint32(len(imageBytes)))
	if _, err := conn.Write(lengthBytes); err != nil {
		return nil, fmt.Errorf("failed to send length: %w", err)
	}

	// Send image bytes
	if _, err := conn.Write(imageBytes); err != nil {
		return nil, fmt.Errorf("failed to send image: %w", err)
	}
	fmt.Println("Receiving response")

	// Read response length (4 bytes)
	responseLengthBytes := make([]byte, 4)
	if _, err := io.ReadFull(conn, responseLengthBytes); err != nil {
		return nil, fmt.Errorf("failed to read response length: %w", err)
	}

	responseLength := binary.BigEndian.Uint32(responseLengthBytes)

	// Read response data
	responseBytes := make([]byte, responseLength)
	if _, err := io.ReadFull(conn, responseBytes); err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Parse JSON response
	var detections []Detection
	if err := json.Unmarshal(responseBytes, &detections); err != nil {
		return nil, fmt.Errorf("failed to parse JSON response: %w", err)
	}

	return detections, nil
}
