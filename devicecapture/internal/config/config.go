package config

import (
	"log"
	"os"
	"strings"
)

type Config struct {
	MqttHost            string
	DbUrl               string
	VideoPath           string
	DetectionServiceUrl string
}

func NewConfig() *Config {
	mh := os.Getenv("MQTT_HOST")
	if mh == "" {
		mh = "tcp://0.0.0.0:1883"
	} else {
		mh = strings.ReplaceAll(mh, "'", "")
	}
	db := os.Getenv("DB_URL")
	if db == "" {
		db = "postgres://postgres:postgres@localhost:5432/openblink"
	}
	log.Printf("MQTT_HOST: %s", mh)
	detectionService := os.Getenv("DETECTION_SERVICE_URL")
	if detectionService == "" {
		detectionService = "http://0.0.0.0:8000"
	}
	log.Printf("DETECTION_SERVICE_URL: %s", detectionService)
	return &Config{
		MqttHost:            mh,
		DbUrl:               db,
		VideoPath:           "/videos",
		DetectionServiceUrl: detectionService,
	}
}
