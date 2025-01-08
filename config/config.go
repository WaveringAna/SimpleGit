//config/config.go

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
	DataDir     string `json:"data_dir"`
	JWTSecret   string `json:"jwt_secret"`
	Domain      string `json:"domain"`
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

	// Set default domain if not specified
	if GlobalConfig.Domain == "" {
		GlobalConfig.Domain = "localhost"
	}
}
