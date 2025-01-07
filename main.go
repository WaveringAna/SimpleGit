package main

import (
	"SimpleGit/config"
	"SimpleGit/handlers"
	"fmt"
	"log"
	"net/http"
)

func main() {
	config.Init()

	server, err := handlers.NewServer("./repositories")
	if err != nil {
		log.Fatal(err)
	}

	if err := server.ScanRepositories(); err != nil {
		log.Fatal(err)
	}

	server.SetupRoutes()

	addr := fmt.Sprintf(":%d", config.GlobalConfig.Port)
	log.Printf("Server starting on %s (dev mode: %v)", addr, config.GlobalConfig.DevMode)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatal(err)
	}
}
