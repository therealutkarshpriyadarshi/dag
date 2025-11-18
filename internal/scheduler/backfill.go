package scheduler

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/therealutkarshpriyadarshi/dag/internal/storage"
	"github.com/therealutkarshpriyadarshi/dag/pkg/models"
)

// BackfillConfig holds configuration for backfill operations
type BackfillConfig struct {
	// MaxConcurrency is the maximum number of concurrent backfill runs
	MaxConcurrency int

	// DryRun if true, will not create actual DAG runs
	DryRun bool

	// ReprocessFailed if true, will recreate failed runs
	ReprocessFailed bool

	// ReprocessSuccessful if true, will recreate successful runs
	ReprocessSuccessful bool
}

// BackfillEngine handles backfilling of DAG runs for missed or historical executions
type BackfillEngine struct {
	dagRepo      storage.DAGRepository
	dagRunRepo   storage.DAGRunRepository
	cronScheduler *CronScheduler
	config       *BackfillConfig
	ctx          context.Context
}

// NewBackfillEngine creates a new backfill engine
func NewBackfillEngine(
	ctx context.Context,
	dagRepo storage.DAGRepository,
	dagRunRepo storage.DAGRunRepository,
	cronScheduler *CronScheduler,
	config *BackfillConfig,
) *BackfillEngine {
	if config == nil {
		config = &BackfillConfig{
			MaxConcurrency:      5,
			DryRun:              false,
			ReprocessFailed:     false,
			ReprocessSuccessful: false,
		}
	}

	return &BackfillEngine{
		dagRepo:       dagRepo,
		dagRunRepo:    dagRunRepo,
		cronScheduler: cronScheduler,
		config:        config,
		ctx:           ctx,
	}
}

// BackfillRequest represents a request to backfill DAG runs
type BackfillRequest struct {
	DAGID     string
	StartDate time.Time
	EndDate   time.Time
}

// BackfillResult represents the result of a backfill operation
type BackfillResult struct {
	DAGID            string
	TotalRuns        int
	CreatedRuns      int
	SkippedRuns      int
	FailedRuns       int
	ExecutionDates   []time.Time
	Errors           []error
	Duration         time.Duration
}

// Backfill performs a backfill operation for a DAG
func (be *BackfillEngine) Backfill(req BackfillRequest) (*BackfillResult, error) {
	startTime := time.Now()

	log.Printf("Starting backfill for DAG %s from %v to %v", req.DAGID, req.StartDate, req.EndDate)

	// Get DAG
	dag, err := be.dagRepo.GetByID(be.ctx, req.DAGID)
	if err != nil {
		return nil, fmt.Errorf("failed to get DAG: %w", err)
	}

	if dag.Schedule == "" {
		return nil, fmt.Errorf("DAG %s has no schedule defined", dag.Name)
	}

	// Calculate execution dates based on schedule
	execDates, err := be.cronScheduler.GetMissedExecutions(dag.Schedule, req.StartDate, req.EndDate, 1000)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate execution dates: %w", err)
	}

	if len(execDates) == 0 {
		log.Printf("No execution dates found for backfill period")
		return &BackfillResult{
			DAGID:     req.DAGID,
			TotalRuns: 0,
			Duration:  time.Since(startTime),
		}, nil
	}

	log.Printf("Found %d potential execution dates for backfill", len(execDates))

	result := &BackfillResult{
		DAGID:          req.DAGID,
		TotalRuns:      len(execDates),
		ExecutionDates: execDates,
	}

	// Process backfill with concurrency control
	if err := be.processBackfillWithConcurrency(dag, execDates, result); err != nil {
		return result, err
	}

	result.Duration = time.Since(startTime)
	log.Printf("Backfill completed: created=%d, skipped=%d, failed=%d, duration=%v",
		result.CreatedRuns, result.SkippedRuns, result.FailedRuns, result.Duration)

	return result, nil
}

// processBackfillWithConcurrency processes backfill runs with concurrency control
func (be *BackfillEngine) processBackfillWithConcurrency(dag *models.DAG, execDates []time.Time, result *BackfillResult) error {
	// Create semaphore for concurrency control
	sem := make(chan struct{}, be.config.MaxConcurrency)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var errors []error

	for _, execDate := range execDates {
		wg.Add(1)
		sem <- struct{}{} // Acquire semaphore

		go func(date time.Time) {
			defer wg.Done()
			defer func() { <-sem }() // Release semaphore

			created, err := be.createBackfillRun(dag, date)

			mu.Lock()
			defer mu.Unlock()

			if err != nil {
				result.FailedRuns++
				result.Errors = append(result.Errors, err)
				errors = append(errors, err)
			} else if created {
				result.CreatedRuns++
			} else {
				result.SkippedRuns++
			}
		}(execDate)
	}

	wg.Wait()

	if len(errors) > 0 {
		return fmt.Errorf("backfill completed with %d errors", len(errors))
	}

	return nil
}

// createBackfillRun creates a single backfill DAG run
func (be *BackfillEngine) createBackfillRun(dag *models.DAG, execDate time.Time) (bool, error) {
	// Check if run already exists
	existing, err := be.dagRunRepo.GetByExecutionDate(be.ctx, dag.ID, execDate)
	if err != nil && err != storage.ErrNotFound {
		return false, fmt.Errorf("failed to check existing run: %w", err)
	}

	// Handle existing runs based on configuration
	if existing != nil {
		shouldReprocess := false

		if existing.State == models.StateFailed && be.config.ReprocessFailed {
			shouldReprocess = true
		} else if existing.State == models.StateSuccess && be.config.ReprocessSuccessful {
			shouldReprocess = true
		}

		if !shouldReprocess {
			log.Printf("Skipping existing run for %v (state: %s)", execDate, existing.State)
			return false, nil
		}

		// Delete existing run if reprocessing
		if err := be.dagRunRepo.Delete(be.ctx, existing.ID); err != nil {
			return false, fmt.Errorf("failed to delete existing run: %w", err)
		}
		log.Printf("Deleted existing run for reprocessing: %s", existing.ID)
	}

	// Dry run - just log what would be created
	if be.config.DryRun {
		log.Printf("[DRY RUN] Would create DAG run for %s at %v", dag.Name, execDate)
		return true, nil
	}

	// Create new DAG run
	dagRun := &models.DAGRun{
		ID:              uuid.New().String(),
		DAGID:           dag.ID,
		ExecutionDate:   execDate,
		State:           models.StateQueued,
		ExternalTrigger: false, // Backfill runs are not external triggers
	}

	if err := be.dagRunRepo.Create(be.ctx, dagRun); err != nil {
		return false, fmt.Errorf("failed to create DAG run: %w", err)
	}

	log.Printf("Created backfill run: %s for execution date %v", dagRun.ID, execDate)
	return true, nil
}

// CancelBackfill cancels an ongoing backfill operation (future enhancement)
func (be *BackfillEngine) CancelBackfill(dagID string) error {
	// This would require maintaining a registry of active backfill operations
	// For now, return not implemented
	return fmt.Errorf("cancel backfill not yet implemented")
}

// GetBackfillStatus returns the status of a backfill operation (future enhancement)
func (be *BackfillEngine) GetBackfillStatus(dagID string) (*BackfillResult, error) {
	// This would require maintaining a registry of backfill operations
	// For now, return not implemented
	return nil, fmt.Errorf("get backfill status not yet implemented")
}

// ValidateBackfillRequest validates a backfill request
func (be *BackfillEngine) ValidateBackfillRequest(req BackfillRequest) error {
	if req.DAGID == "" {
		return fmt.Errorf("DAG ID is required")
	}

	if req.StartDate.IsZero() {
		return fmt.Errorf("start date is required")
	}

	if req.EndDate.IsZero() {
		return fmt.Errorf("end date is required")
	}

	if req.EndDate.Before(req.StartDate) {
		return fmt.Errorf("end date must be after start date")
	}

	// Check if DAG exists
	dag, err := be.dagRepo.GetByID(be.ctx, req.DAGID)
	if err != nil {
		return fmt.Errorf("failed to get DAG: %w", err)
	}

	if dag.Schedule == "" {
		return fmt.Errorf("DAG has no schedule defined")
	}

	return nil
}
