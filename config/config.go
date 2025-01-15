package config

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"

	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	DevMode     bool   `json:"dev_mode" envconfig:"DEV_MODE"`
	Port        int    `json:"port" envconfig:"PORT" default:"3000"`
	DateFormat  string `json:"date_format" envconfig:"DATE_FORMAT" default:"Jan 2, 2006 15:04:05"`
	MaxFileSize int64  `json:"max_file_size" envconfig:"MAX_FILE_SIZE" default:"20485760"`
	DataDir     string `json:"data_dir" envconfig:"DATA_DIR" default:"data"`
	JWTSecret   string `json:"jwt_secret" envconfig:"JWT_SECRET" required:"true"`
	Domain      string `json:"domain" envconfig:"DOMAIN" default:"localhost"`
	SSHPort     int    `json:"ssh_port" envconfig:"SSH_PORT" default:"2222"`
	SSHKeyPath  string `json:"ssh_key_path" envconfig:"SSH_KEY_PATH" default:"ssh/host_key"`
	RepoPath    string `json:"repo_path" envconfig:"REPO_PATH" default:"repositories"`
	DBPath      string `json:"db_path" envconfig:"DB_PATH"`
}

var GlobalConfig Config

// Init initializes the configuration by:
// 1. Loading defaults
// 2. Loading JSON config if present (optional)
// 3. Overriding with environment variables
func Init() {
	// Set initial defaults
	GlobalConfig = Config{
		Port:        3000,
		DateFormat:  "Jan 2, 2006 15:04:05",
		DataDir:     "data",
		Domain:      "localhost",
		SSHPort:     2222,
		SSHKeyPath:  "ssh/host_key",
		RepoPath:    "repositories",
		MaxFileSize: 20485760, // 20MB
	}

	// Try to load JSON config if exists (optional)
	if configData, err := os.ReadFile("config.json"); err == nil {
		if err := json.Unmarshal(configData, &GlobalConfig); err != nil {
			log.Printf("Warning: Failed to parse config.json: %v", err)
		}
	}

	// Override with environment variables (prefixed with SIMPLEGIT_)
	if err := envconfig.Process("SIMPLEGIT", &GlobalConfig); err != nil {
		log.Fatal("Failed to process environment variables:", err)
	}

	// Post-processing
	// If DBPath is not set, use default path in DataDir
	if GlobalConfig.DBPath == "" {
		GlobalConfig.DBPath = filepath.Join(GlobalConfig.DataDir, "githost.db")
	}

	// Ensure all paths are absolute
	if !filepath.IsAbs(GlobalConfig.DataDir) {
		GlobalConfig.DataDir, _ = filepath.Abs(GlobalConfig.DataDir)
	}
	if !filepath.IsAbs(GlobalConfig.RepoPath) {
		GlobalConfig.RepoPath, _ = filepath.Abs(GlobalConfig.RepoPath)
	}
	if !filepath.IsAbs(GlobalConfig.DBPath) {
		GlobalConfig.DBPath, _ = filepath.Abs(GlobalConfig.DBPath)
	}
	if !filepath.IsAbs(GlobalConfig.SSHKeyPath) {
		GlobalConfig.SSHKeyPath, _ = filepath.Abs(GlobalConfig.SSHKeyPath)
	}

	// Validate required configurations
	if GlobalConfig.JWTSecret == "" {
		log.Fatal("JWT secret is required. Set it using SIMPLEGIT_JWT_SECRET environment variable")
	}

	// Create necessary directories
	createDirIfNotExists(GlobalConfig.DataDir)
	createDirIfNotExists(GlobalConfig.RepoPath)
	createDirIfNotExists(filepath.Dir(GlobalConfig.SSHKeyPath))

	// Log configuration (excluding sensitive data)
	log.Printf("Configuration loaded:")
	log.Printf("- HTTP Port: %d", GlobalConfig.Port)
	log.Printf("- SSH Port: %d", GlobalConfig.SSHPort)
	log.Printf("- Domain: %s", GlobalConfig.Domain)
	log.Printf("- Data Directory: %s", GlobalConfig.DataDir)
	log.Printf("- Repository Path: %s", GlobalConfig.RepoPath)
	log.Printf("- Development Mode: %v", GlobalConfig.DevMode)
}

func createDirIfNotExists(path string) {
	if err := os.MkdirAll(path, 0755); err != nil {
		log.Fatalf("Failed to create directory %s: %v", path, err)
	}
}
