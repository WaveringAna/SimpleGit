package database

import (
	"log"
	"os"
	"path/filepath"

	config "SimpleGit/config"
	"SimpleGit/models"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// InitDB initializes the SQLite database and performs migrations
func InitDB(dataDir string) (*gorm.DB, error) {
	// Ensure data directory exists
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, err
	}

	dbPath := config.GlobalConfig.DBPath
	if !filepath.IsAbs(dbPath) {
		dbPath = filepath.Join(dataDir, dbPath)
	}

	// Ensure the directory for the database file exists
	dbDir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		return nil, err
	}

	log.Printf("Using database file: %s", dbPath)

	// If database file doesn't exist, create it (SQLite needs the directory to exist)
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		file, err := os.OpenFile(dbPath, os.O_RDWR|os.O_CREATE, 0600)
		if err != nil {
			return nil, err
		}
		file.Close()
	}

	// Configure GORM to use SQLite
	gormConfig := &gorm.Config{}

	if config.GlobalConfig.DevMode {
		gormConfig.Logger = logger.Default.LogMode(logger.Info)
	} else {
		gormConfig.Logger = logger.Default.LogMode(logger.Silent)
	}

	db, err := gorm.Open(sqlite.Open(dbPath), gormConfig)
	if err != nil {
		return nil, err
	}

	// Auto migrate the schemas
	if err := db.AutoMigrate(&models.User{}, &models.SSHKey{}); err != nil {
		return nil, err
	}

	return db, nil
}
