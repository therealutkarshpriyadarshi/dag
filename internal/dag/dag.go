package dag

import (
	"fmt"

	"github.com/therealutkarshpriyadarshi/dag/pkg/models"
)

// Validator provides DAG validation functionality
type Validator struct{}

// NewValidator creates a new DAG validator
func NewValidator() *Validator {
	return &Validator{}
}

// Validate checks if a DAG is valid
func (v *Validator) Validate(dag *models.DAG) error {
	if dag.Name == "" {
		return fmt.Errorf("DAG name cannot be empty")
	}

	if len(dag.Tasks) == 0 {
		return fmt.Errorf("DAG must have at least one task")
	}

	// Check for duplicate task IDs
	taskIDs := make(map[string]bool)
	for _, task := range dag.Tasks {
		if taskIDs[task.ID] {
			return fmt.Errorf("duplicate task ID: %s", task.ID)
		}
		taskIDs[task.ID] = true
	}

	// Validate task dependencies exist
	for _, task := range dag.Tasks {
		for _, depID := range task.Dependencies {
			if !taskIDs[depID] {
				return fmt.Errorf("task %s depends on non-existent task: %s", task.ID, depID)
			}
		}
	}

	// Check for cycles
	if err := v.detectCycle(dag); err != nil {
		return err
	}

	return nil
}

// detectCycle checks if there are any cycles in the DAG
func (v *Validator) detectCycle(dag *models.DAG) error {
	// Build adjacency list
	adjList := make(map[string][]string)
	for _, task := range dag.Tasks {
		adjList[task.ID] = task.Dependencies
	}

	// Track visit states: 0 = unvisited, 1 = visiting, 2 = visited
	visited := make(map[string]int)

	var dfs func(string) error
	dfs = func(taskID string) error {
		if visited[taskID] == 1 {
			return fmt.Errorf("cycle detected involving task: %s", taskID)
		}
		if visited[taskID] == 2 {
			return nil
		}

		visited[taskID] = 1
		for _, depID := range adjList[taskID] {
			if err := dfs(depID); err != nil {
				return err
			}
		}
		visited[taskID] = 2
		return nil
	}

	for _, task := range dag.Tasks {
		if visited[task.ID] == 0 {
			if err := dfs(task.ID); err != nil {
				return err
			}
		}
	}

	return nil
}

// GetTopologicalOrder returns tasks in topological order
func (v *Validator) GetTopologicalOrder(dag *models.DAG) ([]string, error) {
	// Build adjacency list and in-degree map
	adjList := make(map[string][]string)
	inDegree := make(map[string]int)

	for _, task := range dag.Tasks {
		inDegree[task.ID] = len(task.Dependencies)
		for _, depID := range task.Dependencies {
			adjList[depID] = append(adjList[depID], task.ID)
		}
	}

	// Kahn's algorithm
	var queue []string
	for taskID, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, taskID)
		}
	}

	var result []string
	for len(queue) > 0 {
		taskID := queue[0]
		queue = queue[1:]
		result = append(result, taskID)

		for _, neighbor := range adjList[taskID] {
			inDegree[neighbor]--
			if inDegree[neighbor] == 0 {
				queue = append(queue, neighbor)
			}
		}
	}

	if len(result) != len(dag.Tasks) {
		return nil, fmt.Errorf("cycle detected in DAG")
	}

	return result, nil
}
