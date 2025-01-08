//database/db.go

package database

import (
	"os"
	"path/filepath"

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

	dbPath := filepath.Join(dataDir, "githost.db")

	// Configure GORM to use SQLite
	config := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	}

	db, err := gorm.Open(sqlite.Open(dbPath), config)
	if err != nil {
		return nil, err
	}

	// Auto migrate the schemas
	if err := db.AutoMigrate(&models.User{}); err != nil {
		return nil, err
	}

	return db, nil
}
