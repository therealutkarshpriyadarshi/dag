# Phase 1: Core DAG Engine

This document describes the Phase 1 implementation of the workflow orchestration system, which includes the core DAG (Directed Acyclic Graph) engine with parsing, validation, and graph algorithms.

## Table of Contents

- [Overview](#overview)
- [Features](#features)
- [DAG Definition](#dag-definition)
  - [Using the Builder Pattern (Go DSL)](#using-the-builder-pattern-go-dsl)
  - [Using YAML Files](#using-yaml-files)
  - [Using JSON Files](#using-json-files)
- [DAG Validation](#dag-validation)
- [Graph Algorithms](#graph-algorithms)
  - [Topological Sorting](#topological-sorting)
  - [Parallel Task Detection](#parallel-task-detection)
  - [Critical Path Analysis](#critical-path-analysis)
  - [Task Lineage Tracking](#task-lineage-tracking)
- [API Reference](#api-reference)
- [Examples](#examples)

## Overview

Phase 1 implements the foundational DAG engine that enables users to:

1. Define workflows as DAGs using multiple methods (Go DSL, YAML, JSON)
2. Validate DAG structure and dependencies
3. Analyze DAG structure using graph algorithms
4. Calculate execution plans and critical paths

## Features

### Implemented in Phase 1

✅ **DAG Definition & Parsing**
- Go-based builder pattern (fluent API)
- YAML file parsing
- JSON file parsing
- Support for multiple task types (Bash, HTTP, Python, Go)

✅ **Validation**
- Cycle detection
- Dependency validation
- Orphaned task detection
- Duplicate task ID validation

✅ **Graph Algorithms**
- Topological sorting (Kahn's algorithm)
- Parallel task detection
- Critical path calculation
- Task lineage tracking (upstream/downstream dependencies)
- Root and leaf task identification

## DAG Definition

### Using the Builder Pattern (Go DSL)

The builder pattern provides a fluent, type-safe way to define DAGs in Go code:

```go
package main

import (
    "time"
    "github.com/therealutkarshpriyadarshi/dag/internal/dag"
)

func main() {
    // Create a simple ETL pipeline
    myDAG, err := dag.NewBuilder("etl-pipeline").
        ID("etl-001").
        Description("Daily ETL pipeline for analytics").
        Schedule("0 2 * * *").  // Run at 2 AM daily
        StartDate(time.Now()).
        Tags("production", "etl").
        Task("extract", dag.BashTask("python extract.py").
            Name("Extract Data").
            Retries(3).
            Timeout(30*time.Minute).
            SLA(1*time.Hour)).
        Task("transform", dag.BashTask("python transform.py").
            Name("Transform Data").
            DependsOn("extract").
            Retries(2).
            Timeout(15*time.Minute).
            SLA(30*time.Minute)).
        Task("load", dag.BashTask("python load.py").
            Name("Load Data").
            DependsOn("transform").
            Retries(3).
            Timeout(20*time.Minute).
            SLA(45*time.Minute)).
        Build()

    if err != nil {
        panic(err)
    }

    // DAG is now validated and ready to use
}
```

### Task Types

The builder supports multiple task types:

```go
// Bash/Shell tasks
task1 := dag.BashTask("echo 'Hello World'")

// HTTP/REST tasks
task2 := dag.HTTPTask("https://api.example.com/endpoint")

// Python tasks
task3 := dag.PythonTask("my_script.py")

// Go function tasks
task4 := dag.GoTask("MyFunction")
```

### Using YAML Files

Define DAGs in YAML format for easier configuration management:

```yaml
# etl-pipeline.yaml
id: etl-pipeline
name: ETL Pipeline
description: Daily ETL pipeline for analytics
schedule: "0 2 * * *"
start_date: "2024-01-01"
tags:
  - production
  - etl
tasks:
  - id: extract
    name: Extract Data
    type: bash
    command: python extract.py
    retries: 3
    timeout: 30m
    sla: 1h

  - id: transform
    name: Transform Data
    type: bash
    command: python transform.py
    dependencies:
      - extract
    retries: 2
    timeout: 15m
    sla: 30m

  - id: load
    name: Load Data
    type: bash
    command: python load.py
    dependencies:
      - transform
    retries: 3
    timeout: 20m
    sla: 45m
```

Parse the YAML file:

```go
parser := dag.NewParser()
myDAG, err := parser.ParseYAMLFile("etl-pipeline.yaml")
if err != nil {
    panic(err)
}
```

### Using JSON Files

JSON format is also supported:

```json
{
  "id": "etl-pipeline",
  "name": "ETL Pipeline",
  "description": "Daily ETL pipeline for analytics",
  "schedule": "0 2 * * *",
  "start_date": "2024-01-01T00:00:00Z",
  "tags": ["production", "etl"],
  "tasks": [
    {
      "id": "extract",
      "name": "Extract Data",
      "type": "bash",
      "command": "python extract.py",
      "retries": 3,
      "timeout": "30m",
      "sla": "1h"
    },
    {
      "id": "transform",
      "name": "Transform Data",
      "type": "bash",
      "command": "python transform.py",
      "dependencies": ["extract"],
      "retries": 2,
      "timeout": "15m",
      "sla": "30m"
    },
    {
      "id": "load",
      "name": "Load Data",
      "type": "bash",
      "command": "python load.py",
      "dependencies": ["transform"],
      "retries": 3,
      "timeout": "20m",
      "sla": "45m"
    }
  ]
}
```

Parse the JSON file:

```go
parser := dag.NewParser()
myDAG, err := parser.ParseJSONFile("etl-pipeline.json")
if err != nil {
    panic(err)
}
```

## DAG Validation

The validator automatically checks for:

### 1. Basic Structure Validation
- DAG must have a name
- DAG must have at least one task
- No duplicate task IDs

### 2. Dependency Validation
- All task dependencies must reference existing tasks
- No self-dependencies

### 3. Cycle Detection
- DAG must be acyclic (no circular dependencies)
- Uses depth-first search for cycle detection

### 4. Orphaned Task Detection
- In multi-task DAGs, tasks must be connected
- A task is orphaned if it has no dependencies AND no tasks depend on it
- Single-task DAGs are valid

Example validation:

```go
validator := dag.NewValidator()
err := validator.Validate(myDAG)
if err != nil {
    // Handle validation error
    fmt.Printf("Validation failed: %v\n", err)
}
```

## Graph Algorithms

### Topological Sorting

Get tasks in execution order using Kahn's algorithm:

```go
validator := dag.NewValidator()
order, err := validator.GetTopologicalOrder(myDAG)
if err != nil {
    panic(err)
}

fmt.Println("Execution order:", order)
// Output: [extract transform load]
```

### Parallel Task Detection

Find tasks that can be executed in parallel at any given point:

```go
graph := dag.NewGraph(myDAG)

// Initially, only tasks with no dependencies can run
completed := make(map[string]bool)
parallelTasks := graph.GetParallelTasks(completed)
fmt.Println("Can run in parallel:", parallelTasks)
// Output: [extract]

// After extract completes
completed["extract"] = true
parallelTasks = graph.GetParallelTasks(completed)
fmt.Println("Can run in parallel:", parallelTasks)
// Output: [transform]
```

### Critical Path Analysis

Calculate the critical path for SLA estimation:

```go
graph := dag.NewGraph(myDAG)
result, err := graph.CalculateCriticalPath()
if err != nil {
    panic(err)
}

fmt.Printf("Total estimated duration: %v\n", result.TotalDuration)
fmt.Printf("Critical path: %v\n", result.Path)

// Check which tasks are on the critical path
for taskID, isCritical := range result.IsCriticalTask {
    if isCritical {
        fmt.Printf("Task %s is on critical path (slack: %v)\n",
            taskID, result.Slack[taskID])
    }
}
```

The critical path algorithm:
- Uses task SLA or timeout values as duration estimates
- Calculates earliest and latest start times
- Identifies tasks with zero slack (critical tasks)
- Returns the longest path through the DAG

### Task Lineage Tracking

Track upstream and downstream dependencies:

```go
graph := dag.NewGraph(myDAG)

// Get all upstream tasks (dependencies)
upstream, err := graph.GetUpstreamTasks("load")
fmt.Println("Upstream tasks:", upstream)
// Output: [extract transform]

// Get all downstream tasks (dependents)
downstream, err := graph.GetDownstreamTasks("extract")
fmt.Println("Downstream tasks:", downstream)
// Output: [transform load]

// Get immediate dependencies
immediateDeps, err := graph.GetImmediateDependencies("transform")
fmt.Println("Immediate dependencies:", immediateDeps)
// Output: [extract]

// Get immediate dependents
immediateDependents, err := graph.GetImmediateDependents("extract")
fmt.Println("Immediate dependents:", immediateDependents)
// Output: [transform]
```

### Root and Leaf Tasks

```go
graph := dag.NewGraph(myDAG)

// Get tasks with no dependencies
roots := graph.GetRootTasks()
fmt.Println("Root tasks:", roots)
// Output: [extract]

// Get tasks that nothing depends on
leaves := graph.GetLeafTasks()
fmt.Println("Leaf tasks:", leaves)
// Output: [load]
```

## API Reference

### Builder API

```go
// Create a new DAG builder
builder := dag.NewBuilder(name string) *Builder

// Builder methods (all chainable)
builder.ID(id string) *Builder
builder.Description(desc string) *Builder
builder.Schedule(cronExpr string) *Builder
builder.StartDate(t time.Time) *Builder
builder.EndDate(t time.Time) *Builder
builder.Tags(tags ...string) *Builder
builder.Paused(paused bool) *Builder
builder.Task(id string, taskBuilder *TaskBuilder) *Builder
builder.Build() (*models.DAG, error)
builder.MustBuild() *models.DAG  // Panics on error
```

### Task Builder API

```go
// Create task builders
dag.BashTask(command string) *TaskBuilder
dag.HTTPTask(url string) *TaskBuilder
dag.PythonTask(script string) *TaskBuilder
dag.GoTask(funcName string) *TaskBuilder

// Task builder methods (all chainable)
taskBuilder.Name(name string) *TaskBuilder
taskBuilder.DependsOn(taskIDs ...string) *TaskBuilder
taskBuilder.Retries(count int) *TaskBuilder
taskBuilder.Timeout(duration time.Duration) *TaskBuilder
taskBuilder.SLA(duration time.Duration) *TaskBuilder
```

### Parser API

```go
// Create a parser
parser := dag.NewParser() *Parser

// Parse from files
parser.ParseYAMLFile(filepath string) (*models.DAG, error)
parser.ParseJSONFile(filepath string) (*models.DAG, error)

// Parse from bytes
parser.ParseYAML(data []byte) (*models.DAG, error)
parser.ParseJSON(data []byte) (*models.DAG, error)
```

### Validator API

```go
// Create a validator
validator := dag.NewValidator() *Validator

// Validate a DAG
validator.Validate(dag *models.DAG) error

// Get topological order
validator.GetTopologicalOrder(dag *models.DAG) ([]string, error)
```

### Graph API

```go
// Create a graph
graph := dag.NewGraph(dag *models.DAG) *Graph

// Execution planning
graph.GetParallelTasks(completed map[string]bool) []string

// Lineage tracking
graph.GetUpstreamTasks(taskID string) ([]string, error)
graph.GetDownstreamTasks(taskID string) ([]string, error)
graph.GetImmediateDependencies(taskID string) ([]string, error)
graph.GetImmediateDependents(taskID string) ([]string, error)

// Graph structure
graph.GetRootTasks() []string
graph.GetLeafTasks() []string
graph.GetTaskCount() int
graph.GetTask(taskID string) (*models.Task, error)

// Critical path analysis
graph.CalculateCriticalPath() (*CriticalPathResult, error)
```

## Examples

### Example 1: Simple Linear Pipeline

```go
dag, _ := dag.NewBuilder("simple-pipeline").
    Task("step1", dag.BashTask("echo 'Step 1'")).
    Task("step2", dag.BashTask("echo 'Step 2'").DependsOn("step1")).
    Task("step3", dag.BashTask("echo 'Step 3'").DependsOn("step2")).
    Build()
```

### Example 2: Fan-out/Fan-in Pattern

```go
dag, _ := dag.NewBuilder("fan-pattern").
    Task("start", dag.BashTask("echo 'Starting'")).
    Task("parallel1", dag.BashTask("task1.sh").DependsOn("start")).
    Task("parallel2", dag.BashTask("task2.sh").DependsOn("start")).
    Task("parallel3", dag.BashTask("task3.sh").DependsOn("start")).
    Task("end", dag.BashTask("echo 'Done'").
        DependsOn("parallel1", "parallel2", "parallel3")).
    Build()
```

### Example 3: Complex Data Pipeline

```yaml
id: data-pipeline
name: Complex Data Pipeline
schedule: "0 3 * * *"
start_date: "2024-01-01"
tasks:
  - id: extract_db
    name: Extract from Database
    type: python
    command: extract_db.py
    timeout: 1h
    sla: 2h

  - id: extract_api
    name: Extract from API
    type: http
    command: https://api.example.com/data
    retries: 5
    timeout: 30m
    sla: 1h

  - id: merge_data
    name: Merge Data Sources
    type: python
    command: merge.py
    dependencies:
      - extract_db
      - extract_api
    timeout: 30m
    sla: 1h

  - id: validate
    name: Validate Data
    type: python
    command: validate.py
    dependencies:
      - merge_data
    timeout: 15m
    sla: 30m

  - id: transform
    name: Transform Data
    type: python
    command: transform.py
    dependencies:
      - validate
    timeout: 45m
    sla: 1h30m

  - id: load_warehouse
    name: Load to Warehouse
    type: python
    command: load_warehouse.py
    dependencies:
      - transform
    timeout: 1h
    sla: 2h

  - id: update_cache
    name: Update Cache
    type: python
    command: update_cache.py
    dependencies:
      - transform
    timeout: 10m
    sla: 20m
```

## Testing

Phase 1 includes comprehensive tests with >95% coverage:

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test ./... -coverprofile=coverage.out

# View coverage report
go tool cover -html=coverage.out
```

## Next Steps

Phase 1 provides the foundation for the DAG engine. The next phases will add:

- **Phase 2**: Database layer and state management
- **Phase 3**: Scheduler for cron-based execution
- **Phase 4**: Distributed executor and worker system
- **Phase 5**: Retry logic and error handling
- **Phase 6**: REST API
- **Phase 7**: Web UI

## Troubleshooting

### Common Validation Errors

**"DAG name cannot be empty"**
- Ensure you provide a name when creating the DAG

**"duplicate task ID: xxx"**
- Each task must have a unique ID within the DAG

**"task xxx depends on non-existent task: yyy"**
- Verify all task IDs in dependencies exist

**"cycle detected involving task: xxx"**
- DAGs must be acyclic - remove circular dependencies

**"orphaned task detected: xxx"**
- In multi-task DAGs, ensure all tasks are connected to the graph

### Duration Format

When using YAML/JSON, durations use Go's duration format:
- `1h` = 1 hour
- `30m` = 30 minutes
- `1h30m` = 1 hour 30 minutes
- `2h45m30s` = 2 hours 45 minutes 30 seconds

### Date Format

Supported date formats:
- RFC3339: `2024-01-01T00:00:00Z`
- Date only: `2024-01-01`

## License

MIT License - See LICENSE file for details
