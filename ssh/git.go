package ssh

import (
	"SimpleGit/models"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

func (s *Server) handleChannel(channel ssh.Channel, requests <-chan *ssh.Request) {
	// Print the absolute path of the repository root
	absRepoPath, _ := filepath.Abs(s.repoPath)
	log.Printf("Repository root directory: %s", absRepoPath)

	defer channel.Close()

	for req := range requests {
		if req.Type != "exec" {
			if req.WantReply {
				req.Reply(false, nil)
			}
			continue
		}

		log.Printf("Raw payload: %v", req.Payload)
		cmdStr := string(req.Payload[4:])
		log.Printf("Command string: %s", cmdStr)

		if req.WantReply {
			req.Reply(true, nil)
		}

		parts := strings.SplitN(cmdStr, " ", 2)
		if len(parts) != 2 {
			fmt.Fprintf(channel, "Invalid command format\n")
			channel.SendRequest("exit-status", false, []byte{0, 0, 0, 1})
			return
		}

		cmd := parts[0]
		// Clean up repository path
		repoPath := strings.Trim(parts[1], "'\" /")
		repoPath = strings.TrimSuffix(repoPath, ".git")

		log.Printf("Cleaned command: %s, repo path: %s", cmd, repoPath)

		if err := s.handleGitCommand(cmd, repoPath, channel); err != nil {
			log.Printf("Git command error: %v", err)
			fmt.Fprintf(channel, "Error: %v\n", err)
			channel.SendRequest("exit-status", false, []byte{0, 0, 0, 1})
			return
		}

		channel.SendRequest("exit-status", false, []byte{0, 0, 0, 0})
		return
	}
}

func (s *Server) handleGitCommand(cmd, repoPath string, channel ssh.Channel) error {
	log.Printf("Handling git command: %s %s", cmd, repoPath)

	// Try different repository path variations
	possiblePaths := []string{
		filepath.Join(s.repoPath, repoPath+".git"), // with .git suffix
		filepath.Join(s.repoPath, repoPath),        // without .git suffix
	}

	// Print all possible paths being checked
	for i, path := range possiblePaths {
		absPath, _ := filepath.Abs(path)
		log.Printf("Checking path %d: %s", i+1, absPath)
		if _, err := os.Stat(absPath); !os.IsNotExist(err) {
			log.Printf("Path exists: %s", absPath)
		} else {
			log.Printf("Path does not exist: %s", absPath)
		}
	}

	var fullRepoPath string
	var exists bool
	for _, path := range possiblePaths {
		absPath, _ := filepath.Abs(path)
		if _, err := os.Stat(absPath); !os.IsNotExist(err) {
			// Create a temporary Repository object to use EnsureBare
			repo := &models.Repository{
				Path: absPath,
			}
			if err := repo.EnsureBare(); err != nil {
				return fmt.Errorf("failed to ensure repository is bare: %w", err)
			}
			fullRepoPath = absPath
			exists = true
			break
		}
	}

	if !exists {
		return fmt.Errorf("repository not found: %s (checked paths: %v)", repoPath, possiblePaths)
	}

	log.Printf("Using repository path: %s", fullRepoPath)

	switch cmd {
	case "git-upload-pack", "git-receive-pack":
		return s.executeGitCommand(cmd, fullRepoPath, channel)
	default:
		return fmt.Errorf("unsupported command: %s", cmd)
	}
}

func (s *Server) executeGitCommand(cmd string, repoPath string, channel ssh.Channel) error {
	log.Printf("Executing git command: %s %s", cmd, repoPath)

	gitCmd := exec.Command(cmd, repoPath)
	gitCmd.Dir = repoPath
	gitCmd.Stdin = channel
	gitCmd.Stdout = channel
	gitCmd.Stderr = channel.Stderr()

	log.Printf("Full command: %s %v in directory %s", gitCmd.Path, gitCmd.Args, gitCmd.Dir)

	// Print current working directory
	cwd, _ := os.Getwd()
	log.Printf("Current working directory: %s", cwd)

	if err := gitCmd.Run(); err != nil {
		log.Printf("Command execution error: %v", err)
		return fmt.Errorf("git command failed: %w", err)
	}

	// If this was a receive-pack (push), wait a moment for git to finish writing refs
	// and ensure the repository is in bare format
	if cmd == "git-receive-pack" {
		time.Sleep(100 * time.Millisecond) // Wait for git to finish writing refs

		// Create a temporary Repository object to use EnsureBare
		repo := &models.Repository{
			Path: repoPath,
		}
		if err := repo.EnsureBare(); err != nil {
			log.Printf("Warning: Failed to ensure repository is bare: %v", err)
		}

		// Notify about repository changes
		if s.onUpdate != nil {
			s.onUpdate()
			// Wait another moment for the scan to complete
			time.Sleep(100 * time.Millisecond)
		}
	}

	return nil
}
