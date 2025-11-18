package dag

import (
	"testing"
	"time"

	"github.com/therealutkarshpriyadarshi/dag/pkg/models"
)

func TestValidate_EmptyName(t *testing.T) {
	validator := NewValidator()
	dag := &models.DAG{
		Name:  "",
		Tasks: []models.Task{{ID: "task1", Name: "Task 1", Type: models.TaskTypeBash}},
	}

	err := validator.Validate(dag)
	if err == nil {
		t.Error("Expected error for empty DAG name, got nil")
	}
}

func TestValidate_NoTasks(t *testing.T) {
	validator := NewValidator()
	dag := &models.DAG{
		Name:  "test-dag",
		Tasks: []models.Task{},
	}

	err := validator.Validate(dag)
	if err == nil {
		t.Error("Expected error for DAG with no tasks, got nil")
	}
}

func TestValidate_DuplicateTaskIDs(t *testing.T) {
	validator := NewValidator()
	dag := &models.DAG{
		Name: "test-dag",
		Tasks: []models.Task{
			{ID: "task1", Name: "Task 1", Type: models.TaskTypeBash},
			{ID: "task1", Name: "Task 2", Type: models.TaskTypeBash},
		},
	}

	err := validator.Validate(dag)
	if err == nil {
		t.Error("Expected error for duplicate task IDs, got nil")
	}
}

func TestValidate_NonExistentDependency(t *testing.T) {
	validator := NewValidator()
	dag := &models.DAG{
		Name: "test-dag",
		Tasks: []models.Task{
			{ID: "task1", Name: "Task 1", Type: models.TaskTypeBash, Dependencies: []string{"task2"}},
		},
	}

	err := validator.Validate(dag)
	if err == nil {
		t.Error("Expected error for non-existent dependency, got nil")
	}
}

func TestValidate_ValidDAG(t *testing.T) {
	validator := NewValidator()
	dag := &models.DAG{
		Name: "test-dag",
		Tasks: []models.Task{
			{ID: "task1", Name: "Task 1", Type: models.TaskTypeBash},
			{ID: "task2", Name: "Task 2", Type: models.TaskTypeBash, Dependencies: []string{"task1"}},
		},
	}

	err := validator.Validate(dag)
	if err != nil {
		t.Errorf("Expected no error for valid DAG, got: %v", err)
	}
}

func TestDetectCycle(t *testing.T) {
	validator := NewValidator()
	dag := &models.DAG{
		Name: "test-dag",
		Tasks: []models.Task{
			{ID: "task1", Name: "Task 1", Type: models.TaskTypeBash, Dependencies: []string{"task2"}},
			{ID: "task2", Name: "Task 2", Type: models.TaskTypeBash, Dependencies: []string{"task1"}},
		},
	}

	err := validator.Validate(dag)
	if err == nil {
		t.Error("Expected error for cyclic DAG, got nil")
	}
}

func TestGetTopologicalOrder(t *testing.T) {
	validator := NewValidator()
	dag := &models.DAG{
		Name: "test-dag",
		Tasks: []models.Task{
			{ID: "task1", Name: "Task 1", Type: models.TaskTypeBash},
			{ID: "task2", Name: "Task 2", Type: models.TaskTypeBash, Dependencies: []string{"task1"}},
			{ID: "task3", Name: "Task 3", Type: models.TaskTypeBash, Dependencies: []string{"task1"}},
			{ID: "task4", Name: "Task 4", Type: models.TaskTypeBash, Dependencies: []string{"task2", "task3"}},
		},
		StartDate: time.Now(),
	}

	order, err := validator.GetTopologicalOrder(dag)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(order) != 4 {
		t.Errorf("Expected 4 tasks in topological order, got %d", len(order))
	}

	// task1 should be first
	if order[0] != "task1" {
		t.Errorf("Expected task1 to be first, got %s", order[0])
	}

	// task4 should be last
	if order[3] != "task4" {
		t.Errorf("Expected task4 to be last, got %s", order[3])
	}

	// Verify dependencies are satisfied
	position := make(map[string]int)
	for i, taskID := range order {
		position[taskID] = i
	}

	for _, task := range dag.Tasks {
		for _, depID := range task.Dependencies {
			if position[depID] >= position[task.ID] {
				t.Errorf("Dependency %s should come before %s", depID, task.ID)
			}
		}
	}
}

func TestGetTopologicalOrder_WithCycle(t *testing.T) {
	validator := NewValidator()
	dag := &models.DAG{
		Name: "test-dag",
		Tasks: []models.Task{
			{ID: "task1", Name: "Task 1", Type: models.TaskTypeBash, Dependencies: []string{"task2"}},
			{ID: "task2", Name: "Task 2", Type: models.TaskTypeBash, Dependencies: []string{"task1"}},
		},
	}

	_, err := validator.GetTopologicalOrder(dag)
	if err == nil {
		t.Error("Expected error for cyclic DAG, got nil")
	}
}

func TestValidate_OrphanedTask(t *testing.T) {
	validator := NewValidator()
	dag := &models.DAG{
		Name: "test-dag",
		Tasks: []models.Task{
			{ID: "task1", Name: "Task 1", Type: models.TaskTypeBash},
			{ID: "task2", Name: "Task 2", Type: models.TaskTypeBash},
		},
	}

	err := validator.Validate(dag)
	if err == nil {
		t.Error("Expected error for orphaned task, got nil")
	}
}

func TestValidate_ConnectedDAG(t *testing.T) {
	validator := NewValidator()
	dag := &models.DAG{
		Name: "test-dag",
		Tasks: []models.Task{
			{ID: "task1", Name: "Task 1", Type: models.TaskTypeBash},
			{ID: "task2", Name: "Task 2", Type: models.TaskTypeBash, Dependencies: []string{"task1"}},
			{ID: "task3", Name: "Task 3", Type: models.TaskTypeBash, Dependencies: []string{"task1"}},
		},
	}

	err := validator.Validate(dag)
	if err != nil {
		t.Errorf("Expected no error for connected DAG, got: %v", err)
	}
}

func TestValidate_SingleTask(t *testing.T) {
	validator := NewValidator()
	dag := &models.DAG{
		Name: "test-dag",
		Tasks: []models.Task{
			{ID: "task1", Name: "Task 1", Type: models.TaskTypeBash},
		},
	}

	err := validator.Validate(dag)
	if err != nil {
		t.Errorf("Expected no error for single task DAG, got: %v", err)
	}
}

func TestValidate_ComplexConnectedDAG(t *testing.T) {
	validator := NewValidator()
	dag := &models.DAG{
		Name: "test-dag",
		Tasks: []models.Task{
			{ID: "task1", Name: "Task 1", Type: models.TaskTypeBash},
			{ID: "task2", Name: "Task 2", Type: models.TaskTypeBash, Dependencies: []string{"task1"}},
			{ID: "task3", Name: "Task 3", Type: models.TaskTypeBash, Dependencies: []string{"task1"}},
			{ID: "task4", Name: "Task 4", Type: models.TaskTypeBash, Dependencies: []string{"task2", "task3"}},
		},
		StartDate: time.Now(),
	}

	err := validator.Validate(dag)
	if err != nil {
		t.Errorf("Expected no error for complex connected DAG, got: %v", err)
	}
}
