package ssh

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

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
        filepath.Join(s.repoPath, repoPath+".git"),    // with .git suffix
        filepath.Join(s.repoPath, repoPath),           // without .git suffix
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

    // Verify directory exists and is accessible
    if _, err := os.Stat(repoPath); err != nil {
        return fmt.Errorf("cannot access repository path: %w", err)
    }

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

    return nil
}
