package executor

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/therealutkarshpriyadarshi/dag/pkg/models"
)

// DockerTaskExecutor executes tasks in Docker containers
type DockerTaskExecutor struct {
	defaultImage  string
	network       string
	volumeMounts  []string
	removeOnExit  bool
	memoryLimitMB int64
	cpuQuota      int
}

// DockerTaskConfig represents Docker task configuration
type DockerTaskConfig struct {
	Image        string            `json:"image"`
	Command      string            `json:"command"`
	Env          map[string]string `json:"env,omitempty"`
	Volumes      []string          `json:"volumes,omitempty"`
	WorkingDir   string            `json:"working_dir,omitempty"`
	Network      string            `json:"network,omitempty"`
	MemoryMB     int64             `json:"memory_mb,omitempty"`
	CPUQuota     int               `json:"cpu_quota,omitempty"`
	RemoveOnExit bool              `json:"remove_on_exit,omitempty"`
}

// NewDockerTaskExecutor creates a new Docker task executor
func NewDockerTaskExecutor(defaultImage string) *DockerTaskExecutor {
	return &DockerTaskExecutor{
		defaultImage:  defaultImage,
		network:       "bridge",
		volumeMounts:  []string{},
		removeOnExit:  true,
		memoryLimitMB: 1024,
		cpuQuota:      100,
	}
}

// NewDockerTaskExecutorWithConfig creates a Docker executor with custom config
func NewDockerTaskExecutorWithConfig(config *DockerTaskConfig) *DockerTaskExecutor {
	return &DockerTaskExecutor{
		defaultImage:  config.Image,
		network:       config.Network,
		volumeMounts:  config.Volumes,
		removeOnExit:  config.RemoveOnExit,
		memoryLimitMB: config.MemoryMB,
		cpuQuota:      config.CPUQuota,
	}
}

// Type returns the task type this executor handles
func (e *DockerTaskExecutor) Type() models.TaskType {
	// Docker executor can handle Python tasks by running them in containers
	return models.TaskTypePython
}

// Execute runs a task in a Docker container
func (e *DockerTaskExecutor) Execute(ctx context.Context, task *models.Task, taskInstance *models.TaskInstance) *TaskResult {
	startTime := time.Now()
	hostname, _ := os.Hostname()

	result := &TaskResult{
		State:     models.StateSuccess,
		StartTime: startTime,
		Hostname:  hostname,
	}

	log.Printf("Executing Docker task: %s", task.ID)

	// Check if Docker is available
	if !e.isDockerAvailable() {
		result.EndTime = time.Now()
		result.State = models.StateFailed
		result.ErrorMessage = "Docker is not available or not running"
		return result
	}

	// Parse task configuration
	config, err := e.parseTaskConfig(task)
	if err != nil {
		result.EndTime = time.Now()
		result.State = models.StateFailed
		result.ErrorMessage = fmt.Sprintf("Failed to parse task config: %v", err)
		return result
	}

	// Build docker run command
	args := e.buildDockerCommand(config, task.ID)

	// Execute Docker command
	cmd := exec.CommandContext(ctx, "docker", args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	log.Printf("Running Docker command: docker %s", strings.Join(args, " "))

	err = cmd.Run()
	result.EndTime = time.Now()

	// Combine output
	output := stdout.String()
	if stderr.Len() > 0 {
		output += "\n" + stderr.String()
	}
	result.Output = output

	if err != nil {
		result.State = models.StateFailed
		result.ErrorMessage = fmt.Sprintf("Docker execution failed: %v\nOutput: %s", err, output)
		log.Printf("Docker task %s failed: %v", task.ID, err)
	} else {
		log.Printf("Docker task %s completed successfully", task.ID)
	}

	// Check for context cancellation (timeout)
	if ctx.Err() != nil {
		result.State = models.StateFailed
		result.ErrorMessage = fmt.Sprintf("Task timed out: %v", ctx.Err())
		// Try to stop the container
		e.stopContainer(task.ID)
	}

	return result
}

// parseTaskConfig parses the task command into Docker configuration
func (e *DockerTaskExecutor) parseTaskConfig(task *models.Task) (*DockerTaskConfig, error) {
	config := &DockerTaskConfig{
		Image:        e.defaultImage,
		Command:      task.Command,
		RemoveOnExit: e.removeOnExit,
		MemoryMB:     e.memoryLimitMB,
		CPUQuota:     e.cpuQuota,
		Network:      e.network,
		Volumes:      e.volumeMounts,
	}

	// Try to parse command as JSON for advanced configuration
	if strings.HasPrefix(task.Command, "{") {
		if err := json.Unmarshal([]byte(task.Command), config); err == nil {
			// Successfully parsed JSON config
			return config, nil
		}
	}

	// Simple format: just use the command as is
	return config, nil
}

// buildDockerCommand builds the docker run command arguments
func (e *DockerTaskExecutor) buildDockerCommand(config *DockerTaskConfig, containerName string) []string {
	args := []string{"run"}

	// Container name
	args = append(args, "--name", fmt.Sprintf("dag-task-%s", containerName))

	// Remove container on exit
	if config.RemoveOnExit {
		args = append(args, "--rm")
	}

	// Memory limit
	if config.MemoryMB > 0 {
		args = append(args, "--memory", fmt.Sprintf("%dm", config.MemoryMB))
	}

	// CPU quota
	if config.CPUQuota > 0 && config.CPUQuota < 100 {
		args = append(args, "--cpu-quota", fmt.Sprintf("%d", config.CPUQuota*1000))
	}

	// Network
	if config.Network != "" {
		args = append(args, "--network", config.Network)
	}

	// Volume mounts
	for _, volume := range config.Volumes {
		args = append(args, "-v", volume)
	}

	// Environment variables
	for key, value := range config.Env {
		args = append(args, "-e", fmt.Sprintf("%s=%s", key, value))
	}

	// Working directory
	if config.WorkingDir != "" {
		args = append(args, "-w", config.WorkingDir)
	}

	// Image
	args = append(args, config.Image)

	// Command
	args = append(args, "sh", "-c", config.Command)

	return args
}

// isDockerAvailable checks if Docker is available
func (e *DockerTaskExecutor) isDockerAvailable() bool {
	cmd := exec.Command("docker", "version")
	err := cmd.Run()
	return err == nil
}

// stopContainer stops a running container
func (e *DockerTaskExecutor) stopContainer(taskID string) {
	containerName := fmt.Sprintf("dag-task-%s", taskID)
	cmd := exec.Command("docker", "stop", containerName)
	if err := cmd.Run(); err != nil {
		log.Printf("Failed to stop container %s: %v", containerName, err)
	}
}
