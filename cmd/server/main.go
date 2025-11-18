package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/therealutkarshpriyadarshi/dag/internal/state"
	"github.com/therealutkarshpriyadarshi/dag/internal/storage"
)

const version = "0.2.0"

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

	// Initialize repositories
	dagRepo := storage.NewDAGRepository(db.DB)
	dagRunRepo := storage.NewDAGRunRepository(db.DB, stateManager)
	taskInstanceRepo := storage.NewTaskInstanceRepository(db.DB, stateManager)
	_ = storage.NewTaskLogRepository(db.DB)

	log.Printf("Database initialized successfully")
	log.Printf("Repositories initialized: DAG, DAGRun, TaskInstance, TaskLog")

	// Set Gin mode based on environment
	if env == "production" {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	// Create Gin router
	router := gin.Default()

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
		if !dbHealthy || !redisHealthy {
			status = "degraded"
		}

		c.JSON(200, gin.H{
			"status":   status,
			"version":  version,
			"service":  "workflow-server",
			"database": dbHealthy,
			"redis":    redisHealthy,
		})
	})

	// API v1 routes
	v1 := router.Group("/api/v1")
	{
		v1.GET("/status", func(c *gin.Context) {
			c.JSON(200, gin.H{
				"status":  "ok",
				"version": version,
				"phase":   "2 - Database Layer & State Management",
			})
		})

		// DAG routes
		v1.GET("/dags", func(c *gin.Context) {
			dags, err := dagRepo.List(c.Request.Context(), storage.DAGFilters{Limit: 100})
			if err != nil {
				c.JSON(500, gin.H{"error": err.Error()})
				return
			}
			c.JSON(200, gin.H{"dags": dags, "count": len(dags)})
		})

		v1.GET("/dags/:id", func(c *gin.Context) {
			dag, err := dagRepo.Get(c.Request.Context(), c.Param("id"))
			if err != nil {
				c.JSON(404, gin.H{"error": "DAG not found"})
				return
			}
			c.JSON(200, dag)
		})

		// DAG Run routes
		v1.GET("/dag-runs", func(c *gin.Context) {
			runs, err := dagRunRepo.List(c.Request.Context(), storage.DAGRunFilters{Limit: 100})
			if err != nil {
				c.JSON(500, gin.H{"error": err.Error()})
				return
			}
			c.JSON(200, gin.H{"dag_runs": runs, "count": len(runs)})
		})

		v1.GET("/dag-runs/:id", func(c *gin.Context) {
			run, err := dagRunRepo.Get(c.Request.Context(), c.Param("id"))
			if err != nil {
				c.JSON(404, gin.H{"error": "DAG run not found"})
				return
			}
			c.JSON(200, run)
		})

		// Task Instance routes
		v1.GET("/task-instances", func(c *gin.Context) {
			instances, err := taskInstanceRepo.List(c.Request.Context(), storage.TaskInstanceFilters{Limit: 100})
			if err != nil {
				c.JSON(500, gin.H{"error": err.Error()})
				return
			}
			c.JSON(200, gin.H{"task_instances": instances, "count": len(instances)})
		})

		v1.GET("/task-instances/:id", func(c *gin.Context) {
			instance, err := taskInstanceRepo.Get(c.Request.Context(), c.Param("id"))
			if err != nil {
				c.JSON(404, gin.H{"error": "Task instance not found"})
				return
			}
			c.JSON(200, instance)
		})
	}

	// Start server
	log.Printf("Server listening on port %s in %s mode", port, env)
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
