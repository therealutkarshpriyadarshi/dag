package storage

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type taskLogRepository struct {
	db *gorm.DB
}

// NewTaskLogRepository creates a new task log repository
func NewTaskLogRepository(db *gorm.DB) TaskLogRepository {
	return &taskLogRepository{db: db}
}

func (r *taskLogRepository) Create(ctx context.Context, taskInstanceID, logData string) error {
	instanceID, err := uuid.Parse(taskInstanceID)
	if err != nil {
		return fmt.Errorf("invalid task instance ID: %w", err)
	}

	log := TaskLogModel{
		TaskInstanceID: instanceID,
		LogData:        logData,
	}

	if err := r.db.WithContext(ctx).Create(&log).Error; err != nil {
		return fmt.Errorf("failed to create task log: %w", err)
	}

	return nil
}

func (r *taskLogRepository) List(ctx context.Context, taskInstanceID string, limit int) ([]TaskLogModel, error) {
	instanceID, err := uuid.Parse(taskInstanceID)
	if err != nil {
		return nil, fmt.Errorf("invalid task instance ID: %w", err)
	}

	query := r.db.WithContext(ctx).
		Where("task_instance_id = ?", instanceID).
		Order("timestamp ASC")

	if limit > 0 {
		query = query.Limit(limit)
	}

	var logs []TaskLogModel
	if err := query.Find(&logs).Error; err != nil {
		return nil, fmt.Errorf("failed to list task logs: %w", err)
	}

	return logs, nil
}

func (r *taskLogRepository) Delete(ctx context.Context, taskInstanceID string) error {
	instanceID, err := uuid.Parse(taskInstanceID)
	if err != nil {
		return fmt.Errorf("invalid task instance ID: %w", err)
	}

	if err := r.db.WithContext(ctx).Where("task_instance_id = ?", instanceID).Delete(&TaskLogModel{}).Error; err != nil {
		return fmt.Errorf("failed to delete task logs: %w", err)
	}

	return nil
}
