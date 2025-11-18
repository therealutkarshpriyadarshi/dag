package retry

import (
	"time"
)

// Config holds retry configuration for a task
type Config struct {
	// MaxAttempts is the maximum number of retry attempts (including initial attempt)
	MaxAttempts int

	// Strategy is the retry strategy to use
	Strategy Strategy

	// RetryOnErrorCodes specifies which error codes should trigger a retry
	// Empty means retry on all errors
	RetryOnErrorCodes []string

	// RetryCallback is called when a retry is triggered
	RetryCallback func(attempt int, err error)

	// GiveUpCallback is called when all retries are exhausted
	GiveUpCallback func(err error)
}

// DefaultConfig returns a retry config with sensible defaults
func DefaultConfig() *Config {
	return &Config{
		MaxAttempts:       3,
		Strategy:          DefaultExponentialBackoff(),
		RetryOnErrorCodes: []string{}, // Retry on all errors
	}
}

// NewConfig creates a new retry config
func NewConfig(maxAttempts int, strategy Strategy) *Config {
	return &Config{
		MaxAttempts:       maxAttempts,
		Strategy:          strategy,
		RetryOnErrorCodes: []string{},
	}
}

// WithRetryOnErrorCodes sets the error codes to retry on
func (c *Config) WithRetryOnErrorCodes(codes ...string) *Config {
	c.RetryOnErrorCodes = codes
	return c
}

// WithRetryCallback sets the retry callback
func (c *Config) WithRetryCallback(callback func(attempt int, err error)) *Config {
	c.RetryCallback = callback
	return c
}

// WithGiveUpCallback sets the give up callback
func (c *Config) WithGiveUpCallback(callback func(err error)) *Config {
	c.GiveUpCallback = callback
	return c
}

// ShouldRetryError checks if an error should trigger a retry
func (c *Config) ShouldRetryError(errorCode string) bool {
	// If no specific error codes are specified, retry all errors
	if len(c.RetryOnErrorCodes) == 0 {
		return true
	}

	// Check if error code is in the retry list
	for _, code := range c.RetryOnErrorCodes {
		if code == errorCode {
			return true
		}
	}

	return false
}

// CalculateNextDelay calculates the delay before the next retry
func (c *Config) CalculateNextDelay(attempt int) time.Duration {
	if c.Strategy == nil {
		return 0
	}
	return c.Strategy.NextDelay(attempt)
}

// ShouldRetry checks if a retry should be attempted
func (c *Config) ShouldRetry(attempt int) bool {
	if c.Strategy == nil {
		return false
	}
	return c.Strategy.ShouldRetry(attempt, c.MaxAttempts)
}
