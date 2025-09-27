package config

import "os"

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
