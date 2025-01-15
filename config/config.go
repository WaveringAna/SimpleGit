package config

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"runtime"

	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	DevMode      bool   `json:"dev_mode" envconfig:"DEV_MODE"`
	Port         int    `json:"port" envconfig:"PORT" default:"3000"`
	DateFormat   string `json:"date_format" envconfig:"DATE_FORMAT" default:"Jan 2, 2006 15:04:05"`
	MaxFileSize  int64  `json:"max_file_size" envconfig:"MAX_FILE_SIZE" default:"20485760"`
	DataDir      string `json:"data_dir" envconfig:"DATA_DIR" default:"data"`
	JWTSecret    string `json:"jwt_secret" envconfig:"JWT_SECRET"`
	Domain       string `json:"domain" envconfig:"DOMAIN" default:"localhost"`
	SSHPort      int    `json:"ssh_port" envconfig:"SSH_PORT" default:"2222"`
	SSHKeyPath   string `json:"ssh_key_path" envconfig:"SSH_KEY_PATH" default:"ssh/host_key"`
	RepoPath     string `json:"repo_path" envconfig:"REPO_PATH" default:"repositories"`
	DBPath       string `json:"db_path" envconfig:"DB_PATH"`
	TSServiceURL string `json:"ts_service_url" envconfig:"TS_SERVICE_URL" default:"http://localhost:3001"`
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

	// Try to load JSON config
	if configData, err := findConfigFile(); err == nil {
		if err := json.Unmarshal(configData, &GlobalConfig); err != nil {
			log.Printf("Warning: Failed to parse config.json: %v", err)
		} else {
			log.Printf("Successfully loaded config.json")
			// Debug: print the loaded config
			//prettyConfig, _ := json.MarshalIndent(GlobalConfig, "", "    ")
			//log.Printf("Loaded config: %s", string(prettyConfig))
		}
	} else {
		log.Printf("No config.json found, using defaults and environment variables")
	}

	// Override with environment variables
	if err := envconfig.Process("SIMPLEGIT", &GlobalConfig); err != nil {
		log.Printf("Warning: Environment variable processing error: %v", err)
		// Continue anyway since we might have values from config.json
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

	// Validate JWT secret after all loading is done
	if GlobalConfig.JWTSecret == "" {
		log.Fatal("JWT secret is required. Set it in config.json or using SIMPLEGIT_JWT_SECRET environment variable")
	}

	// Create necessary directories
	createDirIfNotExists(GlobalConfig.DataDir)
	createDirIfNotExists(GlobalConfig.RepoPath)
	createDirIfNotExists(filepath.Dir(GlobalConfig.SSHKeyPath))

	// Log non-sensitive configuration
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

func findConfigFile() ([]byte, error) {
	// Try current directory first
	if data, err := os.ReadFile("config.json"); err == nil {
		return data, nil
	}

	// Try config directory
	if data, err := os.ReadFile("config/config.json"); err == nil {
		return data, nil
	}

	// Try getting config relative to this source file
	_, filename, _, _ := runtime.Caller(0)
	configDir := filepath.Dir(filename)
	if data, err := os.ReadFile(filepath.Join(configDir, "config.json")); err == nil {
		return data, nil
	}

	// Try one directory up from the config package
	if data, err := os.ReadFile(filepath.Join(configDir, "..", "config.json")); err == nil {
		return data, nil
	}

	return nil, os.ErrNotExist
}
