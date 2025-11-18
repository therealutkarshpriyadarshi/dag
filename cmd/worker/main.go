package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/therealutkarshpriyadarshi/dag/internal/executor"
)

const version = "0.4.0"

func main() {
	// Parse command-line flags
	natsURL := flag.String("nats", os.Getenv("NATS_URL"), "NATS server URL")
	workerCount := flag.Int("workers", 5, "Number of concurrent workers")
	timeout := flag.Duration("timeout", 30*time.Minute, "Default task timeout")
	enableDocker := flag.Bool("docker", false, "Enable Docker task executor")
	flag.Parse()

	// Set default NATS URL if not provided
	if *natsURL == "" {
		*natsURL = "nats://localhost:4222"
	}

	log.Printf("Starting Workflow Orchestrator Worker v%s", version)
	log.Printf("NATS URL: %s", *natsURL)
	log.Printf("Worker concurrency: %d", *workerCount)
	log.Printf("Task timeout: %v", *timeout)
	log.Printf("Docker enabled: %v", *enableDocker)

	// Create executor config
	config := &executor.ExecutorConfig{
		WorkerCount:     *workerCount,
		TaskTimeout:     *timeout,
		ShutdownTimeout: 30 * time.Second,
		EnableDocker:    *enableDocker,
		MaxMemoryMB:     1024,
		MaxCPUPercent:   100,
	}

	// Create distributed worker
	worker, err := executor.NewWorker(*natsURL, config)
	if err != nil {
		log.Fatalf("Failed to create worker: %v", err)
	}

	// Register task executors
	worker.RegisterTaskExecutor(executor.NewBashTaskExecutor())
	worker.RegisterTaskExecutor(executor.NewHTTPTaskExecutor(config.TaskTimeout))
	worker.RegisterTaskExecutor(executor.NewGoFuncTaskExecutor())

	if *enableDocker {
		worker.RegisterTaskExecutor(executor.NewDockerTaskExecutor("python:3.11-slim"))
		log.Println("Docker task executor registered")
	}

	// Setup context and signal handling for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start the worker
	if err := worker.Start(ctx); err != nil {
		log.Fatalf("Failed to start worker: %v", err)
	}

	log.Printf("Worker %s started and ready to process tasks", worker.GetID())

	// Wait for shutdown signal
	sig := <-sigChan
	log.Printf("Received signal %v, initiating graceful shutdown...", sig)

	// Stop the worker gracefully
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), config.ShutdownTimeout)
	defer shutdownCancel()

	if err := worker.Stop(shutdownCtx); err != nil {
		log.Printf("Error during shutdown: %v", err)
	}

	log.Println("Worker stopped successfully")
}
