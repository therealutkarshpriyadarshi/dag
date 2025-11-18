.PHONY: help build test clean docker-up docker-down run-server run-worker run-scheduler lint fmt vet tidy install-tools

# Variables
GO := go
DOCKER_COMPOSE := docker-compose
GOLANGCI_LINT := golangci-lint

# Build output directory
BUILD_DIR := bin

# Binary names
SERVER_BIN := $(BUILD_DIR)/server
WORKER_BIN := $(BUILD_DIR)/worker
SCHEDULER_BIN := $(BUILD_DIR)/scheduler

## help: Display this help message
help:
	@echo "Available targets:"
	@grep -E '^## ' Makefile | sed 's/^## /  /'

## install-tools: Install development tools
install-tools:
	@echo "Installing development tools..."
	$(GO) install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	$(GO) install github.com/cosmtrek/air@latest
	@echo "Tools installed successfully"

## tidy: Tidy go.mod and go.sum
tidy:
	@echo "Tidying go modules..."
	$(GO) mod tidy
	@echo "Done"

## fmt: Format Go code
fmt:
	@echo "Formatting Go code..."
	$(GO) fmt ./...
	@echo "Done"

## vet: Run go vet
vet:
	@echo "Running go vet..."
	$(GO) vet ./...
	@echo "Done"

## lint: Run golangci-lint
lint:
	@echo "Running golangci-lint..."
	$(GOLANGCI_LINT) run --timeout=5m
	@echo "Done"

## test: Run all tests
test:
	@echo "Running tests..."
	$(GO) test -v -race -coverprofile=coverage.out ./...
	@echo "Done"

## test-coverage: Run tests with coverage report
test-coverage: test
	@echo "Generating coverage report..."
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

## build: Build all binaries
build: build-server build-worker build-scheduler

## build-server: Build server binary
build-server:
	@echo "Building server..."
	@mkdir -p $(BUILD_DIR)
	$(GO) build -o $(SERVER_BIN) ./cmd/server
	@echo "Server built: $(SERVER_BIN)"

## build-worker: Build worker binary
build-worker:
	@echo "Building worker..."
	@mkdir -p $(BUILD_DIR)
	$(GO) build -o $(WORKER_BIN) ./cmd/worker
	@echo "Worker built: $(WORKER_BIN)"

## build-scheduler: Build scheduler binary
build-scheduler:
	@echo "Building scheduler..."
	@mkdir -p $(BUILD_DIR)
	$(GO) build -o $(SCHEDULER_BIN) ./cmd/scheduler
	@echo "Scheduler built: $(SCHEDULER_BIN)"

## run-server: Run server locally
run-server:
	@echo "Running server..."
	$(GO) run ./cmd/server

## run-worker: Run worker locally
run-worker:
	@echo "Running worker..."
	$(GO) run ./cmd/worker

## run-scheduler: Run scheduler locally
run-scheduler:
	@echo "Running scheduler..."
	$(GO) run ./cmd/scheduler

## docker-up: Start all services with docker-compose
docker-up:
	@echo "Starting services..."
	$(DOCKER_COMPOSE) up -d
	@echo "Services started. Access:"
	@echo "  - Server: http://localhost:8080"
	@echo "  - Grafana: http://localhost:3000 (admin/admin)"
	@echo "  - Prometheus: http://localhost:9090"
	@echo "  - NATS Monitoring: http://localhost:8222"

## docker-down: Stop all services
docker-down:
	@echo "Stopping services..."
	$(DOCKER_COMPOSE) down
	@echo "Services stopped"

## docker-down-volumes: Stop all services and remove volumes
docker-down-volumes:
	@echo "Stopping services and removing volumes..."
	$(DOCKER_COMPOSE) down -v
	@echo "Services stopped and volumes removed"

## docker-logs: Show logs from all services
docker-logs:
	$(DOCKER_COMPOSE) logs -f

## docker-build: Build docker images
docker-build:
	@echo "Building Docker images..."
	$(DOCKER_COMPOSE) build
	@echo "Docker images built"

## docker-rebuild: Rebuild and restart services
docker-rebuild: docker-down docker-build docker-up

## clean: Remove build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html
	@echo "Clean complete"

## dev: Run all checks (fmt, vet, lint, test)
dev: fmt vet lint test

## ci: Run CI checks
ci: tidy vet lint test

## all: Build everything
all: clean tidy fmt vet lint test build

.DEFAULT_GOAL := help
