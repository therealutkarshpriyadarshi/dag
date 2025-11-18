package storage

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/therealutkarshpriyadarshi/dag/pkg/models"
	"gorm.io/gorm"
)

type dagRepository struct {
	db *gorm.DB
}

// NewDAGRepository creates a new DAG repository
func NewDAGRepository(db *gorm.DB) DAGRepository {
	return &dagRepository{db: db}
}

func (r *dagRepository) Create(ctx context.Context, dag *models.DAG) error {
	model, err := FromDAG(dag)
	if err != nil {
		return fmt.Errorf("failed to convert DAG to model: %w", err)
	}

	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		return fmt.Errorf("failed to create DAG: %w", err)
	}

	dag.ID = model.ID.String()
	dag.CreatedAt = model.CreatedAt
	dag.UpdatedAt = model.UpdatedAt

	return nil
}

func (r *dagRepository) Get(ctx context.Context, id string) (*models.DAG, error) {
	dagID, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("invalid DAG ID: %w", err)
	}

	var model DAGModel
	if err := r.db.WithContext(ctx).Where("id = ?", dagID).First(&model).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("DAG not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get DAG: %w", err)
	}

	return model.ToDAG(), nil
}

// GetByID is an alias for Get
func (r *dagRepository) GetByID(ctx context.Context, id string) (*models.DAG, error) {
	return r.Get(ctx, id)
}

func (r *dagRepository) GetByName(ctx context.Context, name string) (*models.DAG, error) {
	var model DAGModel
	if err := r.db.WithContext(ctx).Where("name = ?", name).First(&model).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("DAG not found: %s", name)
		}
		return nil, fmt.Errorf("failed to get DAG by name: %w", err)
	}

	return model.ToDAG(), nil
}

func (r *dagRepository) List(ctx context.Context, filters ...DAGFilters) ([]*models.DAG, error) {
	query := r.db.WithContext(ctx).Model(&DAGModel{})

	// Apply filters if provided
	if len(filters) > 0 {
		filter := filters[0]

		if filter.IsPaused != nil {
			query = query.Where("is_paused = ?", *filter.IsPaused)
		}

		if len(filter.Tags) > 0 {
			// Query for DAGs that have any of the specified tags
			for _, tag := range filter.Tags {
				query = query.Where("tags @> ?", fmt.Sprintf("[\"%s\"]", tag))
			}
		}

		if filter.Limit > 0 {
			query = query.Limit(filter.Limit)
		}

		if filter.Offset > 0 {
			query = query.Offset(filter.Offset)
		}
	}

	var dagModels []DAGModel
	if err := query.Find(&dagModels).Error; err != nil {
		return nil, fmt.Errorf("failed to list DAGs: %w", err)
	}

	dags := make([]*models.DAG, len(dagModels))
	for i, model := range dagModels {
		dags[i] = model.ToDAG()
	}

	return dags, nil
}

func (r *dagRepository) Update(ctx context.Context, dag *models.DAG) error {
	dagID, err := uuid.Parse(dag.ID)
	if err != nil {
		return fmt.Errorf("invalid DAG ID: %w", err)
	}

	model, err := FromDAG(dag)
	if err != nil {
		return fmt.Errorf("failed to convert DAG to model: %w", err)
	}

	model.ID = dagID

	if err := r.db.WithContext(ctx).Model(&DAGModel{}).Where("id = ?", dagID).Updates(model).Error; err != nil {
		return fmt.Errorf("failed to update DAG: %w", err)
	}

	return nil
}

func (r *dagRepository) Delete(ctx context.Context, id string) error {
	dagID, err := uuid.Parse(id)
	if err != nil {
		return fmt.Errorf("invalid DAG ID: %w", err)
	}

	if err := r.db.WithContext(ctx).Delete(&DAGModel{}, "id = ?", dagID).Error; err != nil {
		return fmt.Errorf("failed to delete DAG: %w", err)
	}

	return nil
}

func (r *dagRepository) Pause(ctx context.Context, id string) error {
	dagID, err := uuid.Parse(id)
	if err != nil {
		return fmt.Errorf("invalid DAG ID: %w", err)
	}

	if err := r.db.WithContext(ctx).Model(&DAGModel{}).Where("id = ?", dagID).Update("is_paused", true).Error; err != nil {
		return fmt.Errorf("failed to pause DAG: %w", err)
	}

	return nil
}

func (r *dagRepository) Unpause(ctx context.Context, id string) error {
	dagID, err := uuid.Parse(id)
	if err != nil {
		return fmt.Errorf("invalid DAG ID: %w", err)
	}

	if err := r.db.WithContext(ctx).Model(&DAGModel{}).Where("id = ?", dagID).Update("is_paused", false).Error; err != nil {
		return fmt.Errorf("failed to unpause DAG: %w", err)
	}

	return nil
}
