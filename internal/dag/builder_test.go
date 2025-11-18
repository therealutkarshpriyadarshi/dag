package dag

import (
	"testing"
	"time"

	"github.com/therealutkarshpriyadarshi/dag/pkg/models"
)

func TestBuilder_BasicDAG(t *testing.T) {
	dag, err := NewBuilder("test-dag").
		ID("dag-123").
		Description("A test DAG").
		Schedule("0 0 * * *").
		Tags("test", "example").
		Task("task1", BashTask("echo hello")).
		Build()

	if err != nil {
		t.Fatalf("Failed to build DAG: %v", err)
	}

	if dag.ID != "dag-123" {
		t.Errorf("Expected DAG ID 'dag-123', got '%s'", dag.ID)
	}
	if dag.Name != "test-dag" {
		t.Errorf("Expected DAG name 'test-dag', got '%s'", dag.Name)
	}
	if dag.Description != "A test DAG" {
		t.Errorf("Expected description 'A test DAG', got '%s'", dag.Description)
	}
	if dag.Schedule != "0 0 * * *" {
		t.Errorf("Expected schedule '0 0 * * *', got '%s'", dag.Schedule)
	}
	if len(dag.Tags) != 2 {
		t.Errorf("Expected 2 tags, got %d", len(dag.Tags))
	}
	if len(dag.Tasks) != 1 {
		t.Errorf("Expected 1 task, got %d", len(dag.Tasks))
	}
}

func TestBuilder_ComplexDAG(t *testing.T) {
	startDate := time.Now()
	endDate := startDate.Add(30 * 24 * time.Hour)

	dag, err := NewBuilder("etl-pipeline").
		ID("etl-123").
		Description("ETL Pipeline").
		Schedule("0 2 * * *").
		StartDate(startDate).
		EndDate(endDate).
		Tags("etl", "production").
		Task("extract", BashTask("python extract.py").
			Name("Extract Data").
			Retries(3).
			Timeout(30*time.Minute).
			SLA(1*time.Hour)).
		Task("transform", BashTask("python transform.py").
			Name("Transform Data").
			DependsOn("extract").
			Retries(2).
			Timeout(15*time.Minute).
			SLA(30*time.Minute)).
		Task("load", BashTask("python load.py").
			Name("Load Data").
			DependsOn("transform").
			Retries(3).
			Timeout(20*time.Minute).
			SLA(45*time.Minute)).
		Build()

	if err != nil {
		t.Fatalf("Failed to build DAG: %v", err)
	}

	if len(dag.Tasks) != 3 {
		t.Fatalf("Expected 3 tasks, got %d", len(dag.Tasks))
	}

	// Find tasks by ID
	var extractTask, transformTask, loadTask *models.Task
	for i := range dag.Tasks {
		switch dag.Tasks[i].ID {
		case "extract":
			extractTask = &dag.Tasks[i]
		case "transform":
			transformTask = &dag.Tasks[i]
		case "load":
			loadTask = &dag.Tasks[i]
		}
	}

	// Verify extract task
	if extractTask == nil {
		t.Fatal("Extract task not found")
	}
	if extractTask.Name != "Extract Data" {
		t.Errorf("Expected name 'Extract Data', got '%s'", extractTask.Name)
	}
	if extractTask.Retries != 3 {
		t.Errorf("Expected 3 retries, got %d", extractTask.Retries)
	}
	if extractTask.Timeout != 30*time.Minute {
		t.Errorf("Expected 30m timeout, got %v", extractTask.Timeout)
	}

	// Verify transform task dependencies
	if transformTask == nil {
		t.Fatal("Transform task not found")
	}
	if len(transformTask.Dependencies) != 1 {
		t.Errorf("Expected 1 dependency, got %d", len(transformTask.Dependencies))
	}
	if transformTask.Dependencies[0] != "extract" {
		t.Errorf("Expected dependency 'extract', got '%s'", transformTask.Dependencies[0])
	}

	// Verify load task
	if loadTask == nil {
		t.Fatal("Load task not found")
	}
	if len(loadTask.Dependencies) != 1 {
		t.Errorf("Expected 1 dependency, got %d", len(loadTask.Dependencies))
	}
}

func TestBuilder_TaskTypes(t *testing.T) {
	dag, err := NewBuilder("task-types").
		Task("bash-task", BashTask("echo hello")).
		Task("http-task", HTTPTask("https://api.example.com").DependsOn("bash-task")).
		Task("python-task", PythonTask("script.py").DependsOn("http-task")).
		Task("go-task", GoTask("MyFunction").DependsOn("python-task")).
		Build()

	if err != nil {
		t.Fatalf("Failed to build DAG: %v", err)
	}

	taskTypes := make(map[string]models.TaskType)
	for _, task := range dag.Tasks {
		taskTypes[task.ID] = task.Type
	}

	if taskTypes["bash-task"] != models.TaskTypeBash {
		t.Error("Expected bash task type")
	}
	if taskTypes["http-task"] != models.TaskTypeHTTP {
		t.Error("Expected http task type")
	}
	if taskTypes["python-task"] != models.TaskTypePython {
		t.Error("Expected python task type")
	}
	if taskTypes["go-task"] != models.TaskTypeGo {
		t.Error("Expected go task type")
	}
}

func TestBuilder_MultipleDependencies(t *testing.T) {
	dag, err := NewBuilder("multi-dep").
		Task("task1", BashTask("echo 1")).
		Task("task2", BashTask("echo 2")).
		Task("task3", BashTask("echo 3").DependsOn("task1", "task2")).
		Build()

	if err != nil {
		t.Fatalf("Failed to build DAG: %v", err)
	}

	var task3 *models.Task
	for i := range dag.Tasks {
		if dag.Tasks[i].ID == "task3" {
			task3 = &dag.Tasks[i]
			break
		}
	}

	if task3 == nil {
		t.Fatal("task3 not found")
	}

	if len(task3.Dependencies) != 2 {
		t.Errorf("Expected 2 dependencies, got %d", len(task3.Dependencies))
	}
}

func TestBuilder_InvalidDAG(t *testing.T) {
	// Empty name should fail validation
	_, err := NewBuilder("").
		Task("task1", BashTask("echo hello")).
		Build()

	if err == nil {
		t.Error("Expected error for empty DAG name, got nil")
	}
}

func TestBuilder_CyclicDependency(t *testing.T) {
	// Create a cyclic dependency: task1 -> task2 -> task1
	_, err := NewBuilder("cyclic").
		Task("task1", BashTask("echo 1").DependsOn("task2")).
		Task("task2", BashTask("echo 2").DependsOn("task1")).
		Build()

	if err == nil {
		t.Error("Expected error for cyclic dependency, got nil")
	}
}

func TestBuilder_NonExistentDependency(t *testing.T) {
	_, err := NewBuilder("invalid-dep").
		Task("task1", BashTask("echo 1").DependsOn("nonexistent")).
		Build()

	if err == nil {
		t.Error("Expected error for non-existent dependency, got nil")
	}
}

func TestBuilder_MustBuild(t *testing.T) {
	// Valid DAG should not panic
	dag := NewBuilder("valid").
		Task("task1", BashTask("echo hello")).
		MustBuild()

	if dag == nil {
		t.Error("Expected DAG to be built")
	}
}

func TestBuilder_MustBuild_Panic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected MustBuild to panic on invalid DAG")
		}
	}()

	// Invalid DAG should panic
	NewBuilder("").
		Task("task1", BashTask("echo hello")).
		MustBuild()
}

func TestBuilder_Paused(t *testing.T) {
	dag, err := NewBuilder("paused-dag").
		Paused(true).
		Task("task1", BashTask("echo hello")).
		Build()

	if err != nil {
		t.Fatalf("Failed to build DAG: %v", err)
	}

	if !dag.IsPaused {
		t.Error("Expected DAG to be paused")
	}
}

func TestTaskBuilder_DefaultName(t *testing.T) {
	dag, err := NewBuilder("test").
		Task("my-task", BashTask("echo hello")).
		Build()

	if err != nil {
		t.Fatalf("Failed to build DAG: %v", err)
	}

	// If no name is set, ID should be used as name
	if dag.Tasks[0].Name != "my-task" {
		t.Errorf("Expected task name 'my-task', got '%s'", dag.Tasks[0].Name)
	}
}

func TestTaskBuilder_CustomName(t *testing.T) {
	dag, err := NewBuilder("test").
		Task("task-id", BashTask("echo hello").Name("Custom Name")).
		Build()

	if err != nil {
		t.Fatalf("Failed to build DAG: %v", err)
	}

	if dag.Tasks[0].Name != "Custom Name" {
		t.Errorf("Expected task name 'Custom Name', got '%s'", dag.Tasks[0].Name)
	}
}

func TestBuilder_OrphanedTask(t *testing.T) {
	// DAG with multiple tasks where one is orphaned
	_, err := NewBuilder("orphaned").
		Task("task1", BashTask("echo 1")).
		Task("task2", BashTask("echo 2")).
		Task("task3", BashTask("echo 3").DependsOn("task1")).
		Build()

	if err == nil {
		t.Error("Expected error for orphaned task, got nil")
	}
}

func TestBuilder_ChainedDependencies(t *testing.T) {
	dag, err := NewBuilder("chained").
		Task("start", BashTask("echo start")).
		Task("middle1", BashTask("echo middle1").DependsOn("start")).
		Task("middle2", BashTask("echo middle2").DependsOn("middle1")).
		Task("end", BashTask("echo end").DependsOn("middle2")).
		Build()

	if err != nil {
		t.Fatalf("Failed to build DAG: %v", err)
	}

	// Verify the chain
	graph := NewGraph(dag)
	upstream, _ := graph.GetUpstreamTasks("end")

	if len(upstream) != 3 {
		t.Errorf("Expected 3 upstream tasks for 'end', got %d", len(upstream))
	}
}
