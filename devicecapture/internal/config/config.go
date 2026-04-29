package config

import (
	"log"
	"os"
	"strings"
)

type Config struct {
	MqttHost            string
	MqttUser            string
	MqttPassword        string
	DbUrl               string
	VideoPath           string
	DetectionServiceUrl string
	ThisIp              string // Used to build URLs, ex for images
}

func NewConfig() *Config {
	mh := os.Getenv("MQTT_HOST")
	if mh == "" {
		mh = "tcp://0.0.0.0:1883"
	} else {
		mh = strings.ReplaceAll(mh, "'", "")
	}
	mu := os.Getenv("MQTT_USER")
	mp := os.Getenv("MQTT_PASSWORD")
	ip := os.Getenv("THIS_IP")
	if ip == "" {
		ip = "http://0.0.0.0:4000"
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
		MqttUser:            mu,
		MqttPassword:        mp,
		DbUrl:               db,
		VideoPath:           "/videos",
		DetectionServiceUrl: detectionService,
		ThisIp:              ip,
	}
}
