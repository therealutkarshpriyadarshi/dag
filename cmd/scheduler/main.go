package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/therealutkarshpriyadarshi/dag/internal/scheduler"
	"github.com/therealutkarshpriyadarshi/dag/internal/state"
	"github.com/therealutkarshpriyadarshi/dag/internal/storage"
)

const version = "0.2.0"

var (
	// Database flags
	dbHost     = flag.String("db-host", getEnv("DB_HOST", "localhost"), "Database host")
	dbPort     = flag.String("db-port", getEnv("DB_PORT", "5432"), "Database port")
	dbUser     = flag.String("db-user", getEnv("DB_USER", "workflow"), "Database user")
	dbPassword = flag.String("db-password", getEnv("DB_PASSWORD", "workflow_dev_password"), "Database password")
	dbName     = flag.String("db-name", getEnv("DB_NAME", "workflow_orchestrator"), "Database name")

	// Redis flags
	redisHost     = flag.String("redis-host", getEnv("REDIS_HOST", "localhost"), "Redis host")
	redisPort     = flag.String("redis-port", getEnv("REDIS_PORT", "6379"), "Redis port")
	redisPassword = flag.String("redis-password", getEnv("REDIS_PASSWORD", ""), "Redis password")
	redisDB       = flag.Int("redis-db", 0, "Redis database")

	// Scheduler flags
	scheduleInterval     = flag.Duration("schedule-interval", 10*time.Second, "Schedule check interval")
	maxConcurrentRuns    = flag.Int("max-concurrent-runs", 100, "Maximum concurrent DAG runs")
	enableCatchup        = flag.Bool("enable-catchup", true, "Enable catchup for missed schedules")
	maxCatchupRuns       = flag.Int("max-catchup-runs", 50, "Maximum number of catchup runs")
	timezone             = flag.String("timezone", "UTC", "Default timezone for schedules")

	// Backfill flags
	backfillMode         = flag.Bool("backfill", false, "Run in backfill mode")
	backfillDAGID        = flag.String("backfill-dag-id", "", "DAG ID for backfill")
	backfillStart        = flag.String("backfill-start", "", "Backfill start date (RFC3339)")
	backfillEnd          = flag.String("backfill-end", "", "Backfill end date (RFC3339)")
	backfillConcurrency  = flag.Int("backfill-concurrency", 5, "Backfill concurrency")
	backfillDryRun       = flag.Bool("backfill-dry-run", false, "Backfill dry run")
)

func main() {
	flag.Parse()

	log.Printf("Starting Workflow Orchestrator Scheduler v%s", version)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize database
	db, err := initDatabase()
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	log.Println("Database connection established")

	// Initialize Redis
	redisClient := initRedis()
	log.Println("Redis connection established")

	// Initialize repositories
	stateManager := state.NewManager(nil)
	dagRepo := storage.NewDAGRepository(db.DB)
	dagRunRepo := storage.NewDAGRunRepository(db.DB, stateManager)
	taskInstanceRepo := storage.NewTaskInstanceRepository(db.DB, stateManager)

	// Check if running in backfill mode
	if *backfillMode {
		runBackfill(ctx, dagRepo, dagRunRepo)
		return
	}

	// Initialize concurrency manager
	concurrencyConfig := &scheduler.ConcurrencyConfig{
		MaxGlobalConcurrency:  *maxConcurrentRuns,
		DefaultDAGConcurrency: 16,
		Pools:                 make(map[string]int),
		RedisClient:           redisClient,
		LockTTL:               30 * time.Second,
	}
	concurrencyMgr := scheduler.NewConcurrencyManager(ctx, concurrencyConfig)

	// Initialize scheduler
	schedulerConfig := &scheduler.Config{
		ScheduleInterval:     *scheduleInterval,
		MaxConcurrentDAGRuns: *maxConcurrentRuns,
		DefaultTimezone:      *timezone,
		EnableCatchup:        *enableCatchup,
		MaxCatchupRuns:       *maxCatchupRuns,
	}

	sched := scheduler.New(
		schedulerConfig,
		dagRepo,
		dagRunRepo,
		taskInstanceRepo,
		concurrencyMgr,
	)

	// Start scheduler
	if err := sched.Start(); err != nil {
		log.Fatalf("Failed to start scheduler: %v", err)
	}

	log.Println("Scheduler started successfully")
	log.Printf("Schedule interval: %v", *scheduleInterval)
	log.Printf("Max concurrent runs: %d", *maxConcurrentRuns)
	log.Printf("Catchup enabled: %v", *enableCatchup)
	log.Printf("Timezone: %s", *timezone)

	// Setup signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Wait for shutdown signal
	sig := <-sigChan
	log.Printf("Received signal %v, initiating graceful shutdown...", sig)

	// Stop scheduler
	if err := sched.Stop(); err != nil {
		log.Printf("Error stopping scheduler: %v", err)
	}

	// Close database connection
	sqlDB, _ := db.DB.DB()
	if sqlDB != nil {
		sqlDB.Close()
	}

	// Close Redis connection
	if redisClient != nil {
		redisClient.Close()
	}

	log.Println("Scheduler stopped gracefully")
}

func runBackfill(ctx context.Context, dagRepo storage.DAGRepository, dagRunRepo storage.DAGRunRepository) {
	log.Println("Running in backfill mode")

	// Validate required flags
	if *backfillDAGID == "" {
		log.Fatal("--backfill-dag-id is required for backfill mode")
	}
	if *backfillStart == "" {
		log.Fatal("--backfill-start is required for backfill mode")
	}
	if *backfillEnd == "" {
		log.Fatal("--backfill-end is required for backfill mode")
	}

	// Parse dates
	startDate, err := time.Parse(time.RFC3339, *backfillStart)
	if err != nil {
		log.Fatalf("Invalid backfill start date: %v", err)
	}

	endDate, err := time.Parse(time.RFC3339, *backfillEnd)
	if err != nil {
		log.Fatalf("Invalid backfill end date: %v", err)
	}

	// Initialize backfill engine
	location, err := time.LoadLocation(*timezone)
	if err != nil {
		log.Fatalf("Invalid timezone: %v", err)
	}

	cronScheduler := scheduler.NewCronScheduler(location, nil)
	backfillConfig := &scheduler.BackfillConfig{
		MaxConcurrency:      *backfillConcurrency,
		DryRun:              *backfillDryRun,
		ReprocessFailed:     false,
		ReprocessSuccessful: false,
	}

	backfillEngine := scheduler.NewBackfillEngine(
		ctx,
		dagRepo,
		dagRunRepo,
		cronScheduler,
		backfillConfig,
	)

	// Perform backfill
	req := scheduler.BackfillRequest{
		DAGID:     *backfillDAGID,
		StartDate: startDate,
		EndDate:   endDate,
	}

	log.Printf("Starting backfill for DAG %s from %v to %v", req.DAGID, req.StartDate, req.EndDate)

	result, err := backfillEngine.Backfill(req)
	if err != nil {
		log.Printf("Backfill completed with errors: %v", err)
	}

	// Print results
	log.Printf("Backfill completed:")
	log.Printf("  Total runs: %d", result.TotalRuns)
	log.Printf("  Created: %d", result.CreatedRuns)
	log.Printf("  Skipped: %d", result.SkippedRuns)
	log.Printf("  Failed: %d", result.FailedRuns)
	log.Printf("  Duration: %v", result.Duration)

	if len(result.Errors) > 0 {
		log.Printf("  Errors encountered: %d", len(result.Errors))
		for i, err := range result.Errors {
			if i < 5 { // Show first 5 errors
				log.Printf("    - %v", err)
			}
		}
	}
}

func initDatabase() (*storage.DB, error) {
	config := &storage.Config{
		Host:        *dbHost,
		Port:        *dbPort,
		User:        *dbUser,
		Password:    *dbPassword,
		DBName:      *dbName,
		SSLMode:     "disable",
		MaxConns:    25,
		MinConns:    5,
		MaxIdleTime: 5 * time.Minute,
		MaxLifetime: 30 * time.Minute,
	}

	db, err := storage.NewDB(config)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Run migrations (if migration files exist)
	migrateConfig := &storage.MigrateConfig{
		Host:     *dbHost,
		Port:     *dbPort,
		User:     *dbUser,
		Password: *dbPassword,
		DBName:   *dbName,
		SSLMode:  "disable",
	}
	if err := storage.RunMigrations(migrateConfig, "./migrations"); err != nil {
		log.Printf("Warning: Failed to run migrations (migrations directory may not exist): %v", err)
		// Don't fail if migrations directory doesn't exist in development
	}

	return db, nil
}

func initRedis() *redis.Client {
	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", *redisHost, *redisPort),
		Password: *redisPassword,
		DB:       *redisDB,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		log.Printf("Warning: Redis connection failed: %v", err)
		return nil
	}

	return client
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
