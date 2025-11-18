package circuitbreaker

import (
	"context"
	"errors"
	"sync"
	"time"
)

var (
	// ErrCircuitOpen is returned when the circuit breaker is open
	ErrCircuitOpen = errors.New("circuit breaker is open")

	// ErrTooManyRequests is returned when too many requests are made in half-open state
	ErrTooManyRequests = errors.New("too many requests")
)

// State represents the current state of the circuit breaker
type State int

const (
	// StateClosed allows all requests through
	StateClosed State = iota

	// StateOpen rejects all requests
	StateOpen

	// StateHalfOpen allows limited requests through to test recovery
	StateHalfOpen
)

// String returns the string representation of the state
func (s State) String() string {
	switch s {
	case StateClosed:
		return "closed"
	case StateOpen:
		return "open"
	case StateHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

// Config holds configuration for the circuit breaker
type Config struct {
	// MaxFailures is the number of consecutive failures before opening the circuit
	MaxFailures int

	// Timeout is how long the circuit stays open before transitioning to half-open
	Timeout time.Duration

	// HalfOpenMaxRequests is the max number of requests allowed in half-open state
	HalfOpenMaxRequests int

	// OnStateChange is called when the circuit breaker changes state
	OnStateChange func(from, to State)

	// IsSuccessful determines if a result is considered successful
	// If nil, any non-error result is considered successful
	IsSuccessful func(err error) bool
}

// DefaultConfig returns a config with sensible defaults
func DefaultConfig() *Config {
	return &Config{
		MaxFailures:         5,
		Timeout:             60 * time.Second,
		HalfOpenMaxRequests: 1,
		IsSuccessful: func(err error) bool {
			return err == nil
		},
	}
}

// CircuitBreaker implements the circuit breaker pattern
type CircuitBreaker struct {
	config *Config
	state  State
	mu     sync.RWMutex

	// Counters
	consecutiveFailures int
	consecutiveSuccesses int
	halfOpenRequests    int

	// Timing
	lastFailureTime time.Time
	lastStateChange time.Time
}

// New creates a new circuit breaker
func New(config *Config) *CircuitBreaker {
	if config == nil {
		config = DefaultConfig()
	}

	if config.IsSuccessful == nil {
		config.IsSuccessful = func(err error) bool {
			return err == nil
		}
	}

	return &CircuitBreaker{
		config:          config,
		state:           StateClosed,
		lastStateChange: time.Now(),
	}
}

// Execute runs a function with circuit breaker protection
func (cb *CircuitBreaker) Execute(ctx context.Context, fn func() error) error {
	// Check if we can execute
	if err := cb.beforeRequest(); err != nil {
		return err
	}

	// Execute the function
	err := fn()

	// Record the result
	cb.afterRequest(err)

	return err
}

// ExecuteWithValue runs a function that returns a value with circuit breaker protection
func ExecuteWithValue[T any](ctx context.Context, cb *CircuitBreaker, fn func() (T, error)) (T, error) {
	var result T

	// Check if we can execute
	if err := cb.beforeRequest(); err != nil {
		return result, err
	}

	// Execute the function
	result, err := fn()

	// Record the result
	cb.afterRequest(err)

	return result, err
}

// beforeRequest checks if a request can proceed
func (cb *CircuitBreaker) beforeRequest() error {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case StateClosed:
		return nil

	case StateOpen:
		// Check if we should transition to half-open
		if time.Since(cb.lastFailureTime) >= cb.config.Timeout {
			cb.setState(StateHalfOpen)
			cb.halfOpenRequests = 0
			return nil
		}
		return ErrCircuitOpen

	case StateHalfOpen:
		// Check if we've exceeded half-open request limit
		if cb.halfOpenRequests >= cb.config.HalfOpenMaxRequests {
			return ErrTooManyRequests
		}
		cb.halfOpenRequests++
		return nil

	default:
		return ErrCircuitOpen
	}
}

// afterRequest records the result of a request
func (cb *CircuitBreaker) afterRequest(err error) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	success := cb.config.IsSuccessful(err)

	switch cb.state {
	case StateClosed:
		if success {
			cb.consecutiveFailures = 0
		} else {
			cb.consecutiveFailures++
			cb.lastFailureTime = time.Now()

			if cb.consecutiveFailures >= cb.config.MaxFailures {
				cb.setState(StateOpen)
			}
		}

	case StateHalfOpen:
		if success {
			cb.consecutiveSuccesses++
			// After one success in half-open, transition to closed
			cb.setState(StateClosed)
			cb.consecutiveFailures = 0
			cb.consecutiveSuccesses = 0
		} else {
			// Any failure in half-open goes back to open
			cb.setState(StateOpen)
			cb.lastFailureTime = time.Now()
		}
	}
}

// setState changes the circuit breaker state
func (cb *CircuitBreaker) setState(newState State) {
	if cb.state == newState {
		return
	}

	oldState := cb.state
	cb.state = newState
	cb.lastStateChange = time.Now()

	if cb.config.OnStateChange != nil {
		cb.config.OnStateChange(oldState, newState)
	}
}

// GetState returns the current state
func (cb *CircuitBreaker) GetState() State {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// Reset resets the circuit breaker to closed state
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.setState(StateClosed)
	cb.consecutiveFailures = 0
	cb.consecutiveSuccesses = 0
	cb.halfOpenRequests = 0
}

// GetStats returns current statistics
func (cb *CircuitBreaker) GetStats() Stats {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	return Stats{
		State:                cb.state,
		ConsecutiveFailures:  cb.consecutiveFailures,
		ConsecutiveSuccesses: cb.consecutiveSuccesses,
		LastFailureTime:      cb.lastFailureTime,
		LastStateChange:      cb.lastStateChange,
	}
}

// Stats holds circuit breaker statistics
type Stats struct {
	State                State
	ConsecutiveFailures  int
	ConsecutiveSuccesses int
	LastFailureTime      time.Time
	LastStateChange      time.Time
}
