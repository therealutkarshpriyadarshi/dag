package state

import (
	"testing"

	"github.com/therealutkarshpriyadarshi/dag/pkg/models"
)

func TestStateMachine_CanTransition(t *testing.T) {
	sm := NewStateMachine()

	tests := []struct {
		name     string
		from     models.State
		to       models.State
		expected bool
	}{
		// Valid transitions from Queued
		{"Queued to Running", models.StateQueued, models.StateRunning, true},
		{"Queued to Skipped", models.StateQueued, models.StateSkipped, true},
		{"Queued to Failed", models.StateQueued, models.StateFailed, true},

		// Valid transitions from Running
		{"Running to Success", models.StateRunning, models.StateSuccess, true},
		{"Running to Failed", models.StateRunning, models.StateFailed, true},
		{"Running to Retrying", models.StateRunning, models.StateRetrying, true},
		{"Running to UpstreamFailed", models.StateRunning, models.StateUpstreamFailed, true},

		// Valid transitions from Retrying
		{"Retrying to Running", models.StateRetrying, models.StateRunning, true},
		{"Retrying to Failed", models.StateRetrying, models.StateFailed, true},
		{"Retrying to Success", models.StateRetrying, models.StateSuccess, true},

		// Valid transitions from Failed
		{"Failed to Retrying", models.StateFailed, models.StateRetrying, true},
		{"Failed to Running", models.StateFailed, models.StateRunning, true},

		// Valid transitions from UpstreamFailed
		{"UpstreamFailed to Queued", models.StateUpstreamFailed, models.StateQueued, true},

		// Idempotent transitions (same state)
		{"Queued to Queued", models.StateQueued, models.StateQueued, true},
		{"Running to Running", models.StateRunning, models.StateRunning, true},

		// Invalid transitions
		{"Success to Running", models.StateSuccess, models.StateRunning, false},
		{"Success to Failed", models.StateSuccess, models.StateFailed, false},
		{"Skipped to Running", models.StateSkipped, models.StateRunning, false},
		{"Queued to Success", models.StateQueued, models.StateSuccess, false},
		{"Running to Queued", models.StateRunning, models.StateQueued, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sm.CanTransition(tt.from, tt.to)
			if result != tt.expected {
				t.Errorf("CanTransition(%s, %s) = %v, want %v", tt.from, tt.to, result, tt.expected)
			}
		})
	}
}

func TestStateMachine_ValidateTransition(t *testing.T) {
	sm := NewStateMachine()

	tests := []struct {
		name      string
		from      models.State
		to        models.State
		wantError bool
	}{
		{"Valid: Queued to Running", models.StateQueued, models.StateRunning, false},
		{"Valid: Running to Success", models.StateRunning, models.StateSuccess, false},
		{"Invalid: Success to Running", models.StateSuccess, models.StateRunning, true},
		{"Invalid: Queued to Success", models.StateQueued, models.StateSuccess, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := sm.ValidateTransition(tt.from, tt.to)
			if (err != nil) != tt.wantError {
				t.Errorf("ValidateTransition(%s, %s) error = %v, wantError %v", tt.from, tt.to, err, tt.wantError)
			}
			if err != nil && !tt.wantError {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestStateMachine_GetNextStates(t *testing.T) {
	sm := NewStateMachine()

	tests := []struct {
		name     string
		current  models.State
		expected int // number of valid next states
	}{
		{"Queued has 3 next states", models.StateQueued, 3},
		{"Running has 4 next states", models.StateRunning, 4},
		{"Retrying has 3 next states", models.StateRetrying, 3},
		{"Failed has 2 next states", models.StateFailed, 2},
		{"Success has 0 next states", models.StateSuccess, 0},
		{"Skipped has 0 next states", models.StateSkipped, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			states := sm.GetNextStates(tt.current)
			if len(states) != tt.expected {
				t.Errorf("GetNextStates(%s) returned %d states, want %d", tt.current, len(states), tt.expected)
			}
		})
	}
}

func TestStateMachine_IsTerminalState(t *testing.T) {
	sm := NewStateMachine()

	tests := []struct {
		name     string
		state    models.State
		expected bool
	}{
		{"Success is terminal", models.StateSuccess, true},
		{"Failed is terminal", models.StateFailed, true},
		{"Skipped is terminal", models.StateSkipped, true},
		{"Queued is not terminal", models.StateQueued, false},
		{"Running is not terminal", models.StateRunning, false},
		{"Retrying is not terminal", models.StateRetrying, false},
		{"UpstreamFailed is not terminal", models.StateUpstreamFailed, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sm.IsTerminalState(tt.state)
			if result != tt.expected {
				t.Errorf("IsTerminalState(%s) = %v, want %v", tt.state, result, tt.expected)
			}
		})
	}
}

func TestManager_Transition(t *testing.T) {
	// Mock publisher for testing
	var publishedEvents []TransitionEvent
	mockPublisher := &mockPublisher{
		events: &publishedEvents,
	}

	manager := NewManager(mockPublisher)

	tests := []struct {
		name      string
		entityType string
		entityID  string
		from      models.State
		to        models.State
		metadata  map[string]interface{}
		wantError bool
	}{
		{
			name:       "Valid transition publishes event",
			entityType: "dag_run",
			entityID:   "123",
			from:       models.StateQueued,
			to:         models.StateRunning,
			metadata:   map[string]interface{}{"worker": "worker-1"},
			wantError:  false,
		},
		{
			name:       "Invalid transition returns error",
			entityType: "task_instance",
			entityID:   "456",
			from:       models.StateSuccess,
			to:         models.StateRunning,
			metadata:   nil,
			wantError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			publishedEvents = []TransitionEvent{} // Reset

			err := manager.Transition(tt.entityType, tt.entityID, tt.from, tt.to, tt.metadata)
			if (err != nil) != tt.wantError {
				t.Errorf("Transition() error = %v, wantError %v", err, tt.wantError)
			}

			if !tt.wantError {
				// Check that event was published
				if len(publishedEvents) != 1 {
					t.Errorf("Expected 1 event to be published, got %d", len(publishedEvents))
				} else {
					event := publishedEvents[0]
					if event.EntityType != tt.entityType {
						t.Errorf("Event EntityType = %s, want %s", event.EntityType, tt.entityType)
					}
					if event.EntityID != tt.entityID {
						t.Errorf("Event EntityID = %s, want %s", event.EntityID, tt.entityID)
					}
					if event.OldState != tt.from {
						t.Errorf("Event OldState = %s, want %s", event.OldState, tt.from)
					}
					if event.NewState != tt.to {
						t.Errorf("Event NewState = %s, want %s", event.NewState, tt.to)
					}
				}
			}
		})
	}
}

func TestNoOpPublisher(t *testing.T) {
	publisher := &NoOpPublisher{}
	event := TransitionEvent{
		EntityType: "test",
		EntityID:   "123",
		OldState:   models.StateQueued,
		NewState:   models.StateRunning,
	}

	err := publisher.Publish(event)
	if err != nil {
		t.Errorf("NoOpPublisher.Publish() should never return error, got %v", err)
	}
}

// Mock publisher for testing
type mockPublisher struct {
	events *[]TransitionEvent
}

func (m *mockPublisher) Publish(event TransitionEvent) error {
	*m.events = append(*m.events, event)
	return nil
}
