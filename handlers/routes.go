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
	http.HandleFunc("/repo/", s.handleRepo)
	http.HandleFunc("/file/", s.handleViewFile)
	http.HandleFunc("/commit/", s.handleViewCommit)
	http.HandleFunc("/raw/", s.handleRawFile)

	//Auth Route
	http.HandleFunc("/login", s.handleLogin)
	http.HandleFunc("/logout", s.handleLogout)
	http.HandleFunc("/profile", s.requireAuth(s.handleProfile))

	// API routes
	http.HandleFunc("/api/repos", s.requireAuth(s.handleListRepos))
	http.HandleFunc("/api/ssh-keys", s.requireAuth(s.handleListSSHKeys))
	http.HandleFunc("/api/ssh-keys/add", s.requireAuth(s.handleAddSSHKey))
	http.HandleFunc("/api/ssh-keys/", s.requireAuth(s.handleDeleteSSHKey))

	// Admin routes
	http.HandleFunc("/setup-admin", s.handleAdminSetup)
	http.HandleFunc("/admin", s.requireAdmin(s.handleAdminDashboard))
	http.HandleFunc("/admin/repos", s.requireAdmin(s.handleAdminRepos))
	http.HandleFunc("/admin/users", s.requireAdmin(s.handleAdminUsers))
	http.HandleFunc("/admin/users/create", s.requireAdmin(s.handleCreateUser))
	http.HandleFunc("/admin/repos/create", s.requireAdmin(s.handleCreateRepo))
	http.HandleFunc("/admin/users/", s.requireAdmin(s.handleDeleteUser))
	http.HandleFunc("/admin/repos/", s.requireAdmin(s.handleDeleteRepo))
}
