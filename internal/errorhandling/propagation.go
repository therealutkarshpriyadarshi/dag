package errorhandling

import (
	"context"
	"fmt"

	"github.com/therealutkarshpriyadarshi/dag/pkg/models"
)

// ErrorCallback is a function that is called when a task fails
type ErrorCallback func(ctx context.Context, task *models.Task, taskInstance *models.TaskInstance, err error) error

// PropagationPolicy defines how errors should propagate through the DAG
type PropagationPolicy string

const (
	// PropagationPolicyFail stops the entire DAG on any task failure
	PropagationPolicyFail PropagationPolicy = "fail"

	// PropagationPolicySkipDownstream marks downstream tasks as upstream_failed
	PropagationPolicySkipDownstream PropagationPolicy = "skip_downstream"

	// PropagationPolicyAllowPartial allows other branches to continue executing
	PropagationPolicyAllowPartial PropagationPolicy = "allow_partial"
)

// PropagationConfig holds configuration for error propagation
type PropagationConfig struct {
	// Policy defines how errors should propagate
	Policy PropagationPolicy

	// OnTaskFailure is called when a task fails
	OnTaskFailure ErrorCallback

	// OnDAGFailure is called when the entire DAG fails
	OnDAGFailure func(ctx context.Context, dagRun *models.DAGRun, err error) error

	// AllowPartialSuccess allows DAG to succeed even if some tasks fail
	AllowPartialSuccess bool

	// CriticalTasks lists tasks that must succeed for DAG to succeed
	CriticalTasks []string
}

// DefaultPropagationConfig returns a config with sensible defaults
func DefaultPropagationConfig() *PropagationConfig {
	return &PropagationConfig{
		Policy:              PropagationPolicySkipDownstream,
		AllowPartialSuccess: false,
		CriticalTasks:       []string{},
	}
}

// PropagationHandler handles error propagation logic
type PropagationHandler struct {
	config *PropagationConfig
}

// NewPropagationHandler creates a new error propagation handler
func NewPropagationHandler(config *PropagationConfig) *PropagationHandler {
	if config == nil {
		config = DefaultPropagationConfig()
	}
	return &PropagationHandler{
		config: config,
	}
}

// HandleTaskFailure handles a task failure and determines propagation
func (h *PropagationHandler) HandleTaskFailure(
	ctx context.Context,
	task *models.Task,
	taskInstance *models.TaskInstance,
	dag *models.DAG,
	err error,
) error {
	// Call task failure callback if configured
	if h.config.OnTaskFailure != nil {
		if cbErr := h.config.OnTaskFailure(ctx, task, taskInstance, err); cbErr != nil {
			return fmt.Errorf("task failure callback error: %w", cbErr)
		}
	}

	// Check if this is a critical task
	isCritical := h.isTaskCritical(task.ID)

	// Apply propagation policy
	switch h.config.Policy {
	case PropagationPolicyFail:
		// Always fail the entire DAG
		return fmt.Errorf("task %s failed, stopping DAG execution: %w", task.ID, err)

	case PropagationPolicySkipDownstream:
		// Mark downstream tasks as upstream_failed
		// This is handled by the executor, just return the error
		if isCritical {
			return fmt.Errorf("critical task %s failed, stopping DAG execution: %w", task.ID, err)
		}
		return err

	case PropagationPolicyAllowPartial:
		// Allow other branches to continue
		if isCritical {
			return fmt.Errorf("critical task %s failed, stopping DAG execution: %w", task.ID, err)
		}
		// Don't propagate error, log it instead
		return nil

	default:
		return fmt.Errorf("unknown propagation policy: %s", h.config.Policy)
	}
}

// HandleDAGFailure handles a complete DAG failure
func (h *PropagationHandler) HandleDAGFailure(
	ctx context.Context,
	dagRun *models.DAGRun,
	err error,
) error {
	if h.config.OnDAGFailure != nil {
		return h.config.OnDAGFailure(ctx, dagRun, err)
	}
	return err
}

// ShouldMarkDownstreamFailed determines if downstream tasks should be marked as failed
func (h *PropagationHandler) ShouldMarkDownstreamFailed(task *models.Task) bool {
	return h.config.Policy == PropagationPolicySkipDownstream || h.config.Policy == PropagationPolicyFail
}

// isTaskCritical checks if a task is marked as critical
func (h *PropagationHandler) isTaskCritical(taskID string) bool {
	for _, criticalTask := range h.config.CriticalTasks {
		if criticalTask == taskID {
			return true
		}
	}
	return false
}

// CanDAGSucceed determines if a DAG can succeed with the current task states
func (h *PropagationHandler) CanDAGSucceed(taskInstances []*models.TaskInstance) bool {
	if !h.config.AllowPartialSuccess {
		// All tasks must succeed
		for _, ti := range taskInstances {
			if ti.State == models.StateFailed {
				return false
			}
		}
		return true
	}

	// Check if all critical tasks succeeded
	for _, ti := range taskInstances {
		if h.isTaskCritical(ti.TaskID) && ti.State != models.StateSuccess {
			return false
		}
	}

	return true
}

// ErrorClassifier classifies errors for retry decisions
type ErrorClassifier struct {
	retryableErrors map[string]bool
}

// NewErrorClassifier creates a new error classifier
func NewErrorClassifier() *ErrorClassifier {
	return &ErrorClassifier{
		retryableErrors: map[string]bool{
			"timeout":              true,
			"connection_refused":   true,
			"temporary":            true,
			"rate_limit":           true,
			"service_unavailable":  true,
			"network":              true,
		},
	}
}

// IsRetryable determines if an error is retryable
func (ec *ErrorClassifier) IsRetryable(errorCode string) bool {
	return ec.retryableErrors[errorCode]
}

// AddRetryableError adds an error code to the retryable list
func (ec *ErrorClassifier) AddRetryableError(errorCode string) {
	ec.retryableErrors[errorCode] = true
}

// RemoveRetryableError removes an error code from the retryable list
func (ec *ErrorClassifier) RemoveRetryableError(errorCode string) {
	delete(ec.retryableErrors, errorCode)
}
