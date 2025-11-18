// +build integration

package storage

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/therealutkarshpriyadarshi/dag/pkg/models"
)

func TestDAGRepository_Integration(t *testing.T) {
	db, cleanup := SetupTestDB(t)
	defer cleanup()

	dagRepo, _, _, _ := CreateTestRepositories(db.DB)
	ctx := context.Background()

	t.Run("Create and Get DAG", func(t *testing.T) {
		dag := &models.DAG{
			Name:        "test-dag-" + uuid.New().String(),
			Description: "Test DAG",
			Schedule:    "0 0 * * *",
			Tags:        []string{"test", "integration"},
			StartDate:   time.Now().UTC(),
			IsPaused:    false,
		}

		// Create DAG
		err := dagRepo.Create(ctx, dag)
		if err != nil {
			t.Fatalf("Failed to create DAG: %v", err)
		}

		if dag.ID == "" {
			t.Error("DAG ID should be set after creation")
		}

		// Get DAG by ID
		retrieved, err := dagRepo.Get(ctx, dag.ID)
		if err != nil {
			t.Fatalf("Failed to get DAG: %v", err)
		}

		if retrieved.Name != dag.Name {
			t.Errorf("Retrieved DAG name = %s, want %s", retrieved.Name, dag.Name)
		}
		if retrieved.Description != dag.Description {
			t.Errorf("Retrieved DAG description = %s, want %s", retrieved.Description, dag.Description)
		}

		// Get DAG by name
		byName, err := dagRepo.GetByName(ctx, dag.Name)
		if err != nil {
			t.Fatalf("Failed to get DAG by name: %v", err)
		}

		if byName.ID != dag.ID {
			t.Errorf("Retrieved DAG ID = %s, want %s", byName.ID, dag.ID)
		}
	})

	t.Run("List DAGs with filters", func(t *testing.T) {
		// Create test DAGs
		pausedDAG := &models.DAG{
			Name:      "paused-dag-" + uuid.New().String(),
			Schedule:  "0 0 * * *",
			Tags:      []string{"paused"},
			StartDate: time.Now().UTC(),
			IsPaused:  true,
		}
		err := dagRepo.Create(ctx, pausedDAG)
		if err != nil {
			t.Fatalf("Failed to create paused DAG: %v", err)
		}

		activeDAG := &models.DAG{
			Name:      "active-dag-" + uuid.New().String(),
			Schedule:  "0 0 * * *",
			Tags:      []string{"active"},
			StartDate: time.Now().UTC(),
			IsPaused:  false,
		}
		err = dagRepo.Create(ctx, activeDAG)
		if err != nil {
			t.Fatalf("Failed to create active DAG: %v", err)
		}

		// List all DAGs
		allDAGs, err := dagRepo.List(ctx, DAGFilters{Limit: 100})
		if err != nil {
			t.Fatalf("Failed to list DAGs: %v", err)
		}

		if len(allDAGs) < 2 {
			t.Errorf("Expected at least 2 DAGs, got %d", len(allDAGs))
		}

		// List only paused DAGs
		isPaused := true
		pausedDAGs, err := dagRepo.List(ctx, DAGFilters{IsPaused: &isPaused})
		if err != nil {
			t.Fatalf("Failed to list paused DAGs: %v", err)
		}

		foundPaused := false
		for _, d := range pausedDAGs {
			if !d.IsPaused {
				t.Errorf("DAG %s should be paused", d.Name)
			}
			if d.ID == pausedDAG.ID {
				foundPaused = true
			}
		}
		if !foundPaused {
			t.Error("Paused DAG not found in filtered list")
		}
	})

	t.Run("Pause and Unpause DAG", func(t *testing.T) {
		dag := &models.DAG{
			Name:      "toggle-dag-" + uuid.New().String(),
			Schedule:  "0 0 * * *",
			StartDate: time.Now().UTC(),
			IsPaused:  false,
		}

		err := dagRepo.Create(ctx, dag)
		if err != nil {
			t.Fatalf("Failed to create DAG: %v", err)
		}

		// Pause DAG
		err = dagRepo.Pause(ctx, dag.ID)
		if err != nil {
			t.Fatalf("Failed to pause DAG: %v", err)
		}

		paused, err := dagRepo.Get(ctx, dag.ID)
		if err != nil {
			t.Fatalf("Failed to get DAG: %v", err)
		}

		if !paused.IsPaused {
			t.Error("DAG should be paused")
		}

		// Unpause DAG
		err = dagRepo.Unpause(ctx, dag.ID)
		if err != nil {
			t.Fatalf("Failed to unpause DAG: %v", err)
		}

		unpaused, err := dagRepo.Get(ctx, dag.ID)
		if err != nil {
			t.Fatalf("Failed to get DAG: %v", err)
		}

		if unpaused.IsPaused {
			t.Error("DAG should not be paused")
		}
	})

	t.Run("Update DAG", func(t *testing.T) {
		dag := &models.DAG{
			Name:        "update-dag-" + uuid.New().String(),
			Description: "Original description",
			Schedule:    "0 0 * * *",
			StartDate:   time.Now().UTC(),
		}

		err := dagRepo.Create(ctx, dag)
		if err != nil {
			t.Fatalf("Failed to create DAG: %v", err)
		}

		// Update DAG
		dag.Description = "Updated description"
		err = dagRepo.Update(ctx, dag)
		if err != nil {
			t.Fatalf("Failed to update DAG: %v", err)
		}

		updated, err := dagRepo.Get(ctx, dag.ID)
		if err != nil {
			t.Fatalf("Failed to get updated DAG: %v", err)
		}

		if updated.Description != "Updated description" {
			t.Errorf("DAG description = %s, want 'Updated description'", updated.Description)
		}
	})

	t.Run("Delete DAG", func(t *testing.T) {
		dag := &models.DAG{
			Name:      "delete-dag-" + uuid.New().String(),
			Schedule:  "0 0 * * *",
			StartDate: time.Now().UTC(),
		}

		err := dagRepo.Create(ctx, dag)
		if err != nil {
			t.Fatalf("Failed to create DAG: %v", err)
		}

		// Delete DAG
		err = dagRepo.Delete(ctx, dag.ID)
		if err != nil {
			t.Fatalf("Failed to delete DAG: %v", err)
		}

		// Try to get deleted DAG
		_, err = dagRepo.Get(ctx, dag.ID)
		if err == nil {
			t.Error("Expected error when getting deleted DAG")
		}
	})
}

func TestDAGRunRepository_Integration(t *testing.T) {
	db, cleanup := SetupTestDB(t)
	defer cleanup()

	dagRepo, dagRunRepo, _, _ := CreateTestRepositories(db.DB)
	ctx := context.Background()

	// Create a test DAG first
	dag := &models.DAG{
		Name:      "test-dag-runs-" + uuid.New().String(),
		Schedule:  "0 0 * * *",
		StartDate: time.Now().UTC(),
	}
	err := dagRepo.Create(ctx, dag)
	if err != nil {
		t.Fatalf("Failed to create test DAG: %v", err)
	}

	t.Run("Create and Get DAG Run", func(t *testing.T) {
		run := &models.DAGRun{
			DAGID:         dag.ID,
			ExecutionDate: time.Now().UTC(),
			State:         models.StateQueued,
		}

		err := dagRunRepo.Create(ctx, run)
		if err != nil {
			t.Fatalf("Failed to create DAG run: %v", err)
		}

		if run.ID == "" {
			t.Error("DAG run ID should be set after creation")
		}

		retrieved, err := dagRunRepo.Get(ctx, run.ID)
		if err != nil {
			t.Fatalf("Failed to get DAG run: %v", err)
		}

		if retrieved.DAGID != dag.ID {
			t.Errorf("Retrieved DAG run DAGID = %s, want %s", retrieved.DAGID, dag.ID)
		}
		if retrieved.State != models.StateQueued {
			t.Errorf("Retrieved DAG run state = %s, want %s", retrieved.State, models.StateQueued)
		}
	})

	t.Run("Update DAG Run State", func(t *testing.T) {
		run := &models.DAGRun{
			DAGID:         dag.ID,
			ExecutionDate: time.Now().UTC(),
			State:         models.StateQueued,
		}

		err := dagRunRepo.Create(ctx, run)
		if err != nil {
			t.Fatalf("Failed to create DAG run: %v", err)
		}

		// Update state
		err = dagRunRepo.UpdateState(ctx, run.ID, models.StateQueued, models.StateRunning)
		if err != nil {
			t.Fatalf("Failed to update DAG run state: %v", err)
		}

		updated, err := dagRunRepo.Get(ctx, run.ID)
		if err != nil {
			t.Fatalf("Failed to get updated DAG run: %v", err)
		}

		if updated.State != models.StateRunning {
			t.Errorf("DAG run state = %s, want %s", updated.State, models.StateRunning)
		}

		// Try invalid state transition (should fail)
		err = dagRunRepo.UpdateState(ctx, run.ID, models.StateRunning, models.StateQueued)
		if err == nil {
			t.Error("Expected error for invalid state transition")
		}
	})

	t.Run("List DAG Runs", func(t *testing.T) {
		// Create multiple runs
		for i := 0; i < 3; i++ {
			run := &models.DAGRun{
				DAGID:         dag.ID,
				ExecutionDate: time.Now().UTC().Add(time.Duration(i) * time.Hour),
				State:         models.StateQueued,
			}
			err := dagRunRepo.Create(ctx, run)
			if err != nil {
				t.Fatalf("Failed to create DAG run: %v", err)
			}
		}

		runs, err := dagRunRepo.List(ctx, DAGRunFilters{DAGID: dag.ID, Limit: 10})
		if err != nil {
			t.Fatalf("Failed to list DAG runs: %v", err)
		}

		if len(runs) < 3 {
			t.Errorf("Expected at least 3 DAG runs, got %d", len(runs))
		}
	})

	t.Run("Get Latest Run", func(t *testing.T) {
		latest, err := dagRunRepo.GetLatestRun(ctx, dag.ID)
		if err != nil {
			t.Fatalf("Failed to get latest run: %v", err)
		}

		if latest.DAGID != dag.ID {
			t.Errorf("Latest run DAGID = %s, want %s", latest.DAGID, dag.ID)
		}
	})
}

func TestTaskInstanceRepository_Integration(t *testing.T) {
	db, cleanup := SetupTestDB(t)
	defer cleanup()

	dagRepo, dagRunRepo, taskInstanceRepo, _ := CreateTestRepositories(db.DB)
	ctx := context.Background()

	// Create test DAG and DAG run
	dag := &models.DAG{
		Name:      "test-tasks-" + uuid.New().String(),
		Schedule:  "0 0 * * *",
		StartDate: time.Now().UTC(),
	}
	err := dagRepo.Create(ctx, dag)
	if err != nil {
		t.Fatalf("Failed to create test DAG: %v", err)
	}

	dagRun := &models.DAGRun{
		DAGID:         dag.ID,
		ExecutionDate: time.Now().UTC(),
		State:         models.StateQueued,
	}
	err = dagRunRepo.Create(ctx, dagRun)
	if err != nil {
		t.Fatalf("Failed to create DAG run: %v", err)
	}

	t.Run("Create and Get Task Instance", func(t *testing.T) {
		task := &models.TaskInstance{
			TaskID:   "test-task",
			DAGRunID: dagRun.ID,
			State:    models.StateQueued,
			MaxTries: 3,
		}

		err := taskInstanceRepo.Create(ctx, task)
		if err != nil {
			t.Fatalf("Failed to create task instance: %v", err)
		}

		if task.ID == "" {
			t.Error("Task instance ID should be set after creation")
		}

		retrieved, err := taskInstanceRepo.Get(ctx, task.ID)
		if err != nil {
			t.Fatalf("Failed to get task instance: %v", err)
		}

		if retrieved.TaskID != "test-task" {
			t.Errorf("Retrieved task instance TaskID = %s, want test-task", retrieved.TaskID)
		}
	})

	t.Run("Update Task Instance State", func(t *testing.T) {
		task := &models.TaskInstance{
			TaskID:   "state-task",
			DAGRunID: dagRun.ID,
			State:    models.StateQueued,
			MaxTries: 3,
		}

		err := taskInstanceRepo.Create(ctx, task)
		if err != nil {
			t.Fatalf("Failed to create task instance: %v", err)
		}

		// Update state
		err = taskInstanceRepo.UpdateState(ctx, task.ID, models.StateQueued, models.StateRunning)
		if err != nil {
			t.Fatalf("Failed to update task instance state: %v", err)
		}

		updated, err := taskInstanceRepo.Get(ctx, task.ID)
		if err != nil {
			t.Fatalf("Failed to get updated task instance: %v", err)
		}

		if updated.State != models.StateRunning {
			t.Errorf("Task instance state = %s, want %s", updated.State, models.StateRunning)
		}
	})

	t.Run("List Task Instances by DAG Run", func(t *testing.T) {
		// Create multiple task instances
		for i := 0; i < 3; i++ {
			task := &models.TaskInstance{
				TaskID:   "task-" + string(rune('A'+i)),
				DAGRunID: dagRun.ID,
				State:    models.StateQueued,
				MaxTries: 3,
			}
			err := taskInstanceRepo.Create(ctx, task)
			if err != nil {
				t.Fatalf("Failed to create task instance: %v", err)
			}
		}

		tasks, err := taskInstanceRepo.ListByDAGRun(ctx, dagRun.ID)
		if err != nil {
			t.Fatalf("Failed to list task instances: %v", err)
		}

		if len(tasks) < 3 {
			t.Errorf("Expected at least 3 task instances, got %d", len(tasks))
		}
	})
}
