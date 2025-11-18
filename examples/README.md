# DAG Examples

This directory contains example DAG definitions demonstrating various workflow patterns.

## Examples

### 1. ETL Pipeline (`etl-pipeline.yaml`)

A simple Extract-Transform-Load pipeline demonstrating:
- Linear task dependencies
- Retry configuration
- Timeout and SLA settings
- Cron-based scheduling

**Pattern**: Linear pipeline (A → B → C)

```
extract → transform → load
```

### 2. Data Processing Pipeline (`data-processing.yaml`)

A complex multi-source data processing workflow demonstrating:
- Fan-out pattern (multiple parallel extracts)
- Fan-in pattern (merge multiple sources)
- Parallel transformations
- Multiple output destinations

**Pattern**: Fan-out/Fan-in

```
extract_database ─┐
extract_api ──────┼→ merge → validate ─┬→ transform_customer ─┐
extract_files ────┘                     ├→ transform_orders ───┼→ load_warehouse → notify
                                        └→ transform_products ─┴→ update_cache ──┘
```

### 3. ML Training Pipeline (`ml-training.json`)

A machine learning model training and deployment workflow demonstrating:
- Data fetching and preprocessing
- Model training and evaluation
- Model registry integration
- Automated deployment to staging
- Integration testing
- Notification hooks

**Pattern**: Linear with parallel data fetching

```
fetch_training_data ──┐
                      ├→ preprocess → feature_eng → train → evaluate → register → deploy → test → notify
fetch_validation_data ┘
```

## Using These Examples

### With the Parser

```go
package main

import (
    "fmt"
    "github.com/therealutkarshpriyadarshi/dag/internal/dag"
)

func main() {
    // Parse YAML example
    parser := dag.NewParser()
    dag, err := parser.ParseYAMLFile("examples/etl-pipeline.yaml")
    if err != nil {
        panic(err)
    }

    fmt.Printf("Loaded DAG: %s\n", dag.Name)
    fmt.Printf("Tasks: %d\n", len(dag.Tasks))
}
```

### Validating Examples

```bash
# Create a simple validator program
cat > validate_example.go << 'EOF'
package main

import (
    "fmt"
    "os"
    "github.com/therealutkarshpriyadarshi/dag/internal/dag"
)

func main() {
    if len(os.Args) < 2 {
        fmt.Println("Usage: go run validate_example.go <dag-file.yaml|json>")
        os.Exit(1)
    }

    parser := dag.NewParser()
    var myDAG interface{}
    var err error

    if filepath.Ext(os.Args[1]) == ".yaml" || filepath.Ext(os.Args[1]) == ".yml" {
        myDAG, err = parser.ParseYAMLFile(os.Args[1])
    } else {
        myDAG, err = parser.ParseJSONFile(os.Args[1])
    }

    if err != nil {
        fmt.Printf("❌ Validation failed: %v\n", err)
        os.Exit(1)
    }

    fmt.Printf("✅ DAG is valid!\n")
}
EOF

# Validate an example
go run validate_example.go examples/etl-pipeline.yaml
```

### Analyzing Critical Path

```go
package main

import (
    "fmt"
    "github.com/therealutkarshpriyadarshi/dag/internal/dag"
)

func main() {
    parser := dag.NewParser()
    myDAG, err := parser.ParseYAMLFile("examples/data-processing.yaml")
    if err != nil {
        panic(err)
    }

    graph := dag.NewGraph(myDAG)
    result, err := graph.CalculateCriticalPath()
    if err != nil {
        panic(err)
    }

    fmt.Printf("Total Duration: %v\n", result.TotalDuration)
    fmt.Printf("Critical Path: %v\n", result.Path)

    for taskID, slack := range result.Slack {
        if slack == 0 {
            fmt.Printf("  ⚠️  %s (CRITICAL - no slack)\n", taskID)
        } else {
            fmt.Printf("  ✓ %s (slack: %v)\n", taskID, slack)
        }
    }
}
```

## Creating Your Own DAGs

### YAML Template

```yaml
id: my-dag
name: My DAG
description: Description of what this DAG does
schedule: "0 0 * * *"  # Cron expression
start_date: "2024-01-01"
tags:
  - tag1
  - tag2

tasks:
  - id: task_id
    name: Human-readable task name
    type: bash  # or http, python, go
    command: command to execute
    dependencies:  # Optional
      - other_task_id
    retries: 3  # Optional, default 0
    timeout: 30m  # Optional
    sla: 1h  # Optional
```

### JSON Template

```json
{
  "id": "my-dag",
  "name": "My DAG",
  "description": "Description of what this DAG does",
  "schedule": "0 0 * * *",
  "start_date": "2024-01-01",
  "tags": ["tag1", "tag2"],
  "tasks": [
    {
      "id": "task_id",
      "name": "Human-readable task name",
      "type": "bash",
      "command": "command to execute",
      "dependencies": ["other_task_id"],
      "retries": 3,
      "timeout": "30m",
      "sla": "1h"
    }
  ]
}
```

## Common Patterns

### Linear Pipeline
Tasks execute sequentially, one after another.
```
A → B → C → D
```

### Fan-out
One task triggers multiple parallel tasks.
```
    ┌→ B
A ──┼→ C
    └→ D
```

### Fan-in
Multiple tasks converge to a single task.
```
A ─┐
B ─┼→ D
C ─┘
```

### Diamond
Combination of fan-out and fan-in.
```
    ┌→ B ─┐
A ──┤     ├→ D
    └→ C ─┘
```

## Best Practices

1. **Use meaningful task IDs**: Choose descriptive IDs like `extract_customer_data` instead of `task1`

2. **Set appropriate timeouts**: Always set timeouts to prevent tasks from hanging indefinitely

3. **Configure retries wisely**: Use retries for transient failures, but be careful with operations that aren't idempotent

4. **Define SLAs**: Set SLA expectations to track performance and identify bottlenecks

5. **Tag your DAGs**: Use tags to organize and categorize DAGs (e.g., `production`, `staging`, `etl`, `ml`)

6. **Document your DAGs**: Use the description field to explain what the DAG does and why

7. **Keep tasks focused**: Each task should do one thing well

8. **Plan for failure**: Design your DAG to handle task failures gracefully

## Need Help?

See the [Phase 1 DAG Engine Documentation](../docs/PHASE_1_DAG_ENGINE.md) for complete API reference and detailed examples.
