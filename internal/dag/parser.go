package dag

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/goccy/go-yaml"
	"github.com/therealutkarshpriyadarshi/dag/pkg/models"
)

// Parser handles parsing DAG definitions from various formats
type Parser struct {
	validator *Validator
}

// NewParser creates a new DAG parser
func NewParser() *Parser {
	return &Parser{
		validator: NewValidator(),
	}
}

// dagFile represents the structure of a DAG definition file
type dagFile struct {
	ID          string         `json:"id" yaml:"id"`
	Name        string         `json:"name" yaml:"name"`
	Description string         `json:"description" yaml:"description"`
	Schedule    string         `json:"schedule" yaml:"schedule"`
	StartDate   string         `json:"start_date" yaml:"start_date"`
	EndDate     string         `json:"end_date,omitempty" yaml:"end_date,omitempty"`
	Tags        []string       `json:"tags" yaml:"tags"`
	IsPaused    bool           `json:"is_paused" yaml:"is_paused"`
	Tasks       []taskFile     `json:"tasks" yaml:"tasks"`
}

// taskFile represents the structure of a task in a DAG file
type taskFile struct {
	ID           string   `json:"id" yaml:"id"`
	Name         string   `json:"name" yaml:"name"`
	Type         string   `json:"type" yaml:"type"`
	Command      string   `json:"command" yaml:"command"`
	Dependencies []string `json:"dependencies,omitempty" yaml:"dependencies,omitempty"`
	Retries      int      `json:"retries,omitempty" yaml:"retries,omitempty"`
	Timeout      string   `json:"timeout,omitempty" yaml:"timeout,omitempty"`
	SLA          string   `json:"sla,omitempty" yaml:"sla,omitempty"`
}

// ParseYAMLFile parses a DAG definition from a YAML file
func (p *Parser) ParseYAMLFile(filepath string) (*models.DAG, error) {
	data, err := os.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	return p.ParseYAML(data)
}

// ParseYAML parses a DAG definition from YAML bytes
func (p *Parser) ParseYAML(data []byte) (*models.DAG, error) {
	var df dagFile
	if err := yaml.Unmarshal(data, &df); err != nil {
		return nil, fmt.Errorf("failed to unmarshal YAML: %w", err)
	}

	return p.convertToDAG(&df)
}

// ParseJSONFile parses a DAG definition from a JSON file
func (p *Parser) ParseJSONFile(filepath string) (*models.DAG, error) {
	data, err := os.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	return p.ParseJSON(data)
}

// ParseJSON parses a DAG definition from JSON bytes
func (p *Parser) ParseJSON(data []byte) (*models.DAG, error) {
	var df dagFile
	if err := json.Unmarshal(data, &df); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	return p.convertToDAG(&df)
}

// convertToDAG converts a dagFile to a models.DAG
func (p *Parser) convertToDAG(df *dagFile) (*models.DAG, error) {
	now := time.Now()

	// Parse start date
	var startDate time.Time
	var err error
	if df.StartDate != "" {
		startDate, err = time.Parse(time.RFC3339, df.StartDate)
		if err != nil {
			// Try alternative format
			startDate, err = time.Parse("2006-01-02", df.StartDate)
			if err != nil {
				return nil, fmt.Errorf("invalid start_date format: %w", err)
			}
		}
	} else {
		startDate = now
	}

	// Parse end date if provided
	var endDate *time.Time
	if df.EndDate != "" {
		ed, err := time.Parse(time.RFC3339, df.EndDate)
		if err != nil {
			// Try alternative format
			ed, err = time.Parse("2006-01-02", df.EndDate)
			if err != nil {
				return nil, fmt.Errorf("invalid end_date format: %w", err)
			}
		}
		endDate = &ed
	}

	// Convert tasks
	tasks := make([]models.Task, 0, len(df.Tasks))
	for _, tf := range df.Tasks {
		task, err := p.convertToTask(&tf)
		if err != nil {
			return nil, fmt.Errorf("failed to convert task %s: %w", tf.ID, err)
		}
		tasks = append(tasks, *task)
	}

	dag := &models.DAG{
		ID:          df.ID,
		Name:        df.Name,
		Description: df.Description,
		Schedule:    df.Schedule,
		Tasks:       tasks,
		StartDate:   startDate,
		EndDate:     endDate,
		Tags:        df.Tags,
		IsPaused:    df.IsPaused,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	// Validate the DAG
	if err := p.validator.Validate(dag); err != nil {
		return nil, fmt.Errorf("DAG validation failed: %w", err)
	}

	return dag, nil
}

// convertToTask converts a taskFile to a models.Task
func (p *Parser) convertToTask(tf *taskFile) (*models.Task, error) {
	// Parse task type
	taskType, err := parseTaskType(tf.Type)
	if err != nil {
		return nil, err
	}

	// Parse timeout
	var timeout time.Duration
	if tf.Timeout != "" {
		timeout, err = time.ParseDuration(tf.Timeout)
		if err != nil {
			return nil, fmt.Errorf("invalid timeout format: %w", err)
		}
	}

	// Parse SLA
	var sla time.Duration
	if tf.SLA != "" {
		sla, err = time.ParseDuration(tf.SLA)
		if err != nil {
			return nil, fmt.Errorf("invalid sla format: %w", err)
		}
	}

	task := &models.Task{
		ID:           tf.ID,
		Name:         tf.Name,
		Type:         taskType,
		Command:      tf.Command,
		Dependencies: tf.Dependencies,
		Retries:      tf.Retries,
		Timeout:      timeout,
		SLA:          sla,
	}

	return task, nil
}

// parseTaskType converts a string to a TaskType
func parseTaskType(typeStr string) (models.TaskType, error) {
	switch typeStr {
	case "bash", "shell":
		return models.TaskTypeBash, nil
	case "http", "rest":
		return models.TaskTypeHTTP, nil
	case "python", "py":
		return models.TaskTypePython, nil
	case "go", "golang":
		return models.TaskTypeGo, nil
	default:
		return "", fmt.Errorf("invalid task type: %s", typeStr)
	}
}
