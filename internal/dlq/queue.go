package dlq

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"time"

	"github.com/therealutkarshpriyadarshi/dag/pkg/models"
)

var (
	// ErrNotFound is returned when a DLQ entry is not found
	ErrNotFound = errors.New("dlq entry not found")

	// ErrAlreadyExists is returned when trying to add a duplicate entry
	ErrAlreadyExists = errors.New("dlq entry already exists")
)

// Entry represents a failed task in the dead letter queue
type Entry struct {
	ID              string                 `json:"id"`
	TaskInstanceID  string                 `json:"task_instance_id"`
	TaskID          string                 `json:"task_id"`
	DAGRunID        string                 `json:"dag_run_id"`
	DAGID           string                 `json:"dag_id"`
	FailureReason   string                 `json:"failure_reason"`
	FailureTime     time.Time              `json:"failure_time"`
	Attempts        int                    `json:"attempts"`
	LastAttemptTime time.Time              `json:"last_attempt_time"`
	ErrorMessage    string                 `json:"error_message"`
	Metadata        map[string]interface{} `json:"metadata"`
	Replayed        bool                   `json:"replayed"`
	ReplayedAt      *time.Time             `json:"replayed_at,omitempty"`
}

// Queue represents a dead letter queue for failed tasks
type Queue interface {
	// Add adds an entry to the DLQ
	Add(ctx context.Context, entry *Entry) error

	// Get retrieves an entry by ID
	Get(ctx context.Context, id string) (*Entry, error)

	// List lists all entries with optional filters
	List(ctx context.Context, filters *Filters) ([]*Entry, error)

	// Replay marks an entry as replayed and returns the task to be retried
	Replay(ctx context.Context, id string) error

	// Delete removes an entry from the DLQ
	Delete(ctx context.Context, id string) error

	// Purge removes all entries from the DLQ
	Purge(ctx context.Context) error

	// Count returns the number of entries in the DLQ
	Count(ctx context.Context) (int, error)
}

// Filters holds filtering options for listing DLQ entries
type Filters struct {
	DAGID        string
	TaskID       string
	Replayed     *bool
	After        *time.Time
	Before       *time.Time
	Limit        int
	Offset       int
}

// MemoryQueue is an in-memory implementation of the DLQ (for testing/development)
type MemoryQueue struct {
	mu      sync.RWMutex
	entries map[string]*Entry
}

// NewMemoryQueue creates a new in-memory DLQ
func NewMemoryQueue() *MemoryQueue {
	return &MemoryQueue{
		entries: make(map[string]*Entry),
	}
}

// Add adds an entry to the DLQ
func (q *MemoryQueue) Add(ctx context.Context, entry *Entry) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	if _, exists := q.entries[entry.ID]; exists {
		return ErrAlreadyExists
	}

	q.entries[entry.ID] = entry
	return nil
}

// Get retrieves an entry by ID
func (q *MemoryQueue) Get(ctx context.Context, id string) (*Entry, error) {
	q.mu.RLock()
	defer q.mu.RUnlock()

	entry, exists := q.entries[id]
	if !exists {
		return nil, ErrNotFound
	}

	return entry, nil
}

// List lists all entries with optional filters
func (q *MemoryQueue) List(ctx context.Context, filters *Filters) ([]*Entry, error) {
	q.mu.RLock()
	defer q.mu.RUnlock()

	var result []*Entry
	for _, entry := range q.entries {
		if filters != nil {
			if filters.DAGID != "" && entry.DAGID != filters.DAGID {
				continue
			}
			if filters.TaskID != "" && entry.TaskID != filters.TaskID {
				continue
			}
			if filters.Replayed != nil && entry.Replayed != *filters.Replayed {
				continue
			}
			if filters.After != nil && entry.FailureTime.Before(*filters.After) {
				continue
			}
			if filters.Before != nil && entry.FailureTime.After(*filters.Before) {
				continue
			}
		}

		result = append(result, entry)
	}

	// Apply offset and limit
	if filters != nil {
		if filters.Offset > 0 && filters.Offset < len(result) {
			result = result[filters.Offset:]
		} else if filters.Offset >= len(result) {
			result = []*Entry{}
		}

		if filters.Limit > 0 && filters.Limit < len(result) {
			result = result[:filters.Limit]
		}
	}

	return result, nil
}

// Replay marks an entry as replayed
func (q *MemoryQueue) Replay(ctx context.Context, id string) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	entry, exists := q.entries[id]
	if !exists {
		return ErrNotFound
	}

	now := time.Now()
	entry.Replayed = true
	entry.ReplayedAt = &now

	return nil
}

// Delete removes an entry from the DLQ
func (q *MemoryQueue) Delete(ctx context.Context, id string) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	if _, exists := q.entries[id]; !exists {
		return ErrNotFound
	}

	delete(q.entries, id)
	return nil
}

// Purge removes all entries from the DLQ
func (q *MemoryQueue) Purge(ctx context.Context) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	q.entries = make(map[string]*Entry)
	return nil
}

// Count returns the number of entries in the DLQ
func (q *MemoryQueue) Count(ctx context.Context) (int, error) {
	q.mu.RLock()
	defer q.mu.RUnlock()

	return len(q.entries), nil
}

// Manager manages the dead letter queue
type Manager struct {
	queue            Queue
	onEntryAdded     func(*Entry)
	onThresholdReached func(count int)
	threshold        int
}

// NewManager creates a new DLQ manager
func NewManager(queue Queue, threshold int) *Manager {
	return &Manager{
		queue:     queue,
		threshold: threshold,
	}
}

// AddFailedTask adds a failed task to the DLQ
func (m *Manager) AddFailedTask(ctx context.Context, taskInstance *models.TaskInstance, dag *models.DAG, err error) error {
	errorMessage := ""
	if err != nil {
		errorMessage = err.Error()
	}

	entry := &Entry{
		ID:              taskInstance.ID,
		TaskInstanceID:  taskInstance.ID,
		TaskID:          taskInstance.TaskID,
		DAGRunID:        taskInstance.DAGRunID,
		DAGID:           dag.ID,
		FailureReason:   "max_retries_exceeded",
		FailureTime:     time.Now(),
		Attempts:        taskInstance.TryNumber,
		LastAttemptTime: time.Now(),
		ErrorMessage:    errorMessage,
		Metadata:        make(map[string]interface{}),
		Replayed:        false,
	}

	if err := m.queue.Add(ctx, entry); err != nil {
		return err
	}

	// Call callback if configured
	if m.onEntryAdded != nil {
		m.onEntryAdded(entry)
	}

	// Check threshold
	if m.threshold > 0 {
		count, err := m.queue.Count(ctx)
		if err == nil && count >= m.threshold {
			if m.onThresholdReached != nil {
				m.onThresholdReached(count)
			}
		}
	}

	return nil
}

// OnEntryAdded sets a callback for when an entry is added
func (m *Manager) OnEntryAdded(callback func(*Entry)) {
	m.onEntryAdded = callback
}

// OnThresholdReached sets a callback for when the threshold is reached
func (m *Manager) OnThresholdReached(callback func(count int)) {
	m.onThresholdReached = callback
}

// GetQueue returns the underlying queue
func (m *Manager) GetQueue() Queue {
	return m.queue
}

// ToJSON converts an entry to JSON
func (e *Entry) ToJSON() (string, error) {
	data, err := json.Marshal(e)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// FromJSON creates an entry from JSON
func FromJSON(data string) (*Entry, error) {
	var entry Entry
	err := json.Unmarshal([]byte(data), &entry)
	if err != nil {
		return nil, err
	}
	return &entry, nil
}
