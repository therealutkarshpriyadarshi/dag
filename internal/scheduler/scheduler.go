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

// Config holds scheduler configuration
type Config struct {
	// ScheduleInterval is how often the scheduler checks for new DAG runs
	ScheduleInterval time.Duration

	// MaxConcurrentDAGRuns is the global limit for concurrent DAG runs
	MaxConcurrentDAGRuns int

	// DefaultTimezone is the default timezone for cron schedules
	DefaultTimezone string

	// EnableCatchup enables running missed schedules
	EnableCatchup bool

	// MaxCatchupRuns is the maximum number of catchup runs to create
	MaxCatchupRuns int
}

// DefaultConfig returns the default scheduler configuration
func DefaultConfig() *Config {
	return &Config{
		ScheduleInterval:     10 * time.Second,
		MaxConcurrentDAGRuns: 100,
		DefaultTimezone:      "UTC",
		EnableCatchup:        true,
		MaxCatchupRuns:       50,
	}
}

// Scheduler manages DAG scheduling and execution
type Scheduler struct {
	config            *Config
	dagRepo           storage.DAGRepository
	dagRunRepo        storage.DAGRunRepository
	taskInstanceRepo  storage.TaskInstanceRepository
	cronScheduler     *CronScheduler
	concurrencyMgr    *ConcurrencyManager
	priorityQueue     *PriorityQueue
	mu                sync.RWMutex
	running           bool
	ctx               context.Context
	cancel            context.CancelFunc
	wg                sync.WaitGroup
}

// New creates a new Scheduler instance
func New(
	config *Config,
	dagRepo storage.DAGRepository,
	dagRunRepo storage.DAGRunRepository,
	taskInstanceRepo storage.TaskInstanceRepository,
	concurrencyMgr *ConcurrencyManager,
) *Scheduler {
	if config == nil {
		config = DefaultConfig()
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &Scheduler{
		config:           config,
		dagRepo:          dagRepo,
		dagRunRepo:       dagRunRepo,
		taskInstanceRepo: taskInstanceRepo,
		concurrencyMgr:   concurrencyMgr,
		priorityQueue:    NewPriorityQueue(),
		ctx:              ctx,
		cancel:           cancel,
	}
}

// Start begins the scheduler's operation
func (s *Scheduler) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return fmt.Errorf("scheduler is already running")
	}

	log.Println("Starting scheduler...")

	// Initialize cron scheduler
	location, err := time.LoadLocation(s.config.DefaultTimezone)
	if err != nil {
		return fmt.Errorf("failed to load timezone: %w", err)
	}

	s.cronScheduler = NewCronScheduler(location, s.createDAGRun)

	// Load all active DAGs and register them with cron scheduler
	if err := s.loadAndRegisterDAGs(); err != nil {
		return fmt.Errorf("failed to load DAGs: %w", err)
	}

	s.running = true

	// Start the main scheduling loop
	s.wg.Add(1)
	go s.schedulingLoop()

	log.Println("Scheduler started successfully")
	return nil
}

// Stop gracefully stops the scheduler
func (s *Scheduler) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return fmt.Errorf("scheduler is not running")
	}

	log.Println("Stopping scheduler...")

	// Cancel context to signal shutdown
	s.cancel()

	// Stop cron scheduler
	if s.cronScheduler != nil {
		s.cronScheduler.Stop()
	}

	// Wait for goroutines to finish
	s.wg.Wait()

	s.running = false
	log.Println("Scheduler stopped successfully")
	return nil
}

// IsRunning returns whether the scheduler is currently running
func (s *Scheduler) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.running
}

// TriggerDAG manually triggers a DAG run
func (s *Scheduler) TriggerDAG(dagID string, executionDate time.Time) (*models.DAGRun, error) {
	// Get DAG from repository
	dag, err := s.dagRepo.GetByID(s.ctx, dagID)
	if err != nil {
		return nil, fmt.Errorf("failed to get DAG: %w", err)
	}

	if dag.IsPaused {
		return nil, fmt.Errorf("cannot trigger paused DAG: %s", dag.Name)
	}

	// Create DAG run
	dagRun := &models.DAGRun{
		ID:              uuid.New().String(),
		DAGID:           dagID,
		ExecutionDate:   executionDate,
		State:           models.StateQueued,
		ExternalTrigger: true,
	}

	// Save to database
	if err := s.dagRunRepo.Create(s.ctx, dagRun); err != nil {
		return nil, fmt.Errorf("failed to create DAG run: %w", err)
	}

	// Add to priority queue
	s.priorityQueue.Push(&PriorityQueueItem{
		DAGRunID:      dagRun.ID,
		DAGID:         dagRun.DAGID,
		ExecutionDate: dagRun.ExecutionDate,
		Priority:      PriorityHigh, // External triggers get high priority
		EnqueuedAt:    time.Now(),
	})

	log.Printf("Manually triggered DAG run: %s (DAG: %s)", dagRun.ID, dag.Name)
	return dagRun, nil
}

// schedulingLoop is the main loop that processes scheduled DAG runs
func (s *Scheduler) schedulingLoop() {
	defer s.wg.Done()

	ticker := time.NewTicker(s.config.ScheduleInterval)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.processScheduledRuns()
		}
	}
}

// processScheduledRuns processes pending DAG runs from the priority queue
func (s *Scheduler) processScheduledRuns() {
	// Check global concurrency limit
	if !s.concurrencyMgr.CanScheduleGlobal() {
		log.Println("Global concurrency limit reached, skipping scheduling")
		return
	}

	// Process items from priority queue
	for {
		item := s.priorityQueue.Pop()
		if item == nil {
			break // Queue is empty
		}

		// Check DAG-level concurrency
		if !s.concurrencyMgr.CanScheduleDAG(item.DAGID) {
			// Re-queue for later
			s.priorityQueue.Push(item)
			break
		}

		// Submit DAG run for execution
		if err := s.submitDAGRun(item); err != nil {
			log.Printf("Failed to submit DAG run %s: %v", item.DAGRunID, err)
			continue
		}

		// Increment concurrency counters
		s.concurrencyMgr.IncrementGlobal()
		s.concurrencyMgr.IncrementDAG(item.DAGID)
	}
}

// submitDAGRun submits a DAG run for execution
func (s *Scheduler) submitDAGRun(item *PriorityQueueItem) error {
	// Get the DAG run from database
	dagRun, err := s.dagRunRepo.GetByID(s.ctx, item.DAGRunID)
	if err != nil {
		return fmt.Errorf("failed to get DAG run: %w", err)
	}

	// Update state to running
	now := time.Now()
	dagRun.State = models.StateRunning
	dagRun.StartDate = &now

	if err := s.dagRunRepo.Update(s.ctx, dagRun); err != nil {
		return fmt.Errorf("failed to update DAG run state: %w", err)
	}

	log.Printf("Submitted DAG run %s for execution", item.DAGRunID)
	return nil
}

// loadAndRegisterDAGs loads all active DAGs from the database and registers them with the cron scheduler
func (s *Scheduler) loadAndRegisterDAGs() error {
	// Get all DAGs
	dags, err := s.dagRepo.List(s.ctx)
	if err != nil {
		return fmt.Errorf("failed to list DAGs: %w", err)
	}

	// Register each DAG with a schedule
	for _, dag := range dags {
		if dag.IsPaused {
			log.Printf("Skipping paused DAG: %s", dag.Name)
			continue
		}

		if dag.Schedule == "" {
			log.Printf("Skipping DAG without schedule: %s", dag.Name)
			continue
		}

		if err := s.cronScheduler.AddDAG(dag.ID, dag.Schedule); err != nil {
			log.Printf("Failed to register DAG %s with schedule %s: %v", dag.Name, dag.Schedule, err)
			continue
		}

		// If catchup is enabled, create missed runs
		if s.config.EnableCatchup {
			if err := s.performCatchup(dag); err != nil {
				log.Printf("Failed to perform catchup for DAG %s: %v", dag.Name, err)
			}
		}

		log.Printf("Registered DAG %s with schedule: %s", dag.Name, dag.Schedule)
	}

	return nil
}

// performCatchup creates DAG runs for missed schedules
func (s *Scheduler) performCatchup(dag *models.DAG) error {
	// Get the last DAG run
	lastRun, err := s.dagRunRepo.GetLatestRun(s.ctx, dag.ID)
	if err != nil && err != storage.ErrNotFound {
		return err
	}

	var startTime time.Time
	if lastRun != nil {
		startTime = lastRun.ExecutionDate
	} else {
		startTime = dag.StartDate
	}

	// Calculate missed execution dates
	missedDates, err := s.cronScheduler.GetMissedExecutions(dag.Schedule, startTime, time.Now(), s.config.MaxCatchupRuns)
	if err != nil {
		return err
	}

	if len(missedDates) == 0 {
		return nil
	}

	log.Printf("Creating %d catchup runs for DAG %s", len(missedDates), dag.Name)

	// Create DAG runs for missed executions
	for _, execDate := range missedDates {
		if err := s.createDAGRun(dag.ID, execDate); err != nil {
			log.Printf("Failed to create catchup run for %s at %v: %v", dag.Name, execDate, err)
		}
	}

	return nil
}

// createDAGRun creates a new DAG run for a scheduled execution
func (s *Scheduler) createDAGRun(dagID string, executionDate time.Time) error {
	// Check if DAG run already exists
	existing, err := s.dagRunRepo.GetByExecutionDate(s.ctx, dagID, executionDate)
	if err == nil && existing != nil {
		log.Printf("DAG run already exists for %s at %v", dagID, executionDate)
		return nil
	}

	// Create new DAG run
	dagRun := &models.DAGRun{
		ID:              uuid.New().String(),
		DAGID:           dagID,
		ExecutionDate:   executionDate,
		State:           models.StateQueued,
		ExternalTrigger: false,
	}

	if err := s.dagRunRepo.Create(s.ctx, dagRun); err != nil {
		return fmt.Errorf("failed to create DAG run: %w", err)
	}

	// Add to priority queue
	s.priorityQueue.Push(&PriorityQueueItem{
		DAGRunID:      dagRun.ID,
		DAGID:         dagRun.DAGID,
		ExecutionDate: dagRun.ExecutionDate,
		Priority:      PriorityMedium,
		EnqueuedAt:    time.Now(),
	})

	log.Printf("Created scheduled DAG run: %s (execution date: %v)", dagRun.ID, executionDate)
	return nil
}

// RegisterDAG registers a new DAG with the scheduler
func (s *Scheduler) RegisterDAG(dagID, schedule string) error {
	if s.cronScheduler == nil {
		return fmt.Errorf("scheduler not started")
	}
	return s.cronScheduler.AddDAG(dagID, schedule)
}

// UnregisterDAG removes a DAG from the scheduler
func (s *Scheduler) UnregisterDAG(dagID string) error {
	if s.cronScheduler == nil {
		return fmt.Errorf("scheduler not started")
	}
	s.cronScheduler.RemoveDAG(dagID)
	return nil
}
