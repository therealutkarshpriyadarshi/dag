package state

import (
	"errors"
	"fmt"

	"github.com/therealutkarshpriyadarshi/dag/pkg/models"
)

var (
	// ErrInvalidTransition is returned when an invalid state transition is attempted
	ErrInvalidTransition = errors.New("invalid state transition")

	// ErrOptimisticLock is returned when optimistic locking fails
	ErrOptimisticLock = errors.New("optimistic lock failed - entity was modified")
)

// StateMachine manages state transitions for DAG runs and task instances
type StateMachine struct {
	validTransitions map[models.State][]models.State
}

// NewStateMachine creates a new state machine
func NewStateMachine() *StateMachine {
	return &StateMachine{
		validTransitions: map[models.State][]models.State{
			models.StateQueued: {
				models.StateRunning,
				models.StateSkipped,
				models.StateFailed, // Can fail during queue (e.g., invalid config)
			},
			models.StateRunning: {
				models.StateSuccess,
				models.StateFailed,
				models.StateRetrying,
				models.StateUpstreamFailed,
			},
			models.StateRetrying: {
				models.StateRunning,
				models.StateFailed,
				models.StateSuccess,
			},
			models.StateFailed: {
				models.StateRetrying, // Manual retry
				models.StateRunning,  // Manual retry
			},
			models.StateUpstreamFailed: {
				models.StateQueued, // Retry entire DAG run
			},
			// Terminal states generally don't transition
			models.StateSuccess: {},
			models.StateSkipped: {},
		},
	}
}

// CanTransition checks if a state transition is valid
func (sm *StateMachine) CanTransition(from, to models.State) bool {
	// Allow transition to same state (idempotent)
	if from == to {
		return true
	}

	validStates, exists := sm.validTransitions[from]
	if !exists {
		return false
	}

	for _, state := range validStates {
		if state == to {
			return true
		}
	}

	return false
}

// ValidateTransition validates a state transition and returns an error if invalid
func (sm *StateMachine) ValidateTransition(from, to models.State) error {
	if !sm.CanTransition(from, to) {
		return fmt.Errorf("%w: cannot transition from %s to %s", ErrInvalidTransition, from, to)
	}
	return nil
}

// GetNextStates returns all valid next states from the current state
func (sm *StateMachine) GetNextStates(current models.State) []models.State {
	states, exists := sm.validTransitions[current]
	if !exists {
		return []models.State{}
	}
	return states
}

// IsTerminalState checks if a state is terminal (no further transitions)
func (sm *StateMachine) IsTerminalState(state models.State) bool {
	return state.IsTerminal()
}

// TransitionEvent represents a state transition event
type TransitionEvent struct {
	EntityType string        // "dag_run" or "task_instance"
	EntityID   string        // UUID of the entity
	OldState   models.State
	NewState   models.State
	Metadata   map[string]interface{}
}

// EventPublisher is an interface for publishing state change events
type EventPublisher interface {
	Publish(event TransitionEvent) error
}

// NoOpPublisher is a no-op event publisher for testing
type NoOpPublisher struct{}

// Publish does nothing
func (p *NoOpPublisher) Publish(event TransitionEvent) error {
	return nil
}

// Manager handles state transitions with event publishing
type Manager struct {
	machine   *StateMachine
	publisher EventPublisher
}

// NewManager creates a new state manager
func NewManager(publisher EventPublisher) *Manager {
	if publisher == nil {
		publisher = &NoOpPublisher{}
	}
	return &Manager{
		machine:   NewStateMachine(),
		publisher: publisher,
	}
}

// Transition performs a state transition and publishes an event
func (m *Manager) Transition(entityType, entityID string, from, to models.State, metadata map[string]interface{}) error {
	// Validate transition
	if err := m.machine.ValidateTransition(from, to); err != nil {
		return err
	}

	// Publish event
	event := TransitionEvent{
		EntityType: entityType,
		EntityID:   entityID,
		OldState:   from,
		NewState:   to,
		Metadata:   metadata,
	}

	if err := m.publisher.Publish(event); err != nil {
		return fmt.Errorf("failed to publish state transition event: %w", err)
	}

	return nil
}

// CanTransition delegates to the state machine
func (m *Manager) CanTransition(from, to models.State) bool {
	return m.machine.CanTransition(from, to)
}
