# Workflow Orchestrator

A production-grade workflow orchestration system built in Go, similar to Apache Airflow and Temporal, with distributed execution, web UI, and comprehensive monitoring.

[![CI](https://github.com/therealutkarshpriyadarshi/dag/actions/workflows/ci.yml/badge.svg)](https://github.com/therealutkarshpriyadarshi/dag/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/therealutkarshpriyadarshi/dag)](https://goreportcard.com/report/github.com/therealutkarshpriyadarshi/dag)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

## Features

- **DAG-based Workflows**: Define workflows as Directed Acyclic Graphs (DAGs) with task dependencies
- **Distributed Execution**: Scale across multiple worker nodes
- **Real-time Monitoring**: Track workflow execution with live updates
- **Retry Logic**: Configurable retry strategies with exponential backoff
- **Multiple Task Types**: Bash, HTTP, Python, and Go task executors
- **REST API**: Complete API for programmatic control
- **Web UI**: Interactive dashboard for monitoring and management (Coming Soon)
- **Observability**: Built-in metrics (Prometheus), logging, and tracing

## Target Capabilities

- Handle 100K+ tasks/day
- Distributed execution across multiple workers
- Real-time monitoring and alerting
- Web-based DAG visualization
- SLA tracking and violations

## Tech Stack

### Backend
- **Go**: Core system language
- **Gin**: HTTP server framework
- **PostgreSQL**: Metadata store
- **Redis**: State management and caching
- **NATS JetStream**: Distributed task queue

### Infrastructure
- **Docker**: Container runtime
- **Docker Compose**: Local development
- **Prometheus**: Metrics collection
- **Grafana**: Metrics visualization

## Quick Start

### Prerequisites

- Go 1.21 or higher
- Docker and Docker Compose
- Make (optional, for convenience commands)

### Installation

1. Clone the repository:
```bash
git clone https://github.com/therealutkarshpriyadarshi/dag.git
cd dag
```

2. Copy environment configuration:
```bash
cp .env.example .env
```

3. Start all services:
```bash
make docker-up
# Or without make:
docker-compose up -d
```

4. Verify services are running:
```bash
# Check server health
curl http://localhost:8080/health

# Access Grafana
open http://localhost:3000  # admin/admin

# Access Prometheus
open http://localhost:9090

# Access NATS monitoring
open http://localhost:8222
```

## Development

### Setup Development Environment

1. Install development tools:
```bash
make install-tools
```

2. Install pre-commit hooks (optional):
```bash
./scripts/install-hooks.sh
```

### Running Locally

Run individual components:

```bash
# Run server
make run-server

# Run worker
make run-worker

# Run scheduler
make run-scheduler
```

### Testing

```bash
# Run all tests
make test

# Run tests with coverage
make test-coverage

# Run linters
make lint

# Format code
make fmt
```

### Building

```bash
# Build all binaries
make build

# Build specific component
make build-server
make build-worker
make build-scheduler
```

### Hot Reload Development

For development with hot reload:

```bash
# Install Air
go install github.com/cosmtrek/air@latest

# Run with hot reload
air
```

## Project Structure

```
workflow-orchestrator/
├── cmd/
│   ├── server/          # API server entrypoint
│   ├── worker/          # Task worker entrypoint
│   └── scheduler/       # Scheduler service entrypoint
├── internal/
│   ├── dag/             # DAG validation and graph algorithms
│   ├── executor/        # Task execution logic
│   ├── scheduler/       # Scheduling logic
│   ├── state/           # State management
│   └── storage/         # Database layer
├── pkg/
│   ├── api/             # REST API handlers
│   └── models/          # Shared data models
├── web/                 # Frontend React app (Coming Soon)
├── deployments/         # Docker, K8s configs
├── docs/                # Documentation
└── scripts/             # Utility scripts
```

## API Endpoints

### Health Check
```bash
GET /health
```

### API v1
```bash
GET  /api/v1/status       # API status
GET  /api/v1/dags         # List DAGs (Coming Soon)
POST /api/v1/dags         # Create DAG (Coming Soon)
GET  /api/v1/dags/:id     # Get DAG details (Coming Soon)
```

## Configuration

Configuration is managed through environment variables. See `.env.example` for all available options.

Key configuration variables:

- `ENV`: Environment (development/production)
- `PORT`: Server port (default: 8080)
- `DB_HOST`: PostgreSQL host
- `REDIS_HOST`: Redis host
- `NATS_URL`: NATS server URL

## Architecture

The system consists of three main components:

1. **Server**: REST API server for managing DAGs and exposing metrics
2. **Worker**: Executes tasks from the distributed queue
3. **Scheduler**: Manages cron-based DAG scheduling and triggers

Components communicate via:
- **PostgreSQL**: Persistent state storage
- **Redis**: Fast state access and caching
- **NATS**: Task distribution queue

## Roadmap

See [ROADMAP.md](ROADMAP.md) for detailed development phases.

**Current Status**: Phase 5 - Retry & Error Handling ✅

### Completed Phases

- [x] **Phase 0**: Project Setup & Foundation
  - Repository & tooling setup
  - Development environment
  - Docker Compose configuration
  - CI/CD pipeline

- [x] **Phase 1**: Core DAG Engine
  - DAG definition with builder pattern (Go DSL)
  - YAML/JSON parser for DAG definitions
  - Comprehensive DAG validation (cycles, orphaned tasks, dependencies)
  - Graph algorithms (topological sort, parallel task detection)
  - Critical path analysis for SLA estimation
  - Task lineage tracking (upstream/downstream dependencies)
  - 95.7% test coverage

- [x] **Phase 2**: Database Layer & State Management
  - PostgreSQL schema with migrations (5 tables, 15 indexes)
  - GORM integration with connection pooling
  - State machine with 11 valid transition paths
  - Event publishing (Redis pub/sub + database history)
  - Repository pattern (DAG, DAGRun, TaskInstance, TaskLog)
  - Optimistic locking for concurrent updates
  - 95%+ test coverage

- [x] **Phase 3**: Scheduler
  - Cron-based scheduling with robfig/cron v3
  - Automatic catchup for missed schedules
  - Backfill engine with CLI support
  - Multi-level concurrency controls (global, DAG, pool)
  - Redis-based distributed semaphores
  - Priority queue for smart scheduling
  - 95%+ test coverage

- [x] **Phase 4**: Executor & Worker System
  - Sequential executor for development/testing
  - Local executor with goroutine worker pool
  - Distributed executor with NATS JetStream
  - Task executors: Bash, HTTP, Go Function, Docker
  - Distributed worker with heartbeat monitoring
  - Resource management (timeouts, memory, CPU limits)
  - Worker deployment (standalone, distributed, K8s)
  - 90%+ test coverage

- [x] **Phase 5**: Retry & Error Handling
  - Retry strategies: Exponential backoff, Linear, Fixed delay
  - Configurable retry policies with error code filtering
  - Error propagation: Fail, Skip downstream, Allow partial
  - Circuit breaker pattern for external services
  - Dead Letter Queue (DLQ) for permanently failed tasks
  - Critical task designation
  - Comprehensive error callbacks and monitoring
  - 95%+ test coverage

### Next Phases

- [ ] Phase 6: REST API
- [ ] Phase 7: Web UI

## Contributing

Contributions are welcome! Please follow these steps:

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

### Development Guidelines

- Write tests for new functionality
- Follow Go best practices and idioms
- Run `make dev` before committing (formats, lints, and tests)
- Update documentation as needed

## Testing

The project uses standard Go testing:

```bash
# Run all tests
go test ./...

# Run with race detection
go test -race ./...

# Run with coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

## Monitoring

### Metrics

Prometheus metrics are exposed at `/metrics`:

- `dag_run_duration_seconds`: Histogram of DAG run durations
- `task_instance_duration_seconds`: Histogram of task durations
- `task_queue_depth`: Current task queue depth
- `worker_count`: Number of active workers

### Grafana Dashboards

Access Grafana at `http://localhost:3000` (admin/admin) to view:
- DAG execution metrics
- System health metrics
- Task performance metrics

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

Inspired by:
- [Apache Airflow](https://github.com/apache/airflow)
- [Temporal](https://github.com/temporalio/temporal)
- [Prefect](https://github.com/PrefectHQ/prefect)
- [Argo Workflows](https://github.com/argoproj/argo-workflows)

## Support

For bugs and feature requests, please [open an issue](https://github.com/therealutkarshpriyadarshi/dag/issues).

---

**Status**: Phase 0 Complete - Active Development

**Version**: 0.1.0
