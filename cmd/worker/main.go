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
	log.Printf("Starting Workflow Orchestrator Worker v%s", version)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Worker main loop
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				log.Println("Worker shutting down...")
				return
			case <-ticker.C:
				log.Println("Worker heartbeat - ready to process tasks")
			}
		}
	}()

	// Wait for shutdown signal
	sig := <-sigChan
	log.Printf("Received signal %v, initiating graceful shutdown...", sig)
	cancel()

	// Give workers time to finish current tasks
	time.Sleep(2 * time.Second)
	log.Println("Worker stopped")
}
