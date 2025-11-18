package dag

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/therealutkarshpriyadarshi/dag/pkg/models"
)

func TestParseYAML_ValidDAG(t *testing.T) {
	yamlData := []byte(`
id: test-dag
name: Test DAG
description: A test DAG for unit testing
schedule: "0 0 * * *"
start_date: "2024-01-01"
tags:
  - test
  - example
tasks:
  - id: task1
    name: Task 1
    type: bash
    command: echo hello
    retries: 3
    timeout: 5m
    sla: 10m
  - id: task2
    name: Task 2
    type: bash
    command: echo world
    dependencies:
      - task1
    retries: 2
    timeout: 3m
`)

	parser := NewParser()
	dag, err := parser.ParseYAML(yamlData)

	if err != nil {
		t.Fatalf("Failed to parse YAML: %v", err)
	}

	if dag.ID != "test-dag" {
		t.Errorf("Expected ID 'test-dag', got '%s'", dag.ID)
	}
	if dag.Name != "Test DAG" {
		t.Errorf("Expected name 'Test DAG', got '%s'", dag.Name)
	}
	if dag.Description != "A test DAG for unit testing" {
		t.Errorf("Expected description, got '%s'", dag.Description)
	}
	if dag.Schedule != "0 0 * * *" {
		t.Errorf("Expected schedule '0 0 * * *', got '%s'", dag.Schedule)
	}
	if len(dag.Tags) != 2 {
		t.Errorf("Expected 2 tags, got %d", len(dag.Tags))
	}
	if len(dag.Tasks) != 2 {
		t.Fatalf("Expected 2 tasks, got %d", len(dag.Tasks))
	}

	// Verify task1
	task1 := dag.Tasks[0]
	if task1.ID != "task1" {
		t.Errorf("Expected task ID 'task1', got '%s'", task1.ID)
	}
	if task1.Type != models.TaskTypeBash {
		t.Errorf("Expected task type bash, got '%s'", task1.Type)
	}
	if task1.Retries != 3 {
		t.Errorf("Expected 3 retries, got %d", task1.Retries)
	}
	if task1.Timeout != 5*time.Minute {
		t.Errorf("Expected 5m timeout, got %v", task1.Timeout)
	}
	if task1.SLA != 10*time.Minute {
		t.Errorf("Expected 10m SLA, got %v", task1.SLA)
	}

	// Verify task2
	task2 := dag.Tasks[1]
	if len(task2.Dependencies) != 1 {
		t.Errorf("Expected 1 dependency, got %d", len(task2.Dependencies))
	}
	if task2.Dependencies[0] != "task1" {
		t.Errorf("Expected dependency 'task1', got '%s'", task2.Dependencies[0])
	}
}

func TestParseJSON_ValidDAG(t *testing.T) {
	jsonData := []byte(`{
  "id": "json-dag",
  "name": "JSON DAG",
  "description": "A DAG from JSON",
  "schedule": "0 1 * * *",
  "start_date": "2024-01-01",
  "tags": ["json", "test"],
  "tasks": [
    {
      "id": "task1",
      "name": "Task 1",
      "type": "http",
      "command": "https://api.example.com/endpoint",
      "timeout": "2m",
      "sla": "5m"
    }
  ]
}`)

	parser := NewParser()
	dag, err := parser.ParseJSON(jsonData)

	if err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	if dag.ID != "json-dag" {
		t.Errorf("Expected ID 'json-dag', got '%s'", dag.ID)
	}
	if dag.Name != "JSON DAG" {
		t.Errorf("Expected name 'JSON DAG', got '%s'", dag.Name)
	}
	if len(dag.Tasks) != 1 {
		t.Fatalf("Expected 1 task, got %d", len(dag.Tasks))
	}

	task := dag.Tasks[0]
	if task.Type != models.TaskTypeHTTP {
		t.Errorf("Expected HTTP task type, got '%s'", task.Type)
	}
	if task.Command != "https://api.example.com/endpoint" {
		t.Errorf("Expected URL command, got '%s'", task.Command)
	}
}

func TestParseYAML_TaskTypes(t *testing.T) {
	yamlData := []byte(`
id: task-types
name: Task Types Test
start_date: "2024-01-01"
tasks:
  - id: bash-task
    name: Bash Task
    type: bash
    command: echo hello
  - id: shell-task
    name: Shell Task
    type: shell
    command: ls
    dependencies:
      - bash-task
  - id: http-task
    name: HTTP Task
    type: http
    command: https://example.com
    dependencies:
      - shell-task
  - id: rest-task
    name: REST Task
    type: rest
    command: https://api.example.com
    dependencies:
      - http-task
  - id: python-task
    name: Python Task
    type: python
    command: script.py
    dependencies:
      - rest-task
  - id: py-task
    name: Py Task
    type: py
    command: script.py
    dependencies:
      - python-task
  - id: go-task
    name: Go Task
    type: go
    command: MyFunc
    dependencies:
      - py-task
  - id: golang-task
    name: Golang Task
    type: golang
    command: MyFunc
    dependencies:
      - go-task
`)

	parser := NewParser()
	dag, err := parser.ParseYAML(yamlData)

	if err != nil {
		t.Fatalf("Failed to parse YAML: %v", err)
	}

	expectedTypes := map[string]models.TaskType{
		"bash-task":   models.TaskTypeBash,
		"shell-task":  models.TaskTypeBash,
		"http-task":   models.TaskTypeHTTP,
		"rest-task":   models.TaskTypeHTTP,
		"python-task": models.TaskTypePython,
		"py-task":     models.TaskTypePython,
		"go-task":     models.TaskTypeGo,
		"golang-task": models.TaskTypeGo,
	}

	for _, task := range dag.Tasks {
		expected, ok := expectedTypes[task.ID]
		if !ok {
			t.Errorf("Unexpected task ID: %s", task.ID)
			continue
		}
		if task.Type != expected {
			t.Errorf("Task %s: expected type %s, got %s", task.ID, expected, task.Type)
		}
	}
}

func TestParseYAML_InvalidTaskType(t *testing.T) {
	yamlData := []byte(`
id: invalid-type
name: Invalid Type
start_date: "2024-01-01"
tasks:
  - id: task1
    name: Task 1
    type: invalid-type
    command: echo hello
`)

	parser := NewParser()
	_, err := parser.ParseYAML(yamlData)

	if err == nil {
		t.Error("Expected error for invalid task type, got nil")
	}
}

func TestParseYAML_InvalidTimeout(t *testing.T) {
	yamlData := []byte(`
id: invalid-timeout
name: Invalid Timeout
start_date: "2024-01-01"
tasks:
  - id: task1
    name: Task 1
    type: bash
    command: echo hello
    timeout: invalid
`)

	parser := NewParser()
	_, err := parser.ParseYAML(yamlData)

	if err == nil {
		t.Error("Expected error for invalid timeout, got nil")
	}
}

func TestParseYAML_InvalidSLA(t *testing.T) {
	yamlData := []byte(`
id: invalid-sla
name: Invalid SLA
start_date: "2024-01-01"
tasks:
  - id: task1
    name: Task 1
    type: bash
    command: echo hello
    sla: invalid
`)

	parser := NewParser()
	_, err := parser.ParseYAML(yamlData)

	if err == nil {
		t.Error("Expected error for invalid SLA, got nil")
	}
}

func TestParseYAML_InvalidStartDate(t *testing.T) {
	yamlData := []byte(`
id: invalid-date
name: Invalid Date
start_date: "invalid-date"
tasks:
  - id: task1
    name: Task 1
    type: bash
    command: echo hello
`)

	parser := NewParser()
	_, err := parser.ParseYAML(yamlData)

	if err == nil {
		t.Error("Expected error for invalid start date, got nil")
	}
}

func TestParseYAML_DateFormats(t *testing.T) {
	tests := []struct {
		name      string
		startDate string
		shouldErr bool
	}{
		{"RFC3339", "2024-01-01T00:00:00Z", false},
		{"Date only", "2024-01-01", false},
		{"Invalid", "01-01-2024", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			yamlData := []byte(`
id: date-test
name: Date Test
start_date: "` + tt.startDate + `"
tasks:
  - id: task1
    name: Task 1
    type: bash
    command: echo hello
`)

			parser := NewParser()
			_, err := parser.ParseYAML(yamlData)

			if tt.shouldErr && err == nil {
				t.Error("Expected error, got nil")
			}
			if !tt.shouldErr && err != nil {
				t.Errorf("Expected no error, got: %v", err)
			}
		})
	}
}

func TestParseYAML_WithEndDate(t *testing.T) {
	yamlData := []byte(`
id: end-date-dag
name: End Date DAG
start_date: "2024-01-01"
end_date: "2024-12-31"
tasks:
  - id: task1
    name: Task 1
    type: bash
    command: echo hello
`)

	parser := NewParser()
	dag, err := parser.ParseYAML(yamlData)

	if err != nil {
		t.Fatalf("Failed to parse YAML: %v", err)
	}

	if dag.EndDate == nil {
		t.Error("Expected end date to be set, got nil")
	}
}

func TestParseYAMLFile(t *testing.T) {
	// Create a temporary YAML file
	tmpDir := t.TempDir()
	yamlFile := filepath.Join(tmpDir, "test-dag.yaml")

	yamlContent := []byte(`
id: file-dag
name: File DAG
start_date: "2024-01-01"
tasks:
  - id: task1
    name: Task 1
    type: bash
    command: echo hello
`)

	if err := os.WriteFile(yamlFile, yamlContent, 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	parser := NewParser()
	dag, err := parser.ParseYAMLFile(yamlFile)

	if err != nil {
		t.Fatalf("Failed to parse YAML file: %v", err)
	}

	if dag.ID != "file-dag" {
		t.Errorf("Expected ID 'file-dag', got '%s'", dag.ID)
	}
}

func TestParseJSONFile(t *testing.T) {
	// Create a temporary JSON file
	tmpDir := t.TempDir()
	jsonFile := filepath.Join(tmpDir, "test-dag.json")

	jsonContent := []byte(`{
  "id": "file-dag",
  "name": "File DAG",
  "start_date": "2024-01-01",
  "tasks": [
    {
      "id": "task1",
      "name": "Task 1",
      "type": "bash",
      "command": "echo hello"
    }
  ]
}`)

	if err := os.WriteFile(jsonFile, jsonContent, 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	parser := NewParser()
	dag, err := parser.ParseJSONFile(jsonFile)

	if err != nil {
		t.Fatalf("Failed to parse JSON file: %v", err)
	}

	if dag.ID != "file-dag" {
		t.Errorf("Expected ID 'file-dag', got '%s'", dag.ID)
	}
}

func TestParseYAML_ValidationFailure(t *testing.T) {
	// DAG with cyclic dependency
	yamlData := []byte(`
id: cyclic-dag
name: Cyclic DAG
start_date: "2024-01-01"
tasks:
  - id: task1
    name: Task 1
    type: bash
    command: echo 1
    dependencies:
      - task2
  - id: task2
    name: Task 2
    type: bash
    command: echo 2
    dependencies:
      - task1
`)

	parser := NewParser()
	_, err := parser.ParseYAML(yamlData)

	if err == nil {
		t.Error("Expected validation error for cyclic DAG, got nil")
	}
}

func TestParseJSON_InvalidJSON(t *testing.T) {
	invalidJSON := []byte(`{invalid json}`)

	parser := NewParser()
	_, err := parser.ParseJSON(invalidJSON)

	if err == nil {
		t.Error("Expected error for invalid JSON, got nil")
	}
}

func TestParseYAML_InvalidYAML(t *testing.T) {
	invalidYAML := []byte(`
invalid: yaml: content:
  - unmatched
`)

	parser := NewParser()
	_, err := parser.ParseYAML(invalidYAML)

	if err == nil {
		t.Error("Expected error for invalid YAML, got nil")
	}
}

func TestParseYAMLFile_NonExistentFile(t *testing.T) {
	parser := NewParser()
	_, err := parser.ParseYAMLFile("/nonexistent/file.yaml")

	if err == nil {
		t.Error("Expected error for non-existent file, got nil")
	}
}

func TestParseJSON_ComplexDAG(t *testing.T) {
	jsonData := []byte(`{
  "id": "complex-dag",
  "name": "Complex DAG",
  "description": "A complex DAG with multiple dependencies",
  "schedule": "0 2 * * *",
  "start_date": "2024-01-01T00:00:00Z",
  "tags": ["production", "etl"],
  "is_paused": false,
  "tasks": [
    {
      "id": "extract",
      "name": "Extract Data",
      "type": "python",
      "command": "extract.py",
      "retries": 3,
      "timeout": "30m",
      "sla": "1h"
    },
    {
      "id": "transform",
      "name": "Transform Data",
      "type": "python",
      "command": "transform.py",
      "dependencies": ["extract"],
      "retries": 2,
      "timeout": "15m",
      "sla": "30m"
    },
    {
      "id": "load",
      "name": "Load Data",
      "type": "python",
      "command": "load.py",
      "dependencies": ["transform"],
      "retries": 3,
      "timeout": "20m",
      "sla": "45m"
    }
  ]
}`)

	parser := NewParser()
	dag, err := parser.ParseJSON(jsonData)

	if err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	if len(dag.Tasks) != 3 {
		t.Errorf("Expected 3 tasks, got %d", len(dag.Tasks))
	}

	// Verify the DAG is valid
	graph := NewGraph(dag)
	order, err := graph.topologicalSort()
	if err != nil {
		t.Errorf("Failed to get topological order: %v", err)
	}
	if len(order) != 3 {
		t.Errorf("Expected 3 tasks in order, got %d", len(order))
	}
}
