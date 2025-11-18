package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// RateLimiter stores rate limiters for each client
type RateLimiter struct {
	clients map[string]*rate.Limiter
	mu      sync.RWMutex

	// Rate limit configuration
	requestsPerSecond rate.Limit
	burst             int

	// Cleanup ticker
	ticker *time.Ticker
	done   chan bool
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(requestsPerSecond float64, burst int) *RateLimiter {
	rl := &RateLimiter{
		clients:           make(map[string]*rate.Limiter),
		requestsPerSecond: rate.Limit(requestsPerSecond),
		burst:             burst,
		ticker:            time.NewTicker(5 * time.Minute),
		done:              make(chan bool),
	}

	// Start cleanup goroutine
	go rl.cleanupClients()

	return rl
}

// cleanupClients removes old limiters to prevent memory leaks
func (rl *RateLimiter) cleanupClients() {
	for {
		select {
		case <-rl.ticker.C:
			rl.mu.Lock()
			// In production, you'd track last access time and remove old entries
			// For now, we'll just clear the map periodically if it gets too large
			if len(rl.clients) > 10000 {
				rl.clients = make(map[string]*rate.Limiter)
			}
			rl.mu.Unlock()
		case <-rl.done:
			return
		}
	}
}

// Stop stops the rate limiter cleanup goroutine
func (rl *RateLimiter) Stop() {
	rl.ticker.Stop()
	rl.done <- true
}

// getLimiter returns the rate limiter for a client
func (rl *RateLimiter) getLimiter(clientID string) *rate.Limiter {
	rl.mu.RLock()
	limiter, exists := rl.clients[clientID]
	rl.mu.RUnlock()

	if !exists {
		rl.mu.Lock()
		limiter = rate.NewLimiter(rl.requestsPerSecond, rl.burst)
		rl.clients[clientID] = limiter
		rl.mu.Unlock()
	}

	return limiter
}

// RateLimit returns a middleware that rate limits requests
func (rl *RateLimiter) RateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Use client IP as identifier (in production, use user ID if authenticated)
		clientID := c.ClientIP()

		limiter := rl.getLimiter(clientID)

		if !limiter.Allow() {
			AbortWithError(c, http.StatusTooManyRequests, "RATE_LIMIT_EXCEEDED",
				"Too many requests. Please try again later.")
			return
		}

		c.Next()
	}
}

// GlobalRateLimiter is the default rate limiter instance
var GlobalRateLimiter = NewRateLimiter(10, 20) // 10 requests per second, burst of 20
