package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

const version = "0.1.0"

func main() {
	log.Printf("Starting Workflow Orchestrator Scheduler v%s", version)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Scheduler main loop
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				log.Println("Scheduler shutting down...")
				return
			case <-ticker.C:
				log.Println("Scheduler tick - checking for scheduled DAGs")
			}
		}
	}()

	// Wait for shutdown signal
	sig := <-sigChan
	log.Printf("Received signal %v, initiating graceful shutdown...", sig)
	cancel()

	// Give scheduler time to finish current operations
	time.Sleep(2 * time.Second)
	log.Println("Scheduler stopped")
}
