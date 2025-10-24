package config

import (
	"encoding/json"
	"log"
	"os"
)

type Config struct {
	MqttHost string
	Devices  []DeviceConfig
}

func (c *Config) DeviceIds() []string {
	ids := make([]string, len(c.Devices))
	for i, d := range c.Devices {
		ids[i] = d.DeviceId
	}
	return ids
}

func (c *Config) DeviceMaps() map[string]DeviceConfig {
	m := make(map[string]DeviceConfig)
	for _, d := range c.Devices {
		m[d.DeviceId] = d
	}
	return m
}

type DeviceConfig struct {
	DeviceId  string `json:"device_id"`
	DeviceUrl string `json:"device_url"`
	Name      string `json:"name"`
}

func LoadDevices() ([]DeviceConfig, error) {
	mock := []DeviceConfig{
		DeviceConfig{
			DeviceId:  "mockdevice",
			DeviceUrl: "http://localhost:8080",
			Name:      "Mock Device",
		},
	}
	fileContent, err := os.ReadFile("./devices.json")
	if err != nil {
		return mock, err
	}
	var configs []DeviceConfig
	err = json.Unmarshal(fileContent, &configs)
	if err != nil {
		return mock, err
	}
	cs := append(configs, mock...)
	return cs, nil
}

func NewConfig() *Config {
	mh := os.Getenv("MQTT_HOST")
	if mh == "" {
		mh = "tcp://0.0.0.0:1883"
	}
	log.Printf("MQTT_HOST: %s", mh)
	devices, _ := LoadDevices()
	return &Config{
		MqttHost: mh,
		Devices:  devices,
	}
}
