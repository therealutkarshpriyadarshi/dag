package storage

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/therealutkarshpriyadarshi/dag/internal/state"
	"github.com/therealutkarshpriyadarshi/dag/pkg/models"
	"gorm.io/gorm"
)

type dagRunRepository struct {
	db            *gorm.DB
	stateManager  *state.Manager
}

// NewDAGRunRepository creates a new DAG run repository
func NewDAGRunRepository(db *gorm.DB, stateManager *state.Manager) DAGRunRepository {
	return &dagRunRepository{
		db:           db,
		stateManager: stateManager,
	}
}

func (r *dagRunRepository) Create(ctx context.Context, run *models.DAGRun) error {
	model, err := FromDAGRun(run)
	if err != nil {
		return fmt.Errorf("failed to convert DAG run to model: %w", err)
	}

	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		return fmt.Errorf("failed to create DAG run: %w", err)
	}

	run.ID = model.ID.String()

	return nil
}

func (r *dagRunRepository) Get(ctx context.Context, id string) (*models.DAGRun, error) {
	runID, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("invalid DAG run ID: %w", err)
	}

	var model DAGRunModel
	if err := r.db.WithContext(ctx).Where("id = ?", runID).First(&model).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("DAG run not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get DAG run: %w", err)
	}

	return model.ToDAGRun(), nil
}

func (r *dagRunRepository) List(ctx context.Context, filters DAGRunFilters) ([]*models.DAGRun, error) {
	query := r.db.WithContext(ctx).Model(&DAGRunModel{})

	if filters.DAGID != "" {
		dagID, err := uuid.Parse(filters.DAGID)
		if err != nil {
			return nil, fmt.Errorf("invalid DAG ID: %w", err)
		}
		query = query.Where("dag_id = ?", dagID)
	}

	if filters.State != nil {
		query = query.Where("state = ?", string(*filters.State))
	}

	if filters.After != nil {
		query = query.Where("execution_date > ?", *filters.After)
	}

	if filters.Before != nil {
		query = query.Where("execution_date < ?", *filters.Before)
	}

	query = query.Order("execution_date DESC")

	if filters.Limit > 0 {
		query = query.Limit(filters.Limit)
	}

	if filters.Offset > 0 {
		query = query.Offset(filters.Offset)
	}

	var runModels []DAGRunModel
	if err := query.Find(&runModels).Error; err != nil {
		return nil, fmt.Errorf("failed to list DAG runs: %w", err)
	}

	runs := make([]*models.DAGRun, len(runModels))
	for i, model := range runModels {
		runs[i] = model.ToDAGRun()
	}

	return runs, nil
}

func (r *dagRunRepository) Update(ctx context.Context, run *models.DAGRun) error {
	runID, err := uuid.Parse(run.ID)
	if err != nil {
		return fmt.Errorf("invalid DAG run ID: %w", err)
	}

	model, err := FromDAGRun(run)
	if err != nil {
		return fmt.Errorf("failed to convert DAG run to model: %w", err)
	}

	model.ID = runID

	if err := r.db.WithContext(ctx).Model(&DAGRunModel{}).Where("id = ?", runID).Updates(model).Error; err != nil {
		return fmt.Errorf("failed to update DAG run: %w", err)
	}

	return nil
}

func (r *dagRunRepository) UpdateState(ctx context.Context, id string, oldState, newState models.State) error {
	runID, err := uuid.Parse(id)
	if err != nil {
		return fmt.Errorf("invalid DAG run ID: %w", err)
	}

	// Validate state transition
	if err := r.stateManager.Transition("dag_run", id, oldState, newState, nil); err != nil {
		return fmt.Errorf("invalid state transition: %w", err)
	}

	// Use optimistic locking to prevent concurrent updates
	result := r.db.WithContext(ctx).
		Model(&DAGRunModel{}).
		Where("id = ? AND state = ?", runID, string(oldState)).
		Updates(map[string]interface{}{
			"state":   string(newState),
			"version": gorm.Expr("version + 1"),
		})

	if result.Error != nil {
		return fmt.Errorf("failed to update DAG run state: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return state.ErrOptimisticLock
	}

	return nil
}

func (r *dagRunRepository) Delete(ctx context.Context, id string) error {
	runID, err := uuid.Parse(id)
	if err != nil {
		return fmt.Errorf("invalid DAG run ID: %w", err)
	}

	if err := r.db.WithContext(ctx).Delete(&DAGRunModel{}, "id = ?", runID).Error; err != nil {
		return fmt.Errorf("failed to delete DAG run: %w", err)
	}

	return nil
}

func (r *dagRunRepository) GetLatestRun(ctx context.Context, dagID string) (*models.DAGRun, error) {
	dagUUID, err := uuid.Parse(dagID)
	if err != nil {
		return nil, fmt.Errorf("invalid DAG ID: %w", err)
	}

	var model DAGRunModel
	if err := r.db.WithContext(ctx).
		Where("dag_id = ?", dagUUID).
		Order("execution_date DESC").
		First(&model).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("no runs found for DAG: %s", dagID)
		}
		return nil, fmt.Errorf("failed to get latest DAG run: %w", err)
	}

	return model.ToDAGRun(), nil
}
