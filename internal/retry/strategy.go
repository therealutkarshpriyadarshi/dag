package retry

import (
	"math"
	"math/rand"
	"time"
)

// Strategy defines the interface for retry strategies
type Strategy interface {
	// NextDelay calculates the delay before the next retry attempt
	NextDelay(attempt int) time.Duration

	// ShouldRetry determines if a retry should be attempted
	ShouldRetry(attempt, maxAttempts int) bool
}

// ExponentialBackoff implements exponential backoff retry strategy
type ExponentialBackoff struct {
	BaseDelay  time.Duration
	MaxDelay   time.Duration
	Multiplier float64
	Jitter     bool
}

// NewExponentialBackoff creates a new exponential backoff strategy
func NewExponentialBackoff(baseDelay, maxDelay time.Duration, jitter bool) *ExponentialBackoff {
	return &ExponentialBackoff{
		BaseDelay:  baseDelay,
		MaxDelay:   maxDelay,
		Multiplier: 2.0,
		Jitter:     jitter,
	}
}

// DefaultExponentialBackoff returns an exponential backoff with sensible defaults
func DefaultExponentialBackoff() *ExponentialBackoff {
	return NewExponentialBackoff(1*time.Second, 5*time.Minute, true)
}

// NextDelay calculates the next delay using exponential backoff
func (e *ExponentialBackoff) NextDelay(attempt int) time.Duration {
	// Calculate base delay: baseDelay * multiplier^(attempt-1)
	delay := float64(e.BaseDelay) * math.Pow(e.Multiplier, float64(attempt-1))

	// Cap at max delay
	if delay > float64(e.MaxDelay) {
		delay = float64(e.MaxDelay)
	}

	// Add jitter if enabled (randomize ±25%)
	if e.Jitter {
		jitterFactor := 0.75 + (rand.Float64() * 0.5) // Random between 0.75 and 1.25
		delay = delay * jitterFactor
	}

	return time.Duration(delay)
}

// ShouldRetry checks if we should retry based on attempt count
func (e *ExponentialBackoff) ShouldRetry(attempt, maxAttempts int) bool {
	return attempt < maxAttempts
}

// LinearBackoff implements linear backoff retry strategy
type LinearBackoff struct {
	BaseDelay time.Duration
	MaxDelay  time.Duration
	Increment time.Duration
	Jitter    bool
}

// NewLinearBackoff creates a new linear backoff strategy
func NewLinearBackoff(baseDelay, maxDelay, increment time.Duration, jitter bool) *LinearBackoff {
	return &LinearBackoff{
		BaseDelay: baseDelay,
		MaxDelay:  maxDelay,
		Increment: increment,
		Jitter:    jitter,
	}
}

// NextDelay calculates the next delay using linear backoff
func (l *LinearBackoff) NextDelay(attempt int) time.Duration {
	// Calculate delay: baseDelay + (increment * attempt)
	delay := l.BaseDelay + (l.Increment * time.Duration(attempt-1))

	// Cap at max delay
	if delay > l.MaxDelay {
		delay = l.MaxDelay
	}

	// Add jitter if enabled (randomize ±25%)
	if l.Jitter {
		jitterFactor := 0.75 + (rand.Float64() * 0.5)
		delay = time.Duration(float64(delay) * jitterFactor)
	}

	return delay
}

// ShouldRetry checks if we should retry based on attempt count
func (l *LinearBackoff) ShouldRetry(attempt, maxAttempts int) bool {
	return attempt < maxAttempts
}

// FixedDelay implements fixed delay retry strategy
type FixedDelay struct {
	Delay  time.Duration
	Jitter bool
}

// NewFixedDelay creates a new fixed delay strategy
func NewFixedDelay(delay time.Duration, jitter bool) *FixedDelay {
	return &FixedDelay{
		Delay:  delay,
		Jitter: jitter,
	}
}

// NextDelay returns a fixed delay
func (f *FixedDelay) NextDelay(attempt int) time.Duration {
	delay := f.Delay

	// Add jitter if enabled (randomize ±25%)
	if f.Jitter {
		jitterFactor := 0.75 + (rand.Float64() * 0.5)
		delay = time.Duration(float64(delay) * jitterFactor)
	}

	return delay
}

// ShouldRetry checks if we should retry based on attempt count
func (f *FixedDelay) ShouldRetry(attempt, maxAttempts int) bool {
	return attempt < maxAttempts
}

// NoRetry implements a no-retry strategy
type NoRetry struct{}

// NewNoRetry creates a new no-retry strategy
func NewNoRetry() *NoRetry {
	return &NoRetry{}
}

// NextDelay always returns 0
func (n *NoRetry) NextDelay(attempt int) time.Duration {
	return 0
}

// ShouldRetry always returns false
func (n *NoRetry) ShouldRetry(attempt, maxAttempts int) bool {
	return false
}
