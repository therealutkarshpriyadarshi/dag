package dag

import (
	"testing"
	"time"

	"github.com/therealutkarshpriyadarshi/dag/pkg/models"
)

func createTestDAG() *models.DAG {
	return &models.DAG{
		ID:   "test-dag",
		Name: "Test DAG",
		Tasks: []models.Task{
			{
				ID:           "task1",
				Name:         "Task 1",
				Type:         models.TaskTypeBash,
				Command:      "echo task1",
				Dependencies: []string{},
				Timeout:      1 * time.Minute,
				SLA:          2 * time.Minute,
			},
			{
				ID:           "task2",
				Name:         "Task 2",
				Type:         models.TaskTypeBash,
				Command:      "echo task2",
				Dependencies: []string{"task1"},
				Timeout:      2 * time.Minute,
				SLA:          3 * time.Minute,
			},
			{
				ID:           "task3",
				Name:         "Task 3",
				Type:         models.TaskTypeBash,
				Command:      "echo task3",
				Dependencies: []string{"task1"},
				Timeout:      1 * time.Minute,
				SLA:          2 * time.Minute,
			},
			{
				ID:           "task4",
				Name:         "Task 4",
				Type:         models.TaskTypeBash,
				Command:      "echo task4",
				Dependencies: []string{"task2", "task3"},
				Timeout:      3 * time.Minute,
				SLA:          4 * time.Minute,
			},
		},
		StartDate: time.Now(),
	}
}

func TestNewGraph(t *testing.T) {
	dag := createTestDAG()
	graph := NewGraph(dag)

	if graph == nil {
		t.Fatal("Expected graph to be created, got nil")
	}

	if len(graph.tasks) != 4 {
		t.Errorf("Expected 4 tasks, got %d", len(graph.tasks))
	}

	// Verify adjacency list
	if len(graph.adjList["task1"]) != 2 {
		t.Errorf("Expected task1 to have 2 dependents, got %d", len(graph.adjList["task1"]))
	}
}

func TestGetParallelTasks(t *testing.T) {
	dag := createTestDAG()
	graph := NewGraph(dag)

	// No tasks completed - only task1 should be ready
	completed := make(map[string]bool)
	parallel := graph.GetParallelTasks(completed)

	if len(parallel) != 1 {
		t.Errorf("Expected 1 parallel task, got %d", len(parallel))
	}
	if parallel[0] != "task1" {
		t.Errorf("Expected task1 to be parallel, got %s", parallel[0])
	}

	// task1 completed - task2 and task3 should be ready
	completed["task1"] = true
	parallel = graph.GetParallelTasks(completed)

	if len(parallel) != 2 {
		t.Errorf("Expected 2 parallel tasks, got %d", len(parallel))
	}

	// task1, task2, task3 completed - task4 should be ready
	completed["task2"] = true
	completed["task3"] = true
	parallel = graph.GetParallelTasks(completed)

	if len(parallel) != 1 {
		t.Errorf("Expected 1 parallel task, got %d", len(parallel))
	}
	if parallel[0] != "task4" {
		t.Errorf("Expected task4 to be parallel, got %s", parallel[0])
	}

	// All tasks completed - no tasks should be ready
	completed["task4"] = true
	parallel = graph.GetParallelTasks(completed)

	if len(parallel) != 0 {
		t.Errorf("Expected 0 parallel tasks, got %d", len(parallel))
	}
}

func TestGetUpstreamTasks(t *testing.T) {
	dag := createTestDAG()
	graph := NewGraph(dag)

	// task1 has no upstream
	upstream, err := graph.GetUpstreamTasks("task1")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(upstream) != 0 {
		t.Errorf("Expected 0 upstream tasks for task1, got %d", len(upstream))
	}

	// task2 has task1 upstream
	upstream, err = graph.GetUpstreamTasks("task2")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(upstream) != 1 {
		t.Errorf("Expected 1 upstream task for task2, got %d", len(upstream))
	}

	// task4 has task1, task2, task3 upstream
	upstream, err = graph.GetUpstreamTasks("task4")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(upstream) != 3 {
		t.Errorf("Expected 3 upstream tasks for task4, got %d", len(upstream))
	}

	// Test non-existent task
	_, err = graph.GetUpstreamTasks("nonexistent")
	if err == nil {
		t.Error("Expected error for non-existent task, got nil")
	}
}

func TestGetDownstreamTasks(t *testing.T) {
	dag := createTestDAG()
	graph := NewGraph(dag)

	// task4 has no downstream
	downstream, err := graph.GetDownstreamTasks("task4")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(downstream) != 0 {
		t.Errorf("Expected 0 downstream tasks for task4, got %d", len(downstream))
	}

	// task1 has task2, task3, task4 downstream
	downstream, err = graph.GetDownstreamTasks("task1")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(downstream) != 3 {
		t.Errorf("Expected 3 downstream tasks for task1, got %d", len(downstream))
	}

	// task2 has only task4 downstream
	downstream, err = graph.GetDownstreamTasks("task2")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(downstream) != 1 {
		t.Errorf("Expected 1 downstream task for task2, got %d", len(downstream))
	}

	// Test non-existent task
	_, err = graph.GetDownstreamTasks("nonexistent")
	if err == nil {
		t.Error("Expected error for non-existent task, got nil")
	}
}

func TestGetImmediateDependencies(t *testing.T) {
	dag := createTestDAG()
	graph := NewGraph(dag)

	// task1 has no dependencies
	deps, err := graph.GetImmediateDependencies("task1")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(deps) != 0 {
		t.Errorf("Expected 0 dependencies for task1, got %d", len(deps))
	}

	// task4 has task2 and task3 as dependencies
	deps, err = graph.GetImmediateDependencies("task4")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(deps) != 2 {
		t.Errorf("Expected 2 dependencies for task4, got %d", len(deps))
	}
}

func TestGetImmediateDependents(t *testing.T) {
	dag := createTestDAG()
	graph := NewGraph(dag)

	// task1 has task2 and task3 as dependents
	dependents, err := graph.GetImmediateDependents("task1")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(dependents) != 2 {
		t.Errorf("Expected 2 dependents for task1, got %d", len(dependents))
	}

	// task4 has no dependents
	dependents, err = graph.GetImmediateDependents("task4")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(dependents) != 0 {
		t.Errorf("Expected 0 dependents for task4, got %d", len(dependents))
	}
}

func TestCalculateCriticalPath(t *testing.T) {
	dag := createTestDAG()
	graph := NewGraph(dag)

	result, err := graph.CalculateCriticalPath()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result == nil {
		t.Fatal("Expected critical path result, got nil")
	}

	// Critical path should be: task1 -> task2 -> task4
	// Total: 2m + 3m + 4m = 9m
	expectedDuration := 9 * time.Minute
	if result.TotalDuration != expectedDuration {
		t.Errorf("Expected total duration %v, got %v", expectedDuration, result.TotalDuration)
	}

	// Verify critical tasks
	if !result.IsCriticalTask["task1"] {
		t.Error("Expected task1 to be critical")
	}
	if !result.IsCriticalTask["task2"] {
		t.Error("Expected task2 to be critical")
	}
	if !result.IsCriticalTask["task4"] {
		t.Error("Expected task4 to be critical")
	}

	// task3 should not be critical (has slack)
	if result.IsCriticalTask["task3"] {
		t.Error("Expected task3 to not be critical")
	}

	// Verify slack calculation
	if result.Slack["task1"] != 0 {
		t.Errorf("Expected task1 slack to be 0, got %v", result.Slack["task1"])
	}
	if result.Slack["task3"] == 0 {
		t.Error("Expected task3 to have slack > 0")
	}

	// Verify critical path contains critical tasks
	if len(result.Path) == 0 {
		t.Error("Expected non-empty critical path")
	}
}

func TestGetRootTasks(t *testing.T) {
	dag := createTestDAG()
	graph := NewGraph(dag)

	roots := graph.GetRootTasks()

	if len(roots) != 1 {
		t.Errorf("Expected 1 root task, got %d", len(roots))
	}
	if roots[0] != "task1" {
		t.Errorf("Expected task1 to be root, got %s", roots[0])
	}
}

func TestGetLeafTasks(t *testing.T) {
	dag := createTestDAG()
	graph := NewGraph(dag)

	leaves := graph.GetLeafTasks()

	if len(leaves) != 1 {
		t.Errorf("Expected 1 leaf task, got %d", len(leaves))
	}
	if leaves[0] != "task4" {
		t.Errorf("Expected task4 to be leaf, got %s", leaves[0])
	}
}

func TestGetTaskCount(t *testing.T) {
	dag := createTestDAG()
	graph := NewGraph(dag)

	count := graph.GetTaskCount()
	if count != 4 {
		t.Errorf("Expected 4 tasks, got %d", count)
	}
}

func TestGetTask(t *testing.T) {
	dag := createTestDAG()
	graph := NewGraph(dag)

	task, err := graph.GetTask("task1")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if task.ID != "task1" {
		t.Errorf("Expected task1, got %s", task.ID)
	}

	_, err = graph.GetTask("nonexistent")
	if err == nil {
		t.Error("Expected error for non-existent task, got nil")
	}
}

func TestCriticalPath_LinearDAG(t *testing.T) {
	// Create a linear DAG: task1 -> task2 -> task3
	dag := &models.DAG{
		ID:   "linear-dag",
		Name: "Linear DAG",
		Tasks: []models.Task{
			{ID: "task1", Name: "Task 1", Type: models.TaskTypeBash, SLA: 1 * time.Minute},
			{ID: "task2", Name: "Task 2", Type: models.TaskTypeBash, Dependencies: []string{"task1"}, SLA: 2 * time.Minute},
			{ID: "task3", Name: "Task 3", Type: models.TaskTypeBash, Dependencies: []string{"task2"}, SLA: 3 * time.Minute},
		},
		StartDate: time.Now(),
	}

	graph := NewGraph(dag)
	result, err := graph.CalculateCriticalPath()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// All tasks should be critical in a linear DAG
	for taskID := range graph.tasks {
		if !result.IsCriticalTask[taskID] {
			t.Errorf("Expected %s to be critical in linear DAG", taskID)
		}
	}

	// Total duration should be sum of all SLAs
	expectedDuration := 6 * time.Minute
	if result.TotalDuration != expectedDuration {
		t.Errorf("Expected total duration %v, got %v", expectedDuration, result.TotalDuration)
	}
}

func TestCriticalPath_ParallelDAG(t *testing.T) {
	// Create a parallel DAG: task1 splits to task2 and task3, which join at task4
	dag := &models.DAG{
		ID:   "parallel-dag",
		Name: "Parallel DAG",
		Tasks: []models.Task{
			{ID: "start", Name: "Start", Type: models.TaskTypeBash, SLA: 1 * time.Minute},
			{ID: "fast", Name: "Fast", Type: models.TaskTypeBash, Dependencies: []string{"start"}, SLA: 1 * time.Minute},
			{ID: "slow", Name: "Slow", Type: models.TaskTypeBash, Dependencies: []string{"start"}, SLA: 5 * time.Minute},
			{ID: "end", Name: "End", Type: models.TaskTypeBash, Dependencies: []string{"fast", "slow"}, SLA: 1 * time.Minute},
		},
		StartDate: time.Now(),
	}

	graph := NewGraph(dag)
	result, err := graph.CalculateCriticalPath()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Critical path should go through "slow" task
	if !result.IsCriticalTask["start"] {
		t.Error("Expected start to be critical")
	}
	if !result.IsCriticalTask["slow"] {
		t.Error("Expected slow to be critical")
	}
	if !result.IsCriticalTask["end"] {
		t.Error("Expected end to be critical")
	}

	// "fast" should not be critical (has slack)
	if result.IsCriticalTask["fast"] {
		t.Error("Expected fast to not be critical")
	}

	// Verify slack on fast path
	if result.Slack["fast"] == 0 {
		t.Error("Expected fast to have slack > 0")
	}

	// Total duration: 1 + 5 + 1 = 7 minutes
	expectedDuration := 7 * time.Minute
	if result.TotalDuration != expectedDuration {
		t.Errorf("Expected total duration %v, got %v", expectedDuration, result.TotalDuration)
	}
}
