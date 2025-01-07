package handlers

import (
	"net/http"
	"os"
)

func (s *Server) SetupRoutes() {
	// Static files
	fs := http.FileServer(http.Dir("static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	// Favicon
	http.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		if _, err := os.Stat("static/favicon.ico"); err == nil {
			http.ServeFile(w, r, "static/favicon.ico")
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	})

	// HTML routes
	http.HandleFunc("/", s.handleIndex)
	http.HandleFunc("/repo/", s.handleViewRepo)
	http.HandleFunc("/file/", s.handleViewFile)

	// API routes
	http.HandleFunc("/api/repos", s.handleListRepos)
}
