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
	"regexp"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

type Config struct {
	DevMode     bool   `json:"dev_mode"`
	Port        int    `json:"port"`
	DateFormat  string `json:"date_format"` // e.g. "2006-01-02 15:04:05"
	MaxFileSize int64  `json:"max_file_size"`
}

var config Config

func init() {
	data, err := os.ReadFile("config.json")
	if err != nil {
		log.Fatal("Failed to read config:", err)
	}
	if err := json.Unmarshal(data, &config); err != nil {
		log.Fatal("Failed to parse config:", err)
	}
}

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

type Symbol struct {
	Name string
	Type string // "function", "method", "class", "interface", "const", "var"
	Icon string
	Line int
}

type ErrorType string

const (
	ErrorTypeNotFound      ErrorType = "NOT_FOUND"
	ErrorTypeUnauthorized  ErrorType = "UNAUTHORIZED"
	ErrorTypeBadRequest    ErrorType = "BAD_REQUEST"
	ErrorTypeInternal      ErrorType = "INTERNAL"
	ErrorTypeGit           ErrorType = "GIT_ERROR"
	ErrorTypeInvalidPath   ErrorType = "INVALID_PATH"
	ErrorTypeInvalidBranch ErrorType = "INVALID_BRANCH"
)

// AppError represents a structured application error
type AppError struct {
	Type       ErrorType `json:"type"`
	Message    string    `json:"message"`
	Detail     string    `json:"detail,omitempty"`
	Code       int       `json:"-"` // HTTP status code
	RequestID  string    `json:"request_id,omitempty"`
	File       string    `json:"file,omitempty"`
	Line       int       `json:"line,omitempty"`
	Err        error     `json:"-"`
	ShowInProd bool      `json:"-"` // Whether to show details in production mode
}

// Add a method to mark errors as production-safe
func (e *AppError) ShowInProduction() *AppError {
	e.ShowInProd = true
	return e
}

// Error implements the error interface
func (e *AppError) Error() string {
	if e.Detail != "" {
		return fmt.Sprintf("%s: %s (%s)", e.Type, e.Message, e.Detail)
	}
	return fmt.Sprintf("%s: %s", e.Type, e.Message)
}

// Unwrap implements the errors.Unwrap interface
func (e *AppError) Unwrap() error {
	return e.Err
}

// NewError creates a new AppError with stack trace
func NewError(errType ErrorType, message string, code int) *AppError {
	_, file, line, _ := runtime.Caller(1)
	return &AppError{
		Type:    errType,
		Message: message,
		Code:    code,
		File:    filepath.Base(file),
		Line:    line,
	}
}

// WithDetail adds detail to the error
func (e *AppError) WithDetail(detail string) *AppError {
	e.Detail = detail
	return e
}

// WithRequestID adds a request ID to the error
func (e *AppError) WithRequestID(requestID string) *AppError {
	e.RequestID = requestID
	return e
}

// WithError wraps an underlying error
func (e *AppError) WithError(err error) *AppError {
	e.Err = err
	if err != nil {
		e.Detail = err.Error()
	}
	return e
}

// Common error constructors
func NewNotFoundError(message string) *AppError {
	return NewError(ErrorTypeNotFound, message, http.StatusNotFound)
}

func NewUnauthorizedError(message string) *AppError {
	return NewError(ErrorTypeUnauthorized, message, http.StatusUnauthorized)
}

func NewBadRequestError(message string) *AppError {
	return NewError(ErrorTypeBadRequest, message, http.StatusBadRequest)
}

func NewInternalError(message string) *AppError {
	return NewError(ErrorTypeInternal, message, http.StatusInternalServerError)
}

func NewGitError(message string, err error) *AppError {
	return NewError(ErrorTypeGit, message, http.StatusInternalServerError).WithError(err)
}

// HandleError writes the error to the response in a consistent format
func HandleError(w http.ResponseWriter, r *http.Request, err error) {
	var appErr *AppError
	if !errors.As(err, &appErr) {
		appErr = NewInternalError("Internal Server Error").WithError(err)
	}

	// Build error details for logging
	details := []string{
		fmt.Sprintf("Type: %s", appErr.Type),
		fmt.Sprintf("Message: %s", appErr.Message),
		fmt.Sprintf("Location: %s:%d", appErr.File, appErr.Line),
	}

	if appErr.RequestID != "" {
		details = append(details, fmt.Sprintf("RequestID: %s", appErr.RequestID))
	}
	if appErr.Detail != "" {
		details = append(details, fmt.Sprintf("Detail: %s", appErr.Detail))
	}
	if appErr.Err != nil {
		details = append(details, fmt.Sprintf("Error: %v", appErr.Err))
	}

	// Log all error details
	log.Printf("Error occurred:\n\t%s", strings.Join(details, "\n\t"))

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(appErr.Code)

	if config.DevMode || appErr.ShowInProd {
		json.NewEncoder(w).Encode(appErr)
	} else {
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Internal Server Error",
		})
	}
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
	seenDirs := make(map[string]bool)

	err = tree.Files().ForEach(func(f *object.File) error {
		// Skip files not in the current path
		if !strings.HasPrefix(f.Name, path) {
			return nil
		}

		// Get the relative path from current directory
		relPath := strings.TrimPrefix(f.Name, path)
		if path != "" {
			relPath = strings.TrimPrefix(relPath, "/")
		}

		// If there's no relative path, this file is not in the current directory
		if relPath == "" {
			return nil
		}

		// Split the relative path into parts
		parts := strings.Split(relPath, "/")

		if len(parts) > 1 {
			// This is a file in a subdirectory
			dirName := parts[0]
			dirPath := filepath.Join(path, dirName)

			// Only add directory once
			if !seenDirs[dirPath] {
				seenDirs[dirPath] = true
				entries = append(entries, TreeEntry{
					Name:    dirName,
					Path:    dirPath,
					Type:    "tree",
					Size:    0,
					Commit:  commit.Hash.String(),
					Message: commit.Message,
				})
			}
		} else {
			// This is a file in the current directory
			// Get the last commit for this file
			fileCommit, err := r.git.Log(&git.LogOptions{From: commit.Hash, Order: git.LogOrderCommitterTime, FileName: &f.Name})
			if err != nil {
				return err
			}

			var lastCommitMessage string
			first := true
			err = fileCommit.ForEach(func(c *object.Commit) error {
				if first {
					lastCommitMessage = c.Message
					first = false
					return errors.New("got latest commit") // Stop after first commit since log is ordered by time
				}
				return nil
			})
			if err != nil && err.Error() != "got latest commit" {
				return err
			}

			entries = append(entries, TreeEntry{
				Name:    f.Name[strings.LastIndex(f.Name, "/")+1:],
				Path:    f.Name,
				Type:    "blob",
				Size:    f.Size,
				Commit:  commit.Hash.String(),
				Message: lastCommitMessage,
			})
		}
		return nil
	})

	// Sort entries: directories first, then files, both alphabetically with dot files first
	sort.Slice(entries, func(i, j int) bool {
		// If both are the same type (directory or file)
		if entries[i].Type == entries[j].Type {
			// If both start with dot or both don't start with dot
			iDot := strings.HasPrefix(entries[i].Name, ".")
			jDot := strings.HasPrefix(entries[j].Name, ".")
			if iDot == jDot {
				return strings.ToLower(entries[i].Name) < strings.ToLower(entries[j].Name)
			}
			// Dot files/folders come first
			return iDot && !jDot
		}
		// Directories come before files
		return entries[i].Type == "tree" && entries[j].Type == "blob"
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

func (r *Repository) GetFile(path, branch string) ([]byte, error) {
	if err := r.OpenGit(); err != nil {
		return nil, NewGitError("Failed to open repository", err)
	}

	// Get reference for branch
	refName := plumbing.NewBranchReferenceName(branch)
	ref, err := r.git.Reference(refName, true)
	if err != nil {
		return nil, NewGitError("Failed to get branch reference", err)
	}

	// Get commit
	commit, err := r.git.CommitObject(ref.Hash())
	if err != nil {
		return nil, NewGitError("Failed to get commit", err)
	}

	// Get tree
	tree, err := commit.Tree()
	if err != nil {
		return nil, NewGitError("Failed to get tree", err)
	}

	// Get file
	file, err := tree.File(path)
	if err != nil {
		if err == object.ErrFileNotFound {
			return nil, NewNotFoundError("File not found")
		}
		return nil, NewGitError("Failed to get file", err)
	}

	content, err := file.Contents()
	if err != nil {
		return nil, NewGitError("Failed to read file contents", err)
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
			return t.Format(config.DateFormat)
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
		HandleError(w, r, NewNotFoundError("Page not found").WithDetail(fmt.Sprintf("Path: %s", r.URL.Path)))
		return
	}

	if err := s.tmpl.ExecuteTemplate(w, "index.html", map[string]interface{}{
		"Repos": s.Repos,
	}); err != nil {
		HandleError(w, r, NewInternalError("Failed to render template").WithError(err))
	}
}

func (s *Server) handleListRepos(w http.ResponseWriter, r *http.Request) {
	repos := make([]*Repository, 0, len(s.Repos))
	for _, repo := range s.Repos {
		repos = append(repos, repo)
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(repos); err != nil {
		HandleError(w, r, NewInternalError("Failed to encode response").WithError(err))
	}
}

func (s *Server) handleViewRepo(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 2 {
		HandleError(w, r, NewBadRequestError("Invalid repository path"))
		return
	}

	repoName := parts[1]
	repo, ok := s.Repos[repoName]
	if !ok {
		HandleError(w, r, NewNotFoundError("Repository not found").WithDetail(fmt.Sprintf("Repository: %s", repoName)))
		return
	}

	path := strings.Join(parts[2:], "/")

	// Get repository data
	branches, err := repo.GetBranches()
	if err != nil {
		HandleError(w, r, NewGitError("Failed to get branches", err))
		return
	}

	branch := r.URL.Query().Get("branch")
	if branch == "" && len(branches) > 0 {
		branch = branches[0]
	}

	if branch == "" {
		HandleError(w, r, NewGitError("No branches found", nil))
		return
	}

	ref, err := repo.git.Reference(plumbing.NewBranchReferenceName(branch), true)
	if err != nil {
		HandleError(w, r, NewGitError("Failed to get branch reference", err))
		return
	}

	entries, err := repo.GetTree(path, ref.Hash().String())
	if err != nil {
		HandleError(w, r, NewGitError("Failed to get repository contents", err))
		return
	}

	commits, err := repo.GetCommits(branch, 10)
	if err != nil {
		HandleError(w, r, NewGitError("Failed to get commits", err))
		return
	}

	data := map[string]interface{}{
		"Repo":     repo,
		"Path":     path,
		"Branches": branches,
		"Branch":   branch,
		"Entries":  entries,
		"Commits":  commits,
	}

	if err := s.tmpl.ExecuteTemplate(w, "repo.html", data); err != nil {
		HandleError(w, r, NewInternalError("Failed to render template").WithError(err))
	}
}

// isBinaryFile checks if the content appears to be a binary file
func isBinaryFile(content []byte) bool {
	// Check first 512 bytes for null bytes or non-printable characters
	size := len(content)
	if size > 512 {
		size = 512
	}

	for i := 0; i < size; i++ {
		if content[i] == 0 || (content[i] < 32 && content[i] != '\n' && content[i] != '\r' && content[i] != '\t') {
			return true
		}
	}
	return false
}

func (s *Server) handleViewFile(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 3 {
		HandleError(w, r, NewBadRequestError("Invalid file path"))
		return
	}

	repoName := parts[1]
	repo, ok := s.Repos[repoName]
	if !ok {
		HandleError(w, r, NewNotFoundError("Repository not found").WithDetail(fmt.Sprintf("Repository: %s", repoName)))
		return
	}

	path := strings.Join(parts[2:], "/")

	branch := r.URL.Query().Get("branch")
	if branch == "" {
		branches, err := repo.GetBranches()
		if err != nil {
			HandleError(w, r, NewGitError("Failed to get branches", err))
			return
		}
		if len(branches) == 0 {
			HandleError(w, r, NewGitError("No branches found", nil))
			return
		}
		branch = branches[0]
	}

	content, err := repo.GetFile(path, branch)
	if err != nil {
		if err == plumbing.ErrObjectNotFound {
			HandleError(w, r, NewNotFoundError("File not found"))
		} else {
			HandleError(w, r, NewGitError("Failed to read file", err))
		}
		return
	}

	// Check if file is binary
	if isBinaryFile(content) {
		w.Header().Set("Content-Type", "text/html")
		data := map[string]interface{}{
			"Title":   "Binary File",
			"Message": "This appears to be a binary file and cannot be displayed.",
			"Detail":  fmt.Sprintf("File size: %d bytes\nYou can download or view this file with an appropriate application.", len(content)),
		}
		if err := s.tmpl.ExecuteTemplate(w, "error.html", data); err != nil {
			HandleError(w, r, NewInternalError("Failed to render template").WithError(err))
		}
		return
	}

	// Check file size before rendering
	if int64(len(content)) > config.MaxFileSize {
		w.Header().Set("Content-Type", "text/html")
		data := map[string]interface{}{
			"Title":   "File Too Large",
			"Message": "This file exceeds the maximum size limit for display.",
			"Detail":  fmt.Sprintf("File size: %d bytes\nMaximum allowed: %d bytes", len(content), config.MaxFileSize),
		}
		if err := s.tmpl.ExecuteTemplate(w, "error.html", data); err != nil {
			HandleError(w, r, NewInternalError("Failed to render template").WithError(err))
		}
		return
	}

	// Split content into lines
	lines := strings.Split(string(content), "\n")

	// Parse symbols from content
	symbols := parseSymbols(content)

	data := map[string]interface{}{
		"Repo":    repo,
		"Path":    path,
		"Lines":   lines,
		"Size":    int64(len(content)),
		"Symbols": symbols,
	}

	if err := s.tmpl.ExecuteTemplate(w, "file.html", data); err != nil {
		HandleError(w, r, NewInternalError("Failed to render template").WithError(err))
	}
}

func parseSymbols(content []byte) []Symbol {
	var symbols []Symbol
	lines := strings.Split(string(content), "\n")

	// Common patterns across languages
	patterns := []struct {
		regex string
		typ   string
		icon  string
	}{
		// Functions - catches async, static, public, private, etc.
		{`^[\s]*(?:async\s+)?(?:static\s+)?(?:public\s+)?(?:private\s+)?(?:protected\s+)?(?:function|func)\s+(\w+)`, "function", "ƒ"},

		// Arrow functions with explicit name
		{`^[\s]*(?:export\s+)?(?:const|let|var)\s+(\w+)\s*=\s*(?:async\s+)?\(.*?\)\s*=>`, "function", "ƒ"},

		// Classes/types
		{`^[\s]*(?:export\s+)?(?:abstract\s+)?(?:class|type)\s+(\w+)`, "class", "◇"},

		// Interfaces
		{`^[\s]*(?:export\s+)?interface\s+(\w+)`, "interface", "⬡"},

		// Constants
		{`^[\s]*(?:export\s+)?(?:const|final)\s+(\w+)`, "constant", "□"},

		// Variables
		{`^[\s]*(?:export\s+)?(?:var|let|private|public|protected)\s+(\w+)`, "variable", "○"},

		// Methods
		{`^[\s]*(?:async\s+)?(?:static\s+)?(?:public\s+)?(?:private\s+)?(?:protected\s+)?(?:def|method)\s+(\w+)`, "method", "⌘"},

		// YAML keys (top level)
		{`^(\w+):(?:\s|$)`, "property", "⚑"},

		// YAML anchors
		{`^[\s]*&(\w+)\b`, "anchor", "⚓"},

		// JSON properties (with quotes)
		{`^[\s]*"(\w+)"\s*:`, "property", "⚑"},

		// JSON/YAML nested objects
		{`^[\s]*"?(\w+)"?\s*:\s*{`, "object", "⬡"},

		// JSON/YAML arrays
		{`^[\s]*"?(\w+)"?\s*:\s*\[`, "array", "▤"},
	}

	for lineNum, line := range lines {
		for _, pattern := range patterns {
			re := regexp.MustCompile(pattern.regex)
			if matches := re.FindStringSubmatch(line); matches != nil {
				symbols = append(symbols, Symbol{
					Name: matches[1],
					Type: pattern.typ,
					Icon: pattern.icon,
					Line: lineNum + 1,
				})
			}
		}
	}

	return symbols
}

func (s *Server) setupRoutes() {
	// Static files
	fs := http.FileServer(http.Dir("static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	// Favicon - either serve it or return 404 without logging
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

func main() {
	server, err := NewServer("./repositories")
	if err != nil {
		log.Fatal(err)
	}

	if err := server.ScanRepositories(); err != nil {
		log.Fatal(err)
	}

	server.setupRoutes()

	addr := fmt.Sprintf(":%d", config.Port)
	log.Printf("Server starting on %s (dev mode: %v)", addr, config.DevMode)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatal(err)
	}
}
