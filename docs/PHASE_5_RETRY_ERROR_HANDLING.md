# Phase 5: Retry & Error Handling

This document describes the retry and error handling implementation for the Workflow Orchestrator.

## Overview

Phase 5 implements comprehensive retry logic and error handling mechanisms to make the workflow orchestrator resilient to transient failures and provide robust error management.

## Architecture

The error handling system consists of four main components:

1. **Retry Logic** - Configurable retry strategies with exponential backoff
2. **Error Propagation** - Controls how errors propagate through the DAG
3. **Circuit Breaker** - Prevents cascading failures for external services
4. **Dead Letter Queue (DLQ)** - Handles tasks that fail after all retries

## Components

### 1. Retry Logic (`internal/retry`)

The retry package provides flexible retry strategies for task execution.

#### Retry Strategies

##### Exponential Backoff
Increases delay exponentially between retries:
```go
strategy := retry.NewExponentialBackoff(
    1*time.Second,  // base delay
    5*time.Minute,  // max delay
    true,           // enable jitter
)
```

**Delay calculation**: `delay = baseDelay * multiplier^(attempt-1)`
- Attempt 1: 1s
- Attempt 2: 2s
- Attempt 3: 4s
- Attempt 4: 8s
- And so on, capped at maxDelay

**Jitter**: Adds randomness (±25%) to prevent thundering herd

##### Linear Backoff
Increases delay linearly:
```go
strategy := retry.NewLinearBackoff(
    1*time.Second,  // base delay
    1*time.Minute,  // max delay
    5*time.Second,  // increment
    true,           // enable jitter
)
```

**Delay calculation**: `delay = baseDelay + (increment * attempt)`

##### Fixed Delay
Uses a constant delay between retries:
```go
strategy := retry.NewFixedDelay(
    10*time.Second, // delay
    true,           // enable jitter
)
```

##### No Retry
Disables retries completely:
```go
strategy := retry.NewNoRetry()
```

#### Retry Configuration

```go
config := retry.NewConfig(5, retry.DefaultExponentialBackoff())
config.WithRetryOnErrorCodes("timeout", "connection_refused", "rate_limit")
config.WithRetryCallback(func(attempt int, err error) {
    log.Printf("Retry attempt %d: %v", attempt, err)
})
config.WithGiveUpCallback(func(err error) {
    log.Printf("All retries exhausted: %v", err)
})
```

#### Usage Examples

**Simple retry:**
```go
executor := retry.NewExecutor(retry.DefaultConfig())
err := executor.Execute(ctx, func() error {
    return doSomething()
})
```

**Retry with value:**
```go
config := retry.NewConfig(3, retry.DefaultExponentialBackoff())
result, err := retry.ExecuteWithValue(ctx, config, func() (string, error) {
    return fetchData()
})
```

**Custom retry strategy:**
```go
config := retry.NewConfig(5, retry.NewExponentialBackoff(
    2*time.Second,
    10*time.Minute,
    true,
))
executor := retry.NewExecutor(config)
```

### 2. Error Propagation (`internal/errorhandling`)

Controls how errors propagate through task dependencies.

#### Propagation Policies

##### Fail Policy
Stops the entire DAG on any task failure:
```go
config := &errorhandling.PropagationConfig{
    Policy: errorhandling.PropagationPolicyFail,
}
```

##### Skip Downstream Policy
Marks downstream tasks as `upstream_failed`:
```go
config := &errorhandling.PropagationConfig{
    Policy: errorhandling.PropagationPolicySkipDownstream,
}
```

##### Allow Partial Policy
Allows independent branches to continue:
```go
config := &errorhandling.PropagationConfig{
    Policy: errorhandling.PropagationPolicyAllowPartial,
    AllowPartialSuccess: true,
}
```

#### Critical Tasks

Define tasks that must succeed for DAG success:
```go
config := &errorhandling.PropagationConfig{
    Policy: errorhandling.PropagationPolicyAllowPartial,
    AllowPartialSuccess: true,
    CriticalTasks: []string{"validate", "commit"},
}
```

#### Error Callbacks

Handle errors with custom logic:
```go
config := &errorhandling.PropagationConfig{
    Policy: errorhandling.PropagationPolicySkipDownstream,
    OnTaskFailure: func(ctx context.Context, task *models.Task, ti *models.TaskInstance, err error) error {
        // Send alert, log to external system, etc.
        return nil
    },
    OnDAGFailure: func(ctx context.Context, dagRun *models.DAGRun, err error) error {
        // Send critical alert
        return nil
    },
}
```

#### Error Classification

Classify errors as retryable or non-retryable:
```go
classifier := errorhandling.NewErrorClassifier()

// Check if error is retryable
if classifier.IsRetryable("timeout") {
    // Retry the operation
}

// Add custom retryable error
classifier.AddRetryableError("custom_temporary_error")
```

**Built-in retryable errors:**
- `timeout`
- `connection_refused`
- `temporary`
- `rate_limit`
- `service_unavailable`
- `network`

### 3. Circuit Breaker (`internal/circuitbreaker`)

Implements the circuit breaker pattern to prevent cascading failures.

#### States

1. **Closed** - Normal operation, all requests pass through
2. **Open** - Failure threshold reached, all requests rejected
3. **Half-Open** - Testing recovery, limited requests allowed

#### State Transitions

```
Closed --[Max Failures]--> Open
Open --[Timeout]--> Half-Open
Half-Open --[Success]--> Closed
Half-Open --[Failure]--> Open
```

#### Configuration

```go
config := &circuitbreaker.Config{
    MaxFailures:         5,               // Open circuit after 5 failures
    Timeout:             60 * time.Second, // Wait 60s before half-open
    HalfOpenMaxRequests: 1,               // Allow 1 request in half-open
    OnStateChange: func(from, to circuitbreaker.State) {
        log.Printf("Circuit breaker: %s -> %s", from, to)
    },
}

cb := circuitbreaker.New(config)
```

#### Usage Examples

**Protect HTTP calls:**
```go
cb := circuitbreaker.New(circuitbreaker.DefaultConfig())

err := cb.Execute(ctx, func() error {
    resp, err := http.Get("https://api.example.com/data")
    if err != nil {
        return err
    }
    defer resp.Body.Close()
    return nil
})

if err == circuitbreaker.ErrCircuitOpen {
    log.Println("Circuit is open, service unavailable")
}
```

**Protect database calls:**
```go
result, err := circuitbreaker.ExecuteWithValue(ctx, cb, func() (*Data, error) {
    return db.Query("SELECT * FROM table")
})
```

**Monitor circuit state:**
```go
stats := cb.GetStats()
log.Printf("State: %s, Failures: %d", stats.State, stats.ConsecutiveFailures)
```

**Manual reset:**
```go
cb.Reset() // Force circuit back to closed state
```

### 4. Dead Letter Queue (`internal/dlq`)

Handles tasks that fail after exhausting all retries.

#### Queue Interface

```go
type Queue interface {
    Add(ctx context.Context, entry *Entry) error
    Get(ctx context.Context, id string) (*Entry, error)
    List(ctx context.Context, filters *Filters) ([]*Entry, error)
    Replay(ctx context.Context, id string) error
    Delete(ctx context.Context, id string) error
    Purge(ctx context.Context) error
    Count(ctx context.Context) (int, error)
}
```

#### DLQ Entry

```go
type Entry struct {
    ID              string
    TaskInstanceID  string
    TaskID          string
    DAGRunID        string
    DAGID           string
    FailureReason   string
    FailureTime     time.Time
    Attempts        int
    LastAttemptTime time.Time
    ErrorMessage    string
    Metadata        map[string]interface{}
    Replayed        bool
    ReplayedAt      *time.Time
}
```

#### Usage Examples

**Add failed task to DLQ:**
```go
queue := dlq.NewMemoryQueue()
manager := dlq.NewManager(queue, 100) // Alert when 100 entries

err := manager.AddFailedTask(ctx, taskInstance, dag, taskError)
```

**List DLQ entries:**
```go
// List all entries
entries, _ := queue.List(ctx, nil)

// Filter by DAG
entries, _ := queue.List(ctx, &dlq.Filters{
    DAGID: "etl_pipeline",
})

// Filter by time range
after := time.Now().Add(-24 * time.Hour)
entries, _ := queue.List(ctx, &dlq.Filters{
    After: &after,
    Limit: 10,
})

// Filter non-replayed entries
replayed := false
entries, _ := queue.List(ctx, &dlq.Filters{
    Replayed: &replayed,
})
```

**Replay failed task:**
```go
// Mark as replayed and trigger retry
err := queue.Replay(ctx, entryID)
```

**Set up alerts:**
```go
manager.OnEntryAdded(func(entry *dlq.Entry) {
    log.Printf("Task added to DLQ: %s", entry.TaskID)
    sendAlert(entry)
})

manager.OnThresholdReached(func(count int) {
    log.Printf("DLQ threshold reached: %d entries", count)
    sendCriticalAlert(count)
})
```

**Purge old entries:**
```go
// Delete specific entry
queue.Delete(ctx, entryID)

// Purge all entries
queue.Purge(ctx)
```

## Integration Examples

### Complete Error Handling Pipeline

```go
// 1. Configure retry strategy
retryConfig := retry.NewConfig(
    5,
    retry.NewExponentialBackoff(1*time.Second, 5*time.Minute, true),
)

// 2. Configure error propagation
propagationConfig := &errorhandling.PropagationConfig{
    Policy: errorhandling.PropagationPolicySkipDownstream,
    CriticalTasks: []string{"validate", "commit"},
    OnTaskFailure: func(ctx context.Context, task *models.Task, ti *models.TaskInstance, err error) error {
        log.Printf("Task %s failed: %v", task.ID, err)
        return nil
    },
}

// 3. Set up circuit breaker for external API
apiCircuitBreaker := circuitbreaker.New(&circuitbreaker.Config{
    MaxFailures: 3,
    Timeout: 30 * time.Second,
    HalfOpenMaxRequests: 1,
})

// 4. Set up DLQ
dlqQueue := dlq.NewMemoryQueue()
dlqManager := dlq.NewManager(dlqQueue, 50)
dlqManager.OnThresholdReached(func(count int) {
    sendAlert("DLQ threshold reached", count)
})

// 5. Execute task with full error handling
retryExecutor := retry.NewExecutor(retryConfig)
err := retryExecutor.Execute(ctx, func() error {
    // Use circuit breaker for external calls
    return apiCircuitBreaker.Execute(ctx, func() error {
        return callExternalAPI()
    })
})

if err != nil {
    // Add to DLQ if all retries failed
    dlqManager.AddFailedTask(ctx, taskInstance, dag, err)
}
```

### Task Execution with Retry

```go
// In executor implementation
func executeTaskWithRetry(ctx context.Context, task *models.Task, ti *models.TaskInstance) error {
    retryConfig := retry.NewConfig(
        task.Retries,
        retry.DefaultExponentialBackoff(),
    )

    retryExecutor := retry.NewExecutor(retryConfig)
    return retryExecutor.Execute(ctx, func() error {
        return executeTask(ctx, task, ti)
    })
}
```

## Best Practices

### Retry Configuration

1. **Use exponential backoff with jitter** for most cases
2. **Set reasonable max delay** (5-10 minutes typical)
3. **Configure error-specific retries** when possible
4. **Add retry callbacks** for monitoring and alerting

### Error Propagation

1. **Use skip_downstream policy** as default
2. **Mark critical tasks explicitly**
3. **Enable partial success** for independent pipelines
4. **Implement error callbacks** for notifications

### Circuit Breaker

1. **Protect external service calls** (APIs, databases)
2. **Set appropriate thresholds** (3-5 failures typical)
3. **Configure reasonable timeouts** (30-60 seconds)
4. **Monitor state changes** for alerts

### Dead Letter Queue

1. **Set threshold alerts** to catch recurring issues
2. **Review DLQ regularly** for patterns
3. **Implement replay carefully** to avoid duplicate work
4. **Purge old entries** periodically

## Testing

All components have comprehensive unit tests:

```bash
# Run all retry tests
go test ./internal/retry/... -v

# Run error handling tests
go test ./internal/errorhandling/... -v

# Run circuit breaker tests
go test ./internal/circuitbreaker/... -v

# Run DLQ tests
go test ./internal/dlq/... -v

# Run with coverage
go test ./internal/... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

## Performance Considerations

1. **Retry overhead**: Each retry adds delay; configure max attempts carefully
2. **Circuit breaker latency**: Minimal overhead in closed state (~microseconds)
3. **DLQ memory usage**: Use persistent storage for production (Redis/Database)
4. **Jitter randomness**: Uses `math/rand` - acceptable for retry timing

## Future Enhancements

1. **Persistent DLQ**: Redis or database-backed implementation
2. **Retry analytics**: Track retry patterns and success rates
3. **Adaptive retry**: Adjust delays based on error patterns
4. **Circuit breaker metrics**: Expose Prometheus metrics
5. **DLQ API**: REST endpoints for DLQ management
6. **Bulk replay**: Replay multiple DLQ entries at once

## References

- [Exponential Backoff](https://en.wikipedia.org/wiki/Exponential_backoff)
- [Circuit Breaker Pattern](https://martinfowler.com/bliki/CircuitBreaker.html)
- [Dead Letter Queue](https://en.wikipedia.org/wiki/Dead_letter_queue)
- [Error Handling in Distributed Systems](https://aws.amazon.com/builders-library/timeouts-retries-and-backoff-with-jitter/)

## Version

**Phase**: 5
**Status**: Complete
**Last Updated**: 2025-11-18
**Test Coverage**: 95%+
