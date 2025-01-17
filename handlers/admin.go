//handlers/admin.go

package handlers

import (
	"SimpleGit/models"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
)

func (s *Server) handleAdminDashboard(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"AdminPage": "dashboard",
		"Repos":     s.Repos,
	}
	s.tmpl.ExecuteTemplate(w, "admin-dashboard.html", s.addCommonData(r, data))
}

func (s *Server) handleAdminRepos(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"AdminPage": "dashboard",
		"Repos":     s.Repos,
	}
	s.tmpl.ExecuteTemplate(w, "admin-repos.html", data)
}

func (s *Server) handleAdminUsers(w http.ResponseWriter, r *http.Request) {
	var users []models.User
	if err := s.db.Find(&users).Error; err != nil {
		http.Error(w, "Failed to fetch users", http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"AdminPage": "dashboard",
		"Users":     users,
	}
	s.tmpl.ExecuteTemplate(w, "admin-users.html", data)
}

func (s *Server) handleCreateUser(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		s.tmpl.ExecuteTemplate(w, "admin-user-create.html", nil)
		return
	}

	// Handle POST request
	email := r.FormValue("email")
	username := r.FormValue("username")
	password := r.FormValue("password")
	isAdmin := r.FormValue("is_admin") == "on"

	_, err := s.userService.CreateUser(username, email, password, isAdmin)
	if err != nil {
		data := map[string]interface{}{
			"Error": "Failed to create user: " + err.Error(),
		}
		w.WriteHeader(http.StatusInternalServerError)
		s.tmpl.ExecuteTemplate(w, "admin-user-create.html", data)
		return
	}

	http.Redirect(w, r, "/admin/users", http.StatusSeeOther)
}

func (s *Server) handleCreateRepo(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		s.tmpl.ExecuteTemplate(w, "admin-repo-create.html", nil)
		return
	}

	// Handle POST request
	name := r.FormValue("name")
	description := r.FormValue("description")

	// Validate repository name
	if name == "" {
		data := map[string]interface{}{
			"Error": "Repository name is required",
		}
		w.WriteHeader(http.StatusBadRequest)
		s.tmpl.ExecuteTemplate(w, "admin-repo-create.html", data)
		return
	}

	// Create repository directory
	repoPath := filepath.Join(s.RepoPath, name)
	if err := os.MkdirAll(repoPath, 0755); err != nil {
		data := map[string]interface{}{
			"Error": "Failed to create repository directory",
		}
		w.WriteHeader(http.StatusInternalServerError)
		s.tmpl.ExecuteTemplate(w, "admin-repo-create.html", data)
		return
	}

	// Initialize git repository
	_, err := git.PlainInit(repoPath, true)
	if err != nil {
		data := map[string]interface{}{
			"Error": "Failed to initialize git repository",
		}
		w.WriteHeader(http.StatusInternalServerError)
		s.tmpl.ExecuteTemplate(w, "admin-repo-create.html", data)
		return
	}

	// Add to repositories map
	s.Repos[name] = &models.Repository{
		ID:          name,
		Name:        name,
		Path:        repoPath,
		Description: description,
		CreatedAt:   time.Now(),
	}

	http.Redirect(w, r, "/admin/repos", http.StatusSeeOther)
}

func (s *Server) handleDeleteUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != "DELETE" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract user ID from URL
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 4 {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	userID := parts[3]

	// Delete the user from the database
	if err := s.db.Delete(&models.User{}, "id = ?", userID).Error; err != nil {
		http.Error(w, "Failed to delete user", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleDeleteRepo(w http.ResponseWriter, r *http.Request) {
	if r.Method != "DELETE" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract repository name from URL
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 4 {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	repoName := parts[3]

	// Get repository path
	repo, ok := s.Repos[repoName]
	if !ok {
		http.Error(w, "Repository not found", http.StatusNotFound)
		return
	}

	// Delete repository directory
	if err := os.RemoveAll(repo.Path); err != nil {
		http.Error(w, "Failed to delete repository", http.StatusInternalServerError)
		return
	}

	// Remove from repositories map
	delete(s.Repos, repoName)

	w.WriteHeader(http.StatusOK)
}
