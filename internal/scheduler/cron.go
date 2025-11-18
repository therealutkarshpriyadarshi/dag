package scheduler

import (
	"fmt"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
)

// DAGRunCreator is a function type for creating DAG runs
type DAGRunCreator func(dagID string, executionDate time.Time) error

// CronScheduler manages cron-based scheduling for DAGs
type CronScheduler struct {
	cron      *cron.Cron
	location  *time.Location
	creator   DAGRunCreator
	entries   map[string]cron.EntryID // dagID -> entryID
	mu        sync.RWMutex
}

// NewCronScheduler creates a new cron scheduler
func NewCronScheduler(location *time.Location, creator DAGRunCreator) *CronScheduler {
	return &CronScheduler{
		cron:     cron.New(cron.WithLocation(location), cron.WithSeconds()),
		location: location,
		creator:  creator,
		entries:  make(map[string]cron.EntryID),
	}
}

// Start starts the cron scheduler
func (cs *CronScheduler) Start() {
	cs.cron.Start()
}

// Stop stops the cron scheduler
func (cs *CronScheduler) Stop() {
	ctx := cs.cron.Stop()
	<-ctx.Done() // Wait for all jobs to complete
}

// AddDAG adds a DAG to the cron scheduler
func (cs *CronScheduler) AddDAG(dagID, schedule string) error {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	// Check if DAG is already registered
	if _, exists := cs.entries[dagID]; exists {
		return fmt.Errorf("DAG %s is already registered", dagID)
	}

	// Parse and validate cron expression
	_, err := cron.ParseStandard(schedule)
	if err != nil {
		return fmt.Errorf("invalid cron expression %s: %w", schedule, err)
	}

	// Add job to cron
	entryID, err := cs.cron.AddFunc(schedule, func() {
		executionDate := time.Now().In(cs.location)
		if err := cs.creator(dagID, executionDate); err != nil {
			// Log error but don't stop the scheduler
			fmt.Printf("Error creating DAG run for %s: %v\n", dagID, err)
		}
	})

	if err != nil {
		return fmt.Errorf("failed to add cron job: %w", err)
	}

	cs.entries[dagID] = entryID
	return nil
}

// RemoveDAG removes a DAG from the cron scheduler
func (cs *CronScheduler) RemoveDAG(dagID string) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	if entryID, exists := cs.entries[dagID]; exists {
		cs.cron.Remove(entryID)
		delete(cs.entries, dagID)
	}
}

// GetScheduledDAGs returns all currently scheduled DAG IDs
func (cs *CronScheduler) GetScheduledDAGs() []string {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	dagIDs := make([]string, 0, len(cs.entries))
	for dagID := range cs.entries {
		dagIDs = append(dagIDs, dagID)
	}
	return dagIDs
}

// GetNextExecution returns the next scheduled execution time for a DAG
func (cs *CronScheduler) GetNextExecution(dagID string) (*time.Time, error) {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	entryID, exists := cs.entries[dagID]
	if !exists {
		return nil, fmt.Errorf("DAG %s is not registered", dagID)
	}

	entry := cs.cron.Entry(entryID)
	if entry.ID == 0 {
		return nil, fmt.Errorf("entry not found for DAG %s", dagID)
	}

	nextTime := entry.Next
	return &nextTime, nil
}

// GetMissedExecutions calculates execution times that were missed between start and end times
func (cs *CronScheduler) GetMissedExecutions(schedule string, startTime, endTime time.Time, maxRuns int) ([]time.Time, error) {
	// Parse cron schedule
	sched, err := cron.ParseStandard(schedule)
	if err != nil {
		return nil, fmt.Errorf("invalid cron expression: %w", err)
	}

	var executions []time.Time
	current := startTime

	// Generate execution times until we reach endTime or maxRuns
	for len(executions) < maxRuns {
		next := sched.Next(current)
		if next.After(endTime) {
			break
		}
		executions = append(executions, next)
		current = next
	}

	return executions, nil
}

// IsRegistered checks if a DAG is registered with the scheduler
func (cs *CronScheduler) IsRegistered(dagID string) bool {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	_, exists := cs.entries[dagID]
	return exists
}

// UpdateSchedule updates the cron schedule for a DAG
func (cs *CronScheduler) UpdateSchedule(dagID, newSchedule string) error {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	// Remove existing entry
	if entryID, exists := cs.entries[dagID]; exists {
		cs.cron.Remove(entryID)
		delete(cs.entries, dagID)
	}

	// Add new entry
	cs.mu.Unlock() // Unlock before calling AddDAG to avoid deadlock
	err := cs.AddDAG(dagID, newSchedule)
	cs.mu.Lock() // Relock before returning

	return err
}
