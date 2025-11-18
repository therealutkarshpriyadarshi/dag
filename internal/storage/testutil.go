package storage

import (
	"fmt"
	"os"
	"testing"

	"github.com/therealutkarshpriyadarshi/dag/internal/state"
	"gorm.io/gorm"
)

// SetupTestDB creates a test database for integration tests
func SetupTestDB(t *testing.T) (*DB, func()) {
	t.Helper()

	// Use environment variables if available, otherwise skip tests
	host := os.Getenv("DB_HOST")
	if host == "" {
		host = "localhost"
	}

	port := os.Getenv("DB_PORT")
	if port == "" {
		port = "5432"
	}

	user := os.Getenv("DB_USER")
	if user == "" {
		user = "workflow"
	}

	password := os.Getenv("DB_PASSWORD")
	if password == "" {
		password = "workflow_dev_password"
	}

	dbname := os.Getenv("DB_NAME")
	if dbname == "" {
		dbname = "workflow_orchestrator"
	}

	cfg := &Config{
		Host:     host,
		Port:     port,
		User:     user,
		Password: password,
		DBName:   dbname,
		SSLMode:  "disable",
		MaxConns: 10,
		MinConns: 2,
	}

	db, err := NewDB(cfg)
	if err != nil {
		t.Skipf("Failed to connect to test database: %v. Set DB_HOST, DB_PORT, etc. to run integration tests", err)
	}

	// Run migrations
	migrateCfg := &MigrateConfig{
		Host:     cfg.Host,
		Port:     cfg.Port,
		User:     cfg.User,
		Password: cfg.Password,
		DBName:   cfg.DBName,
		SSLMode:  cfg.SSLMode,
	}

	if err := RunMigrations(migrateCfg, "./../../migrations"); err != nil {
		// Try relative path from different location
		if err := RunMigrations(migrateCfg, "../../../migrations"); err != nil {
			t.Logf("Warning: Failed to run migrations: %v", err)
		}
	}

	cleanup := func() {
		// Clean up test data
		db.Exec("TRUNCATE TABLE task_logs CASCADE")
		db.Exec("TRUNCATE TABLE state_history CASCADE")
		db.Exec("TRUNCATE TABLE task_instances CASCADE")
		db.Exec("TRUNCATE TABLE dag_runs CASCADE")
		db.Exec("TRUNCATE TABLE dags CASCADE")
		db.Close()
	}

	return db, cleanup
}

// CreateTestRepositories creates test repositories with a shared state manager
func CreateTestRepositories(db *gorm.DB) (DAGRepository, DAGRunRepository, TaskInstanceRepository, TaskLogRepository) {
	stateManager := state.NewManager(&state.NoOpPublisher{})

	dagRepo := NewDAGRepository(db)
	dagRunRepo := NewDAGRunRepository(db, stateManager)
	taskInstanceRepo := NewTaskInstanceRepository(db, stateManager)
	taskLogRepo := NewTaskLogRepository(db)

	return dagRepo, dagRunRepo, taskInstanceRepo, taskLogRepo
}

// PrintTestDatabaseInfo prints information about connecting to the test database
func PrintTestDatabaseInfo() {
	fmt.Println("Integration tests require a PostgreSQL database.")
	fmt.Println("Set the following environment variables to configure:")
	fmt.Println("  DB_HOST (default: localhost)")
	fmt.Println("  DB_PORT (default: 5432)")
	fmt.Println("  DB_USER (default: workflow)")
	fmt.Println("  DB_PASSWORD (default: workflow_dev_password)")
	fmt.Println("  DB_NAME (default: workflow_orchestrator)")
	fmt.Println("\nOr run: docker-compose up -d postgres")
}
