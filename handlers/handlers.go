//handlers/handlers.go

package handlers

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"SimpleGit/config"
	"SimpleGit/models"
	"SimpleGit/utils"

	"github.com/go-git/go-git/v5/plumbing"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Server represents the server configuration and dependencies.
//
// Parameters:
//   - RepoPath: The path to the repository directory.
//   - Repos: A map of repository names to repository objects.
//   - tmpl: The template engine instance.
//   - userService: The user service instance.
//   - db: The database instance.
type Server struct {
	RepoPath    string
	Repos       map[string]*models.Repository
	tmpl        *template.Template
	userService *models.UserService
	db          *gorm.DB
}

// NewServer creates a new server instance with the given repository path.
//
// Parameters:
//   - repoPath: The path to the repository directory.
func NewServer(repoPath string) (*Server, error) {
	if err := os.MkdirAll(repoPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create repo directory: %w", err)
	}

	s := &Server{
		RepoPath: repoPath,
		Repos:    make(map[string]*models.Repository),
	}

	// Create template functions
	funcMap := template.FuncMap{
		"formatSize": func(size int64) string {
			if size < 1024 {
				return fmt.Sprintf("%d B", size)
			}
			if size < 1024*1024 {
				return fmt.Sprintf("%.1f KB", float64(size)/1024)
			}
			return fmt.Sprintf("%.1f MB", float64(size)/(1024*1024))
		},
		"formatDate": func(t time.Time) string {
			return t.Format(config.GlobalConfig.DateFormat)
		},
		"split": strings.Split,
		"dir":   filepath.Dir,
		"sub": func(a, b int) int {
			return a - b
		},
		"add": func(a, b int) int {
			return a + b
		},
		"firstLine": func(s string) string {
			if i := strings.Index(s, "\n"); i != -1 {
				return s[:i]
			}
			return s
		},
		"getFileIcon": utils.GetFileIcon,
	}

	// Parse templates
	tmpl, err := template.New("").Funcs(funcMap).ParseGlob("templates/*.html")
	if err != nil {
		return nil, fmt.Errorf("failed to parse templates: %w", err)
	}
	s.tmpl = tmpl

	return s, nil
}

// addCommonData adds common data to the given map.
//
// Parameters:
//   - r: The HTTP request.
//   - data: The map of data to add to.
//
// Returns:
//
//	The updated map of data.
func (s *Server) addCommonData(r *http.Request, data map[string]interface{}) map[string]interface{} {
	if data == nil {
		data = make(map[string]interface{})
	}

	user, _ := s.getUserFromRequest(r)
	data["User"] = user

	return data
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		models.HandleError(w, r, models.NewNotFoundError("Page not found").WithDetail(fmt.Sprintf("Path: %s", r.URL.Path)))
		return
	}

	// Ensure all repositories are loaded
	if err := s.ScanRepositories(); err != nil {
		models.HandleError(w, r, models.NewInternalError("Failed to scan repositories").WithError(err))
		return
	}

	data := map[string]interface{}{
		"Repos": s.Repos,
	}

	s.tmpl.ExecuteTemplate(w, "index.html", s.addCommonData(r, data))
}

// handleListRepos handles the request to list all repositories.
//
// Parameters:
//   - w: The HTTP response writer.
//   - r: The HTTP request.
func (s *Server) handleListRepos(w http.ResponseWriter, r *http.Request) {
	repos := make([]*models.Repository, 0, len(s.Repos))
	for _, repo := range s.Repos {
		repos = append(repos, repo)
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(repos); err != nil {
		models.HandleError(w, r, models.NewInternalError("Failed to encode response").WithError(err))
	}
}

// handleViewRepo handles the request to view a repository.
//
// Parameters:
//   - w: The HTTP response writer.
//   - r: The HTTP request.
func (s *Server) handleViewRepo(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 2 {
		models.HandleError(w, r, models.NewBadRequestError("Invalid repository path"))
		return
	}

	repoName := parts[1]
	repo, ok := s.Repos[repoName]
	if !ok {
		// Try to rescan repositories in case it was just created
		if err := s.ScanRepositories(); err != nil {
			models.HandleError(w, r, models.NewInternalError("Failed to scan repositories").WithError(err))
			return
		}

		repo, ok = s.Repos[repoName]
		if !ok {
			models.HandleError(w, r, models.NewNotFoundError("Repository not found").WithDetail(fmt.Sprintf("Repository: %s", repoName)))
			return
		}
	}

	path := strings.Join(parts[2:], "/")

	// Get repository data
	branches, err := repo.GetBranches()
	if err != nil {
		// Don't treat this as an error for empty repos
		branches = []string{}
	}

	// Handle empty repository case
	if len(branches) == 0 {
		data := map[string]interface{}{
			"Repo":     repo,
			"Path":     path,
			"Branches": []string{},
			"Branch":   "",
			"Entries":  []models.TreeEntry{},
			"Commits":  []models.Commit{},
			"IsEmpty":  true,
		}

		if err := s.tmpl.ExecuteTemplate(w, "repo.html", data); err != nil {
			models.HandleError(w, r, models.NewInternalError("Failed to render template").WithError(err))
		}
		return
	}

	branch := r.URL.Query().Get("branch")
	if branch == "" {
		branch = branches[0]
	}

	gitRepo, err := repo.Git()
	if err != nil {
		models.HandleError(w, r, models.NewGitError("Failed to get git repository", err))
		return
	}

	ref, err := gitRepo.Reference(plumbing.NewBranchReferenceName(branch), true)
	if err != nil {
		models.HandleError(w, r, models.NewGitError("Failed to get branch reference", err))
		return
	}

	hashStr := ref.Hash().String()

	entries, err := repo.GetTree(path, hashStr)
	if err != nil {
		models.HandleError(w, r, models.NewGitError("Failed to get repository contents", err))
		return
	}

	commits, err := repo.GetCommits(branch, 10)
	if err != nil {
		models.HandleError(w, r, models.NewGitError("Failed to get commits", err))
		return
	}

	data := map[string]interface{}{
		"Repo":     repo,
		"Path":     path,
		"Branches": branches,
		"Branch":   branch,
		"Entries":  entries,
		"Commits":  commits,
		"IsEmpty":  false,
	}

	if err := s.tmpl.ExecuteTemplate(w, "repo.html", s.addCommonData(r, data)); err != nil {
		models.HandleError(w, r, models.NewInternalError("Failed to render template").WithError(err))
	}
}

// handleViewFile handles the request to view a file.
//
// Parameters:
//   - w: The HTTP response writer.
//   - r: The HTTP request.
func (s *Server) handleViewFile(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 3 {
		models.HandleError(w, r, models.NewBadRequestError("Invalid file path"))
		return
	}

	repoName := parts[1]
	repo, ok := s.Repos[repoName]
	if !ok {
		models.HandleError(w, r, models.NewNotFoundError("Repository not found").WithDetail(fmt.Sprintf("Repository: %s", repoName)))
		return
	}

	path := strings.Join(parts[2:], "/")

	branch := r.URL.Query().Get("branch")
	if branch == "" {
		branches, err := repo.GetBranches()
		if err != nil {
			models.HandleError(w, r, models.NewGitError("Failed to get branches", err))
			return
		}
		if len(branches) == 0 {
			models.HandleError(w, r, models.NewGitError("No branches found", nil))
			return
		}
		branch = branches[0]
	}

	content, err := repo.GetFile(path, branch)
	if err != nil {
		if err == plumbing.ErrObjectNotFound {
			models.HandleError(w, r, models.NewNotFoundError("File not found"))
		} else {
			models.HandleError(w, r, models.NewGitError("Failed to read file", err))
		}
		return
	}

	// Check if file is binary
	if utils.IsBinaryFile(content) {
		w.Header().Set("Content-Type", "text/html")
		data := map[string]interface{}{
			"Title":   "Binary File",
			"Message": "This appears to be a binary file and cannot be displayed.",
			"Detail":  fmt.Sprintf("File size: %d bytes\nYou can download or view this file with an appropriate application.", len(content)),
		}
		if err := s.tmpl.ExecuteTemplate(w, "error.html", data); err != nil {
			models.HandleError(w, r, models.NewInternalError("Failed to render template").WithError(err))
		}
		return
	}

	// Check file size before rendering
	if int64(len(content)) > config.GlobalConfig.MaxFileSize {
		w.Header().Set("Content-Type", "text/html")
		data := map[string]interface{}{
			"Title":   "File Too Large",
			"Message": "This file exceeds the maximum size limit for display.",
			"Detail":  fmt.Sprintf("File size: %d bytes\nMaximum allowed: %d bytes", len(content), config.GlobalConfig.MaxFileSize),
		}
		if err := s.tmpl.ExecuteTemplate(w, "error.html", data); err != nil {
			models.HandleError(w, r, models.NewInternalError("Failed to render template").WithError(err))
		}
		return
	}

	// Split content into lines
	lines := strings.Split(string(content), "\n")

	// Parse symbols from content
	symbols := utils.ParseSymbols(content)

	data := map[string]interface{}{
		"Repo":    repo,
		"Path":    path,
		"Lines":   lines,
		"Size":    int64(len(content)),
		"Symbols": symbols,
	}

	if err := s.tmpl.ExecuteTemplate(w, "file.html", data); err != nil {
		models.HandleError(w, r, models.NewInternalError("Failed to render template").WithError(err))
	}
}

// ScanRepositories scans the repository directory and updates the server's repository map.
func (s *Server) ScanRepositories() error {
	entries, err := os.ReadDir(s.RepoPath)
	if err != nil {
		return fmt.Errorf("failed to read repo directory: %w", err)
	}

	// Create a new map to avoid duplicates
	repos := make(map[string]*models.Repository)

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		name := entry.Name()
		path := filepath.Join(s.RepoPath, name)

		// Check if it's a git repository (has .git directory or is a bare repo)
		_, errGit := os.Stat(filepath.Join(path, ".git"))
		_, errBare := os.Stat(filepath.Join(path, "HEAD"))

		if os.IsNotExist(errGit) && os.IsNotExist(errBare) {
			continue
		}

		// Get repository info
		info, err := entry.Info()
		if err != nil {
			continue
		}

		// Add or update repository
		if existing, ok := s.Repos[name]; ok {
			// Update existing repository
			existing.Path = path
			existing.Size = info.Size()
			repos[name] = existing
		} else {
			// Create new repository entry
			repos[name] = &models.Repository{
				ID:        name,
				Name:      name,
				Path:      path,
				CreatedAt: info.ModTime(),
				Size:      info.Size(),
			}
		}
	}

	// Update the server's repository map
	s.Repos = repos
	return nil
}

// InitAdminSetup checks if an admin user exists and creates a setup token if not.
func (s *Server) InitAdminSetup() error {
	// Check if admin exists
	adminCount, err := s.userService.GetAdminCount()
	if err != nil {
		return err
	}

	if adminCount == 0 {
		// Generate setup token
		setupToken := uuid.New().String()
		if err := os.WriteFile("admin_setup_token.txt", []byte(setupToken), 0600); err != nil {
			return fmt.Errorf("failed to create admin setup token: %w", err)
		}
		fmt.Printf("Admin setup token created: %s\n", setupToken)
	}

	return nil
}

// SetDB sets the database instance for the server.
func (s *Server) SetDB(db *gorm.DB) {
	s.db = db
}

// SetUserService sets the user service instance for the server.
func (s *Server) SetUserService(userService *models.UserService) {
	s.userService = userService
}

// getUserFromRequest gets the user from the request header.
//
// Parameters:
//   - r: The HTTP request.
//
// Returns:
//
//	The user object or nil if not found.
func (s *Server) getUserFromRequest(r *http.Request) (*models.User, error) {
	userID := r.Header.Get("User-ID")
	if userID == "" {
		return nil, nil
	}

	var user models.User
	if err := s.db.First(&user, "id = ?", userID).Error; err != nil {
		return nil, err
	}

	return &user, nil
}
