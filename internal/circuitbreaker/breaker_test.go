package circuitbreaker

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestCircuitBreaker_InitialState(t *testing.T) {
	cb := New(DefaultConfig())

	if cb.GetState() != StateClosed {
		t.Errorf("Initial state should be Closed, got %v", cb.GetState())
	}
}

func TestCircuitBreaker_OpenAfterMaxFailures(t *testing.T) {
	config := &Config{
		MaxFailures:         3,
		Timeout:             1 * time.Second,
		HalfOpenMaxRequests: 1,
	}
	cb := New(config)

	// Execute 3 failures
	for i := 0; i < 3; i++ {
		cb.Execute(context.Background(), func() error {
			return errors.New("error")
		})
	}

	// Circuit should now be open
	if cb.GetState() != StateOpen {
		t.Errorf("Circuit should be Open after %d failures, got %v", config.MaxFailures, cb.GetState())
	}

	// Next request should fail immediately
	err := cb.Execute(context.Background(), func() error {
		return nil
	})

	if err != ErrCircuitOpen {
		t.Errorf("Expected ErrCircuitOpen, got %v", err)
	}
}

func TestCircuitBreaker_HalfOpenAfterTimeout(t *testing.T) {
	config := &Config{
		MaxFailures:         2,
		Timeout:             100 * time.Millisecond,
		HalfOpenMaxRequests: 1,
	}
	cb := New(config)

	// Open the circuit
	for i := 0; i < 2; i++ {
		cb.Execute(context.Background(), func() error {
			return errors.New("error")
		})
	}

	if cb.GetState() != StateOpen {
		t.Fatalf("Circuit should be Open")
	}

	// Wait for timeout
	time.Sleep(150 * time.Millisecond)

	// Next request should transition to half-open
	cb.Execute(context.Background(), func() error {
		return nil
	})

	stats := cb.GetStats()
	if stats.State != StateClosed {
		t.Errorf("Circuit should transition to Closed after successful half-open request, got %v", stats.State)
	}
}

func TestCircuitBreaker_HalfOpenToClosedOnSuccess(t *testing.T) {
	config := &Config{
		MaxFailures:         2,
		Timeout:             100 * time.Millisecond,
		HalfOpenMaxRequests: 1,
	}
	cb := New(config)

	// Open the circuit
	for i := 0; i < 2; i++ {
		cb.Execute(context.Background(), func() error {
			return errors.New("error")
		})
	}

	// Wait for timeout
	time.Sleep(150 * time.Millisecond)

	// Successful request in half-open should close the circuit
	err := cb.Execute(context.Background(), func() error {
		return nil
	})

	if err != nil {
		t.Errorf("Expected successful execution, got error: %v", err)
	}

	if cb.GetState() != StateClosed {
		t.Errorf("Circuit should be Closed after successful half-open request, got %v", cb.GetState())
	}
}

func TestCircuitBreaker_HalfOpenToOpenOnFailure(t *testing.T) {
	config := &Config{
		MaxFailures:         2,
		Timeout:             100 * time.Millisecond,
		HalfOpenMaxRequests: 1,
	}
	cb := New(config)

	// Open the circuit
	for i := 0; i < 2; i++ {
		cb.Execute(context.Background(), func() error {
			return errors.New("error")
		})
	}

	// Wait for timeout
	time.Sleep(150 * time.Millisecond)

	// Failed request in half-open should reopen the circuit
	cb.Execute(context.Background(), func() error {
		return errors.New("error")
	})

	if cb.GetState() != StateOpen {
		t.Errorf("Circuit should be Open after failed half-open request, got %v", cb.GetState())
	}
}

func TestCircuitBreaker_TooManyRequestsInHalfOpen(t *testing.T) {
	config := &Config{
		MaxFailures:         2,
		Timeout:             100 * time.Millisecond,
		HalfOpenMaxRequests: 1,
	}
	cb := New(config)

	// Open the circuit
	for i := 0; i < 2; i++ {
		cb.Execute(context.Background(), func() error {
			return errors.New("error")
		})
	}

	if cb.GetState() != StateOpen {
		t.Fatalf("Circuit should be Open")
	}

	// Wait for timeout to transition to half-open
	time.Sleep(150 * time.Millisecond)

	// First beforeRequest should transition to half-open and succeed
	cb.mu.Lock()
	if time.Since(cb.lastFailureTime) >= cb.config.Timeout {
		cb.setState(StateHalfOpen)
		cb.halfOpenRequests = 0
	}
	cb.mu.Unlock()

	// First request increments counter
	err1 := cb.beforeRequest()
	if err1 != nil {
		t.Errorf("First request should be allowed, got error: %v", err1)
	}

	// Second request should fail with TooManyRequests
	err2 := cb.beforeRequest()
	if err2 != ErrTooManyRequests {
		t.Errorf("Expected ErrTooManyRequests, got %v", err2)
	}
}

func TestCircuitBreaker_Reset(t *testing.T) {
	config := &Config{
		MaxFailures:         2,
		Timeout:             1 * time.Second,
		HalfOpenMaxRequests: 1,
	}
	cb := New(config)

	// Open the circuit
	for i := 0; i < 2; i++ {
		cb.Execute(context.Background(), func() error {
			return errors.New("error")
		})
	}

	if cb.GetState() != StateOpen {
		t.Fatalf("Circuit should be Open")
	}

	// Reset the circuit
	cb.Reset()

	if cb.GetState() != StateClosed {
		t.Errorf("Circuit should be Closed after reset, got %v", cb.GetState())
	}

	stats := cb.GetStats()
	if stats.ConsecutiveFailures != 0 {
		t.Errorf("ConsecutiveFailures should be 0 after reset, got %d", stats.ConsecutiveFailures)
	}
}

func TestCircuitBreaker_OnStateChange(t *testing.T) {
	stateChanges := []State{}
	config := &Config{
		MaxFailures:         2,
		Timeout:             1 * time.Second,
		HalfOpenMaxRequests: 1,
		OnStateChange: func(from, to State) {
			stateChanges = append(stateChanges, to)
		},
	}
	cb := New(config)

	// Open the circuit
	for i := 0; i < 2; i++ {
		cb.Execute(context.Background(), func() error {
			return errors.New("error")
		})
	}

	if len(stateChanges) != 1 || stateChanges[0] != StateOpen {
		t.Errorf("Expected state change to Open, got %v", stateChanges)
	}

	// Reset to test more state changes
	cb.Reset()

	if len(stateChanges) != 2 || stateChanges[1] != StateClosed {
		t.Errorf("Expected state change to Closed, got %v", stateChanges)
	}
}

func TestCircuitBreaker_ExecuteWithValue(t *testing.T) {
	cb := New(DefaultConfig())

	result, err := ExecuteWithValue(context.Background(), cb, func() (string, error) {
		return "success", nil
	})

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if result != "success" {
		t.Errorf("Expected result 'success', got %v", result)
	}
}

func TestCircuitBreaker_ExecuteWithValue_CircuitOpen(t *testing.T) {
	config := &Config{
		MaxFailures:         2,
		Timeout:             1 * time.Second,
		HalfOpenMaxRequests: 1,
	}
	cb := New(config)

	// Open the circuit
	for i := 0; i < 2; i++ {
		ExecuteWithValue(context.Background(), cb, func() (string, error) {
			return "", errors.New("error")
		})
	}

	result, err := ExecuteWithValue(context.Background(), cb, func() (string, error) {
		return "should not execute", nil
	})

	if err != ErrCircuitOpen {
		t.Errorf("Expected ErrCircuitOpen, got %v", err)
	}

	if result != "" {
		t.Errorf("Expected empty result, got %v", result)
	}
}

func TestCircuitBreaker_SuccessResetsFailureCount(t *testing.T) {
	config := &Config{
		MaxFailures:         3,
		Timeout:             1 * time.Second,
		HalfOpenMaxRequests: 1,
	}
	cb := New(config)

	// Two failures
	for i := 0; i < 2; i++ {
		cb.Execute(context.Background(), func() error {
			return errors.New("error")
		})
	}

	stats := cb.GetStats()
	if stats.ConsecutiveFailures != 2 {
		t.Errorf("Expected 2 consecutive failures, got %d", stats.ConsecutiveFailures)
	}

	// Success should reset failure count
	cb.Execute(context.Background(), func() error {
		return nil
	})

	stats = cb.GetStats()
	if stats.ConsecutiveFailures != 0 {
		t.Errorf("Expected 0 consecutive failures after success, got %d", stats.ConsecutiveFailures)
	}

	if cb.GetState() != StateClosed {
		t.Errorf("Circuit should remain Closed, got %v", cb.GetState())
	}
}

func BenchmarkCircuitBreaker_Execute_Closed(b *testing.B) {
	cb := New(DefaultConfig())

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cb.Execute(context.Background(), func() error {
			return nil
		})
	}
}

func BenchmarkCircuitBreaker_Execute_Open(b *testing.B) {
	config := &Config{
		MaxFailures:         1,
		Timeout:             1 * time.Hour,
		HalfOpenMaxRequests: 1,
	}
	cb := New(config)

	// Open the circuit
	cb.Execute(context.Background(), func() error {
		return errors.New("error")
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cb.Execute(context.Background(), func() error {
			return nil
		})
	}
}
