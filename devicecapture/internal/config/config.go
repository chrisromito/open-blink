package config

import (
	"os"
	"strconv"
	"strings"

	"devicecapture/internal/logger"
)

type Config struct {
	MqttHost            string
	MqttUser            string
	MqttPassword        string
	DbUrl               string
	VideoPath           string
	DetectionServiceUrl string
	ThisIp              string // Used to build URLs, ex for images
	ArchivePath         string // Used by archivist, empty = disabled
	ArchiveDays         int    // Used by archivist
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
	logger.Debug().Msgf("MQTT_HOST: %s", mh)
	detectionService := os.Getenv("DETECTION_SERVICE_URL")
	if detectionService == "" {
		detectionService = "http://0.0.0.0:8000"
	}
	logger.Debug().Msgf("DETECTION_SERVICE_URL: %s", detectionService)
	archivePath := os.Getenv("ARCHIVE_DIR")
	logger.Debug().Msgf("ARCHIVE_DIR: %s", archivePath)
	archiveValue := os.Getenv("ARCHIVE_DAYS")
	var archiveDays int
	if archiveValue == "" {
		archiveDays = 1
	} else {
		days, err := strconv.Atoi(archiveValue)
		if err != nil {
			archiveDays = 1
		} else {
			archiveDays = days
		}
	}
	logger.Debug().Str("archiveValue", archiveValue).Msgf("Archive days: %d", archiveDays)
	return &Config{
		MqttHost:            mh,
		MqttUser:            mu,
		MqttPassword:        mp,
		DbUrl:               db,
		VideoPath:           "/static/videos",
		DetectionServiceUrl: detectionService,
		ThisIp:              ip,
		ArchivePath:         archivePath,
		ArchiveDays:         archiveDays,
	}
}

func GetTestConfig(videoPath string, archivePath string) *Config {
	if videoPath == "" {
		videoPath = "/tmp/videos"
	}
	return &Config{
		MqttHost:            "",
		DbUrl:               "postgres://postgres:postgres@postgres:5432/test_openblink",
		VideoPath:           videoPath,
		DetectionServiceUrl: "http://0.0.0.0:4000",
		ThisIp:              "http://0.0.0.0:8000",
		ArchivePath:         archivePath,
		ArchiveDays:         1,
	}
}
