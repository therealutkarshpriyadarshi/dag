package dag

import (
	"fmt"
	"time"

	"github.com/therealutkarshpriyadarshi/dag/pkg/models"
)

// Graph represents a DAG as an adjacency list with advanced graph algorithms
type Graph struct {
	tasks      map[string]*models.Task
	adjList    map[string][]string // taskID -> list of dependent task IDs
	revAdjList map[string][]string // taskID -> list of dependency task IDs
}

// NewGraph creates a new Graph from a DAG
func NewGraph(dag *models.DAG) *Graph {
	g := &Graph{
		tasks:      make(map[string]*models.Task),
		adjList:    make(map[string][]string),
		revAdjList: make(map[string][]string),
	}

	// Build task map
	for i := range dag.Tasks {
		task := &dag.Tasks[i]
		g.tasks[task.ID] = task
		g.adjList[task.ID] = []string{}
		g.revAdjList[task.ID] = task.Dependencies
	}

	// Build adjacency list (forward edges: task -> tasks that depend on it)
	for _, task := range dag.Tasks {
		for _, depID := range task.Dependencies {
			g.adjList[depID] = append(g.adjList[depID], task.ID)
		}
	}

	return g
}

// GetParallelTasks returns tasks that can be executed in parallel
// These are tasks with no pending dependencies
func (g *Graph) GetParallelTasks(completed map[string]bool) []string {
	var parallelTasks []string

	for taskID := range g.tasks {
		// Skip already completed tasks
		if completed[taskID] {
			continue
		}

		// Check if all dependencies are completed
		allDepsCompleted := true
		for _, depID := range g.revAdjList[taskID] {
			if !completed[depID] {
				allDepsCompleted = false
				break
			}
		}

		if allDepsCompleted {
			parallelTasks = append(parallelTasks, taskID)
		}
	}

	return parallelTasks
}

// GetUpstreamTasks returns all tasks that this task depends on (directly or indirectly)
func (g *Graph) GetUpstreamTasks(taskID string) ([]string, error) {
	if _, exists := g.tasks[taskID]; !exists {
		return nil, fmt.Errorf("task not found: %s", taskID)
	}

	upstream := make(map[string]bool)
	visited := make(map[string]bool)

	var dfs func(string)
	dfs = func(id string) {
		if visited[id] {
			return
		}
		visited[id] = true

		for _, depID := range g.revAdjList[id] {
			upstream[depID] = true
			dfs(depID)
		}
	}

	dfs(taskID)

	result := make([]string, 0, len(upstream))
	for id := range upstream {
		result = append(result, id)
	}

	return result, nil
}

// GetDownstreamTasks returns all tasks that depend on this task (directly or indirectly)
func (g *Graph) GetDownstreamTasks(taskID string) ([]string, error) {
	if _, exists := g.tasks[taskID]; !exists {
		return nil, fmt.Errorf("task not found: %s", taskID)
	}

	downstream := make(map[string]bool)
	visited := make(map[string]bool)

	var dfs func(string)
	dfs = func(id string) {
		if visited[id] {
			return
		}
		visited[id] = true

		for _, depTaskID := range g.adjList[id] {
			downstream[depTaskID] = true
			dfs(depTaskID)
		}
	}

	dfs(taskID)

	result := make([]string, 0, len(downstream))
	for id := range downstream {
		result = append(result, id)
	}

	return result, nil
}

// GetImmediateDependencies returns direct dependencies of a task
func (g *Graph) GetImmediateDependencies(taskID string) ([]string, error) {
	if _, exists := g.tasks[taskID]; !exists {
		return nil, fmt.Errorf("task not found: %s", taskID)
	}

	return g.revAdjList[taskID], nil
}

// GetImmediateDependents returns tasks that directly depend on this task
func (g *Graph) GetImmediateDependents(taskID string) ([]string, error) {
	if _, exists := g.tasks[taskID]; !exists {
		return nil, fmt.Errorf("task not found: %s", taskID)
	}

	return g.adjList[taskID], nil
}

// CriticalPathResult contains the critical path analysis results
type CriticalPathResult struct {
	Path              []string      // Task IDs in the critical path
	TotalDuration     time.Duration // Sum of SLAs/Timeouts on the critical path
	TaskDurations     map[string]time.Duration
	EarliestStart     map[string]time.Duration
	LatestStart       map[string]time.Duration
	Slack             map[string]time.Duration // Latest - Earliest start
	IsCriticalTask    map[string]bool
}

// CalculateCriticalPath computes the critical path for SLA estimation
// The critical path is the longest path through the DAG based on task SLAs/timeouts
func (g *Graph) CalculateCriticalPath() (*CriticalPathResult, error) {
	// Get topological order
	topoOrder, err := g.topologicalSort()
	if err != nil {
		return nil, err
	}

	result := &CriticalPathResult{
		TaskDurations:  make(map[string]time.Duration),
		EarliestStart:  make(map[string]time.Duration),
		LatestStart:    make(map[string]time.Duration),
		Slack:          make(map[string]time.Duration),
		IsCriticalTask: make(map[string]bool),
	}

	// Calculate task durations (use SLA if set, otherwise Timeout, otherwise default 1 minute)
	for taskID, task := range g.tasks {
		if task.SLA > 0 {
			result.TaskDurations[taskID] = task.SLA
		} else if task.Timeout > 0 {
			result.TaskDurations[taskID] = task.Timeout
		} else {
			result.TaskDurations[taskID] = 1 * time.Minute // default
		}
	}

	// Forward pass: Calculate earliest start times
	for _, taskID := range topoOrder {
		maxPredecessorFinish := time.Duration(0)
		for _, depID := range g.revAdjList[taskID] {
			predecessorFinish := result.EarliestStart[depID] + result.TaskDurations[depID]
			if predecessorFinish > maxPredecessorFinish {
				maxPredecessorFinish = predecessorFinish
			}
		}
		result.EarliestStart[taskID] = maxPredecessorFinish
	}

	// Find the project completion time (maximum earliest finish time)
	projectDuration := time.Duration(0)
	var endTasks []string
	for taskID := range g.tasks {
		earliestFinish := result.EarliestStart[taskID] + result.TaskDurations[taskID]
		if earliestFinish > projectDuration {
			projectDuration = earliestFinish
			endTasks = []string{taskID}
		} else if earliestFinish == projectDuration {
			endTasks = append(endTasks, taskID)
		}
	}
	result.TotalDuration = projectDuration

	// Backward pass: Calculate latest start times
	for i := len(topoOrder) - 1; i >= 0; i-- {
		taskID := topoOrder[i]

		// Check if this is an end task (no successors or all successors have later starts)
		successors := g.adjList[taskID]
		if len(successors) == 0 {
			// End task: latest start = project duration - task duration
			result.LatestStart[taskID] = projectDuration - result.TaskDurations[taskID]
		} else {
			// Calculate minimum latest start among successors
			minSuccessorStart := time.Duration(1<<63 - 1) // max duration
			for _, succID := range successors {
				if result.LatestStart[succID] < minSuccessorStart {
					minSuccessorStart = result.LatestStart[succID]
				}
			}
			result.LatestStart[taskID] = minSuccessorStart - result.TaskDurations[taskID]
		}
	}

	// Calculate slack and identify critical tasks
	for taskID := range g.tasks {
		slack := result.LatestStart[taskID] - result.EarliestStart[taskID]
		result.Slack[taskID] = slack

		// A task is critical if it has zero slack
		if slack == 0 {
			result.IsCriticalTask[taskID] = true
		}
	}

	// Build the critical path by following tasks with zero slack
	result.Path = g.buildCriticalPath(result)

	return result, nil
}

// buildCriticalPath constructs the critical path from the analysis results
func (g *Graph) buildCriticalPath(result *CriticalPathResult) []string {
	// Start from tasks with no dependencies and zero slack
	var path []string
	visited := make(map[string]bool)

	// Find the starting critical task
	var startTasks []string
	for taskID := range g.tasks {
		if result.IsCriticalTask[taskID] && len(g.revAdjList[taskID]) == 0 {
			startTasks = append(startTasks, taskID)
		}
	}

	// If no start task found, find any critical task with earliest start = 0
	if len(startTasks) == 0 {
		for taskID := range g.tasks {
			if result.IsCriticalTask[taskID] && result.EarliestStart[taskID] == 0 {
				startTasks = append(startTasks, taskID)
			}
		}
	}

	// DFS through critical tasks
	var dfs func(string)
	dfs = func(taskID string) {
		if visited[taskID] {
			return
		}
		visited[taskID] = true
		path = append(path, taskID)

		// Find the next critical task in the path
		for _, succID := range g.adjList[taskID] {
			if result.IsCriticalTask[succID] && !visited[succID] {
				dfs(succID)
				return
			}
		}
	}

	// Build path from each start task
	for _, startTask := range startTasks {
		if !visited[startTask] {
			dfs(startTask)
		}
	}

	return path
}

// topologicalSort returns tasks in topological order using Kahn's algorithm
func (g *Graph) topologicalSort() ([]string, error) {
	inDegree := make(map[string]int)

	for taskID := range g.tasks {
		inDegree[taskID] = len(g.revAdjList[taskID])
	}

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

		for _, neighbor := range g.adjList[taskID] {
			inDegree[neighbor]--
			if inDegree[neighbor] == 0 {
				queue = append(queue, neighbor)
			}
		}
	}

	if len(result) != len(g.tasks) {
		return nil, fmt.Errorf("cycle detected in DAG")
	}

	return result, nil
}

// GetRootTasks returns all tasks with no dependencies
func (g *Graph) GetRootTasks() []string {
	var roots []string
	for taskID := range g.tasks {
		if len(g.revAdjList[taskID]) == 0 {
			roots = append(roots, taskID)
		}
	}
	return roots
}

// GetLeafTasks returns all tasks that no other task depends on
func (g *Graph) GetLeafTasks() []string {
	var leaves []string
	for taskID := range g.tasks {
		if len(g.adjList[taskID]) == 0 {
			leaves = append(leaves, taskID)
		}
	}
	return leaves
}

// GetTaskCount returns the total number of tasks in the graph
func (g *Graph) GetTaskCount() int {
	return len(g.tasks)
}

// GetTask returns a task by ID
func (g *Graph) GetTask(taskID string) (*models.Task, error) {
	task, exists := g.tasks[taskID]
	if !exists {
		return nil, fmt.Errorf("task not found: %s", taskID)
	}
	return task, nil
}
