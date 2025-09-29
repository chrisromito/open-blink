package client

import (
	"errors"
	"fmt"
	"github.com/mattn/go-mjpeg"
	"image"
	"log"
	"net/http"
)

func GetImages(c Client) (image.Image, error) {
	req, err := http.NewRequestWithContext(c.ctx, "GET", c.DeviceUrl, nil)
	if err != nil {
		fmt.Printf("Error creating request: %v", err)
		return nil, err
	}
	// Send request
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		log.Fatalf("Status code is not OK: %v (%s)", resp.StatusCode, resp.Status)
		return nil, errors.New("status code is not OK")
	}

	dec, err := mjpeg.NewDecoderFromResponse(resp)
	if err != nil {
		return nil, err
	}
	return dec.Decode()
}
