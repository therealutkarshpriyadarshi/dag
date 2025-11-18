package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/therealutkarshpriyadarshi/dag/pkg/api/middleware"
)

func TestGenerateToken(t *testing.T) {
	config := middleware.DefaultJWTConfig()

	t.Run("successful token generation", func(t *testing.T) {
		token, err := middleware.GenerateToken(config, "user123", "john_doe", []string{"admin", "user"})
		assert.NoError(t, err)
		assert.NotEmpty(t, token)
	})
}

func TestValidateToken(t *testing.T) {
	config := middleware.DefaultJWTConfig()

	t.Run("valid token", func(t *testing.T) {
		token, _ := middleware.GenerateToken(config, "user123", "john_doe", []string{"admin"})

		claims, err := middleware.ValidateToken(config, token)
		assert.NoError(t, err)
		assert.Equal(t, "user123", claims.UserID)
		assert.Equal(t, "john_doe", claims.Username)
		assert.Contains(t, claims.Roles, "admin")
	})

	t.Run("invalid token", func(t *testing.T) {
		claims, err := middleware.ValidateToken(config, "invalid-token")
		assert.Error(t, err)
		assert.Nil(t, claims)
	})
}

func TestJWTAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)
	config := middleware.DefaultJWTConfig()

	t.Run("valid token in header", func(t *testing.T) {
		token, _ := middleware.GenerateToken(config, "user123", "john_doe", []string{"admin"})

		router := gin.Default()
		router.Use(middleware.JWTAuth(config))
		router.GET("/test", func(c *gin.Context) {
			userID, _ := c.Get("user_id")
			c.JSON(200, gin.H{"user_id": userID})
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("missing authorization header", func(t *testing.T) {
		router := gin.Default()
		router.Use(middleware.JWTAuth(config))
		router.GET("/test", func(c *gin.Context) {
			c.JSON(200, gin.H{"status": "ok"})
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("invalid token format", func(t *testing.T) {
		router := gin.Default()
		router.Use(middleware.JWTAuth(config))
		router.GET("/test", func(c *gin.Context) {
			c.JSON(200, gin.H{"status": "ok"})
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("Authorization", "InvalidFormat")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

func TestRequireRole(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("user has required role", func(t *testing.T) {
		router := gin.Default()
		router.Use(func(c *gin.Context) {
			c.Set("roles", []string{"admin", "user"})
			c.Next()
		})
		router.Use(middleware.RequireRole("admin"))
		router.GET("/test", func(c *gin.Context) {
			c.JSON(200, gin.H{"status": "ok"})
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("user does not have required role", func(t *testing.T) {
		router := gin.Default()
		router.Use(func(c *gin.Context) {
			c.Set("roles", []string{"user"})
			c.Next()
		})
		router.Use(middleware.RequireRole("admin"))
		router.GET("/test", func(c *gin.Context) {
			c.JSON(200, gin.H{"status": "ok"})
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
	})
}
