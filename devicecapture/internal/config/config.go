package config

import (
	"log"
	"os"
	"strings"
)

type Config struct {
	MqttHost  string
	DbUrl     string
	VideoPath string
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
		db = "postgres://postgres:postgres@localhost:5432/postgres"
	}
	log.Printf("MQTT_HOST: %s", mh)
	return &Config{
		MqttHost:  mh,
		DbUrl:     db,
		VideoPath: "/videos",
	}
}
