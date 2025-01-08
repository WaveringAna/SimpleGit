//models/repository.go

package models

import (
	config "SimpleGit/config"
	"errors"
	"fmt"
	"path/filepath"
	"sort"
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
	Size        int64     `json:"size"`
	git         *git.Repository
}

type TreeEntry struct {
	Name    string `json:"name"`
	Path    string `json:"path"`
	Type    string `json:"type"`
	Size    int64  `json:"size"`
	Commit  string `json:"commit"`
	Message string `json:"message"`
}

type Commit struct {
	Hash      string
	Message   string
	Author    string
	Email     string
	Date      time.Time
	ShortHash string // First 7 characters of the hash
}

type CommitInfo struct {
	Hash      string    `json:"hash"`
	Author    string    `json:"author"`
	Email     string    `json:"email"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
}

func (r *Repository) initGit() error {
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

func (r *Repository) Git() (*git.Repository, error) {
	if err := r.initGit(); err != nil {
		return nil, err
	}
	return r.git, nil
}

func (r *Repository) GetTree(path, ref string) ([]TreeEntry, error) {
	if err := r.initGit(); err != nil {
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
	if err := r.initGit(); err != nil {
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
	if err := r.initGit(); err != nil {
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

func (r *Repository) GetBranches() ([]string, error) {
	if err := r.initGit(); err != nil {
		return nil, err
	}

	// If repository is empty, return empty slice
	if r.git == nil {
		return []string{}, nil
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
	if err != nil {
		return nil, err
	}

	return branches, nil
}

func (r *Repository) CloneURL() string {
	// For development/local setup
	if config.GlobalConfig.Domain == "localhost" {
		return fmt.Sprintf("http://localhost:%d/repo/%s.git",
			config.GlobalConfig.Port,
			r.Name)
	}

	// For production setup
	return fmt.Sprintf("http://%s/repo/%s.git",
		config.GlobalConfig.Domain,
		r.Name)
}
