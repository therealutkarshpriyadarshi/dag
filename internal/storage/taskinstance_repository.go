package storage

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/therealutkarshpriyadarshi/dag/internal/state"
	"github.com/therealutkarshpriyadarshi/dag/pkg/models"
	"gorm.io/gorm"
)

type taskInstanceRepository struct {
	db           *gorm.DB
	stateManager *state.Manager
}

// NewTaskInstanceRepository creates a new task instance repository
func NewTaskInstanceRepository(db *gorm.DB, stateManager *state.Manager) TaskInstanceRepository {
	return &taskInstanceRepository{
		db:           db,
		stateManager: stateManager,
	}
}

func (r *taskInstanceRepository) Create(ctx context.Context, instance *models.TaskInstance) error {
	model, err := FromTaskInstance(instance)
	if err != nil {
		return fmt.Errorf("failed to convert task instance to model: %w", err)
	}

	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		return fmt.Errorf("failed to create task instance: %w", err)
	}

	instance.ID = model.ID.String()

	return nil
}

func (r *taskInstanceRepository) Get(ctx context.Context, id string) (*models.TaskInstance, error) {
	instanceID, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("invalid task instance ID: %w", err)
	}

	var model TaskInstanceModel
	if err := r.db.WithContext(ctx).Where("id = ?", instanceID).First(&model).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("task instance not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get task instance: %w", err)
	}

	return model.ToTaskInstance(), nil
}

func (r *taskInstanceRepository) GetByTaskID(ctx context.Context, dagRunID, taskID string) (*models.TaskInstance, error) {
	runID, err := uuid.Parse(dagRunID)
	if err != nil {
		return nil, fmt.Errorf("invalid DAG run ID: %w", err)
	}

	var model TaskInstanceModel
	if err := r.db.WithContext(ctx).
		Where("dag_run_id = ? AND task_id = ?", runID, taskID).
		Order("try_number DESC").
		First(&model).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("task instance not found for task %s in DAG run %s", taskID, dagRunID)
		}
		return nil, fmt.Errorf("failed to get task instance: %w", err)
	}

	return model.ToTaskInstance(), nil
}

func (r *taskInstanceRepository) List(ctx context.Context, filters TaskInstanceFilters) ([]*models.TaskInstance, error) {
	query := r.db.WithContext(ctx).Model(&TaskInstanceModel{})

	if filters.DAGRunID != "" {
		runID, err := uuid.Parse(filters.DAGRunID)
		if err != nil {
			return nil, fmt.Errorf("invalid DAG run ID: %w", err)
		}
		query = query.Where("dag_run_id = ?", runID)
	}

	if filters.TaskID != "" {
		query = query.Where("task_id = ?", filters.TaskID)
	}

	if filters.State != nil {
		query = query.Where("state = ?", string(*filters.State))
	}

	query = query.Order("created_at DESC")

	if filters.Limit > 0 {
		query = query.Limit(filters.Limit)
	}

	if filters.Offset > 0 {
		query = query.Offset(filters.Offset)
	}

	var instanceModels []TaskInstanceModel
	if err := query.Find(&instanceModels).Error; err != nil {
		return nil, fmt.Errorf("failed to list task instances: %w", err)
	}

	instances := make([]*models.TaskInstance, len(instanceModels))
	for i, model := range instanceModels {
		instances[i] = model.ToTaskInstance()
	}

	return instances, nil
}

func (r *taskInstanceRepository) Update(ctx context.Context, instance *models.TaskInstance) error {
	instanceID, err := uuid.Parse(instance.ID)
	if err != nil {
		return fmt.Errorf("invalid task instance ID: %w", err)
	}

	model, err := FromTaskInstance(instance)
	if err != nil {
		return fmt.Errorf("failed to convert task instance to model: %w", err)
	}

	model.ID = instanceID

	if err := r.db.WithContext(ctx).Model(&TaskInstanceModel{}).Where("id = ?", instanceID).Updates(model).Error; err != nil {
		return fmt.Errorf("failed to update task instance: %w", err)
	}

	return nil
}

func (r *taskInstanceRepository) UpdateState(ctx context.Context, id string, oldState, newState models.State) error {
	instanceID, err := uuid.Parse(id)
	if err != nil {
		return fmt.Errorf("invalid task instance ID: %w", err)
	}

	// Validate state transition
	if err := r.stateManager.Transition("task_instance", id, oldState, newState, nil); err != nil {
		return fmt.Errorf("invalid state transition: %w", err)
	}

	// Use optimistic locking to prevent concurrent updates
	result := r.db.WithContext(ctx).
		Model(&TaskInstanceModel{}).
		Where("id = ? AND state = ?", instanceID, string(oldState)).
		Updates(map[string]interface{}{
			"state":   string(newState),
			"version": gorm.Expr("version + 1"),
		})

	if result.Error != nil {
		return fmt.Errorf("failed to update task instance state: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return state.ErrOptimisticLock
	}

	return nil
}

func (r *taskInstanceRepository) Delete(ctx context.Context, id string) error {
	instanceID, err := uuid.Parse(id)
	if err != nil {
		return fmt.Errorf("invalid task instance ID: %w", err)
	}

	if err := r.db.WithContext(ctx).Delete(&TaskInstanceModel{}, "id = ?", instanceID).Error; err != nil {
		return fmt.Errorf("failed to delete task instance: %w", err)
	}

	return nil
}

func (r *taskInstanceRepository) ListByDAGRun(ctx context.Context, dagRunID string) ([]*models.TaskInstance, error) {
	runID, err := uuid.Parse(dagRunID)
	if err != nil {
		return nil, fmt.Errorf("invalid DAG run ID: %w", err)
	}

	var instanceModels []TaskInstanceModel
	if err := r.db.WithContext(ctx).
		Where("dag_run_id = ?", runID).
		Order("created_at ASC").
		Find(&instanceModels).Error; err != nil {
		return nil, fmt.Errorf("failed to list task instances by DAG run: %w", err)
	}

	instances := make([]*models.TaskInstance, len(instanceModels))
	for i, model := range instanceModels {
		instances[i] = model.ToTaskInstance()
	}

	return instances, nil
}
