# Development Guide

This guide covers everything you need to know to contribute to the Workflow Orchestrator project.

## Table of Contents

- [Prerequisites](#prerequisites)
- [Getting Started](#getting-started)
- [Project Structure](#project-structure)
- [Development Workflow](#development-workflow)
- [Testing](#testing)
- [Code Quality](#code-quality)
- [Docker Development](#docker-development)
- [Troubleshooting](#troubleshooting)

## Prerequisites

### Required

- **Go 1.21+**: [Installation Guide](https://golang.org/doc/install)
- **Docker**: [Installation Guide](https://docs.docker.com/get-docker/)
- **Docker Compose**: [Installation Guide](https://docs.docker.com/compose/install/)

### Recommended

- **Make**: For running convenience commands
- **golangci-lint**: For code linting
- **Air**: For hot-reload development
- **pre-commit**: For git hooks

## Getting Started

### 1. Clone the Repository

```bash
git clone https://github.com/therealutkarshpriyadarshi/dag.git
cd dag
```

### 2. Install Development Tools

```bash
make install-tools
```

This will install:
- golangci-lint
- Air (hot-reload tool)

### 3. Setup Pre-commit Hooks (Optional)

```bash
./scripts/install-hooks.sh
```

### 4. Copy Environment Configuration

```bash
cp .env.example .env
```

### 5. Start Development Environment

```bash
make docker-up
```

This starts:
- PostgreSQL (port 5432)
- Redis (port 6379)
- NATS (port 4222)
- Prometheus (port 9090)
- Grafana (port 3000)

## Project Structure

```
.
├── cmd/                    # Application entrypoints
│   ├── server/            # API server
│   ├── worker/            # Task worker
│   └── scheduler/         # Scheduler service
├── internal/              # Private application code
│   ├── dag/              # DAG logic and validation
│   ├── executor/         # Task execution
│   ├── scheduler/        # Scheduling logic
│   ├── state/            # State management
│   └── storage/          # Database layer
├── pkg/                   # Public libraries
│   ├── api/              # API handlers
│   └── models/           # Shared data models
├── web/                   # Frontend (Future)
├── deployments/           # Deployment configs
│   ├── Dockerfile.*      # Docker build files
│   ├── prometheus/       # Prometheus config
│   └── grafana/          # Grafana dashboards
├── docs/                  # Documentation
├── scripts/               # Utility scripts
├── .github/workflows/     # CI/CD pipelines
├── Makefile              # Build commands
├── docker-compose.yml    # Local dev environment
├── .golangci.yml         # Linter configuration
└── .air.toml             # Hot-reload configuration
```

## Development Workflow

### Running Services Locally

#### Option 1: Run with Go (Recommended for Development)

```bash
# Terminal 1 - Run server
make run-server

# Terminal 2 - Run worker
make run-worker

# Terminal 3 - Run scheduler
make run-scheduler
```

#### Option 2: Run with Hot Reload

```bash
air
```

#### Option 3: Run in Docker

```bash
make docker-up
```

### Making Changes

1. **Create a Feature Branch**
   ```bash
   git checkout -b feature/your-feature-name
   ```

2. **Make Your Changes**
   - Write code following Go best practices
   - Add tests for new functionality
   - Update documentation as needed

3. **Run Quality Checks**
   ```bash
   make dev
   ```
   This runs:
   - `go fmt` - Format code
   - `go vet` - Static analysis
   - `golangci-lint` - Comprehensive linting
   - `go test` - Run tests

4. **Commit Your Changes**
   ```bash
   git add .
   git commit -m "feat: add new feature"
   ```

   Follow [Conventional Commits](https://www.conventionalcommits.org/):
   - `feat:` - New feature
   - `fix:` - Bug fix
   - `docs:` - Documentation changes
   - `test:` - Test additions or changes
   - `refactor:` - Code refactoring
   - `chore:` - Maintenance tasks

5. **Push and Create PR**
   ```bash
   git push origin feature/your-feature-name
   ```

## Testing

### Running Tests

```bash
# Run all tests
make test

# Run specific package tests
go test ./internal/dag/...

# Run with race detection
go test -race ./...

# Run with coverage
make test-coverage
```

### Writing Tests

Example test structure:

```go
package dag

import "testing"

func TestValidate_ValidDAG(t *testing.T) {
    validator := NewValidator()
    dag := &models.DAG{
        Name: "test-dag",
        Tasks: []models.Task{
            {ID: "task1", Name: "Task 1", Type: models.TaskTypeBash},
        },
    }

    err := validator.Validate(dag)
    if err != nil {
        t.Errorf("Expected no error, got: %v", err)
    }
}
```

### Test Coverage Guidelines

- Aim for >80% code coverage
- Focus on critical paths and edge cases
- Use table-driven tests for multiple scenarios
- Mock external dependencies

## Code Quality

### Formatting

```bash
# Format all Go files
make fmt

# Or use gofmt directly
gofmt -w .
```

### Linting

```bash
# Run golangci-lint
make lint

# Fix auto-fixable issues
golangci-lint run --fix
```

### Static Analysis

```bash
# Run go vet
make vet

# Or directly
go vet ./...
```

### Pre-commit Checks

All these run automatically if you installed pre-commit hooks:
- Trailing whitespace removal
- End-of-file fixer
- YAML validation
- Go formatting
- Go imports
- Go vet
- golangci-lint

## Docker Development

### Building Docker Images

```bash
# Build all images
make docker-build

# Build specific image
docker build -f deployments/Dockerfile.server -t workflow-server .
```

### Managing Services

```bash
# Start all services
make docker-up

# Stop all services
make docker-down

# Stop and remove volumes
make docker-down-volumes

# View logs
make docker-logs

# Rebuild and restart
make docker-rebuild
```

### Accessing Services

| Service | URL | Credentials |
|---------|-----|-------------|
| API Server | http://localhost:8080 | - |
| Grafana | http://localhost:3000 | admin/admin |
| Prometheus | http://localhost:9090 | - |
| NATS Monitoring | http://localhost:8222 | - |
| PostgreSQL | localhost:5432 | workflow/workflow_dev_password |
| Redis | localhost:6379 | - |

## Troubleshooting

### Common Issues

#### Port Already in Use

```bash
# Find process using port
lsof -i :8080

# Kill process
kill -9 <PID>
```

#### Docker Services Won't Start

```bash
# Remove all containers and volumes
make docker-down-volumes

# Rebuild and start
make docker-rebuild
```

#### Tests Failing

```bash
# Clean build cache
go clean -testcache

# Re-run tests
make test
```

#### Import Issues

```bash
# Tidy dependencies
make tidy

# Verify dependencies
go mod verify
```

### Debug Mode

Enable debug logging:

```bash
# Set in .env
ENV=development
LOG_LEVEL=debug

# Or set when running
ENV=development LOG_LEVEL=debug make run-server
```

### Database Issues

```bash
# Connect to PostgreSQL
docker exec -it workflow-postgres psql -U workflow -d workflow_orchestrator

# View tables
\dt

# Exit
\q
```

### Redis Issues

```bash
# Connect to Redis
docker exec -it workflow-redis redis-cli

# Check keys
KEYS *

# Exit
exit
```

## Performance Profiling

### CPU Profiling

```go
import _ "net/http/pprof"

// Access profiles at http://localhost:8080/debug/pprof/
```

### Memory Profiling

```bash
# Generate memory profile
go test -memprofile=mem.out ./...

# Analyze
go tool pprof mem.out
```

## Contributing Guidelines

1. **Code Style**
   - Follow [Effective Go](https://golang.org/doc/effective_go)
   - Use meaningful variable names
   - Write descriptive comments for exported functions
   - Keep functions small and focused

2. **Testing**
   - Write tests for all new features
   - Ensure tests pass before submitting PR
   - Include both positive and negative test cases

3. **Documentation**
   - Update README.md for user-facing changes
   - Update DEVELOPMENT.md for development changes
   - Add godoc comments for exported types and functions

4. **Commit Messages**
   - Use conventional commits format
   - Be descriptive but concise
   - Reference issues when applicable

## Additional Resources

- [Go Documentation](https://golang.org/doc/)
- [Gin Framework](https://gin-gonic.com/docs/)
- [PostgreSQL Documentation](https://www.postgresql.org/docs/)
- [NATS Documentation](https://docs.nats.io/)
- [Docker Documentation](https://docs.docker.com/)

## Getting Help

- Check existing [Issues](https://github.com/therealutkarshpriyadarshi/dag/issues)
- Create a new issue with detailed information
- Join discussions in Pull Requests
