package scheduler

import (
	"container/heap"
	"sync"
	"time"
)

// Priority levels for DAG runs
type Priority int

const (
	PriorityLow    Priority = 0
	PriorityMedium Priority = 1
	PriorityHigh   Priority = 2
)

// PriorityQueueItem represents an item in the priority queue
type PriorityQueueItem struct {
	DAGRunID      string
	DAGID         string
	ExecutionDate time.Time
	Priority      Priority
	EnqueuedAt    time.Time
	index         int // Index in the heap
}

// priorityQueueHeap implements heap.Interface
type priorityQueueHeap []*PriorityQueueItem

func (pq priorityQueueHeap) Len() int { return len(pq) }

func (pq priorityQueueHeap) Less(i, j int) bool {
	// Higher priority comes first
	if pq[i].Priority != pq[j].Priority {
		return pq[i].Priority > pq[j].Priority
	}
	// For same priority, older enqueued items come first (FIFO)
	return pq[i].EnqueuedAt.Before(pq[j].EnqueuedAt)
}

func (pq priorityQueueHeap) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].index = i
	pq[j].index = j
}

func (pq *priorityQueueHeap) Push(x interface{}) {
	n := len(*pq)
	item := x.(*PriorityQueueItem)
	item.index = n
	*pq = append(*pq, item)
}

func (pq *priorityQueueHeap) Pop() interface{} {
	old := *pq
	n := len(old)
	item := old[n-1]
	old[n-1] = nil  // Avoid memory leak
	item.index = -1 // For safety
	*pq = old[0 : n-1]
	return item
}

// PriorityQueue is a thread-safe priority queue for DAG runs
type PriorityQueue struct {
	heap priorityQueueHeap
	mu   sync.Mutex
}

// NewPriorityQueue creates a new priority queue
func NewPriorityQueue() *PriorityQueue {
	pq := &PriorityQueue{
		heap: make(priorityQueueHeap, 0),
	}
	heap.Init(&pq.heap)
	return pq
}

// Push adds an item to the priority queue
func (pq *PriorityQueue) Push(item *PriorityQueueItem) {
	pq.mu.Lock()
	defer pq.mu.Unlock()
	heap.Push(&pq.heap, item)
}

// Pop removes and returns the highest priority item from the queue
func (pq *PriorityQueue) Pop() *PriorityQueueItem {
	pq.mu.Lock()
	defer pq.mu.Unlock()

	if pq.heap.Len() == 0 {
		return nil
	}
	return heap.Pop(&pq.heap).(*PriorityQueueItem)
}

// Peek returns the highest priority item without removing it
func (pq *PriorityQueue) Peek() *PriorityQueueItem {
	pq.mu.Lock()
	defer pq.mu.Unlock()

	if pq.heap.Len() == 0 {
		return nil
	}
	return pq.heap[0]
}

// Len returns the number of items in the queue
func (pq *PriorityQueue) Len() int {
	pq.mu.Lock()
	defer pq.mu.Unlock()
	return pq.heap.Len()
}

// IsEmpty returns true if the queue is empty
func (pq *PriorityQueue) IsEmpty() bool {
	return pq.Len() == 0
}

// Clear removes all items from the queue
func (pq *PriorityQueue) Clear() {
	pq.mu.Lock()
	defer pq.mu.Unlock()
	pq.heap = make(priorityQueueHeap, 0)
	heap.Init(&pq.heap)
}

// Items returns a copy of all items in the queue (for inspection/debugging)
func (pq *PriorityQueue) Items() []*PriorityQueueItem {
	pq.mu.Lock()
	defer pq.mu.Unlock()

	items := make([]*PriorityQueueItem, len(pq.heap))
	copy(items, pq.heap)
	return items
}
