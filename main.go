//main.go

package main

import (
	"SimpleGit/config"
	"SimpleGit/database"
	"SimpleGit/handlers"
	"SimpleGit/models"
	"SimpleGit/ssh"
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"sync"
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

	server, err := handlers.NewServer(config.GlobalConfig.RepoPath)
	if err != nil {
		log.Fatal(err)
	}

	server.HighlightCache = handlers.NewHighlightCache(1000)

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

	// Create SSH server
	repoPath := filepath.Join(".", "repositories")
	absRepoPath, err := filepath.Abs(repoPath)
	if err != nil {
		log.Fatal("Failed to get absolute repository path:", err)
	}
	log.Printf("Using repository path: %s", absRepoPath)

	sshServer, err := ssh.NewServer(
		absRepoPath, // Use absolute path
		userService,
		func() {
			// This callback will be called after git-receive-pack operations
			if err := server.ScanRepositories(); err != nil {
				log.Printf("Error rescanning repositories after update: %v", err)
			} else {
				log.Printf("Successfully rescanned repositories after update")
			}
		},
	)
	if err != nil {
		log.Fatal("Failed to create SSH server:", err)
	}

	// Use WaitGroup to keep the main function from exiting
	var wg sync.WaitGroup
	wg.Add(2)

	// Start HTTP server in a goroutine
	go func() {
		defer wg.Done()
		addr := fmt.Sprintf(":%d", config.GlobalConfig.Port)
		log.Printf("HTTP server starting on %s (dev mode: %v)", addr, config.GlobalConfig.DevMode)
		if err := http.ListenAndServe(addr, nil); err != nil {
			log.Fatal("HTTP server error:", err)
		}
	}()

	// Start SSH server in a goroutine
	go func() {
		defer wg.Done()
		addr := fmt.Sprintf(":%d", config.GlobalConfig.SSHPort)
		log.Printf("SSH server starting on %s", addr)
		if err := sshServer.ListenAndServe(addr); err != nil {
			log.Fatal("SSH server error:", err)
		}
	}()

	// Wait for both servers
	wg.Wait()
}
