package main

import (
	"fmt"
	"log"
	"os"

	"github.com/gin-gonic/gin"
)

const version = "0.1.0"

func main() {
	log.Printf("Starting Workflow Orchestrator Server v%s", version)

	// Check if we're in development mode
	if os.Getenv("ENV") != "production" {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.Default()

	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "healthy",
			"version": version,
		})
	})

	// API v1 routes will be added here
	v1 := router.Group("/api/v1")
	{
		v1.GET("/status", func(c *gin.Context) {
			c.JSON(200, gin.H{
				"message": "API v1 is ready",
			})
		})
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server listening on port %s", port)
	if err := router.Run(fmt.Sprintf(":%s", port)); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
