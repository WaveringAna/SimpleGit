// handlers/git.go
package handlers

import (
	"SimpleGit/models"
	"fmt"
	"net/http"
	"os/exec"
	"strings"
)

func (s *Server) handleGitRequest(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 2 {
		http.Error(w, "Invalid repository path", http.StatusBadRequest)
		return
	}

	repoName := strings.TrimSuffix(parts[1], ".git")
	repo, ok := s.Repos[repoName]
	if !ok {
		http.NotFound(w, r)
		return
	}

	switch {
	case strings.HasSuffix(r.URL.Path, "/info/refs"):
		s.handleInfoRefs(w, r, repo)
	case strings.HasSuffix(r.URL.Path, "/git-upload-pack"):
		s.handleUploadPack(w, r, repo)
	default:
		http.NotFound(w, r)
	}
}

func (s *Server) handleInfoRefs(w http.ResponseWriter, r *http.Request, repo *models.Repository) {
	service := r.URL.Query().Get("service")
	if service != "git-upload-pack" && service != "git-receive-pack" {
		http.Error(w, "Service not available", http.StatusForbidden)
		return
	}

	w.Header().Set("Content-Type", fmt.Sprintf("application/x-%s-advertisement", service))
	w.WriteHeader(http.StatusOK)

	// Write packet-line format header
	fmt.Fprintf(w, "%04x# service=%s\n", len(fmt.Sprintf("# service=%s\n", service))+4, service)
	fmt.Fprintf(w, "0000")

	cmd := exec.Command("git", strings.TrimPrefix(service, "git-"), "--advertise-refs", ".")
	cmd.Dir = repo.Path
	cmd.Stdout = w
	cmd.Run()
}

func (s *Server) handleUploadPack(w http.ResponseWriter, r *http.Request, repo *models.Repository) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/x-git-upload-pack-result")

	cmd := exec.Command("git", "upload-pack", "--stateless-rpc", ".")
	cmd.Dir = repo.Path
	cmd.Stdin = r.Body
	cmd.Stdout = w
	cmd.Run()
}

func (s *Server) handleReceivePack(w http.ResponseWriter, r *http.Request, repo *models.Repository) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/x-git-receive-pack-result")

	cmd := exec.Command("git", "receive-pack", "--stateless-rpc", ".")
	cmd.Dir = repo.Path
	cmd.Stdin = r.Body
	cmd.Stdout = w
	cmd.Run()
}
