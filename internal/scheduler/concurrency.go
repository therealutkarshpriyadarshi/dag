package scheduler

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// PoolConfig defines configuration for a task pool
type PoolConfig struct {
	Name  string
	Slots int
}

// ConcurrencyConfig holds concurrency control configuration
type ConcurrencyConfig struct {
	// MaxGlobalConcurrency is the maximum number of concurrent DAG runs globally
	MaxGlobalConcurrency int

	// DefaultDAGConcurrency is the default maximum concurrent runs per DAG
	DefaultDAGConcurrency int

	// Pools defines named resource pools for task-level concurrency
	Pools map[string]int // pool name -> max slots

	// RedisClient for distributed locking (optional)
	RedisClient *redis.Client

	// LockTTL is the TTL for Redis locks
	LockTTL time.Duration
}

// ConcurrencyManager manages concurrency limits at various levels
type ConcurrencyManager struct {
	config              *ConcurrencyConfig
	globalCount         int
	dagCounts           map[string]int // dagID -> current count
	dagLimits           map[string]int // dagID -> max concurrent
	poolCounts          map[string]int // pool name -> current count
	mu                  sync.RWMutex
	redis               *redis.Client
	ctx                 context.Context
}

// NewConcurrencyManager creates a new concurrency manager
func NewConcurrencyManager(ctx context.Context, config *ConcurrencyConfig) *ConcurrencyManager {
	if config == nil {
		config = &ConcurrencyConfig{
			MaxGlobalConcurrency:  100,
			DefaultDAGConcurrency: 16,
			Pools:                 make(map[string]int),
			LockTTL:               30 * time.Second,
		}
	}

	return &ConcurrencyManager{
		config:     config,
		globalCount: 0,
		dagCounts:  make(map[string]int),
		dagLimits:  make(map[string]int),
		poolCounts: make(map[string]int),
		redis:      config.RedisClient,
		ctx:        ctx,
	}
}

// CanScheduleGlobal checks if a new DAG run can be scheduled globally
func (cm *ConcurrencyManager) CanScheduleGlobal() bool {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.globalCount < cm.config.MaxGlobalConcurrency
}

// CanScheduleDAG checks if a new run can be scheduled for a specific DAG
func (cm *ConcurrencyManager) CanScheduleDAG(dagID string) bool {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	currentCount := cm.dagCounts[dagID]
	limit := cm.getDAGLimit(dagID)
	return currentCount < limit
}

// CanAcquirePool checks if a slot is available in the specified pool
func (cm *ConcurrencyManager) CanAcquirePool(poolName string) bool {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	maxSlots, exists := cm.config.Pools[poolName]
	if !exists {
		return true // If pool doesn't exist, allow unlimited
	}

	currentCount := cm.poolCounts[poolName]
	return currentCount < maxSlots
}

// IncrementGlobal increments the global concurrency counter
func (cm *ConcurrencyManager) IncrementGlobal() {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.globalCount++
}

// DecrementGlobal decrements the global concurrency counter
func (cm *ConcurrencyManager) DecrementGlobal() {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	if cm.globalCount > 0 {
		cm.globalCount--
	}
}

// IncrementDAG increments the concurrency counter for a specific DAG
func (cm *ConcurrencyManager) IncrementDAG(dagID string) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.dagCounts[dagID]++
}

// DecrementDAG decrements the concurrency counter for a specific DAG
func (cm *ConcurrencyManager) DecrementDAG(dagID string) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	if count, exists := cm.dagCounts[dagID]; exists && count > 0 {
		cm.dagCounts[dagID]--
	}
}

// AcquirePool acquires a slot in the specified pool
func (cm *ConcurrencyManager) AcquirePool(poolName string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	maxSlots, exists := cm.config.Pools[poolName]
	if !exists {
		return fmt.Errorf("pool %s does not exist", poolName)
	}

	currentCount := cm.poolCounts[poolName]
	if currentCount >= maxSlots {
		return fmt.Errorf("pool %s is full (%d/%d)", poolName, currentCount, maxSlots)
	}

	cm.poolCounts[poolName]++
	return nil
}

// ReleasePool releases a slot in the specified pool
func (cm *ConcurrencyManager) ReleasePool(poolName string) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if count, exists := cm.poolCounts[poolName]; exists && count > 0 {
		cm.poolCounts[poolName]--
	}
}

// SetDAGLimit sets the concurrency limit for a specific DAG
func (cm *ConcurrencyManager) SetDAGLimit(dagID string, limit int) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.dagLimits[dagID] = limit
}

// GetDAGLimit returns the concurrency limit for a specific DAG
func (cm *ConcurrencyManager) GetDAGLimit(dagID string) int {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.getDAGLimit(dagID)
}

// getDAGLimit is the internal (non-locking) version
func (cm *ConcurrencyManager) getDAGLimit(dagID string) int {
	if limit, exists := cm.dagLimits[dagID]; exists {
		return limit
	}
	return cm.config.DefaultDAGConcurrency
}

// GetGlobalCount returns the current global concurrency count
func (cm *ConcurrencyManager) GetGlobalCount() int {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.globalCount
}

// GetDAGCount returns the current concurrency count for a specific DAG
func (cm *ConcurrencyManager) GetDAGCount(dagID string) int {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.dagCounts[dagID]
}

// GetPoolCount returns the current count for a specific pool
func (cm *ConcurrencyManager) GetPoolCount(poolName string) int {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.poolCounts[poolName]
}

// CreatePool creates a new resource pool
func (cm *ConcurrencyManager) CreatePool(poolName string, maxSlots int) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.config.Pools[poolName] = maxSlots
	cm.poolCounts[poolName] = 0
}

// DeletePool removes a resource pool
func (cm *ConcurrencyManager) DeletePool(poolName string) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	delete(cm.config.Pools, poolName)
	delete(cm.poolCounts, poolName)
}

// AcquireDistributedLock acquires a distributed lock using Redis (if configured)
func (cm *ConcurrencyManager) AcquireDistributedLock(key string) (bool, error) {
	if cm.redis == nil {
		return false, fmt.Errorf("redis client not configured")
	}

	// Try to set key with NX (only if not exists) and expiration
	result, err := cm.redis.SetNX(cm.ctx, key, "locked", cm.config.LockTTL).Result()
	if err != nil {
		return false, fmt.Errorf("failed to acquire lock: %w", err)
	}

	return result, nil
}

// ReleaseDistributedLock releases a distributed lock using Redis
func (cm *ConcurrencyManager) ReleaseDistributedLock(key string) error {
	if cm.redis == nil {
		return fmt.Errorf("redis client not configured")
	}

	_, err := cm.redis.Del(cm.ctx, key).Result()
	if err != nil {
		return fmt.Errorf("failed to release lock: %w", err)
	}

	return nil
}

// IncrementDistributedCounter increments a counter in Redis for distributed concurrency tracking
func (cm *ConcurrencyManager) IncrementDistributedCounter(key string) (int64, error) {
	if cm.redis == nil {
		return 0, fmt.Errorf("redis client not configured")
	}

	val, err := cm.redis.Incr(cm.ctx, key).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to increment counter: %w", err)
	}

	// Set expiration if this is a new key
	cm.redis.Expire(cm.ctx, key, 24*time.Hour)

	return val, nil
}

// DecrementDistributedCounter decrements a counter in Redis
func (cm *ConcurrencyManager) DecrementDistributedCounter(key string) (int64, error) {
	if cm.redis == nil {
		return 0, fmt.Errorf("redis client not configured")
	}

	val, err := cm.redis.Decr(cm.ctx, key).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to decrement counter: %w", err)
	}

	return val, nil
}

// GetDistributedCounter gets the current value of a counter in Redis
func (cm *ConcurrencyManager) GetDistributedCounter(key string) (int64, error) {
	if cm.redis == nil {
		return 0, fmt.Errorf("redis client not configured")
	}

	val, err := cm.redis.Get(cm.ctx, key).Int64()
	if err == redis.Nil {
		return 0, nil // Key doesn't exist, return 0
	}
	if err != nil {
		return 0, fmt.Errorf("failed to get counter: %w", err)
	}

	return val, nil
}

// Reset resets all concurrency counters
func (cm *ConcurrencyManager) Reset() {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	cm.globalCount = 0
	cm.dagCounts = make(map[string]int)
	cm.poolCounts = make(map[string]int)
}
