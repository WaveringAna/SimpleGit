package config

import (
	"encoding/json"
	"log"
	"os"
)

type Config struct {
	DevMode     bool   `json:"dev_mode"`
	Port        int    `json:"port"`
	DateFormat  string `json:"date_format"`
	MaxFileSize int64  `json:"max_file_size"`
}

var GlobalConfig Config

func Init() {
	data, err := os.ReadFile("config.json")
	if err != nil {
		log.Fatal("Failed to read config:", err)
	}
	if err := json.Unmarshal(data, &GlobalConfig); err != nil {
		log.Fatal("Failed to parse config:", err)
	}
}
