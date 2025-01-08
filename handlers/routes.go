//handlers/routes.go

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

	//Auth Route
	http.HandleFunc("/login", s.handleLogin)
	http.HandleFunc("/logout", s.handleLogout)

	// API routes
	http.HandleFunc("/api/repos", s.requireAuth(s.handleListRepos))

	// Admin routes
	http.HandleFunc("/setup-admin", s.handleAdminSetup)
	http.HandleFunc("/admin", s.requireAdmin(s.handleAdminDashboard))
	http.HandleFunc("/admin/repos", s.requireAdmin(s.handleAdminRepos))
	http.HandleFunc("/admin/users", s.requireAdmin(s.handleAdminUsers))
	http.HandleFunc("/admin/users/create", s.requireAdmin(s.handleCreateUser))
	http.HandleFunc("/admin/repos/create", s.requireAdmin(s.handleCreateRepo))
}
