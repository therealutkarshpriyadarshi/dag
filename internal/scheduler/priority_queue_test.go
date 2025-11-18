package scheduler

import (
	"testing"
	"time"
)

func TestPriorityQueue(t *testing.T) {
	t.Run("NewPriorityQueue creates empty queue", func(t *testing.T) {
		pq := NewPriorityQueue()
		if pq == nil {
			t.Fatal("expected non-nil priority queue")
		}
		if pq.Len() != 0 {
			t.Errorf("expected empty queue, got length %d", pq.Len())
		}
		if !pq.IsEmpty() {
			t.Error("expected queue to be empty")
		}
	})

	t.Run("Push and Pop single item", func(t *testing.T) {
		pq := NewPriorityQueue()
		item := &PriorityQueueItem{
			DAGRunID:      "run-1",
			DAGID:         "dag-1",
			ExecutionDate: time.Now(),
			Priority:      PriorityMedium,
			EnqueuedAt:    time.Now(),
		}

		pq.Push(item)
		if pq.Len() != 1 {
			t.Errorf("expected length 1, got %d", pq.Len())
		}

		popped := pq.Pop()
		if popped == nil {
			t.Fatal("expected non-nil item")
		}
		if popped.DAGRunID != "run-1" {
			t.Errorf("expected DAGRunID run-1, got %s", popped.DAGRunID)
		}
		if pq.Len() != 0 {
			t.Errorf("expected empty queue, got length %d", pq.Len())
		}
	})

	t.Run("Items ordered by priority", func(t *testing.T) {
		pq := NewPriorityQueue()
		now := time.Now()

		// Add items in reverse priority order
		lowItem := &PriorityQueueItem{
			DAGRunID:      "run-low",
			DAGID:         "dag-1",
			ExecutionDate: now,
			Priority:      PriorityLow,
			EnqueuedAt:    now,
		}
		medItem := &PriorityQueueItem{
			DAGRunID:      "run-med",
			DAGID:         "dag-1",
			ExecutionDate: now,
			Priority:      PriorityMedium,
			EnqueuedAt:    now,
		}
		highItem := &PriorityQueueItem{
			DAGRunID:      "run-high",
			DAGID:         "dag-1",
			ExecutionDate: now,
			Priority:      PriorityHigh,
			EnqueuedAt:    now,
		}

		pq.Push(lowItem)
		pq.Push(medItem)
		pq.Push(highItem)

		// Should pop in priority order: high, medium, low
		first := pq.Pop()
		if first.DAGRunID != "run-high" {
			t.Errorf("expected run-high first, got %s", first.DAGRunID)
		}

		second := pq.Pop()
		if second.DAGRunID != "run-med" {
			t.Errorf("expected run-med second, got %s", second.DAGRunID)
		}

		third := pq.Pop()
		if third.DAGRunID != "run-low" {
			t.Errorf("expected run-low third, got %s", third.DAGRunID)
		}
	})

	t.Run("FIFO order for same priority", func(t *testing.T) {
		pq := NewPriorityQueue()
		now := time.Now()

		// Add multiple items with same priority
		item1 := &PriorityQueueItem{
			DAGRunID:      "run-1",
			DAGID:         "dag-1",
			ExecutionDate: now,
			Priority:      PriorityMedium,
			EnqueuedAt:    now,
		}
		item2 := &PriorityQueueItem{
			DAGRunID:      "run-2",
			DAGID:         "dag-1",
			ExecutionDate: now,
			Priority:      PriorityMedium,
			EnqueuedAt:    now.Add(1 * time.Second),
		}

		pq.Push(item1)
		pq.Push(item2)

		// Should pop in FIFO order for same priority
		first := pq.Pop()
		if first.DAGRunID != "run-1" {
			t.Errorf("expected run-1 first, got %s", first.DAGRunID)
		}
	})

	t.Run("Peek without removing", func(t *testing.T) {
		pq := NewPriorityQueue()
		item := &PriorityQueueItem{
			DAGRunID:      "run-1",
			DAGID:         "dag-1",
			ExecutionDate: time.Now(),
			Priority:      PriorityHigh,
			EnqueuedAt:    time.Now(),
		}

		pq.Push(item)

		peeked := pq.Peek()
		if peeked == nil {
			t.Fatal("expected non-nil item")
		}
		if peeked.DAGRunID != "run-1" {
			t.Errorf("expected DAGRunID run-1, got %s", peeked.DAGRunID)
		}

		// Queue should still have the item
		if pq.Len() != 1 {
			t.Errorf("expected length 1 after peek, got %d", pq.Len())
		}
	})

	t.Run("Clear removes all items", func(t *testing.T) {
		pq := NewPriorityQueue()
		now := time.Now()

		for i := 0; i < 5; i++ {
			pq.Push(&PriorityQueueItem{
				DAGRunID:      "run",
				DAGID:         "dag-1",
				ExecutionDate: now,
				Priority:      PriorityMedium,
				EnqueuedAt:    now,
			})
		}

		if pq.Len() != 5 {
			t.Errorf("expected length 5, got %d", pq.Len())
		}

		pq.Clear()

		if pq.Len() != 0 {
			t.Errorf("expected length 0 after clear, got %d", pq.Len())
		}
		if !pq.IsEmpty() {
			t.Error("expected queue to be empty after clear")
		}
	})

	t.Run("Pop from empty queue returns nil", func(t *testing.T) {
		pq := NewPriorityQueue()
		item := pq.Pop()
		if item != nil {
			t.Error("expected nil from empty queue")
		}
	})

	t.Run("Peek empty queue returns nil", func(t *testing.T) {
		pq := NewPriorityQueue()
		item := pq.Peek()
		if item != nil {
			t.Error("expected nil from empty queue")
		}
	})

	t.Run("Items returns copy of queue", func(t *testing.T) {
		pq := NewPriorityQueue()
		now := time.Now()

		pq.Push(&PriorityQueueItem{
			DAGRunID:      "run-1",
			DAGID:         "dag-1",
			ExecutionDate: now,
			Priority:      PriorityHigh,
			EnqueuedAt:    now,
		})

		items := pq.Items()
		if len(items) != 1 {
			t.Errorf("expected 1 item, got %d", len(items))
		}

		// Original queue should still have item
		if pq.Len() != 1 {
			t.Error("Items() should not modify queue")
		}
	})
}

func TestPriorityQueueConcurrency(t *testing.T) {
	t.Run("Concurrent push and pop", func(t *testing.T) {
		pq := NewPriorityQueue()
		now := time.Now()
		done := make(chan bool)

		// Push items concurrently
		for i := 0; i < 100; i++ {
			go func(id int) {
				pq.Push(&PriorityQueueItem{
					DAGRunID:      "run",
					DAGID:         "dag-1",
					ExecutionDate: now,
					Priority:      PriorityMedium,
					EnqueuedAt:    now,
				})
			}(i)
		}

		// Pop items concurrently
		go func() {
			for i := 0; i < 100; i++ {
				pq.Pop()
			}
			done <- true
		}()

		<-done

		// Queue should be empty
		if !pq.IsEmpty() {
			t.Error("expected empty queue after concurrent operations")
		}
	})
}
