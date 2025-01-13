/file config
/file database
/file handlers
/file models
/file static/css
/file templates
/file util
/file main.go

Here's a gameplan for adding Git SSH server support to your SimpleGit application:

1. **Add SSH Server Configuration**
```go
// config/config.go
type Config struct {
    // ... existing fields ...
    SSHPort    int    `env:"SSH_PORT" envDefault:"2222"`
    SSHKeyPath string `env:"SSH_KEY_PATH" envDefault:"ssh/host_key"`
}
```

2. **Create SSH Package Structure**
```
SimpleGit/
└── ssh/
    ├── server.go      // SSH server implementation
    ├── auth.go        // SSH authentication
    └── git.go         // Git command handling
```

3. **Implement SSH Server (ssh/server.go)**
```go
package ssh

import (
    "golang.org/x/crypto/ssh"
    "net"
    "SimpleGit/models"
)

type Server struct {
    config     *ssh.ServerConfig
    userService *models.UserService
    repoPath    string
}

func NewServer(keyPath string, userService *models.UserService, repoPath string) (*Server, error) {
    // Load host keys
    // Set up SSH server config
    // Configure authentication
    // Return server instance
}

func (s *Server) ListenAndServe(port string) error {
    // Start SSH server
    // Handle incoming connections
}
```

4. **Implement Authentication (ssh/auth.go)**
```go
package ssh

import (
    "golang.org/x/crypto/ssh"
    "SimpleGit/models"
)

func (s *Server) authenticateKey(conn ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
    // Verify SSH key against user's stored keys
}

func (s *Server) authenticatePassword(conn ssh.ConnMetadata, password []byte) (*ssh.Permissions, error) {
    // Verify username/password
}
```

5. **Add Git Command Handling (ssh/git.go)**
```go
package ssh

import (
    "os/exec"
    "path/filepath"
)

func (s *Server) handleGitCommand(cmd string, args []string, chans *ssh.Channels) error {
    // Handle git-upload-pack and git-receive-pack commands
}
```

6. **Add User SSH Key Management**
```go
// models/user.go
type SSHKey struct {
    ID          string
    UserID      string
    Name        string
    PublicKey   string
    Fingerprint string
    CreatedAt   time.Time
}

// Add methods to manage SSH keys
```

7. **Update Database Schema**
```go
// database/db.go
func InitDB(dataDir string) (*gorm.DB, error) {
    // ... existing code ...
    if err := db.AutoMigrate(&models.SSHKey{}); err != nil {
        return nil, err
    }
    return db, nil
}
```

8. **Add API Endpoints for Key Management**
```go
// handlers/ssh.go
func (s *Server) handleAddSSHKey(w http.ResponseWriter, r *http.Request)
func (s *Server) handleListSSHKeys(w http.ResponseWriter, r *http.Request)
func (s *Server) handleDeleteSSHKey(w http.ResponseWriter, r *http.Request)
```

9. **Update Main Application**
```go
func main() {
    // ... existing setup ...

    // Initialize SSH server
    sshServer, err := ssh.NewServer(
        config.GlobalConfig.SSHKeyPath,
        userService,
        "./repositories",
    )
    if err != nil {
        log.Fatal(err)
    }

    // Start SSH server in goroutine
    go func() {
        if err := sshServer.ListenAndServe(fmt.Sprintf(":%d", config.GlobalConfig.SSHPort)); err != nil {
            log.Fatal(err)
        }
    }()

    // ... continue with HTTP server setup ...
}
```

10. **Add UI for SSH Key Management**
- Create templates for key management
- Add key management to user settings page
- Update CSS for new UI elements

Implementation Steps:
1. Start with basic SSH server setup
2. Add authentication mechanisms
3. Implement Git command handling
4. Add database support for SSH keys
5. Create API endpoints
6. Add UI components
7. Test with git client
8. Add logging and monitoring
9. Implement security best practices
10. Add documentation

Here's a detailed game plan for implementing Git SSH server support with user key management:

1. **Add SSH Key Management Models**
```go
// models/ssh_key.go
type SSHKey struct {
    ID          string    `gorm:"primarykey"`
    UserID      string    `gorm:"index"`
    Name        string
    PublicKey   string    `gorm:"type:text"`
    Fingerprint string    `gorm:"unique"`
    CreatedAt   time.Time
    UpdatedAt   time.Time
}

// Add SSHKey to DB migrations
func InitDB() {
    db.AutoMigrate(&SSHKey{})
}
```

2. **Create SSH Server Package**
```go
// ssh/server.go
type Server struct {
    config       *ssh.ServerConfig
    userService  *models.UserService
    repoPath     string
    authorizedKeys map[string][]ssh.PublicKey
}

func NewServer(keyPath string, userService *models.UserService) (*Server, error) {
    // Initialize SSH server
    // Load host keys
    // Set up authentication handlers
}
```

3. **Add SSH Key Management API Endpoints**
```go
// handlers/ssh_keys.go
func (s *Server) handleListSSHKeys(w http.ResponseWriter, r *http.Request)
func (s *Server) handleAddSSHKey(w http.ResponseWriter, r *http.Request)
func (s *Server) handleDeleteSSHKey(w http.ResponseWriter, r *http.Request)
```

4. **Add SSH Key Management UI**
```html
<!-- templates/user-ssh-keys.html -->
<div class="ssh-keys">
    <h2>SSH Keys</h2>
    <div class="key-list">
        {{range .SSHKeys}}
        <div class="key-item">
            <div class="key-info">
                <span class="key-name">{{.Name}}</span>
                <span class="key-fingerprint">{{.Fingerprint}}</span>
            </div>
            <button onclick="deleteKey('{{.ID}}')" class="delete-btn">Delete</button>
        </div>
        {{end}}
    </div>
    <form class="add-key-form">
        <input type="text" name="name" placeholder="Key name">
        <textarea name="key" placeholder="Public key content"></textarea>
        <button type="submit">Add Key</button>
    </form>
</div>
```

5. **Implement Push Authentication**
```go
// ssh/auth.go
func (s *Server) authenticateKey(conn ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
    // Check if key is authorized for user
    // Return permissions including user info
}

func (s *Server) authorizeGitOperation(user *models.User, repo string, operation string) bool {
    // Check if user has SSH key and permission for operation
}
```

6. **Update Repository Model**
```go
// models/repository.go
type Repository struct {
    // ... existing fields ...
    WriteUsers []string `gorm:"type:json"` // Users with push access
}

func (r *Repository) CanUserWrite(userID string) bool {
    // Check if user has write access
}
```

7. **Add Admin Management for Repository Permissions**
```html
<!-- templates/admin-repo-permissions.html -->
<div class="repo-permissions">
    <h2>Repository Access</h2>
    <div class="user-list">
        {{range .Users}}
        <div class="user-item">
            <span>{{.Email}}</span>
            <label>
                <input type="checkbox"
                       onchange="updateAccess('{{.ID}}')"
                       {{if .HasAccess}}checked{{end}}>
                Write Access
            </label>
        </div>
        {{end}}
    </div>
</div>
```

8. **Implement Git Operations Over SSH**
```go
// ssh/git.go
func (s *Server) handleGitCommand(cmd string, args []string, channel ssh.Channel) error {
    // Handle git-upload-pack and git-receive-pack
    // Check permissions before allowing push
}
```

9. **Update Configuration**
```go
// config/config.go
type Config struct {
    // ... existing fields ...
    SSHPort     int    `env:"SSH_PORT" envDefault:"2222"`
    SSHHostKey  string `env:"SSH_HOST_KEY" envDefault:"ssh/host_key"`
}
```

10. **Update Main Application**
```go
func main() {
    // ... existing setup ...

    // Initialize SSH server
    sshServer, err := ssh.NewServer(
        config.GlobalConfig.SSHHostKey,
        userService,
        "./repositories",
    )
    if err != nil {
        log.Fatal(err)
    }

    // Start SSH server
    go func() {
        if err := sshServer.ListenAndServe(); err != nil {
            log.Fatal(err)
        }
    }()
}
```

Implementation Steps:
1. Start with database schema updates
2. Add SSH key management endpoints
3. Create SSH server implementation
4. Add UI for key management
5. Implement repository access control
6. Add admin interface for permissions
7. Test with git client
8. Add documentation

Would you like me to provide detailed implementation for any of these components?
