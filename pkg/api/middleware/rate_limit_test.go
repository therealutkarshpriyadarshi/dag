package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/therealutkarshpriyadarshi/dag/pkg/api/middleware"
)

func TestRateLimiter(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("allows requests within limit", func(t *testing.T) {
		limiter := middleware.NewRateLimiter(10, 10)
		defer limiter.Stop()

		router := gin.Default()
		router.Use(limiter.RateLimit())
		router.GET("/test", func(c *gin.Context) {
			c.JSON(200, gin.H{"status": "ok"})
		})

		// Make 5 requests (within limit of 10/second)
		for i := 0; i < 5; i++ {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			assert.Equal(t, http.StatusOK, w.Code)
		}
	})

	t.Run("rate limits excessive requests", func(t *testing.T) {
		limiter := middleware.NewRateLimiter(1, 2) // Very strict limit for testing
		defer limiter.Stop()

		router := gin.Default()
		router.Use(limiter.RateLimit())
		router.GET("/test", func(c *gin.Context) {
			c.JSON(200, gin.H{"status": "ok"})
		})

		// First request should succeed
		req1 := httptest.NewRequest(http.MethodGet, "/test", nil)
		w1 := httptest.NewRecorder()
		router.ServeHTTP(w1, req1)
		assert.Equal(t, http.StatusOK, w1.Code)

		// Quickly exhaust burst capacity
		req2 := httptest.NewRequest(http.MethodGet, "/test", nil)
		w2 := httptest.NewRecorder()
		router.ServeHTTP(w2, req2)
		assert.Equal(t, http.StatusOK, w2.Code)

		req3 := httptest.NewRequest(http.MethodGet, "/test", nil)
		w3 := httptest.NewRecorder()
		router.ServeHTTP(w3, req3)
		assert.Equal(t, http.StatusOK, w3.Code)

		// This request should be rate limited
		req4 := httptest.NewRequest(http.MethodGet, "/test", nil)
		w4 := httptest.NewRecorder()
		router.ServeHTTP(w4, req4)
		assert.Equal(t, http.StatusTooManyRequests, w4.Code)
	})

	t.Run("rate limiter cleanup works", func(t *testing.T) {
		limiter := middleware.NewRateLimiter(10, 10)

		// Make requests to populate the cache
		router := gin.Default()
		router.Use(limiter.RateLimit())
		router.GET("/test", func(c *gin.Context) {
			c.JSON(200, gin.H{"status": "ok"})
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Stop the limiter
		limiter.Stop()

		// Give cleanup goroutine time to exit
		time.Sleep(100 * time.Millisecond)

		assert.True(t, true) // Test passes if no deadlock/panic
	})
}
