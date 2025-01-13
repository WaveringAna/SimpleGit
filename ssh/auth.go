package ssh

import (
	"fmt"

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
		if userKey.PublicKey == providedKey {
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
