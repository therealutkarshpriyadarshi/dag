package scheduler

import (
	"context"
	"testing"
)

func TestConcurrencyManager(t *testing.T) {
	ctx := context.Background()

	t.Run("NewConcurrencyManager with default config", func(t *testing.T) {
		cm := NewConcurrencyManager(ctx, nil)
		if cm == nil {
			t.Fatal("expected non-nil concurrency manager")
		}
		if cm.config.MaxGlobalConcurrency != 100 {
			t.Errorf("expected default max global concurrency 100, got %d", cm.config.MaxGlobalConcurrency)
		}
	})

	t.Run("Global concurrency limits", func(t *testing.T) {
		config := &ConcurrencyConfig{
			MaxGlobalConcurrency: 2,
		}
		cm := NewConcurrencyManager(ctx, config)

		// Should allow scheduling up to limit
		if !cm.CanScheduleGlobal() {
			t.Error("expected to allow first schedule")
		}

		cm.IncrementGlobal()
		if !cm.CanScheduleGlobal() {
			t.Error("expected to allow second schedule")
		}

		cm.IncrementGlobal()
		if cm.CanScheduleGlobal() {
			t.Error("expected to block third schedule")
		}

		// Decrement should allow scheduling again
		cm.DecrementGlobal()
		if !cm.CanScheduleGlobal() {
			t.Error("expected to allow schedule after decrement")
		}
	})

	t.Run("DAG-level concurrency limits", func(t *testing.T) {
		config := &ConcurrencyConfig{
			DefaultDAGConcurrency: 2,
		}
		cm := NewConcurrencyManager(ctx, config)
		dagID := "test-dag"

		// Should allow scheduling up to limit
		if !cm.CanScheduleDAG(dagID) {
			t.Error("expected to allow first DAG schedule")
		}

		cm.IncrementDAG(dagID)
		cm.IncrementDAG(dagID)

		if cm.CanScheduleDAG(dagID) {
			t.Error("expected to block third DAG schedule")
		}

		// Decrement should allow scheduling again
		cm.DecrementDAG(dagID)
		if !cm.CanScheduleDAG(dagID) {
			t.Error("expected to allow DAG schedule after decrement")
		}
	})

	t.Run("Custom DAG limits", func(t *testing.T) {
		cm := NewConcurrencyManager(ctx, &ConcurrencyConfig{
			DefaultDAGConcurrency: 2,
		})

		dagID := "special-dag"
		cm.SetDAGLimit(dagID, 5)

		limit := cm.GetDAGLimit(dagID)
		if limit != 5 {
			t.Errorf("expected DAG limit 5, got %d", limit)
		}
	})

	t.Run("Pool management", func(t *testing.T) {
		cm := NewConcurrencyManager(ctx, nil)
		poolName := "test-pool"

		// Create pool
		cm.CreatePool(poolName, 2)

		// Should be able to acquire slots
		if err := cm.AcquirePool(poolName); err != nil {
			t.Errorf("expected to acquire pool slot: %v", err)
		}

		if cm.GetPoolCount(poolName) != 1 {
			t.Errorf("expected pool count 1, got %d", cm.GetPoolCount(poolName))
		}

		// Acquire second slot
		if err := cm.AcquirePool(poolName); err != nil {
			t.Errorf("expected to acquire second pool slot: %v", err)
		}

		// Should fail to acquire third slot
		if err := cm.AcquirePool(poolName); err == nil {
			t.Error("expected error when pool is full")
		}

		// Release slot
		cm.ReleasePool(poolName)
		if cm.GetPoolCount(poolName) != 1 {
			t.Errorf("expected pool count 1 after release, got %d", cm.GetPoolCount(poolName))
		}

		// Should be able to acquire again
		if err := cm.AcquirePool(poolName); err != nil {
			t.Errorf("expected to acquire pool slot after release: %v", err)
		}
	})

	t.Run("Pool acquire/release checks", func(t *testing.T) {
		cm := NewConcurrencyManager(ctx, nil)
		poolName := "db-pool"
		cm.CreatePool(poolName, 3)

		// Check can acquire
		if !cm.CanAcquirePool(poolName) {
			t.Error("expected to be able to acquire pool slot")
		}

		// Acquire all slots
		cm.AcquirePool(poolName)
		cm.AcquirePool(poolName)
		cm.AcquirePool(poolName)

		// Should not be able to acquire more
		if cm.CanAcquirePool(poolName) {
			t.Error("expected pool to be full")
		}
	})

	t.Run("Non-existent pool allows unlimited", func(t *testing.T) {
		cm := NewConcurrencyManager(ctx, nil)

		if !cm.CanAcquirePool("nonexistent-pool") {
			t.Error("expected non-existent pool to allow acquisition")
		}
	})

	t.Run("Delete pool", func(t *testing.T) {
		cm := NewConcurrencyManager(ctx, nil)
		poolName := "temp-pool"
		cm.CreatePool(poolName, 5)

		if _, exists := cm.config.Pools[poolName]; !exists {
			t.Error("expected pool to exist")
		}

		cm.DeletePool(poolName)

		if _, exists := cm.config.Pools[poolName]; exists {
			t.Error("expected pool to be deleted")
		}
	})

	t.Run("Reset clears all counters", func(t *testing.T) {
		cm := NewConcurrencyManager(ctx, nil)

		cm.IncrementGlobal()
		cm.IncrementGlobal()
		cm.IncrementDAG("dag1")
		cm.IncrementDAG("dag2")

		cm.CreatePool("pool1", 10)
		cm.AcquirePool("pool1")

		cm.Reset()

		if cm.GetGlobalCount() != 0 {
			t.Errorf("expected global count 0 after reset, got %d", cm.GetGlobalCount())
		}

		if cm.GetDAGCount("dag1") != 0 {
			t.Errorf("expected DAG count 0 after reset, got %d", cm.GetDAGCount("dag1"))
		}

		if cm.GetPoolCount("pool1") != 0 {
			t.Errorf("expected pool count 0 after reset, got %d", cm.GetPoolCount("pool1"))
		}
	})
}
