package dlq

import (
	"context"
	"testing"
	"time"

	"github.com/therealutkarshpriyadarshi/dag/pkg/models"
)

func TestMemoryQueue_AddAndGet(t *testing.T) {
	q := NewMemoryQueue()
	ctx := context.Background()

	entry := &Entry{
		ID:             "entry1",
		TaskInstanceID: "ti1",
		TaskID:         "task1",
		DAGRunID:       "dr1",
		DAGID:          "dag1",
		FailureReason:  "timeout",
		FailureTime:    time.Now(),
		Attempts:       3,
		ErrorMessage:   "task timed out",
		Replayed:       false,
	}

	// Add entry
	err := q.Add(ctx, entry)
	if err != nil {
		t.Fatalf("Failed to add entry: %v", err)
	}

	// Get entry
	retrieved, err := q.Get(ctx, "entry1")
	if err != nil {
		t.Fatalf("Failed to get entry: %v", err)
	}

	if retrieved.ID != entry.ID {
		t.Errorf("Expected ID %s, got %s", entry.ID, retrieved.ID)
	}
}

func TestMemoryQueue_AddDuplicate(t *testing.T) {
	q := NewMemoryQueue()
	ctx := context.Background()

	entry := &Entry{
		ID:             "entry1",
		TaskInstanceID: "ti1",
		TaskID:         "task1",
		DAGRunID:       "dr1",
		DAGID:          "dag1",
		FailureReason:  "timeout",
		FailureTime:    time.Now(),
	}

	// Add entry
	err := q.Add(ctx, entry)
	if err != nil {
		t.Fatalf("Failed to add entry: %v", err)
	}

	// Try to add duplicate
	err = q.Add(ctx, entry)
	if err != ErrAlreadyExists {
		t.Errorf("Expected ErrAlreadyExists, got %v", err)
	}
}

func TestMemoryQueue_GetNotFound(t *testing.T) {
	q := NewMemoryQueue()
	ctx := context.Background()

	_, err := q.Get(ctx, "nonexistent")
	if err != ErrNotFound {
		t.Errorf("Expected ErrNotFound, got %v", err)
	}
}

func TestMemoryQueue_List(t *testing.T) {
	q := NewMemoryQueue()
	ctx := context.Background()

	// Add multiple entries
	entries := []*Entry{
		{ID: "entry1", DAGID: "dag1", TaskID: "task1", FailureTime: time.Now()},
		{ID: "entry2", DAGID: "dag1", TaskID: "task2", FailureTime: time.Now()},
		{ID: "entry3", DAGID: "dag2", TaskID: "task1", FailureTime: time.Now()},
	}

	for _, entry := range entries {
		q.Add(ctx, entry)
	}

	// List all
	all, err := q.List(ctx, nil)
	if err != nil {
		t.Fatalf("Failed to list entries: %v", err)
	}

	if len(all) != 3 {
		t.Errorf("Expected 3 entries, got %d", len(all))
	}
}

func TestMemoryQueue_ListWithFilters(t *testing.T) {
	q := NewMemoryQueue()
	ctx := context.Background()

	// Add multiple entries
	entries := []*Entry{
		{ID: "entry1", DAGID: "dag1", TaskID: "task1", FailureTime: time.Now(), Replayed: false},
		{ID: "entry2", DAGID: "dag1", TaskID: "task2", FailureTime: time.Now(), Replayed: false},
		{ID: "entry3", DAGID: "dag2", TaskID: "task1", FailureTime: time.Now(), Replayed: true},
	}

	for _, entry := range entries {
		q.Add(ctx, entry)
	}

	// Filter by DAGID
	filtered, err := q.List(ctx, &Filters{DAGID: "dag1"})
	if err != nil {
		t.Fatalf("Failed to list entries: %v", err)
	}

	if len(filtered) != 2 {
		t.Errorf("Expected 2 entries for dag1, got %d", len(filtered))
	}

	// Filter by TaskID
	filtered, err = q.List(ctx, &Filters{TaskID: "task1"})
	if err != nil {
		t.Fatalf("Failed to list entries: %v", err)
	}

	if len(filtered) != 2 {
		t.Errorf("Expected 2 entries for task1, got %d", len(filtered))
	}

	// Filter by Replayed
	replayed := false
	filtered, err = q.List(ctx, &Filters{Replayed: &replayed})
	if err != nil {
		t.Fatalf("Failed to list entries: %v", err)
	}

	if len(filtered) != 2 {
		t.Errorf("Expected 2 non-replayed entries, got %d", len(filtered))
	}
}

func TestMemoryQueue_ListWithPagination(t *testing.T) {
	q := NewMemoryQueue()
	ctx := context.Background()

	// Add multiple entries
	for i := 0; i < 10; i++ {
		entry := &Entry{
			ID:          string(rune('a' + i)),
			DAGID:       "dag1",
			TaskID:      "task1",
			FailureTime: time.Now(),
		}
		q.Add(ctx, entry)
	}

	// Test limit
	limited, err := q.List(ctx, &Filters{Limit: 5})
	if err != nil {
		t.Fatalf("Failed to list entries: %v", err)
	}

	if len(limited) != 5 {
		t.Errorf("Expected 5 entries with limit, got %d", len(limited))
	}

	// Test offset
	offset, err := q.List(ctx, &Filters{Offset: 5})
	if err != nil {
		t.Fatalf("Failed to list entries: %v", err)
	}

	if len(offset) != 5 {
		t.Errorf("Expected 5 entries with offset, got %d", len(offset))
	}

	// Test limit and offset
	page, err := q.List(ctx, &Filters{Offset: 5, Limit: 3})
	if err != nil {
		t.Fatalf("Failed to list entries: %v", err)
	}

	if len(page) != 3 {
		t.Errorf("Expected 3 entries with offset and limit, got %d", len(page))
	}
}

func TestMemoryQueue_Replay(t *testing.T) {
	q := NewMemoryQueue()
	ctx := context.Background()

	entry := &Entry{
		ID:             "entry1",
		TaskInstanceID: "ti1",
		TaskID:         "task1",
		DAGRunID:       "dr1",
		DAGID:          "dag1",
		FailureReason:  "timeout",
		FailureTime:    time.Now(),
		Replayed:       false,
	}

	q.Add(ctx, entry)

	// Replay entry
	err := q.Replay(ctx, "entry1")
	if err != nil {
		t.Fatalf("Failed to replay entry: %v", err)
	}

	// Verify replayed status
	retrieved, _ := q.Get(ctx, "entry1")
	if !retrieved.Replayed {
		t.Error("Entry should be marked as replayed")
	}

	if retrieved.ReplayedAt == nil {
		t.Error("ReplayedAt should be set")
	}
}

func TestMemoryQueue_Delete(t *testing.T) {
	q := NewMemoryQueue()
	ctx := context.Background()

	entry := &Entry{
		ID:             "entry1",
		TaskInstanceID: "ti1",
		TaskID:         "task1",
		DAGRunID:       "dr1",
		DAGID:          "dag1",
		FailureReason:  "timeout",
		FailureTime:    time.Now(),
	}

	q.Add(ctx, entry)

	// Delete entry
	err := q.Delete(ctx, "entry1")
	if err != nil {
		t.Fatalf("Failed to delete entry: %v", err)
	}

	// Verify deletion
	_, err = q.Get(ctx, "entry1")
	if err != ErrNotFound {
		t.Errorf("Expected ErrNotFound after deletion, got %v", err)
	}
}

func TestMemoryQueue_Purge(t *testing.T) {
	q := NewMemoryQueue()
	ctx := context.Background()

	// Add multiple entries
	for i := 0; i < 5; i++ {
		entry := &Entry{
			ID:          string(rune('a' + i)),
			DAGID:       "dag1",
			TaskID:      "task1",
			FailureTime: time.Now(),
		}
		q.Add(ctx, entry)
	}

	// Purge all entries
	err := q.Purge(ctx)
	if err != nil {
		t.Fatalf("Failed to purge entries: %v", err)
	}

	// Verify purge
	count, _ := q.Count(ctx)
	if count != 0 {
		t.Errorf("Expected 0 entries after purge, got %d", count)
	}
}

func TestMemoryQueue_Count(t *testing.T) {
	q := NewMemoryQueue()
	ctx := context.Background()

	// Add entries
	for i := 0; i < 5; i++ {
		entry := &Entry{
			ID:          string(rune('a' + i)),
			DAGID:       "dag1",
			TaskID:      "task1",
			FailureTime: time.Now(),
		}
		q.Add(ctx, entry)
	}

	count, err := q.Count(ctx)
	if err != nil {
		t.Fatalf("Failed to count entries: %v", err)
	}

	if count != 5 {
		t.Errorf("Expected 5 entries, got %d", count)
	}
}

func TestManager_AddFailedTask(t *testing.T) {
	q := NewMemoryQueue()
	m := NewManager(q, 10)
	ctx := context.Background()

	taskInstance := &models.TaskInstance{
		ID:        "ti1",
		TaskID:    "task1",
		DAGRunID:  "dr1",
		TryNumber: 3,
	}

	dag := &models.DAG{
		ID:   "dag1",
		Name: "Test DAG",
	}

	err := m.AddFailedTask(ctx, taskInstance, dag, nil)
	if err != nil {
		t.Fatalf("Failed to add failed task: %v", err)
	}

	// Verify entry was added
	entry, err := q.Get(ctx, "ti1")
	if err != nil {
		t.Fatalf("Failed to get entry: %v", err)
	}

	if entry.TaskID != "task1" {
		t.Errorf("Expected TaskID task1, got %s", entry.TaskID)
	}
}

func TestManager_OnEntryAdded(t *testing.T) {
	q := NewMemoryQueue()
	m := NewManager(q, 10)
	ctx := context.Background()

	callbackCalled := false
	m.OnEntryAdded(func(entry *Entry) {
		callbackCalled = true
	})

	taskInstance := &models.TaskInstance{
		ID:       "ti1",
		TaskID:   "task1",
		DAGRunID: "dr1",
	}

	dag := &models.DAG{ID: "dag1", Name: "Test DAG"}

	m.AddFailedTask(ctx, taskInstance, dag, nil)

	if !callbackCalled {
		t.Error("OnEntryAdded callback was not called")
	}
}

func TestManager_OnThresholdReached(t *testing.T) {
	q := NewMemoryQueue()
	m := NewManager(q, 3)
	ctx := context.Background()

	thresholdReached := false
	m.OnThresholdReached(func(count int) {
		thresholdReached = true
	})

	// Add entries up to threshold
	for i := 0; i < 3; i++ {
		taskInstance := &models.TaskInstance{
			ID:       string(rune('a' + i)),
			TaskID:   "task1",
			DAGRunID: "dr1",
		}
		dag := &models.DAG{ID: "dag1", Name: "Test DAG"}
		m.AddFailedTask(ctx, taskInstance, dag, nil)
	}

	if !thresholdReached {
		t.Error("OnThresholdReached callback was not called")
	}
}

func TestEntry_ToJSON(t *testing.T) {
	entry := &Entry{
		ID:             "entry1",
		TaskInstanceID: "ti1",
		TaskID:         "task1",
		DAGRunID:       "dr1",
		DAGID:          "dag1",
		FailureReason:  "timeout",
		FailureTime:    time.Now(),
		Attempts:       3,
		ErrorMessage:   "task timed out",
		Metadata:       map[string]interface{}{"key": "value"},
		Replayed:       false,
	}

	jsonStr, err := entry.ToJSON()
	if err != nil {
		t.Fatalf("Failed to convert to JSON: %v", err)
	}

	if jsonStr == "" {
		t.Error("JSON string should not be empty")
	}
}

func TestFromJSON(t *testing.T) {
	jsonStr := `{
		"id": "entry1",
		"task_instance_id": "ti1",
		"task_id": "task1",
		"dag_run_id": "dr1",
		"dag_id": "dag1",
		"failure_reason": "timeout",
		"failure_time": "2024-01-01T00:00:00Z",
		"attempts": 3,
		"last_attempt_time": "2024-01-01T00:00:00Z",
		"error_message": "task timed out",
		"metadata": {},
		"replayed": false
	}`

	entry, err := FromJSON(jsonStr)
	if err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	if entry.ID != "entry1" {
		t.Errorf("Expected ID entry1, got %s", entry.ID)
	}

	if entry.TaskID != "task1" {
		t.Errorf("Expected TaskID task1, got %s", entry.TaskID)
	}
}
