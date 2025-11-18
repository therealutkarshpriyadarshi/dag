package executor

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/therealutkarshpriyadarshi/dag/pkg/models"
)

func TestHTTPTaskExecutor_Execute_GET_Success(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("Expected GET request, got %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"success"}`))
	}))
	defer server.Close()

	executor := NewHTTPTaskExecutor(10 * time.Second)

	task := &models.Task{
		ID:      "test-task",
		Type:    models.TaskTypeHTTP,
		Command: "GET " + server.URL,
		Timeout: 10 * time.Second,
	}

	taskInstance := &models.TaskInstance{
		ID:     "test-instance",
		TaskID: "test-task",
	}

	ctx := context.Background()
	result := executor.Execute(ctx, task, taskInstance)

	if result.State != models.StateSuccess {
		t.Errorf("Expected state Success, got %s. Error: %s", result.State, result.ErrorMessage)
	}

	if !strings.Contains(result.Output, "200") {
		t.Errorf("Expected output to contain status 200, got: %s", result.Output)
	}
}

func TestHTTPTaskExecutor_Execute_POST_Success(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"id":"123"}`))
	}))
	defer server.Close()

	executor := NewHTTPTaskExecutor(10 * time.Second)

	task := &models.Task{
		ID:      "test-task",
		Type:    models.TaskTypeHTTP,
		Command: `POST ` + server.URL + ` {"name":"test"}`,
		Timeout: 10 * time.Second,
	}

	taskInstance := &models.TaskInstance{
		ID:     "test-instance",
		TaskID: "test-task",
	}

	ctx := context.Background()
	result := executor.Execute(ctx, task, taskInstance)

	if result.State != models.StateSuccess {
		t.Errorf("Expected state Success, got %s. Error: %s", result.State, result.ErrorMessage)
	}

	if !strings.Contains(result.Output, "201") {
		t.Errorf("Expected output to contain status 201, got: %s", result.Output)
	}
}

func TestHTTPTaskExecutor_Execute_Error_Status(t *testing.T) {
	// Create test server that returns 404
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error":"not found"}`))
	}))
	defer server.Close()

	executor := NewHTTPTaskExecutor(10 * time.Second)

	task := &models.Task{
		ID:      "test-task",
		Type:    models.TaskTypeHTTP,
		Command: "GET " + server.URL,
		Timeout: 10 * time.Second,
	}

	taskInstance := &models.TaskInstance{
		ID:     "test-instance",
		TaskID: "test-task",
	}

	ctx := context.Background()
	result := executor.Execute(ctx, task, taskInstance)

	if result.State != models.StateFailed {
		t.Errorf("Expected state Failed for 404 response, got %s", result.State)
	}

	if !strings.Contains(result.ErrorMessage, "404") {
		t.Errorf("Expected error message to contain 404, got: %s", result.ErrorMessage)
	}
}

func TestHTTPTaskExecutor_ParseCommand(t *testing.T) {
	executor := NewHTTPTaskExecutor(10 * time.Second)

	tests := []struct {
		name        string
		command     string
		expectError bool
		expectMethod string
		expectURL   string
	}{
		{
			name:        "GET without body",
			command:     "GET https://api.example.com/users",
			expectError: false,
			expectMethod: "GET",
			expectURL:   "https://api.example.com/users",
		},
		{
			name:        "POST with body",
			command:     `POST https://api.example.com/users {"name":"test"}`,
			expectError: false,
			expectMethod: "POST",
			expectURL:   "https://api.example.com/users",
		},
		{
			name:        "Invalid format",
			command:     "INVALID",
			expectError: true,
		},
		{
			name:        "Invalid method",
			command:     "INVALID_METHOD https://example.com",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, err := executor.parseCommand(tt.command)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if config.Method != tt.expectMethod {
					t.Errorf("Expected method %s, got %s", tt.expectMethod, config.Method)
				}
				if config.URL != tt.expectURL {
					t.Errorf("Expected URL %s, got %s", tt.expectURL, config.URL)
				}
			}
		})
	}
}

func TestHTTPTaskExecutor_Type(t *testing.T) {
	executor := NewHTTPTaskExecutor(10 * time.Second)

	if executor.Type() != models.TaskTypeHTTP {
		t.Errorf("Expected type HTTP, got %s", executor.Type())
	}
}
