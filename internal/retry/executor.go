package retry

import (
	"context"
	"fmt"
	"time"
)

// Executor executes a function with retry logic
type Executor struct {
	config *Config
}

// NewExecutor creates a new retry executor
func NewExecutor(config *Config) *Executor {
	if config == nil {
		config = DefaultConfig()
	}
	return &Executor{
		config: config,
	}
}

// Execute runs a function with retry logic
func (e *Executor) Execute(ctx context.Context, fn func() error) error {
	var lastErr error

	for attempt := 1; attempt <= e.config.MaxAttempts; attempt++ {
		// Execute the function
		err := fn()

		// Success
		if err == nil {
			return nil
		}

		lastErr = err

		// Check if we should retry
		if attempt >= e.config.MaxAttempts {
			// No more retries
			if e.config.GiveUpCallback != nil {
				e.config.GiveUpCallback(err)
			}
			return fmt.Errorf("all retry attempts exhausted after %d tries: %w", attempt, err)
		}

		// Call retry callback if provided
		if e.config.RetryCallback != nil {
			e.config.RetryCallback(attempt, err)
		}

		// Calculate delay
		delay := e.config.CalculateNextDelay(attempt)

		// Wait for delay or context cancellation
		select {
		case <-ctx.Done():
			return fmt.Errorf("retry cancelled: %w", ctx.Err())
		case <-time.After(delay):
			// Continue to next attempt
		}
	}

	return fmt.Errorf("all retry attempts exhausted: %w", lastErr)
}

// ExecuteWithValue runs a function that returns a value with retry logic
func ExecuteWithValue[T any](ctx context.Context, config *Config, fn func() (T, error)) (T, error) {
	var result T
	var lastErr error

	if config == nil {
		config = DefaultConfig()
	}

	for attempt := 1; attempt <= config.MaxAttempts; attempt++ {
		// Execute the function
		val, err := fn()

		// Success
		if err == nil {
			return val, nil
		}

		lastErr = err

		// Check if we should retry
		if attempt >= config.MaxAttempts {
			// No more retries
			if config.GiveUpCallback != nil {
				config.GiveUpCallback(err)
			}
			return result, fmt.Errorf("all retry attempts exhausted after %d tries: %w", attempt, err)
		}

		// Call retry callback if provided
		if config.RetryCallback != nil {
			config.RetryCallback(attempt, err)
		}

		// Calculate delay
		delay := config.CalculateNextDelay(attempt)

		// Wait for delay or context cancellation
		select {
		case <-ctx.Done():
			return result, fmt.Errorf("retry cancelled: %w", ctx.Err())
		case <-time.After(delay):
			// Continue to next attempt
		}
	}

	return result, fmt.Errorf("all retry attempts exhausted: %w", lastErr)
}
