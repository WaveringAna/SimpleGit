package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

type Repository struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Path        string    `json:"path"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	git         *git.Repository
}

type TreeEntry struct {
	Name    string `json:"name"`
	Path    string `json:"path"`
	Type    string `json:"type"` // "tree" or "blob"
	Size    int64  `json:"size"`
	Commit  string `json:"commit"`
	Message string `json:"message"`
}

type CommitInfo struct {
	Hash      string    `json:"hash"`
	Author    string    `json:"author"`
	Email     string    `json:"email"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
}

func (r *Repository) OpenGit() error {
	if r.git != nil {
		return nil
	}

	repo, err := git.PlainOpen(r.Path)
	if err != nil {
		return err
	}
	r.git = repo
	return nil
}

func (r *Repository) GetBranches() ([]string, error) {
	if err := r.OpenGit(); err != nil {
		return nil, err
	}

	branches := []string{}
	branchIter, err := r.git.Branches()
	if err != nil {
		return nil, err
	}

	err = branchIter.ForEach(func(ref *plumbing.Reference) error {
		name := ref.Name().Short()
		branches = append(branches, name)
		return nil
	})

	return branches, err
}

func (r *Repository) GetTree(path, ref string) ([]TreeEntry, error) {
	if err := r.OpenGit(); err != nil {
		return nil, err
	}

	hash := plumbing.NewHash(ref)
	commit, err := r.git.CommitObject(hash)
	if err != nil {
		return nil, err
	}

	tree, err := commit.Tree()
	if err != nil {
		return nil, err
	}

	var entries []TreeEntry
	err = tree.Files().ForEach(func(f *object.File) error {
		// Skip files not in the current path
		if path != "" && !strings.HasPrefix(f.Name, path) {
			return nil
		}

		entries = append(entries, TreeEntry{
			Name:    filepath.Base(f.Name),
			Path:    f.Name,
			Type:    "blob",
			Size:    f.Size,
			Commit:  commit.Hash.String(),
			Message: commit.Message,
		})
		return nil
	})

	return entries, err
}

func (r *Repository) GetCommits(ref string, limit int) ([]CommitInfo, error) {
	if err := r.OpenGit(); err != nil {
		return nil, err
	}

	var commits []CommitInfo

	hash := plumbing.NewHash(ref)
	cIter, err := r.git.Log(&git.LogOptions{From: hash, Order: git.LogOrderCommitterTime})
	if err != nil {
		return nil, err
	}

	err = cIter.ForEach(func(c *object.Commit) error {
		if limit > 0 && len(commits) >= limit {
			return errors.New("limit reached")
		}

		commits = append(commits, CommitInfo{
			Hash:      c.Hash.String(),
			Author:    c.Author.Name,
			Email:     c.Author.Email,
			Message:   c.Message,
			Timestamp: c.Author.When,
		})
		return nil
	})

	if err != nil && err.Error() != "limit reached" {
		return nil, err
	}

	return commits, nil
}

func (r *Repository) GetFileContent(path, ref string) ([]byte, error) {
	if err := r.OpenGit(); err != nil {
		return nil, err
	}

	hash := plumbing.NewHash(ref)
	commit, err := r.git.CommitObject(hash)
	if err != nil {
		return nil, err
	}

	tree, err := commit.Tree()
	if err != nil {
		return nil, err
	}

	file, err := tree.File(path)
	if err != nil {
		return nil, err
	}

	content, err := file.Contents()
	if err != nil {
		return nil, err
	}
	return []byte(content), nil
}

type Server struct {
	RepoPath string                 // Base path for git repositories
	Repos    map[string]*Repository // Cache of repositories
	tmpl     *template.Template     // HTML templates
}

func NewServer(repoPath string) (*Server, error) {
	if err := os.MkdirAll(repoPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create repo directory: %w", err)
	}

	s := &Server{
		RepoPath: repoPath,
		Repos:    make(map[string]*Repository),
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
			return t.Format("Jan 02, 2006 15:04:05")
		},
		"split": strings.Split,
		"dir":   filepath.Dir,
		"sub": func(a, b int) int {
			return a - b
		},
	}

	// Parse templates
	tmpl, err := template.New("").Funcs(funcMap).ParseGlob("templates/*.html")
	if err != nil {
		return nil, fmt.Errorf("failed to parse templates: %w", err)
	}
	s.tmpl = tmpl

	return s, nil
}

func (s *Server) ScanRepositories() error {
	entries, err := os.ReadDir(s.RepoPath)
	if err != nil {
		return fmt.Errorf("failed to read repo directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		// Check if it's a git repository
		gitDir := filepath.Join(s.RepoPath, entry.Name(), ".git")
		if _, err := os.Stat(gitDir); os.IsNotExist(err) {
			continue
		}

		// Add to our cache
		s.Repos[entry.Name()] = &Repository{
			ID:        entry.Name(), // Use name as ID for now
			Name:      entry.Name(),
			Path:      filepath.Join(s.RepoPath, entry.Name()),
			CreatedAt: time.Now(), // We'll get this from git later
		}
	}

	return nil
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	if err := s.tmpl.ExecuteTemplate(w, "index.html", map[string]interface{}{
		"Repos": s.Repos,
	}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) handleListRepos(w http.ResponseWriter, r *http.Request) {
	repos := make([]*Repository, 0, len(s.Repos))
	for _, repo := range s.Repos {
		repos = append(repos, repo)
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(repos); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) handleViewRepo(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 2 {
		http.NotFound(w, r)
		return
	}

	repoName := parts[1]
	repo, ok := s.Repos[repoName]
	if !ok {
		http.NotFound(w, r)
		return
	}

	// Get current branch and ref
	branch := r.URL.Query().Get("branch")
	if branch == "" {
		branch = "main" // fallback to main
	}

	if err := repo.OpenGit(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Get repository data
	branches, err := repo.GetBranches()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	ref, err := repo.git.Reference(plumbing.NewBranchReferenceName(branch), true)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Get file tree
	path := strings.Join(parts[2:], "/")
	entries, err := repo.GetTree(path, ref.Hash().String())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Get recent commits
	commits, err := repo.GetCommits(ref.Hash().String(), 10)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"Repo":     repo,
		"Branches": branches,
		"Branch":   branch,
		"Path":     path,
		"Entries":  entries,
		"Commits":  commits,
	}

	if err := s.tmpl.ExecuteTemplate(w, "repo.html", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) handleViewFile(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 3 {
		http.NotFound(w, r)
		return
	}

	repoName := parts[1]
	repo, ok := s.Repos[repoName]
	if !ok {
		http.NotFound(w, r)
		return
	}

	branch := r.URL.Query().Get("branch")
	if branch == "" {
		branch = "main"
	}

	if err := repo.OpenGit(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	ref, err := repo.git.Reference(plumbing.NewBranchReferenceName(branch), true)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	path := strings.Join(parts[2:], "/")
	content, err := repo.GetFileContent(path, ref.Hash().String())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"Repo":    repo,
		"Path":    path,
		"Content": string(content),
		"Size":    int64(len(content)),
	}

	if err := s.tmpl.ExecuteTemplate(w, "file.html", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) setupRoutes() {
	// Static files
	fs := http.FileServer(http.Dir("static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	// HTML routes
	http.HandleFunc("/", s.handleIndex)
	http.HandleFunc("/repo/", s.handleViewRepo)
	http.HandleFunc("/file/", s.handleViewFile)

	// API routes
	http.HandleFunc("/api/repos", s.handleListRepos)
}

func main() {
	server, err := NewServer("./repositories")
	if err != nil {
		log.Fatal(err)
	}

	if err := server.ScanRepositories(); err != nil {
		log.Fatal(err)
	}

	server.setupRoutes()

	log.Println("Server starting on :3000")
	if err := http.ListenAndServe(":3000", nil); err != nil {
		log.Fatal(err)
	}
}
