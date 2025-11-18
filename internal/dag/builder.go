package dag

import (
	"fmt"
	"time"

	"github.com/therealutkarshpriyadarshi/dag/pkg/models"
)

// Builder provides a fluent API for building DAGs
type Builder struct {
	dag    *models.DAG
	tasks  map[string]*models.Task
	errors []error
}

// NewBuilder creates a new DAG builder
func NewBuilder(name string) *Builder {
	return &Builder{
		dag: &models.DAG{
			Name:      name,
			Tasks:     []models.Task{},
			Tags:      []string{},
			IsPaused:  false,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		tasks:  make(map[string]*models.Task),
		errors: []error{},
	}
}

// ID sets the DAG ID
func (b *Builder) ID(id string) *Builder {
	b.dag.ID = id
	return b
}

// Description sets the DAG description
func (b *Builder) Description(desc string) *Builder {
	b.dag.Description = desc
	return b
}

// Schedule sets the cron schedule for the DAG
func (b *Builder) Schedule(cronExpr string) *Builder {
	b.dag.Schedule = cronExpr
	return b
}

// StartDate sets the DAG start date
func (b *Builder) StartDate(t time.Time) *Builder {
	b.dag.StartDate = t
	return b
}

// EndDate sets the DAG end date
func (b *Builder) EndDate(t time.Time) *Builder {
	b.dag.EndDate = &t
	return b
}

// Tags adds tags to the DAG
func (b *Builder) Tags(tags ...string) *Builder {
	b.dag.Tags = append(b.dag.Tags, tags...)
	return b
}

// Paused sets whether the DAG is paused
func (b *Builder) Paused(paused bool) *Builder {
	b.dag.IsPaused = paused
	return b
}

// Task adds a task to the DAG
func (b *Builder) Task(id string, taskBuilder *TaskBuilder) *Builder {
	task := taskBuilder.build(id)
	b.tasks[id] = task
	return b
}

// Build constructs the final DAG and validates it
func (b *Builder) Build() (*models.DAG, error) {
	// Convert tasks map to slice
	b.dag.Tasks = make([]models.Task, 0, len(b.tasks))
	for _, task := range b.tasks {
		b.dag.Tasks = append(b.dag.Tasks, *task)
	}

	// Validate the DAG
	validator := NewValidator()
	if err := validator.Validate(b.dag); err != nil {
		return nil, fmt.Errorf("DAG validation failed: %w", err)
	}

	return b.dag, nil
}

// MustBuild builds the DAG and panics if there's an error (useful for testing)
func (b *Builder) MustBuild() *models.DAG {
	dag, err := b.Build()
	if err != nil {
		panic(err)
	}
	return dag
}

// TaskBuilder provides a fluent API for building tasks
type TaskBuilder struct {
	name         string
	taskType     models.TaskType
	command      string
	dependencies []string
	retries      int
	timeout      time.Duration
	sla          time.Duration
}

// BashTask creates a new Bash task builder
func BashTask(command string) *TaskBuilder {
	return &TaskBuilder{
		taskType: models.TaskTypeBash,
		command:  command,
		retries:  0,
	}
}

// HTTPTask creates a new HTTP task builder
func HTTPTask(url string) *TaskBuilder {
	return &TaskBuilder{
		taskType: models.TaskTypeHTTP,
		command:  url,
		retries:  0,
	}
}

// PythonTask creates a new Python task builder
func PythonTask(script string) *TaskBuilder {
	return &TaskBuilder{
		taskType: models.TaskTypePython,
		command:  script,
		retries:  0,
	}
}

// GoTask creates a new Go task builder
func GoTask(funcName string) *TaskBuilder {
	return &TaskBuilder{
		taskType: models.TaskTypeGo,
		command:  funcName,
		retries:  0,
	}
}

// Name sets the task name
func (tb *TaskBuilder) Name(name string) *TaskBuilder {
	tb.name = name
	return tb
}

// DependsOn sets task dependencies
func (tb *TaskBuilder) DependsOn(taskIDs ...string) *TaskBuilder {
	tb.dependencies = append(tb.dependencies, taskIDs...)
	return tb
}

// Retries sets the number of retries
func (tb *TaskBuilder) Retries(count int) *TaskBuilder {
	tb.retries = count
	return tb
}

// Timeout sets the task timeout
func (tb *TaskBuilder) Timeout(duration time.Duration) *TaskBuilder {
	tb.timeout = duration
	return tb
}

// SLA sets the task SLA
func (tb *TaskBuilder) SLA(duration time.Duration) *TaskBuilder {
	tb.sla = duration
	return tb
}

// build constructs the final task
func (tb *TaskBuilder) build(id string) *models.Task {
	name := tb.name
	if name == "" {
		name = id
	}

	return &models.Task{
		ID:           id,
		Name:         name,
		Type:         tb.taskType,
		Command:      tb.command,
		Dependencies: tb.dependencies,
		Retries:      tb.retries,
		Timeout:      tb.timeout,
		SLA:          tb.sla,
	}
}
