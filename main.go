//main.go

package main

import (
	"SimpleGit/config"
	"SimpleGit/database"
	"SimpleGit/handlers"
	"SimpleGit/models"
	"fmt"
	"log"
	"net/http"
)

func main() {
	config.Init()

	// Initialize database
	db, err := database.InitDB(config.GlobalConfig.DataDir)
	if err != nil {
		log.Fatal(err)
	}

	// Initialize user service with JWT key
	userService := models.NewUserService(db, []byte(config.GlobalConfig.JWTSecret))

	server, err := handlers.NewServer("./repositories")
	if err != nil {
		log.Fatal(err)
	}

	// Set database and user service
	server.SetDB(db)
	server.SetUserService(userService)

	if err := server.ScanRepositories(); err != nil {
		log.Fatal(err)
	}

	if err := server.InitAdminSetup(); err != nil {
		log.Fatal(err)
	}

	server.SetupRoutes()

	addr := fmt.Sprintf(":%d", config.GlobalConfig.Port)
	log.Printf("Server starting on %s (dev mode: %v)", addr, config.GlobalConfig.DevMode)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatal(err)
	}
}
