package config

import (
	"encoding/json"
	"log"
	"os"
)

var Config Configuration

type Configuration struct {
	ListenRange string `json:"listen_ip"`
	Port        int    `json:"port"`
}

func LoadConfig() {
	file, err := os.ReadFile("config.json")
	if err != nil {
		log.Fatalf("%s", err.Error())
	}

	err = json.Unmarshal(file, &Config)
	if err != nil {
		log.Fatalf("%s", err.Error())
	}
}
