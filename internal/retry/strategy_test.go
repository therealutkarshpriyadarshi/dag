package retry

import (
	"testing"
	"time"
)

func TestExponentialBackoff_NextDelay(t *testing.T) {
	tests := []struct {
		name       string
		strategy   *ExponentialBackoff
		attempt    int
		minDelay   time.Duration
		maxDelay   time.Duration
	}{
		{
			name:       "first attempt",
			strategy:   NewExponentialBackoff(1*time.Second, 1*time.Minute, false),
			attempt:    1,
			minDelay:   1 * time.Second,
			maxDelay:   1 * time.Second,
		},
		{
			name:       "second attempt",
			strategy:   NewExponentialBackoff(1*time.Second, 1*time.Minute, false),
			attempt:    2,
			minDelay:   2 * time.Second,
			maxDelay:   2 * time.Second,
		},
		{
			name:       "third attempt",
			strategy:   NewExponentialBackoff(1*time.Second, 1*time.Minute, false),
			attempt:    3,
			minDelay:   4 * time.Second,
			maxDelay:   4 * time.Second,
		},
		{
			name:       "exceeds max delay",
			strategy:   NewExponentialBackoff(1*time.Second, 10*time.Second, false),
			attempt:    10,
			minDelay:   10 * time.Second,
			maxDelay:   10 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			delay := tt.strategy.NextDelay(tt.attempt)
			if delay < tt.minDelay || delay > tt.maxDelay {
				t.Errorf("NextDelay() = %v, want between %v and %v", delay, tt.minDelay, tt.maxDelay)
			}
		})
	}
}

func TestExponentialBackoff_NextDelayWithJitter(t *testing.T) {
	strategy := NewExponentialBackoff(1*time.Second, 1*time.Minute, true)

	// Test multiple attempts to ensure jitter varies
	delays := make(map[time.Duration]bool)
	for i := 0; i < 10; i++ {
		delay := strategy.NextDelay(2)
		delays[delay] = true

		// With jitter, delay should be within ±25% of 2 seconds
		if delay < 1500*time.Millisecond || delay > 2500*time.Millisecond {
			t.Errorf("NextDelay() with jitter = %v, want between 1.5s and 2.5s", delay)
		}
	}

	// We should see some variation (at least 3 different delays)
	if len(delays) < 3 {
		t.Errorf("Jitter not working properly, only got %d unique delays", len(delays))
	}
}

func TestExponentialBackoff_ShouldRetry(t *testing.T) {
	strategy := DefaultExponentialBackoff()

	tests := []struct {
		name        string
		attempt     int
		maxAttempts int
		want        bool
	}{
		{"first attempt", 1, 3, true},
		{"second attempt", 2, 3, true},
		{"last attempt", 3, 3, false},
		{"exceeded attempts", 4, 3, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := strategy.ShouldRetry(tt.attempt, tt.maxAttempts)
			if got != tt.want {
				t.Errorf("ShouldRetry() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLinearBackoff_NextDelay(t *testing.T) {
	strategy := NewLinearBackoff(1*time.Second, 10*time.Second, 1*time.Second, false)

	tests := []struct {
		name     string
		attempt  int
		expected time.Duration
	}{
		{"first attempt", 1, 1 * time.Second},
		{"second attempt", 2, 2 * time.Second},
		{"third attempt", 3, 3 * time.Second},
		{"tenth attempt", 10, 10 * time.Second},
		{"exceeds max", 20, 10 * time.Second},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			delay := strategy.NextDelay(tt.attempt)
			if delay != tt.expected {
				t.Errorf("NextDelay() = %v, want %v", delay, tt.expected)
			}
		})
	}
}

func TestFixedDelay_NextDelay(t *testing.T) {
	strategy := NewFixedDelay(5*time.Second, false)

	for attempt := 1; attempt <= 10; attempt++ {
		delay := strategy.NextDelay(attempt)
		if delay != 5*time.Second {
			t.Errorf("NextDelay(attempt=%d) = %v, want 5s", attempt, delay)
		}
	}
}

func TestFixedDelay_NextDelayWithJitter(t *testing.T) {
	strategy := NewFixedDelay(5*time.Second, true)

	delays := make(map[time.Duration]bool)
	for i := 0; i < 10; i++ {
		delay := strategy.NextDelay(1)
		delays[delay] = true

		// With jitter, delay should be within ±25% of 5 seconds
		if delay < 3750*time.Millisecond || delay > 6250*time.Millisecond {
			t.Errorf("NextDelay() with jitter = %v, want between 3.75s and 6.25s", delay)
		}
	}

	// We should see some variation
	if len(delays) < 3 {
		t.Errorf("Jitter not working properly, only got %d unique delays", len(delays))
	}
}

func TestNoRetry_ShouldRetry(t *testing.T) {
	strategy := NewNoRetry()

	for attempt := 1; attempt <= 10; attempt++ {
		if strategy.ShouldRetry(attempt, 10) {
			t.Errorf("NoRetry.ShouldRetry() should always return false")
		}

		if strategy.NextDelay(attempt) != 0 {
			t.Errorf("NoRetry.NextDelay() should always return 0")
		}
	}
}

func BenchmarkExponentialBackoff_NextDelay(b *testing.B) {
	strategy := DefaultExponentialBackoff()
	for i := 0; i < b.N; i++ {
		strategy.NextDelay(i % 10)
	}
}

func BenchmarkLinearBackoff_NextDelay(b *testing.B) {
	strategy := NewLinearBackoff(1*time.Second, 1*time.Minute, 1*time.Second, true)
	for i := 0; i < b.N; i++ {
		strategy.NextDelay(i % 10)
	}
}
