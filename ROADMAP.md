# Workflow Orchestration System - Development Roadmap

## Project Overview
Build a production-grade workflow orchestration system in Go, similar to Airflow/Temporal, with distributed execution, web UI, and comprehensive monitoring.

**Target Capabilities:**
- Handle 100K+ tasks/day
- Distributed execution across multiple workers
- Real-time monitoring and alerting
- Web-based DAG visualization
- Retry logic with exponential backoff
- SLA tracking and violations

---

## Tech Stack Summary

### Backend (Go)
- **Framework**: Gin (HTTP server)
- **Database**: PostgreSQL (metadata store)
- **Cache/Queue**: Redis (state management, task queue)
- **Message Queue**: NATS JetStream (distributed task distribution)
- **Scheduler**: robfig/cron (cron expressions)
- **ORM**: GORM (database abstraction)

### Frontend (React)
- **UI Library**: React + TypeScript
- **DAG Visualization**: React Flow
- **State Management**: Zustand
- **Data Fetching**: TanStack Query
- **Styling**: Tailwind CSS
- **WebSockets**: gorilla/websocket (real-time updates)

### Infrastructure
- **Containerization**: Docker + Docker Compose
- **Metrics**: Prometheus + Grafana
- **Logging**: Zerolog + Loki
- **Storage**: MinIO (S3-compatible for logs)

---

## Development Phases

### **Phase 0: Project Setup & Foundation** (Week 1)

#### Milestone 0.1: Repository & Tooling Setup
- [ ] Initialize Go module (`go mod init github.com/yourusername/workflow-orchestrator`)
- [ ] Setup project structure:
  ```
  workflow-orchestrator/
  ├── cmd/
  │   ├── server/          # API server
  │   ├── worker/          # Task worker
  │   └── scheduler/       # Scheduler service
  ├── internal/
  │   ├── dag/             # DAG logic
  │   ├── executor/        # Task execution
  │   ├── scheduler/       # Scheduling logic
  │   ├── state/           # State management
  │   └── storage/         # Database layer
  ├── pkg/
  │   ├── api/             # REST API handlers
  │   └── models/          # Shared data models
  ├── web/                 # Frontend React app
  ├── deployments/         # Docker, K8s configs
  └── docs/                # Documentation
  ```
- [ ] Configure linting (golangci-lint)
- [ ] Setup CI/CD pipeline (GitHub Actions)
- [ ] Initialize git hooks (pre-commit)

#### Milestone 0.2: Development Environment
- [ ] Create `docker-compose.yml` with:
  - PostgreSQL 15
  - Redis 7
  - NATS 2.10
  - Grafana + Prometheus
- [ ] Setup hot-reload for Go (air)
- [ ] Create Makefile for common commands
- [ ] Write development setup docs

**Deliverable**: Development environment that spins up with `docker-compose up`

---

### **Phase 1: Core DAG Engine** (Weeks 2-3)

#### Milestone 1.1: DAG Definition & Parsing
- [ ] Define DAG struct with metadata:
  ```go
  type DAG struct {
      ID          string
      Name        string
      Description string
      Schedule    string    // Cron expression
      Tasks       []Task
      StartDate   time.Time
      EndDate     *time.Time
      Tags        []string
  }
  ```
- [ ] Implement Task struct:
  ```go
  type Task struct {
      ID           string
      Name         string
      Type         TaskType  // Bash, HTTP, Python, Go
      Command      string
      Dependencies []string
      Retries      int
      Timeout      time.Duration
      SLA          time.Duration
  }
  ```
- [ ] Build DAG parser from YAML/JSON
- [ ] Create Go-based DSL (builder pattern):
  ```go
  dag := NewDAG("etl_pipeline").
      Schedule("0 0 * * *").
      Task("extract", BashTask("extract.sh")).
      Task("transform", BashTask("transform.sh"), DependsOn("extract")).
      Task("load", BashTask("load.sh"), DependsOn("transform"))
  ```

#### Milestone 1.2: Dependency Graph Logic
- [ ] Implement adjacency list representation
- [ ] Build topological sort (Kahn's algorithm)
- [ ] Detect cycles in DAG
- [ ] Validate DAG integrity:
  - No orphaned tasks
  - All dependencies exist
  - No duplicate task IDs
- [ ] Write comprehensive unit tests (>90% coverage)

#### Milestone 1.3: Graph Algorithms
- [ ] Implement parallel task finder (tasks with no pending deps)
- [ ] Calculate critical path for SLA estimation
- [ ] Build task lineage tracker (upstream/downstream tasks)

**Deliverable**: DAG parser that validates and builds execution graph

---

### **Phase 2: Database Layer & State Management** (Week 4)

#### Milestone 2.1: Database Schema Design
- [ ] Create migration system (golang-migrate)
- [ ] Define schema:
  ```sql
  -- dags table
  CREATE TABLE dags (
      id UUID PRIMARY KEY,
      name VARCHAR(255) UNIQUE NOT NULL,
      description TEXT,
      schedule VARCHAR(100),
      is_paused BOOLEAN DEFAULT false,
      tags JSONB,
      created_at TIMESTAMP,
      updated_at TIMESTAMP
  );

  -- dag_runs table
  CREATE TABLE dag_runs (
      id UUID PRIMARY KEY,
      dag_id UUID REFERENCES dags(id),
      execution_date TIMESTAMP,
      state VARCHAR(50), -- queued, running, success, failed
      start_date TIMESTAMP,
      end_date TIMESTAMP,
      external_trigger BOOLEAN DEFAULT false
  );

  -- task_instances table
  CREATE TABLE task_instances (
      id UUID PRIMARY KEY,
      task_id VARCHAR(255),
      dag_run_id UUID REFERENCES dag_runs(id),
      state VARCHAR(50),
      try_number INT DEFAULT 1,
      max_tries INT,
      start_date TIMESTAMP,
      end_date TIMESTAMP,
      duration INTERVAL,
      hostname VARCHAR(255),
      error_message TEXT
  );

  -- task_logs table
  CREATE TABLE task_logs (
      id UUID PRIMARY KEY,
      task_instance_id UUID REFERENCES task_instances(id),
      log_data TEXT,
      timestamp TIMESTAMP
  );
  ```
- [ ] Add indexes on frequently queried columns
- [ ] Setup connection pooling (pgx)

#### Milestone 2.2: State Machine Implementation
- [ ] Define state enum:
  ```go
  const (
      StateQueued   = "queued"
      StateRunning  = "running"
      StateSuccess  = "success"
      StateFailed   = "failed"
      StateRetrying = "retrying"
      StateSkipped  = "skipped"
      StateUpstream = "upstream_failed"
  )
  ```
- [ ] Implement atomic state transitions with optimistic locking
- [ ] Build state change event publisher (for UI updates)
- [ ] Create state history tracking

#### Milestone 2.3: Repository Pattern
- [ ] Implement DAGRepository interface
- [ ] Create TaskInstanceRepository
- [ ] Add transaction support
- [ ] Write integration tests with test database

**Deliverable**: Persistent state management with ACID guarantees

---

### **Phase 3: Scheduler** (Weeks 5-6)

#### Milestone 3.1: Cron Scheduler
- [ ] Integrate robfig/cron library
- [ ] Parse cron expressions from DAG definitions
- [ ] Build DAG run creator (creates dag_run on schedule trigger)
- [ ] Implement timezone support
- [ ] Add catchup logic (run missed schedules)

#### Milestone 3.2: Backfill Engine
- [ ] CLI command for backfill: `./scheduler backfill --dag-id=etl --start=2024-01-01 --end=2024-01-31`
- [ ] Parallel backfill with concurrency limits
- [ ] Progress tracking and cancellation support

#### Milestone 3.3: Concurrency Controls
- [ ] DAG-level concurrency (max parallel dag_runs per DAG)
- [ ] Global concurrency limit (max tasks across all DAGs)
- [ ] Task-level pools (e.g., "database_pool" with 5 slots)
- [ ] Implement semaphore-based slot management with Redis

#### Milestone 3.4: Smart Scheduling
- [ ] Priority queue for DAG runs (high/medium/low)
- [ ] Fair scheduling algorithm (prevent starvation)
- [ ] Resource-aware scheduling (CPU/memory requirements)

**Deliverable**: Scheduler that triggers DAG runs based on cron schedules

---

### **Phase 4: Executor & Worker System** (Weeks 7-8)

#### Milestone 4.1: Sequential Executor (Development)
- [ ] Simple in-process executor for testing
- [ ] Execute tasks in topological order
- [ ] No parallelism, just sequential execution

#### Milestone 4.2: Local Executor (Multi-process)
- [ ] Worker pool using goroutines
- [ ] Task queue (buffered channel)
- [ ] Graceful shutdown handling
- [ ] Health check endpoint

#### Milestone 4.3: Distributed Executor (NATS)
- [ ] Setup NATS JetStream streams:
  - `tasks.pending` (task queue)
  - `tasks.results` (result stream)
- [ ] Worker registration system
- [ ] Heartbeat mechanism (detect dead workers)
- [ ] Task assignment with at-least-once delivery
- [ ] Duplicate task detection (idempotency)

#### Milestone 4.4: Task Executors
- [ ] BashTaskExecutor (execute shell commands)
- [ ] HTTPTaskExecutor (make HTTP requests)
- [ ] GoFuncTaskExecutor (execute Go functions)
- [ ] DockerTaskExecutor (run tasks in containers)
- [ ] Executor plugin system (for extensibility)

#### Milestone 4.5: Resource Management
- [ ] Cgroup-based CPU/memory limits
- [ ] Task timeout enforcement
- [ ] OOM handling
- [ ] Disk space checks before execution

**Deliverable**: Distributed worker system that executes tasks across multiple nodes

---

### **Phase 5: Retry & Error Handling** (Week 9)

#### Milestone 5.1: Retry Logic
- [ ] Implement retry strategies:
  ```go
  type RetryStrategy interface {
      NextDelay(attempt int) time.Duration
  }

  type ExponentialBackoff struct {
      BaseDelay time.Duration
      MaxDelay  time.Duration
      Jitter    bool
  }
  ```
- [ ] Configure max retries per task
- [ ] Retry on specific error codes
- [ ] Manual retry trigger from UI

#### Milestone 5.2: Error Propagation
- [ ] Upstream failure handling (skip downstream tasks)
- [ ] Partial DAG success (allow some tasks to fail)
- [ ] Error callbacks (trigger on failure)

#### Milestone 5.3: Circuit Breaker
- [ ] Implement circuit breaker for external services
- [ ] Automatic circuit opening on repeated failures
- [ ] Half-open state for recovery testing

#### Milestone 5.4: Dead Letter Queue
- [ ] Move failed tasks (after max retries) to DLQ
- [ ] DLQ inspection and replay API
- [ ] Alert on DLQ accumulation

**Deliverable**: Robust error handling with configurable retry policies

---

### **Phase 6: REST API** (Week 10)

#### Milestone 6.1: Core API Endpoints
- [ ] `POST /api/v1/dags` - Create DAG
- [ ] `GET /api/v1/dags` - List DAGs
- [ ] `GET /api/v1/dags/:id` - Get DAG details
- [ ] `PATCH /api/v1/dags/:id` - Update DAG
- [ ] `DELETE /api/v1/dags/:id` - Delete DAG
- [ ] `POST /api/v1/dags/:id/pause` - Pause DAG
- [ ] `POST /api/v1/dags/:id/unpause` - Unpause DAG

#### Milestone 6.2: DAG Run Endpoints
- [ ] `POST /api/v1/dags/:id/trigger` - Manual trigger
- [ ] `GET /api/v1/dag-runs` - List runs (with filters)
- [ ] `GET /api/v1/dag-runs/:id` - Get run details
- [ ] `POST /api/v1/dag-runs/:id/cancel` - Cancel run

#### Milestone 6.3: Task Instance Endpoints
- [ ] `GET /api/v1/dag-runs/:id/tasks` - List tasks in run
- [ ] `GET /api/v1/task-instances/:id` - Task details
- [ ] `GET /api/v1/task-instances/:id/logs` - Stream logs
- [ ] `POST /api/v1/task-instances/:id/retry` - Manual retry

#### Milestone 6.4: API Infrastructure
- [ ] Request validation (go-playground/validator)
- [ ] Error handling middleware
- [ ] Rate limiting (per-IP and per-user)
- [ ] API versioning strategy
- [ ] OpenAPI/Swagger documentation
- [ ] Authentication (JWT tokens)
- [ ] Authorization (RBAC)

**Deliverable**: RESTful API for all core operations

---

### **Phase 7: Web UI** (Weeks 11-13)

#### Milestone 7.1: Frontend Setup
- [ ] Initialize React app with Vite
- [ ] Configure TypeScript, ESLint, Prettier
- [ ] Setup Tailwind CSS
- [ ] Configure React Router
- [ ] Setup TanStack Query for API calls

#### Milestone 7.2: DAG List View
- [ ] DAG cards with status indicators
- [ ] Search and filter (by tag, status)
- [ ] Pause/unpause toggle
- [ ] Trigger DAG button
- [ ] Recent runs timeline

#### Milestone 7.3: DAG Visualization
- [ ] Integrate React Flow
- [ ] Render DAG as directed graph
- [ ] Color-code tasks by status (green=success, red=failed, blue=running)
- [ ] Click task to view details
- [ ] Zoom and pan controls
- [ ] Auto-layout algorithm (dagre)

#### Milestone 7.4: Execution Dashboard
- [ ] Real-time run status grid
- [ ] Gantt chart view (task timeline)
- [ ] Duration metrics
- [ ] Success/failure rates
- [ ] Active workers count

#### Milestone 7.5: Task Log Viewer
- [ ] Stream logs via WebSocket
- [ ] Syntax highlighting
- [ ] Download logs button
- [ ] Log level filtering
- [ ] Search in logs

#### Milestone 7.6: Manual Controls
- [ ] Trigger DAG form (with config override)
- [ ] Clear task instance (reset state)
- [ ] Mark task as success/failed
- [ ] Kill running task

**Deliverable**: Full-featured web UI for monitoring and control

---

### **Phase 8: Monitoring & Observability** (Week 14)

#### Milestone 8.1: Metrics Collection
- [ ] Integrate Prometheus client
- [ ] Expose metrics endpoint (`/metrics`)
- [ ] Key metrics:
  ```
  - dag_run_duration_seconds (histogram)
  - task_instance_duration_seconds (histogram)
  - task_queue_depth (gauge)
  - worker_count (gauge)
  - task_success_total (counter)
  - task_failure_total (counter)
  - scheduler_latency_seconds (histogram)
  ```

#### Milestone 8.2: Logging Infrastructure
- [ ] Integrate zerolog for structured logging
- [ ] Log levels per component
- [ ] Request ID tracing
- [ ] Ship logs to Loki/Elasticsearch
- [ ] Log rotation and retention policies

#### Milestone 8.3: SLA Monitoring
- [ ] Define SLA per task/DAG
- [ ] Calculate actual vs expected duration
- [ ] Record SLA misses in database
- [ ] SLA dashboard in UI
- [ ] Alert on SLA violations

#### Milestone 8.4: Alerting
- [ ] Webhook alerting system
- [ ] Slack integration
- [ ] Email notifications (SMTP)
- [ ] PagerDuty integration
- [ ] Alert rules:
  - DAG failure after max retries
  - Worker down
  - Queue backup
  - SLA violations

#### Milestone 8.5: Distributed Tracing
- [ ] Integrate OpenTelemetry
- [ ] Trace DAG run lifecycle
- [ ] Span per task execution
- [ ] Export to Jaeger/Tempo

**Deliverable**: Comprehensive observability stack

---

### **Phase 9: Advanced Features** (Weeks 15-16)

#### Milestone 9.1: Dynamic DAGs
- [ ] Hot-reload DAG definitions from Git repo
- [ ] DAG versioning (track changes)
- [ ] Rollback to previous DAG version

#### Milestone 9.2: XCom (Cross-Task Communication)
- [ ] Key-value store for task outputs
- [ ] Tasks push data: `xcom.push("result", data)`
- [ ] Tasks pull data: `xcom.pull("result", from_task="extract")`
- [ ] Store in Redis with TTL

#### Milestone 9.3: Sensors
- [ ] Sensor tasks (wait for external condition):
  - FileSensor (wait for file to exist)
  - HTTPSensor (poll endpoint until success)
  - TimeSensor (wait until specific time)
  - ExternalTaskSensor (wait for another task)
- [ ] Poke mode (blocking) vs reschedule mode (async)

#### Milestone 9.4: Branching & Conditionals
- [ ] BranchTask (choose downstream task based on condition)
- [ ] Skip tasks based on XCom values
- [ ] Dynamic task generation

#### Milestone 9.5: Sub-DAGs
- [ ] DAG within a DAG
- [ ] Reusable workflow components
- [ ] Isolated execution context

**Deliverable**: Advanced workflow patterns for complex pipelines

---

### **Phase 10: Performance & Scale** (Week 17)

#### Milestone 10.1: Load Testing
- [ ] Simulate 1M tasks/day
- [ ] Benchmark scheduler latency (<100ms)
- [ ] Test worker autoscaling
- [ ] Database query optimization
- [ ] Connection pool tuning

#### Milestone 10.2: Horizontal Scaling
- [ ] Multiple scheduler instances (leader election)
- [ ] Worker autoscaling based on queue depth
- [ ] Database read replicas
- [ ] Redis cluster mode

#### Milestone 10.3: Caching Strategy
- [ ] Cache DAG definitions in Redis
- [ ] Cache task instance states
- [ ] Invalidation on updates

#### Milestone 10.4: Profiling & Optimization
- [ ] pprof CPU profiling
- [ ] Memory leak detection
- [ ] Goroutine leak detection
- [ ] Database query analysis (EXPLAIN)

**Deliverable**: System handles 100K+ tasks/day with <1s p99 latency

---

### **Phase 11: Security & Compliance** (Week 18)

#### Milestone 11.1: Authentication & Authorization
- [ ] JWT-based authentication
- [ ] OAuth2 integration (Google, GitHub)
- [ ] RBAC (roles: admin, operator, viewer)
- [ ] API key management

#### Milestone 11.2: Secrets Management
- [ ] Integration with HashiCorp Vault
- [ ] Encrypted secrets in database (AES-256)
- [ ] Secret rotation
- [ ] Audit log for secret access

#### Milestone 11.3: Network Security
- [ ] TLS/HTTPS everywhere
- [ ] mTLS for worker-server communication
- [ ] Network policies (if on K8s)

#### Milestone 11.4: Audit Logging
- [ ] Log all state changes
- [ ] Track user actions
- [ ] Tamper-proof logs (write-once storage)

**Deliverable**: Enterprise-grade security posture

---

### **Phase 12: Production Deployment** (Week 19)

#### Milestone 12.1: Kubernetes Deployment
- [ ] Helm chart for full stack
- [ ] StatefulSet for scheduler (with leader election)
- [ ] Deployment for workers (with HPA)
- [ ] Service mesh integration (Istio)

#### Milestone 12.2: High Availability
- [ ] Multi-AZ deployment
- [ ] Database failover (Patroni)
- [ ] Redis Sentinel
- [ ] Zero-downtime deployments

#### Milestone 12.3: Backup & Disaster Recovery
- [ ] Automated database backups (pgBackRest)
- [ ] Point-in-time recovery testing
- [ ] Backup retention policy (7 daily, 4 weekly, 12 monthly)

#### Milestone 12.4: Documentation
- [ ] Architecture diagram (C4 model)
- [ ] Deployment guide
- [ ] Operations runbook
- [ ] API documentation
- [ ] User tutorials

**Deliverable**: Production-ready deployment on Kubernetes

---

### **Phase 13: Testing & Quality** (Ongoing)

#### Unit Tests (Target: >80% coverage)
- [ ] DAG parsing and validation
- [ ] State machine transitions
- [ ] Scheduler logic
- [ ] Executor task assignment

#### Integration Tests
- [ ] End-to-end DAG execution
- [ ] Database transactions
- [ ] Worker communication

#### E2E Tests
- [ ] UI workflow tests (Playwright)
- [ ] API contract tests
- [ ] Load tests (k6)

#### Chaos Engineering
- [ ] Kill random workers
- [ ] Database connection drops
- [ ] Network partitions
- [ ] Resource exhaustion

---

## Timeline Summary

| Phase | Duration | Key Deliverable |
|-------|----------|-----------------|
| 0. Setup | 1 week | Dev environment |
| 1. DAG Engine | 2 weeks | DAG parser & validator |
| 2. Database | 1 week | State persistence |
| 3. Scheduler | 2 weeks | Cron-based triggers |
| 4. Executor | 2 weeks | Distributed workers |
| 5. Error Handling | 1 week | Retry logic |
| 6. REST API | 1 week | Full CRUD API |
| 7. Web UI | 3 weeks | React dashboard |
| 8. Monitoring | 1 week | Metrics & logs |
| 9. Advanced Features | 2 weeks | XCom, sensors |
| 10. Performance | 1 week | Load testing |
| 11. Security | 1 week | Auth & secrets |
| 12. Production | 1 week | K8s deployment |
| **Total** | **19 weeks** | **Production system** |

---

## Success Criteria

### Functional Requirements
- [ ] Execute 100,000+ tasks/day reliably
- [ ] Support DAGs with 100+ tasks
- [ ] Scheduler latency <100ms p99
- [ ] Task start latency <500ms p99
- [ ] 99.9% uptime SLA

### Non-Functional Requirements
- [ ] Horizontal scaling (add workers without downtime)
- [ ] Graceful degradation (system remains usable if workers fail)
- [ ] Data durability (no lost task results)
- [ ] Security (encrypted secrets, RBAC)

### Developer Experience
- [ ] DAG definition in <20 lines of code
- [ ] Hot-reload DAGs without restart
- [ ] Clear error messages
- [ ] Comprehensive API docs

---

## Risk Mitigation

### Risk: Database becomes bottleneck
**Mitigation**:
- Use read replicas for queries
- Implement caching layer (Redis)
- Partition tables by date

### Risk: Worker overload
**Mitigation**:
- Autoscaling based on queue depth
- Circuit breakers for failing tasks
- Resource limits per task

### Risk: Data loss on crash
**Mitigation**:
- WAL archiving for PostgreSQL
- Redis persistence (AOF + RDB)
- Task idempotency (safe to retry)

### Risk: Scheduler single point of failure
**Mitigation**:
- Leader election (etcd/Consul)
- Multiple scheduler replicas
- Fast failover (<30s)

---

## Next Steps

1. **Week 1**: Complete Phase 0 (project setup)
2. **Week 2**: Start Phase 1 (DAG engine)
3. **Review checkpoint**: End of Phase 4 (basic execution working)
4. **MVP target**: End of Phase 7 (functional UI)
5. **Production ready**: End of Phase 12

---

## References & Inspiration

- **Airflow**: https://github.com/apache/airflow
- **Temporal**: https://github.com/temporalio/temporal
- **Prefect**: https://github.com/PrefectHQ/prefect
- **Argo Workflows**: https://github.com/argoproj/argo-workflows
- **Flyte**: https://github.com/flyteorg/flyte

---

**Last Updated**: 2025-11-18
**Version**: 1.0
**Status**: Planning Phase
