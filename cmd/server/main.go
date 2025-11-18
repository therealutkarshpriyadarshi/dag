package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	"github.com/therealutkarshpriyadarshi/dag/internal/dag"
	"github.com/therealutkarshpriyadarshi/dag/internal/executor"
	"github.com/therealutkarshpriyadarshi/dag/internal/state"
	"github.com/therealutkarshpriyadarshi/dag/internal/storage"
	"github.com/therealutkarshpriyadarshi/dag/pkg/api/dto"
	"github.com/therealutkarshpriyadarshi/dag/pkg/api/handlers"
	"github.com/therealutkarshpriyadarshi/dag/pkg/api/middleware"
)

const version = "0.6.0"

func main() {
	log.Printf("Starting Workflow Orchestrator Server v%s", version)

	// Get environment
	env := os.Getenv("ENV")
	if env == "" {
		env = "development"
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Database configuration from environment
	dbCfg := &storage.Config{
		Host:        getEnv("DB_HOST", "localhost"),
		Port:        getEnv("DB_PORT", "5432"),
		User:        getEnv("DB_USER", "workflow"),
		Password:    getEnv("DB_PASSWORD", "workflow_dev_password"),
		DBName:      getEnv("DB_NAME", "workflow_orchestrator"),
		SSLMode:     getEnv("DB_SSLMODE", "disable"),
		MaxConns:    25,
		MinConns:    5,
		MaxIdleTime: 5 * time.Minute,
		MaxLifetime: 30 * time.Minute,
	}

	// Initialize database connection
	db, err := storage.NewDB(dbCfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Run migrations
	migrateCfg := &storage.MigrateConfig{
		Host:     dbCfg.Host,
		Port:     dbCfg.Port,
		User:     dbCfg.User,
		Password: dbCfg.Password,
		DBName:   dbCfg.DBName,
		SSLMode:  dbCfg.SSLMode,
	}

	if err := storage.RunMigrations(migrateCfg, "./migrations"); err != nil {
		log.Printf("Warning: Failed to run migrations: %v", err)
	}

	// Initialize Redis client
	redisClient := redis.NewClient(&redis.Options{
		Addr: fmt.Sprintf("%s:%s", getEnv("REDIS_HOST", "localhost"), getEnv("REDIS_PORT", "6379")),
	})
	defer redisClient.Close()

	// Test Redis connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Printf("Warning: Failed to connect to Redis: %v", err)
	}

	// Initialize state management
	redisPublisher := state.NewRedisPublisher(redisClient)
	historyPublisher := state.NewHistoryPublisher(db.DB)
	multiPublisher := state.NewMultiPublisher(redisPublisher, historyPublisher)
	stateManager := state.NewManager(multiPublisher)
	stateMachine := state.NewMachine()

	// Initialize repositories
	dagRepo := storage.NewDAGRepository(db.DB)
	dagRunRepo := storage.NewDAGRunRepository(db.DB, stateManager)
	taskInstanceRepo := storage.NewTaskInstanceRepository(db.DB, stateManager)
	taskLogRepo := storage.NewTaskLogRepository(db.DB)

	// Initialize DAG engine
	dagEngine := dag.NewEngine()

	// Initialize executor
	executorCfg := &executor.ExecutorConfig{
		WorkerCount:  4,
		QueueSize:    100,
		MaxRetries:   3,
		RetryDelay:   5 * time.Second,
		TaskTimeout:  30 * time.Minute,
		ShutdownTimeout: 1 * time.Minute,
	}

	localExecutor := executor.NewLocalExecutor(
		taskInstanceRepo,
		dagRunRepo,
		stateMachine,
		executorCfg,
	)

	// Register task executors
	localExecutor.RegisterTaskExecutor(executor.NewBashTaskExecutor())
	localExecutor.RegisterTaskExecutor(executor.NewHTTPTaskExecutor())
	localExecutor.RegisterTaskExecutor(executor.NewGoFuncTaskExecutor())
	// Note: DockerTaskExecutor requires Docker client setup

	// Start executor
	executorCtx := context.Background()
	if err := localExecutor.Start(executorCtx); err != nil {
		log.Printf("Warning: Failed to start executor: %v", err)
	}
	defer localExecutor.Stop(executorCtx)

	log.Printf("Database initialized successfully")
	log.Printf("Repositories initialized: DAG, DAGRun, TaskInstance, TaskLog")
	log.Printf("Executor started with %d workers", executorCfg.WorkerCount)

	// Set Gin mode based on environment
	if env == "production" {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	// Create logger
	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})
	if env == "development" {
		logger.SetLevel(logrus.DebugLevel)
	} else {
		logger.SetLevel(logrus.InfoLevel)
	}

	// Create Gin router
	router := gin.New()

	// Apply global middleware
	router.Use(gin.Recovery())
	router.Use(middleware.ErrorHandler())
	router.Use(middleware.Logger(logger))
	router.Use(middleware.CORS())

	// Initialize handlers
	dagHandler := handlers.NewDAGHandler(dagRepo, dagEngine)
	dagRunHandler := handlers.NewDAGRunHandler(dagRepo, dagRunRepo, taskInstanceRepo, localExecutor)
	taskInstanceHandler := handlers.NewTaskInstanceHandler(taskInstanceRepo, taskLogRepo)

	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		// Check database health
		dbHealthy := true
		if err := db.Health(c.Request.Context()); err != nil {
			dbHealthy = false
		}

		// Check Redis health
		redisHealthy := true
		if err := redisClient.Ping(c.Request.Context()).Err(); err != nil {
			redisHealthy = false
		}

		status := "healthy"
		services := map[string]string{
			"database": "healthy",
			"redis":    "healthy",
			"executor": "healthy",
		}

		if !dbHealthy {
			status = "degraded"
			services["database"] = "unhealthy"
		}
		if !redisHealthy {
			status = "degraded"
			services["redis"] = "unhealthy"
		}
		if !localExecutor.GetStatus().Running {
			status = "degraded"
			services["executor"] = "stopped"
		}

		c.JSON(200, dto.HealthResponse{
			Status:   status,
			Services: services,
		})
	})

	// JWT configuration
	jwtConfig := middleware.DefaultJWTConfig()

	// Public API routes (no authentication required)
	public := router.Group("/api/v1")
	{
		public.GET("/status", func(c *gin.Context) {
			c.JSON(200, gin.H{
				"status":  "ok",
				"version": version,
				"phase":   "6 - REST API",
			})
		})

		// Health endpoint (public)
		public.GET("/health", func(c *gin.Context) {
			c.Redirect(301, "/health")
		})
	}

	// Protected API routes (authentication required)
	// For now, we'll make all endpoints public to enable easy testing
	// In production, you would use: api := router.Group("/api/v1", middleware.JWTAuth(jwtConfig))
	api := router.Group("/api/v1")
	api.Use(middleware.OptionalAuth(jwtConfig)) // Use optional auth for development
	api.Use(middleware.GlobalRateLimiter.RateLimit())

	// DAG routes
	dags := api.Group("/dags")
	{
		dags.POST("", dagHandler.CreateDAG)
		dags.GET("", dagHandler.ListDAGs)
		dags.GET("/:id", dagHandler.GetDAG)
		dags.PATCH("/:id", dagHandler.UpdateDAG)
		dags.DELETE("/:id", dagHandler.DeleteDAG)
		dags.POST("/:id/pause", dagHandler.PauseDAG)
		dags.POST("/:id/unpause", dagHandler.UnpauseDAG)
		dags.POST("/:id/trigger", dagRunHandler.TriggerDAG)
	}

	// DAG Run routes
	dagRuns := api.Group("/dag-runs")
	{
		dagRuns.GET("", dagRunHandler.ListDAGRuns)
		dagRuns.GET("/:id", dagRunHandler.GetDAGRun)
		dagRuns.POST("/:id/cancel", dagRunHandler.CancelDAGRun)
		dagRuns.GET("/:id/tasks", taskInstanceHandler.ListDAGRunTasks)
	}

	// Task Instance routes
	taskInstances := api.Group("/task-instances")
	{
		taskInstances.GET("", taskInstanceHandler.ListTaskInstances)
		taskInstances.GET("/:id", taskInstanceHandler.GetTaskInstance)
		taskInstances.GET("/:id/logs", taskInstanceHandler.GetTaskInstanceLogs)
		taskInstances.POST("/:id/retry", taskInstanceHandler.RetryTaskInstance)
	}

	// Start server
	log.Printf("Server listening on port %s in %s mode", port, env)
	log.Printf("Phase 6: REST API with authentication, rate limiting, and validation")
	log.Printf("API Documentation: http://localhost:%s/api/v1/status", port)

	if err := router.Run(fmt.Sprintf(":%s", port)); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
