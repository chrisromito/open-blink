package client

import "C"
import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

type Client struct {
	ctx        context.Context
	DeviceId   string
	DeviceUrl  string
	Capturing  bool
	Connecting bool
	StartedAt  int64
	StoppedAt  int64
}

func NewClient(ctx context.Context, DeviceId string, DeviceUrl string) *Client {
	return &Client{
		ctx:        ctx,
		DeviceId:   DeviceId,
		DeviceUrl:  DeviceUrl,
		Capturing:  false,
		Connecting: false,
	}
}

func (c *Client) Start() error {
	if c.Connecting {
		return nil
	}
	c.Connecting = true
	c.StartedAt = time.Now().UnixMilli()
	return c.capture()
}

func (c *Client) capture() error {
	outputFileName := fmt.Sprintf("output-%s-%v.mjpeg", c.DeviceId, c.StartedAt)
	hc := http.Client{}
	// Start Connection
	req, err := http.NewRequestWithContext(c.ctx, "GET", c.DeviceUrl, nil)
	if err != nil {
		fmt.Printf("Error creating request: %v", err)
		return err
	}
	// Send request
	resp, err := hc.Do(req)
	if err != nil {
		fmt.Printf("Error sending request: %v", err)
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Fatalf("Received non-OK status code: %d", resp.StatusCode)
	}
	// Create the output file
	outFile, err := os.Create(outputFileName)
	if err != nil {
		log.Fatalf("Error creating file: %v", err)
		return err
	}
	defer outFile.Close()
	// Use a goroutine to copy the stream to the file.
	// This ensures the main goroutine can continue and the context timeout will work correctly.
	go func() {
		_, err := io.Copy(outFile, resp.Body)
		if err != nil && err != context.Canceled {
			fmt.Printf("Error writing to file: %v\n", err)
			return
		}
	}()
	// Wait for the context to finish.
	// The copy will be cancelled when the timeout is reached.
	<-c.ctx.Done()
	return nil
}

func (c *Client) Stop() {
	if !c.Connecting {
		return
	}
	c.Connecting = false
	c.StoppedAt = time.Now().UnixMilli()
}

func (c *Client) IsConnected() bool {
	return c.Connecting
}

func (c *Client) IsCapturing() bool {
	return c.Capturing
}
