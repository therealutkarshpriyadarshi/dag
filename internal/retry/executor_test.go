package retry

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestExecutor_Execute_Success(t *testing.T) {
	config := DefaultConfig()
	executor := NewExecutor(config)

	callCount := 0
	fn := func() error {
		callCount++
		return nil
	}

	err := executor.Execute(context.Background(), fn)
	if err != nil {
		t.Errorf("Execute() error = %v, want nil", err)
	}

	if callCount != 1 {
		t.Errorf("Function called %d times, want 1", callCount)
	}
}

func TestExecutor_Execute_SuccessAfterRetries(t *testing.T) {
	config := NewConfig(5, DefaultExponentialBackoff())
	executor := NewExecutor(config)

	callCount := 0
	fn := func() error {
		callCount++
		if callCount < 3 {
			return errors.New("temporary error")
		}
		return nil
	}

	err := executor.Execute(context.Background(), fn)
	if err != nil {
		t.Errorf("Execute() error = %v, want nil", err)
	}

	if callCount != 3 {
		t.Errorf("Function called %d times, want 3", callCount)
	}
}

func TestExecutor_Execute_AllRetriesFailed(t *testing.T) {
	config := NewConfig(3, NewFixedDelay(10*time.Millisecond, false))
	executor := NewExecutor(config)

	callCount := 0
	fn := func() error {
		callCount++
		return errors.New("persistent error")
	}

	err := executor.Execute(context.Background(), fn)
	if err == nil {
		t.Error("Execute() error = nil, want error")
	}

	if callCount != 3 {
		t.Errorf("Function called %d times, want 3", callCount)
	}
}

func TestExecutor_Execute_ContextCancellation(t *testing.T) {
	config := NewConfig(5, NewFixedDelay(100*time.Millisecond, false))
	executor := NewExecutor(config)

	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()

	callCount := 0
	fn := func() error {
		callCount++
		return errors.New("error")
	}

	err := executor.Execute(ctx, fn)
	if err == nil {
		t.Error("Execute() error = nil, want context error")
	}

	// Should fail after 1-2 attempts due to timeout
	if callCount > 3 {
		t.Errorf("Function called %d times, want <= 3", callCount)
	}
}

func TestExecutor_Execute_RetryCallback(t *testing.T) {
	retryCallbackCalled := false
	config := NewConfig(3, NewFixedDelay(10*time.Millisecond, false))
	config.WithRetryCallback(func(attempt int, err error) {
		retryCallbackCalled = true
	})

	executor := NewExecutor(config)

	callCount := 0
	fn := func() error {
		callCount++
		if callCount < 2 {
			return errors.New("error")
		}
		return nil
	}

	err := executor.Execute(context.Background(), fn)
	if err != nil {
		t.Errorf("Execute() error = %v, want nil", err)
	}

	if !retryCallbackCalled {
		t.Error("RetryCallback was not called")
	}
}

func TestExecutor_Execute_GiveUpCallback(t *testing.T) {
	giveUpCallbackCalled := false
	config := NewConfig(2, NewFixedDelay(10*time.Millisecond, false))
	config.WithGiveUpCallback(func(err error) {
		giveUpCallbackCalled = true
	})

	executor := NewExecutor(config)

	fn := func() error {
		return errors.New("persistent error")
	}

	err := executor.Execute(context.Background(), fn)
	if err == nil {
		t.Error("Execute() error = nil, want error")
	}

	if !giveUpCallbackCalled {
		t.Error("GiveUpCallback was not called")
	}
}

func TestExecuteWithValue_Success(t *testing.T) {
	config := DefaultConfig()

	callCount := 0
	fn := func() (string, error) {
		callCount++
		return "success", nil
	}

	result, err := ExecuteWithValue(context.Background(), config, fn)
	if err != nil {
		t.Errorf("ExecuteWithValue() error = %v, want nil", err)
	}

	if result != "success" {
		t.Errorf("ExecuteWithValue() result = %v, want 'success'", result)
	}

	if callCount != 1 {
		t.Errorf("Function called %d times, want 1", callCount)
	}
}

func TestExecuteWithValue_SuccessAfterRetries(t *testing.T) {
	config := NewConfig(5, NewFixedDelay(10*time.Millisecond, false))

	callCount := 0
	fn := func() (int, error) {
		callCount++
		if callCount < 3 {
			return 0, errors.New("temporary error")
		}
		return 42, nil
	}

	result, err := ExecuteWithValue(context.Background(), config, fn)
	if err != nil {
		t.Errorf("ExecuteWithValue() error = %v, want nil", err)
	}

	if result != 42 {
		t.Errorf("ExecuteWithValue() result = %v, want 42", result)
	}

	if callCount != 3 {
		t.Errorf("Function called %d times, want 3", callCount)
	}
}

func TestExecuteWithValue_AllRetriesFailed(t *testing.T) {
	config := NewConfig(3, NewFixedDelay(10*time.Millisecond, false))

	fn := func() (string, error) {
		return "", errors.New("persistent error")
	}

	result, err := ExecuteWithValue(context.Background(), config, fn)
	if err == nil {
		t.Error("ExecuteWithValue() error = nil, want error")
	}

	if result != "" {
		t.Errorf("ExecuteWithValue() result = %v, want empty string", result)
	}
}

func BenchmarkExecutor_Execute_NoRetries(b *testing.B) {
	config := NewConfig(1, NewNoRetry())
	executor := NewExecutor(config)

	fn := func() error {
		return nil
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		executor.Execute(context.Background(), fn)
	}
}

func BenchmarkExecutor_Execute_WithRetries(b *testing.B) {
	config := NewConfig(3, NewFixedDelay(1*time.Millisecond, false))
	executor := NewExecutor(config)

	callCount := 0
	fn := func() error {
		callCount++
		if callCount%2 == 0 {
			return nil
		}
		return errors.New("error")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		executor.Execute(context.Background(), fn)
	}
}
