# SimpleGit

SimpleGit is a lightweight, self-hosted Git server written in Go that provides a clean web interface for browsing repositories and Git operations over HTTP and SSH protocols.

![RepoView](readme_img/repoview.png)

![FileView](readme_img/fileview.png)

## Features

- **Web Interface**
  - Repository browser with syntax highlighting
  - Commit history viewer
  - File content viewer with symbol navigation
  - Support for multiple branches
  - Admin dashboard
  - Dark theme with Nord color scheme

- **Git Operations**
  - HTTP Git protocol support (clone, push, pull)
  - SSH Git protocol support
  - User-managed SSH key authentication
  - Support for bare repositories

- **User Management**
  - User authentication with JWT
  - Role-based access control (Admin/User)
  - SSH key management per user
  - Admin user creation system

- **Technical Features**
  - SQLite database for user management
  - Docker and Docker Compose support
  - Configurable through environment variables or JSON
  - Syntax highlighting for multiple languages
  - Symbol detection and navigation for code files

## Quick Start

### Manual Setup

1. Install Go 1.21 or later
2. Clone the repository:
```bash
git clone https://github.com/yourusername/simplegit.git
cd simplegit
```

3. Create the configuration file:
```bash
cp config.json.example config.json
```

4. Edit `config.json` with your settings

5. Build and run:
```bash
go build
./simplegit
```

## Configuration

### Configuration

The server can be configured through environment variables or a JSON config file. Environment variables take precedence over the JSON configuration.

Environment variables:

- `SIMPLEGIT_DEV_MODE`: Enable development mode (boolean)
- `SIMPLEGIT_PORT`: HTTP server port
- `SIMPLEGIT_SSH_PORT`: SSH server port (default: 2222)
- `SIMPLEGIT_DATE_FORMAT`: Date format string
- `SIMPLEGIT_MAX_FILE_SIZE`: Maximum file size for web display in bytes
- `SIMPLEGIT_DATA_DIR`: Directory for database and data storage
- `SIMPLEGIT_JWT_SECRET`: Secret key for JWT tokens
- `SIMPLEGIT_DOMAIN`: Server domain name (default: localhost)
- `SIMPLEGIT_SSH_KEY_PATH`: Path to SSH host key
- `SIMPLEGIT_REPO_PATH`: Path to store Git repositories
- `SIMPLEGIT_DB_PATH`: Path to SQLite database file

### Docker Configuration

The included Docker setup provides:
- Multi-stage build for minimal image size
- Volume mounting for repositories and data
- Environment-based configuration
- Automatic repository directory creation
- Non-root user for security

Example docker-compose.yml:
```yaml
version: "3.8"

services:
  simplegit:
    build: .
    ports:
      - "3000:3000"
      - "2222:2222"
    volumes:
      - ./repositories:/app/repositories
      - ./data:/app/data
    environment:
      - SIMPLEGIT_PORT=3000
      - SIMPLEGIT_SSH_PORT=2222
      - SIMPLEGIT_JWT_SECRET=your-secure-secret
      - SIMPLEGIT_DOMAIN=git.yourdomain.com
      - SIMPLEGIT_MAX_FILE_SIZE=10485760
    restart: unless-stopped
    deploy:
      resources:
        limits:
          memory: 512M
        reservations:
          memory: 128M
```

Custom Docker run command:
```bash
docker run -d \
  --name simplegit \
  -p 3000:3000 \
  -p 2222:2222 \
  -v ./repositories:/app/repositories \
  -v ./data:/app/data \
  -e SIMPLEGIT_PORT=3000 \
  -e SIMPLEGIT_SSH_PORT=2222 \
  -e SIMPLEGIT_JWT_SECRET=your-secure-secret \
  -e SIMPLEGIT_DOMAIN=localhost \
  -e SIMPLEGIT_DATA_DIR=/app/data \
  -e SIMPLEGIT_REPO_PATH=/app/repositories \
  -e SIMPLEGIT_DB_PATH=/app/data/githost.db \
  simplegit
```

## Initial Setup

1. Start the server for the first time
2. Look for the admin setup token in the console output
3. Visit http://localhost:3000/setup-admin
4. Use the setup token to create the admin account
5. Log in with the admin account
6. Create repositories and users through the admin interface

## Repository Management

### Create a Repository

1. Log in as admin
2. Go to Admin → Repositories
3. Click "Create Repository"
4. Enter repository name and description

### Clone a Repository

HTTP:
```bash
git clone http://localhost:3000/repo/example.git
```

SSH:
```bash
git clone ssh://git@localhost:2222/example.git
```

## Development

### Project Structure

```
simplegit/
├── config/        # Configuration handling
├── database/      # Database initialization
├── handlers/      # HTTP request handlers
├── models/        # Data models and business logic
├── ssh/          # SSH server implementation
├── static/       # Static web assets
├── templates/    # HTML templates
└── utils/        # Utility functions
```

### Technology Stack

- Backend:
  - Go 1.21+
  - go-git (Git operations)
  - GORM (ORM)
  - SQLite (Database)
  - Gorilla Mux (Routing)
  - JWT (Authentication)

- Frontend:
  - HTMX (Dynamic updates)
  - Highlight.js (Syntax highlighting)
  - Font Awesome (Icons)
  - Nord theme (Color scheme)

## Contributing

1. Fork the repository
2. Create a feature branch
3. Commit your changes
4. Push to the branch
5. Create a Pull Request

## License

This project is released under the Unlicense. See the LICENSE file for details.

## Acknowledgments

Built with these excellent libraries:
- highlight.js - BSD Three Clause License
- htmx - BSD Zero Clause License
- go-git - Apache License 2.0