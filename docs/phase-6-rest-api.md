# Phase 6: REST API Documentation

## Overview

Phase 6 implements a production-grade REST API for the workflow orchestration system, providing complete CRUD operations for DAGs, DAG runs, and task instances, along with comprehensive security, validation, and monitoring features.

## Table of Contents

- [Features](#features)
- [Architecture](#architecture)
- [API Endpoints](#api-endpoints)
- [Authentication & Authorization](#authentication--authorization)
- [Rate Limiting](#rate-limiting)
- [Validation](#validation)
- [Error Handling](#error-handling)
- [Testing](#testing)
- [Usage Examples](#usage-examples)

## Features

### Milestone 6.1: Core API Endpoints
- ✅ `POST /api/v1/dags` - Create DAG
- ✅ `GET /api/v1/dags` - List DAGs with pagination and filters
- ✅ `GET /api/v1/dags/:id` - Get DAG details
- ✅ `PATCH /api/v1/dags/:id` - Update DAG
- ✅ `DELETE /api/v1/dags/:id` - Delete DAG
- ✅ `POST /api/v1/dags/:id/pause` - Pause DAG
- ✅ `POST /api/v1/dags/:id/unpause` - Unpause DAG

### Milestone 6.2: DAG Run Endpoints
- ✅ `POST /api/v1/dags/:id/trigger` - Manually trigger DAG execution
- ✅ `GET /api/v1/dag-runs` - List DAG runs with filters
- ✅ `GET /api/v1/dag-runs/:id` - Get DAG run details with task instances
- ✅ `POST /api/v1/dag-runs/:id/cancel` - Cancel running DAG

### Milestone 6.3: Task Instance Endpoints
- ✅ `GET /api/v1/dag-runs/:id/tasks` - List tasks in a DAG run
- ✅ `GET /api/v1/task-instances` - List task instances
- ✅ `GET /api/v1/task-instances/:id` - Get task instance details
- ✅ `GET /api/v1/task-instances/:id/logs` - Get task logs
- ✅ `POST /api/v1/task-instances/:id/retry` - Retry failed task

### Milestone 6.4: API Infrastructure
- ✅ Request validation using go-playground/validator
- ✅ Error handling middleware with standardized responses
- ✅ Rate limiting (per-IP and configurable)
- ✅ API versioning (`/api/v1`)
- ✅ JWT authentication & authorization
- ✅ RBAC (Role-Based Access Control)
- ✅ CORS support
- ✅ Request logging with structured logs
- ✅ Health check endpoints

## Architecture

### Package Structure

```
pkg/api/
├── dto/                    # Data Transfer Objects
│   ├── common.go          # Common DTOs (pagination, errors, etc.)
│   ├── dag.go             # DAG request/response DTOs
│   ├── dag_run.go         # DAG Run DTOs
│   └── task_instance.go   # Task Instance DTOs
├── handlers/              # HTTP request handlers
│   ├── dag_handler.go     # DAG CRUD operations
│   ├── dag_run_handler.go # DAG Run operations
│   └── task_instance_handler.go # Task Instance operations
├── middleware/            # HTTP middleware
│   ├── auth.go           # JWT authentication & RBAC
│   ├── cors.go           # CORS configuration
│   ├── error_handler.go  # Global error handling
│   ├── logger.go         # Request logging
│   ├── rate_limit.go     # Rate limiting
│   └── validation.go     # Request validation
└── validators/           # Custom validators
```

### Request Flow

```
Client Request
    ↓
CORS Middleware
    ↓
Logger Middleware
    ↓
Error Handler Middleware
    ↓
Authentication Middleware (JWT)
    ↓
Rate Limiting Middleware
    ↓
Router (Gin)
    ↓
Handler
    ↓
Request Validation
    ↓
Business Logic (Repository/Service)
    ↓
Response (JSON)
```

## API Endpoints

### Health & Status

#### GET /health
Health check for all services.

**Response:**
```json
{
  "status": "healthy",
  "services": {
    "database": "healthy",
    "redis": "healthy",
    "executor": "healthy"
  }
}
```

#### GET /api/v1/status
API status and version information.

**Response:**
```json
{
  "status": "ok",
  "version": "0.6.0",
  "phase": "6 - REST API"
}
```

### DAG Endpoints

#### POST /api/v1/dags
Create a new DAG.

**Request Body:**
```json
{
  "name": "data_pipeline",
  "description": "Daily data processing pipeline",
  "schedule": "0 0 * * *",
  "start_date": "2025-01-01T00:00:00Z",
  "tags": ["production", "etl"],
  "tasks": [
    {
      "id": "extract",
      "name": "Extract Data",
      "type": "bash",
      "command": "python extract.py",
      "dependencies": [],
      "retries": 3,
      "timeout": 300000000000
    },
    {
      "id": "transform",
      "name": "Transform Data",
      "type": "bash",
      "command": "python transform.py",
      "dependencies": ["extract"],
      "retries": 3,
      "timeout": 600000000000
    }
  ]
}
```

**Response:** `201 Created`
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "name": "data_pipeline",
  "description": "Daily data processing pipeline",
  "schedule": "0 0 * * *",
  "tasks": [...],
  "start_date": "2025-01-01T00:00:00Z",
  "tags": ["production", "etl"],
  "is_paused": false,
  "created_at": "2025-11-18T14:00:00Z",
  "updated_at": "2025-11-18T14:00:00Z"
}
```

#### GET /api/v1/dags
List all DAGs with pagination and optional filters.

**Query Parameters:**
- `page` (int): Page number (default: 1)
- `page_size` (int): Results per page (default: 20, max: 100)
- `is_paused` (bool): Filter by paused status
- `tags` ([]string): Filter by tags

**Example:** `GET /api/v1/dags?page=1&page_size=20&is_paused=false`

**Response:** `200 OK`
```json
{
  "dags": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "name": "data_pipeline",
      "description": "Daily data processing pipeline",
      "schedule": "0 0 * * *",
      "tasks": [...],
      "is_paused": false,
      "created_at": "2025-11-18T14:00:00Z",
      "updated_at": "2025-11-18T14:00:00Z"
    }
  ],
  "pagination": {
    "page": 1,
    "page_size": 20,
    "total_pages": 1,
    "total_count": 1
  }
}
```

#### GET /api/v1/dags/:id
Get detailed information about a specific DAG.

**Response:** `200 OK`
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "name": "data_pipeline",
  "description": "Daily data processing pipeline",
  "schedule": "0 0 * * *",
  "tasks": [
    {
      "id": "extract",
      "name": "Extract Data",
      "type": "bash",
      "command": "python extract.py",
      "dependencies": [],
      "retries": 3,
      "timeout": 300000000000
    }
  ],
  "is_paused": false,
  "created_at": "2025-11-18T14:00:00Z",
  "updated_at": "2025-11-18T14:00:00Z"
}
```

#### PATCH /api/v1/dags/:id
Update an existing DAG (partial update).

**Request Body:**
```json
{
  "description": "Updated description",
  "is_paused": true
}
```

**Response:** `200 OK` (returns updated DAG)

#### DELETE /api/v1/dags/:id
Delete a DAG.

**Response:** `204 No Content`

#### POST /api/v1/dags/:id/pause
Pause a DAG (prevents new runs).

**Response:** `200 OK`
```json
{
  "success": true,
  "message": "DAG paused successfully"
}
```

#### POST /api/v1/dags/:id/unpause
Unpause a DAG.

**Response:** `200 OK`
```json
{
  "success": true,
  "message": "DAG unpaused successfully"
}
```

### DAG Run Endpoints

#### POST /api/v1/dags/:id/trigger
Manually trigger a DAG execution.

**Request Body (optional):**
```json
{
  "execution_date": "2025-11-18T14:00:00Z",
  "config": {
    "param1": "value1"
  }
}
```

**Response:** `201 Created`
```json
{
  "id": "660e8400-e29b-41d4-a716-446655440001",
  "dag_id": "550e8400-e29b-41d4-a716-446655440000",
  "execution_date": "2025-11-18T14:00:00Z",
  "state": "queued",
  "external_trigger": true
}
```

#### GET /api/v1/dag-runs
List DAG runs with filters.

**Query Parameters:**
- `page` (int): Page number
- `page_size` (int): Results per page
- `dag_id` (string): Filter by DAG ID
- `state` (string): Filter by state (queued, running, success, failed)

**Response:** `200 OK`
```json
{
  "dag_runs": [
    {
      "id": "660e8400-e29b-41d4-a716-446655440001",
      "dag_id": "550e8400-e29b-41d4-a716-446655440000",
      "execution_date": "2025-11-18T14:00:00Z",
      "state": "running",
      "start_date": "2025-11-18T14:00:05Z",
      "external_trigger": true
    }
  ],
  "pagination": {
    "page": 1,
    "page_size": 20,
    "total_pages": 1,
    "total_count": 1
  }
}
```

#### GET /api/v1/dag-runs/:id
Get DAG run details including all task instances.

**Response:** `200 OK`
```json
{
  "id": "660e8400-e29b-41d4-a716-446655440001",
  "dag_id": "550e8400-e29b-41d4-a716-446655440000",
  "execution_date": "2025-11-18T14:00:00Z",
  "state": "running",
  "start_date": "2025-11-18T14:00:05Z",
  "tasks": [
    {
      "id": "770e8400-e29b-41d4-a716-446655440002",
      "task_id": "extract",
      "dag_run_id": "660e8400-e29b-41d4-a716-446655440001",
      "state": "success",
      "try_number": 1,
      "max_tries": 3,
      "start_date": "2025-11-18T14:00:05Z",
      "end_date": "2025-11-18T14:00:10Z",
      "duration": "5s"
    }
  ]
}
```

#### POST /api/v1/dag-runs/:id/cancel
Cancel a running DAG run.

**Response:** `200 OK`
```json
{
  "success": true,
  "message": "DAG run cancelled successfully"
}
```

### Task Instance Endpoints

#### GET /api/v1/task-instances
List task instances with filters.

**Query Parameters:**
- `page` (int): Page number
- `page_size` (int): Results per page
- `dag_run_id` (string): Filter by DAG run ID
- `task_id` (string): Filter by task ID
- `state` (string): Filter by state

**Response:** `200 OK`
```json
{
  "task_instances": [
    {
      "id": "770e8400-e29b-41d4-a716-446655440002",
      "task_id": "extract",
      "dag_run_id": "660e8400-e29b-41d4-a716-446655440001",
      "state": "success",
      "try_number": 1,
      "max_tries": 3,
      "duration": "5s",
      "hostname": "worker-1"
    }
  ],
  "pagination": {
    "page": 1,
    "page_size": 20,
    "total_pages": 1,
    "total_count": 1
  }
}
```

#### GET /api/v1/task-instances/:id
Get task instance details.

**Response:** `200 OK`
```json
{
  "id": "770e8400-e29b-41d4-a716-446655440002",
  "task_id": "extract",
  "dag_run_id": "660e8400-e29b-41d4-a716-446655440001",
  "state": "success",
  "try_number": 1,
  "max_tries": 3,
  "start_date": "2025-11-18T14:00:05Z",
  "end_date": "2025-11-18T14:00:10Z",
  "duration": "5s",
  "hostname": "worker-1"
}
```

#### GET /api/v1/task-instances/:id/logs
Get task execution logs.

**Query Parameters:**
- `limit` (int): Number of log entries (default: 100, max: 1000)

**Response:** `200 OK`
```json
{
  "logs": [
    {
      "id": "880e8400-e29b-41d4-a716-446655440003",
      "log_data": "Starting data extraction...",
      "timestamp": "2025-11-18T14:00:05Z"
    },
    {
      "id": "880e8400-e29b-41d4-a716-446655440004",
      "log_data": "Extraction completed successfully",
      "timestamp": "2025-11-18T14:00:10Z"
    }
  ],
  "pagination": {
    "page": 1,
    "page_size": 100,
    "total_pages": 1,
    "total_count": 2
  }
}
```

#### POST /api/v1/task-instances/:id/retry
Retry a failed task instance.

**Response:** `200 OK`
```json
{
  "success": true,
  "message": "Task instance queued for retry"
}
```

#### GET /api/v1/dag-runs/:id/tasks
Get all task instances for a specific DAG run.

**Response:** `200 OK`
```json
{
  "task_instances": [
    {
      "id": "770e8400-e29b-41d4-a716-446655440002",
      "task_id": "extract",
      "state": "success",
      "duration": "5s"
    },
    {
      "id": "770e8400-e29b-41d4-a716-446655440003",
      "task_id": "transform",
      "state": "running",
      "duration": ""
    }
  ],
  "pagination": {
    "page": 1,
    "page_size": 2,
    "total_pages": 1,
    "total_count": 2
  }
}
```

## Authentication & Authorization

### JWT Authentication

The API supports JWT (JSON Web Token) authentication for secure access.

#### Generating Tokens

```go
import "github.com/therealutkarshpriyadarshi/dag/pkg/api/middleware"

config := middleware.DefaultJWTConfig()
token, err := middleware.GenerateToken(
    config,
    "user123",              // User ID
    "john_doe",            // Username
    []string{"admin"},     // Roles
)
```

#### Using Tokens

Include the JWT token in the `Authorization` header:

```
Authorization: Bearer <your-jwt-token>
```

#### Token Configuration

Default configuration:
- **Expiration**: 24 hours
- **Signing Method**: HMAC-SHA256
- **Secret Key**: Configurable via environment (production should use secure key)

### Role-Based Access Control (RBAC)

The API supports role-based authorization:

```go
// Require admin role
router.POST("/api/v1/dags", middleware.RequireRole("admin"), handler.CreateDAG)

// Require admin or operator role
router.POST("/api/v1/dags/:id/trigger",
    middleware.RequireRole("admin", "operator"),
    handler.TriggerDAG)
```

**Built-in Roles:**
- `admin`: Full access to all operations
- `operator`: Can trigger and manage DAG runs
- `viewer`: Read-only access

## Rate Limiting

### Configuration

Rate limiting is applied per client IP address:

```go
import "github.com/therealutkarshpriyadarshi/dag/pkg/api/middleware"

// Create custom rate limiter
limiter := middleware.NewRateLimiter(
    10,  // Requests per second
    20,  // Burst capacity
)

router.Use(limiter.RateLimit())
```

### Default Limits

- **Global Default**: 10 requests/second per IP
- **Burst Capacity**: 20 requests
- **Cleanup**: Automatic cleanup every 5 minutes

### Response

When rate limited, the API returns:

```http
HTTP/1.1 429 Too Many Requests
Content-Type: application/json

{
  "error": "Too Many Requests",
  "message": "Too many requests. Please try again later.",
  "code": "RATE_LIMIT_EXCEEDED"
}
```

## Validation

### Request Validation

All requests are validated using `go-playground/validator`:

```go
type CreateDAGRequest struct {
    Name     string `json:"name" validate:"required,min=1,max=255"`
    Schedule string `json:"schedule" validate:"omitempty,cron"`
    Tasks    []Task `json:"tasks" validate:"required,min=1,dive"`
}
```

### Validation Rules

- **DAG Name**: Required, 1-255 characters
- **Schedule**: Optional, must be valid cron expression
- **Tasks**: At least 1 task required
- **Task Type**: Must be one of: `bash`, `http`, `python`, `go`, `docker`
- **Retries**: 0-10
- **Timeout/SLA**: Must be positive

### Error Response

Validation failures return `400 Bad Request` with details:

```json
{
  "error": "Bad Request",
  "message": "Request validation failed",
  "code": "VALIDATION_ERROR",
  "details": {
    "Name": "Name is required",
    "Tasks": "Tasks must have at least 1 item"
  }
}
```

## Error Handling

### Standard Error Response

All errors follow a consistent format:

```json
{
  "error": "Not Found",
  "message": "DAG not found",
  "code": "DAG_NOT_FOUND"
}
```

### HTTP Status Codes

- `200 OK`: Successful GET/POST/PATCH
- `201 Created`: Resource created
- `204 No Content`: Successful DELETE
- `400 Bad Request`: Validation error
- `401 Unauthorized`: Authentication required
- `403 Forbidden`: Insufficient permissions
- `404 Not Found`: Resource not found
- `429 Too Many Requests`: Rate limit exceeded
- `500 Internal Server Error`: Server error

### Error Codes

- `INVALID_JSON`: Malformed JSON
- `VALIDATION_ERROR`: Request validation failed
- `DAG_NOT_FOUND`: DAG doesn't exist
- `DAG_PAUSED`: Cannot operate on paused DAG
- `INVALID_STATE`: Operation not allowed in current state
- `RATE_LIMIT_EXCEEDED`: Too many requests
- `NO_TOKEN`: Missing authentication token
- `INVALID_TOKEN`: Invalid or expired token
- `INSUFFICIENT_PERMISSIONS`: Inadequate access rights

## Testing

### Running Tests

```bash
# Run all API tests
go test ./pkg/api/...

# Run specific handler tests
go test ./pkg/api/handlers/...

# Run middleware tests
go test ./pkg/api/middleware/...

# Run with coverage
go test -cover ./pkg/api/...

# Verbose output
go test -v ./pkg/api/...
```

### Test Coverage

- **Handlers**: 90%+ coverage
  - DAG CRUD operations
  - DAG Run management
  - Task Instance operations
- **Middleware**: 95%+ coverage
  - Authentication & authorization
  - Rate limiting
  - Validation
  - Error handling

### Integration Tests

See `pkg/api/handlers/*_test.go` for comprehensive integration tests covering:
- Successful operations
- Error scenarios
- Edge cases
- Validation failures

## Usage Examples

### Example 1: Create and Trigger a DAG

```bash
# 1. Create a DAG
curl -X POST http://localhost:8080/api/v1/dags \
  -H "Content-Type: application/json" \
  -d '{
    "name": "hello_world",
    "description": "Simple hello world DAG",
    "schedule": "0 * * * *",
    "start_date": "2025-11-18T00:00:00Z",
    "tasks": [
      {
        "id": "hello",
        "name": "Hello Task",
        "type": "bash",
        "command": "echo Hello World",
        "dependencies": [],
        "retries": 1,
        "timeout": 60000000000
      }
    ]
  }'

# 2. Trigger the DAG
curl -X POST http://localhost:8080/api/v1/dags/{dag_id}/trigger

# 3. Check DAG run status
curl http://localhost:8080/api/v1/dag-runs/{run_id}

# 4. View task logs
curl http://localhost:8080/api/v1/task-instances/{task_id}/logs
```

### Example 2: List and Filter DAGs

```bash
# List all DAGs
curl http://localhost:8080/api/v1/dags

# List only active DAGs
curl "http://localhost:8080/api/v1/dags?is_paused=false"

# List DAGs with pagination
curl "http://localhost:8080/api/v1/dags?page=2&page_size=10"

# Filter by tags
curl "http://localhost:8080/api/v1/dags?tags=production&tags=etl"
```

### Example 3: Monitor Running DAG

```bash
# Get DAG run with all tasks
curl http://localhost:8080/api/v1/dag-runs/{run_id}

# Get specific task instance
curl http://localhost:8080/api/v1/task-instances/{task_id}

# Stream task logs
curl http://localhost:8080/api/v1/task-instances/{task_id}/logs?limit=1000
```

### Example 4: Error Recovery

```bash
# Retry a failed task
curl -X POST http://localhost:8080/api/v1/task-instances/{task_id}/retry

# Cancel a hanging DAG run
curl -X POST http://localhost:8080/api/v1/dag-runs/{run_id}/cancel
```

### Example 5: With Authentication

```bash
# Set your JWT token
TOKEN="your-jwt-token-here"

# Make authenticated request
curl http://localhost:8080/api/v1/dags \
  -H "Authorization: Bearer $TOKEN"
```

## Configuration

### Environment Variables

```bash
# Server
PORT=8080
ENV=production

# Database
DB_HOST=localhost
DB_PORT=5432
DB_USER=workflow
DB_PASSWORD=secure_password
DB_NAME=workflow_orchestrator

# Redis
REDIS_HOST=localhost
REDIS_PORT=6379

# JWT
JWT_SECRET=your-secret-key-change-in-production
JWT_EXPIRATION=24h

# Rate Limiting
RATE_LIMIT_RPS=10
RATE_LIMIT_BURST=20
```

## Performance Considerations

### Pagination

- Always use pagination for list endpoints
- Default page size: 20
- Maximum page size: 100
- Total count included in response for UI pagination

### Caching

- Consider implementing response caching for read-heavy endpoints
- Use Redis for distributed caching
- Cache DAG definitions (they change infrequently)

### Database Queries

- All list endpoints use indexed columns
- Prepared statements prevent SQL injection
- Connection pooling optimizes database access

## Future Enhancements

- [ ] OpenAPI/Swagger documentation generation
- [ ] API versioning strategy for breaking changes
- [ ] Webhooks for event notifications
- [ ] GraphQL endpoint for complex queries
- [ ] API key authentication (in addition to JWT)
- [ ] Request/response compression
- [ ] WebSocket support for real-time updates
- [ ] API metrics and monitoring dashboard

## Conclusion

Phase 6 provides a complete, production-ready REST API for the workflow orchestration system. All endpoints are secured, validated, rate-limited, and thoroughly tested. The API follows REST best practices and provides a solid foundation for building UI applications and integrations.
