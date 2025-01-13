//config/config.go

package config

import (
	"encoding/json"
	"log"
	"os"

	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	DevMode     bool   `json:"dev_mode"`
	Port        int    `json:"port"`
	DateFormat  string `json:"date_format"`
	MaxFileSize int64  `json:"max_file_size"`
	DataDir     string `json:"data_dir"`
	JWTSecret   string `json:"jwt_secret"`
	Domain      string `json:"domain"`
	SSHPort     int    `json:"ssh_port" envconfig:"SSH_PORT" default:"2222"`
	SSHKeyPath  string `json:"ssh_key_path" envconfig:"SSH_KEY_PATH" default:"ssh/host_key"`
}

var GlobalConfig Config

func Init() {
    // Load JSON config first
    data, err := os.ReadFile("config.json")
    if err != nil {
        log.Fatal("Failed to read config:", err)
    }
    if err := json.Unmarshal(data, &GlobalConfig); err != nil {
        log.Fatal("Failed to parse config:", err)
    }

    // Override with environment variables
    if err := envconfig.Process("", &GlobalConfig); err != nil {
        log.Fatal("Failed to process environment variables:", err)
    }

    // Set default domain if not specified
    if GlobalConfig.Domain == "" {
        GlobalConfig.Domain = "localhost"
    }

    // Log the configuration
    log.Printf("Configuration loaded: HTTP port: %d, SSH port: %d",
        GlobalConfig.Port,
        GlobalConfig.SSHPort)
}
