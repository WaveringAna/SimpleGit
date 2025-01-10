package handlers

import (
	"SimpleGit/models"
	"net/http"
	"strings"
	"time"

	"github.com/go-git/go-git/v5/plumbing"
)

type CommitInfo struct {
	Hash      string    `json:"hash"`
	Author    string    `json:"author"`
	Email     string    `json:"email"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
}

func (s *Server) handleViewCommit(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 3 {
		models.HandleError(w, r, models.NewBadRequestError("Invalid commit path"))
		return
	}

	repoName := parts[1]
	commitHash := parts[2]

	repo, ok := s.Repos[repoName]
	if !ok {
		models.HandleError(w, r, models.NewNotFoundError("Repository not found"))
		return
	}

	gitRepo, err := repo.Git()
	if err != nil {
		models.HandleError(w, r, models.NewGitError("Failed to open repository", err))
		return
	}

	hash := plumbing.NewHash(commitHash)
	commit, err := gitRepo.CommitObject(hash)
	if err != nil {
		models.HandleError(w, r, models.NewNotFoundError("Commit not found"))
		return
	}

	data := map[string]interface{}{
		"Repo": repo,
		"Commit": CommitInfo{
			Hash:      commit.Hash.String(),
			Author:    commit.Author.Name,
			Email:     commit.Author.Email,
			Message:   commit.Message,
			Timestamp: commit.Author.When,
		},
	}

	if err := s.tmpl.ExecuteTemplate(w, "commit.html", s.addCommonData(r, data)); err != nil {
		models.HandleError(w, r, models.NewInternalError("Failed to render template").WithError(err))
	}
}
