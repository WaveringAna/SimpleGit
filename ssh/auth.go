package ssh

import (
	"fmt"
	"strings"

	"golang.org/x/crypto/ssh"
)

func (s *Server) authenticateKey(conn ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
	// Use username instead of email for SSH authentication
	username := conn.User()
	user, err := s.userService.GetUserByUsername(username)
	if err != nil {
		return nil, fmt.Errorf("authentication failed")
	}

	// Get user's SSH keys
	keys, err := s.userService.GetUserSSHKeys(user.ID)
	if err != nil {
		return nil, fmt.Errorf("authentication failed")
	}

	// Convert provided key to authorized key format
	providedKey := string(ssh.MarshalAuthorizedKey(key))

	// Check if the key matches any of the user's keys
	for _, userKey := range keys {
		// Split the stored key and take only the type and key parts
		storedKeyParts := strings.Fields(userKey.PublicKey)
		if len(storedKeyParts) < 2 {
			continue // Invalid key format
		}
		storedKey := storedKeyParts[0] + " " + storedKeyParts[1]

		// Split the provided key the same way
		providedKeyParts := strings.Fields(providedKey)
		if len(providedKeyParts) < 2 {
			continue // Invalid key format
		}
		trimmedProvidedKey := providedKeyParts[0] + " " + providedKeyParts[1]

		if storedKey == trimmedProvidedKey {
			return &ssh.Permissions{
				Extensions: map[string]string{
					"user_id":  user.ID,
					"username": user.Username,
				},
			}, nil
		}
	}
	return nil, fmt.Errorf("authentication failed")
}

func (s *Server) authenticatePassword(conn ssh.ConnMetadata, password []byte) (*ssh.Permissions, error) {
	user, _, err := s.userService.AuthenticateUser(conn.User(), string(password))
	if err != nil {
		return nil, fmt.Errorf("authentication failed")
	}

	return &ssh.Permissions{
		Extensions: map[string]string{
			"user_id": user.ID,
		},
	}, nil
}

func (s *Server) keyboardInteractiveCallback(conn ssh.ConnMetadata, client ssh.KeyboardInteractiveChallenge) (*ssh.Permissions, error) {
	// Prompt for password using keyboard-interactive
	answers, err := client("", "", []string{"Password:"}, []bool{false})
	if err != nil {
		return nil, fmt.Errorf("authentication failed")
	}

	if len(answers) != 1 {
		return nil, fmt.Errorf("authentication failed")
	}

	// Use the same authentication logic as password auth
	user, _, err := s.userService.AuthenticateUser(conn.User(), answers[0])
	if err != nil {
		return nil, fmt.Errorf("authentication failed")
	}

	return &ssh.Permissions{
		Extensions: map[string]string{
			"user_id":  user.ID,
			"username": user.Username,
		},
	}, nil
}
