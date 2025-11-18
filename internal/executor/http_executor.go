package executor

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/therealutkarshpriyadarshi/dag/pkg/models"
)

// HTTPTaskExecutor executes HTTP requests
type HTTPTaskExecutor struct {
	client  *http.Client
	timeout time.Duration
}

// HTTPTaskConfig represents HTTP task configuration
// The command field should be formatted as: METHOD URL [BODY]
// Example: "GET https://api.example.com/data"
// Example: "POST https://api.example.com/data {\"key\":\"value\"}"
type HTTPTaskConfig struct {
	Method  string
	URL     string
	Body    string
	Headers map[string]string
}

// NewHTTPTaskExecutor creates a new HTTP task executor
func NewHTTPTaskExecutor(timeout time.Duration) *HTTPTaskExecutor {
	return &HTTPTaskExecutor{
		client: &http.Client{
			Timeout: timeout,
		},
		timeout: timeout,
	}
}

// Type returns the task type this executor handles
func (e *HTTPTaskExecutor) Type() models.TaskType {
	return models.TaskTypeHTTP
}

// Execute makes an HTTP request and returns the result
func (e *HTTPTaskExecutor) Execute(ctx context.Context, task *models.Task, taskInstance *models.TaskInstance) *TaskResult {
	startTime := time.Now()
	hostname, _ := os.Hostname()

	result := &TaskResult{
		State:     models.StateSuccess,
		StartTime: startTime,
		Hostname:  hostname,
	}

	log.Printf("Executing HTTP task: %s, command: %s", task.ID, task.Command)

	// Parse command to extract HTTP config
	config, err := e.parseCommand(task.Command)
	if err != nil {
		result.EndTime = time.Now()
		result.State = models.StateFailed
		result.ErrorMessage = fmt.Sprintf("Failed to parse HTTP command: %v", err)
		return result
	}

	// Create request
	var bodyReader io.Reader
	if config.Body != "" {
		bodyReader = bytes.NewBufferString(config.Body)
	}

	req, err := http.NewRequestWithContext(ctx, config.Method, config.URL, bodyReader)
	if err != nil {
		result.EndTime = time.Now()
		result.State = models.StateFailed
		result.ErrorMessage = fmt.Sprintf("Failed to create request: %v", err)
		return result
	}

	// Set headers
	for key, value := range config.Headers {
		req.Header.Set(key, value)
	}

	// Set default content type for POST/PUT
	if (config.Method == "POST" || config.Method == "PUT") && req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json")
	}

	// Execute request
	resp, err := e.client.Do(req)
	result.EndTime = time.Now()

	if err != nil {
		result.State = models.StateFailed
		result.ErrorMessage = fmt.Sprintf("HTTP request failed: %v", err)
		log.Printf("HTTP task %s failed: %v", task.ID, err)
		return result
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		result.State = models.StateFailed
		result.ErrorMessage = fmt.Sprintf("Failed to read response: %v", err)
		return result
	}

	result.Output = fmt.Sprintf("Status: %d %s\nBody: %s", resp.StatusCode, resp.Status, string(body))

	// Check status code
	if resp.StatusCode >= 400 {
		result.State = models.StateFailed
		result.ErrorMessage = fmt.Sprintf("HTTP request returned error status: %d", resp.StatusCode)
		log.Printf("HTTP task %s failed with status %d", task.ID, resp.StatusCode)
	} else {
		log.Printf("HTTP task %s completed successfully with status %d", task.ID, resp.StatusCode)
	}

	return result
}

// parseCommand parses the command string into HTTP configuration
// Format: METHOD URL [BODY]
func (e *HTTPTaskExecutor) parseCommand(command string) (*HTTPTaskConfig, error) {
	parts := strings.SplitN(command, " ", 3)
	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid HTTP command format, expected: METHOD URL [BODY]")
	}

	config := &HTTPTaskConfig{
		Method:  strings.ToUpper(parts[0]),
		URL:     parts[1],
		Headers: make(map[string]string),
	}

	if len(parts) == 3 {
		config.Body = parts[2]
	}

	// Validate HTTP method
	validMethods := map[string]bool{
		"GET": true, "POST": true, "PUT": true, "DELETE": true,
		"PATCH": true, "HEAD": true, "OPTIONS": true,
	}

	if !validMethods[config.Method] {
		return nil, fmt.Errorf("invalid HTTP method: %s", config.Method)
	}

	return config, nil
}

// ParseHTTPCommand is a helper function to parse HTTP command with JSON config
// Supports both simple format (METHOD URL BODY) and JSON format
func ParseHTTPCommand(command string) (*HTTPTaskConfig, error) {
	// Try to parse as JSON first
	var config HTTPTaskConfig
	if err := json.Unmarshal([]byte(command), &config); err == nil {
		return &config, nil
	}

	// Fall back to simple format
	parts := strings.SplitN(command, " ", 3)
	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid HTTP command format")
	}

	config = HTTPTaskConfig{
		Method:  strings.ToUpper(parts[0]),
		URL:     parts[1],
		Headers: make(map[string]string),
	}

	if len(parts) == 3 {
		config.Body = parts[2]
	}

	return &config, nil
}
