package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	//"./client"
	"devicecapture/client"
)

func captureStream() {
	conf := NewConfig()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cameraClient := client.NewClient(ctx, "1", conf.DeviceUrl)
	err := cameraClient.Start()
	if err != nil {
		log.Fatalf("Error starting camera: %v", err)
	}
	//<-ctx.Done()
	cameraClient.Stop()
}

func main() {
	conf := NewConfig()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	c := client.NewClient(ctx, "1", conf.DeviceUrl)
	err := c.Start()
	if err != nil {
		log.Fatalf("Error starting camera: %v", err)
	}
	defer c.Stop()
	<-ctx.Done()
	fmt.Println("Recording stopped after 30 seconds.")
	//outputFileName := "output.mjpeg"
	//ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	//defer cancel()
	//
	//hc := http.Client{}
	//req, err := http.NewRequestWithContext(ctx, "GET", conf.DeviceUrl, nil)
	//if err != nil {
	//	fmt.Printf("Error creating request: %v", err)
	//	return
	//}
	//
	//// Send request
	//resp, err := hc.Do(req)
	//if err != nil {
	//	fmt.Printf("Error sending request: %v", err)
	//	return
	//}
	//defer resp.Body.Close()
	//if resp.StatusCode != http.StatusOK {
	//	log.Fatalf("Received non-OK status code: %d", resp.StatusCode)
	//	return
	//} else {
	//	fmt.Printf("Status: %s", resp.Status)
	//}
	//
	//// Create the output file
	//outFile, err := os.Create(outputFileName)
	//if err != nil {
	//	log.Fatalf("Error creating file: %v", err)
	//}
	//defer outFile.Close()
	//// Use a goroutine to copy the stream to the file.
	//// This ensures the main goroutine can continue and the context timeout will work correctly.
	//go func() {
	//	_, err := io.Copy(outFile, resp.Body)
	//	if err != nil && err != context.Canceled {
	//		fmt.Printf("Error writing to file: %v\n", err)
	//	}
	//}()
	//
	//// Wait for the context to finish.
	//// The copy will be cancelled when the timeout is reached.
	//<-ctx.Done()
	//fmt.Println("Recording stopped after 30 seconds.")
}

type Config struct {
	DeviceUrl string
}

func NewConfig() *Config {
	deviceUrl := os.Getenv("DEVICE_URL")
	if deviceUrl == "" {
		deviceUrl = "http://192.168.0.22/stream"
	}

	return &Config{
		DeviceUrl: deviceUrl,
	}
}
