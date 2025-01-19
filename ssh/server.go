package ssh

import (
	"SimpleGit/config"

	"SimpleGit/models"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"

	"golang.org/x/crypto/ssh"
)

type Server struct {
	config      *ssh.ServerConfig
	userService *models.UserService
	repoPath    string
	onUpdate    func()
}

func NewServer(repoPath string, userService *models.UserService, onUpdate func()) (*Server, error) {
	server := &Server{
		userService: userService,
		repoPath:    repoPath,
		onUpdate:    onUpdate,
	}

	// TODO figure out how to default to PublicKeyCallback while allowing KeyboardInteractiveCallback
	sshConfig := &ssh.ServerConfig{
		PublicKeyCallback: server.authenticateKey,
		//KeyboardInteractiveCallback: server.keyboardInteractiveCallback,
		ServerVersion: "SSH-2.0-SimpleGit",
	}

	keyPath := config.GlobalConfig.SSHKeyPath
	if err := os.MkdirAll(filepath.Dir(keyPath), 0700); err != nil {
		return nil, fmt.Errorf("failed to create SSH key directory: %w", err)
	}

	hostKey, err := loadOrGenerateHostKey(keyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to setup host key: %w", err)
	}

	sshConfig.AddHostKey(hostKey)
	server.config = sshConfig
	return server, nil
}

func (s *Server) ListenAndServe(addr string) error {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", addr, err)
	}
	defer listener.Close()

	for {
		conn, err := listener.Accept()
		if err != nil {
			return fmt.Errorf("failed to accept connection: %w", err)
		}

		go s.handleConnection(conn)
	}
}

func (s *Server) handleConnection(conn net.Conn) {
	defer conn.Close()

	// Perform SSH handshake
	sshConn, chans, reqs, err := ssh.NewServerConn(conn, s.config)
	if err != nil {
		fmt.Printf("Failed to handshake: %v\n", err)
		return
	}
	defer sshConn.Close()

	// Handle incoming requests
	go ssh.DiscardRequests(reqs)

	// Handle channels
	for newChannel := range chans {
		if newChannel.ChannelType() != "session" {
			newChannel.Reject(ssh.UnknownChannelType, "unknown channel type")
			continue
		}

		channel, requests, err := newChannel.Accept()
		if err != nil {
			fmt.Printf("Failed to accept channel: %v\n", err)
			continue
		}

		go s.handleChannel(channel, requests)
	}
}

func loadOrGenerateHostKey(path string) (ssh.Signer, error) {
	keyBytes, err := ioutil.ReadFile(path)
	if err == nil {
		return ssh.ParsePrivateKey(keyBytes)
	}

	if !os.IsNotExist(err) {
		return nil, err
	}

	// Generate new key
	key, err := generateHostKey()
	if err != nil {
		return nil, err
	}

	// Save the key
	if err := ioutil.WriteFile(path, key, 0600); err != nil {
		return nil, err
	}

	return ssh.ParsePrivateKey(key)
}

func generateHostKey() ([]byte, error) {
	// Generate RSA key
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}

	// Convert to PEM format
	privateKey := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	}

	return pem.EncodeToMemory(privateKey), nil
}
