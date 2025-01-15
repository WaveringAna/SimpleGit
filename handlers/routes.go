//handlers/routes.go

package handlers

import (
	"fmt"
	"net/http"
	"os"
)

func (s *Server) addUserData(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Try to get user from cookie, but don't require it
		cookie, err := r.Cookie("auth_token")
		if err == nil {
			user, err := s.userService.VerifyToken(cookie.Value)
			if err == nil {
				// Add user to request context
				r.Header.Set("User-ID", user.ID)
				r.Header.Set("User-Admin", fmt.Sprintf("%v", user.IsAdmin))
			}
		}
		next.ServeHTTP(w, r)
	}
}

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
	http.HandleFunc("/", s.addUserData(s.handleIndex))
	http.HandleFunc("/repo/", s.addUserData(s.handleRepo))
	http.HandleFunc("/file/", s.addUserData(s.handleViewFile))
	http.HandleFunc("/commit/", s.addUserData(s.handleViewCommit))
	http.HandleFunc("/raw/", s.addUserData(s.handleRawFile))

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
