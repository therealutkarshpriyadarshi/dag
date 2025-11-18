package middleware_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/therealutkarshpriyadarshi/dag/pkg/api/middleware"
)

type TestRequest struct {
	Name  string `json:"name" validate:"required,min=3,max=50"`
	Email string `json:"email" validate:"required,email"`
	Age   int    `json:"age" validate:"min=0,max=150"`
}

func TestValidateRequest(t *testing.T) {
	t.Run("valid request", func(t *testing.T) {
		req := TestRequest{
			Name:  "John Doe",
			Email: "john@example.com",
			Age:   30,
		}

		err := middleware.ValidateRequest(req)
		assert.NoError(t, err)
	})

	t.Run("missing required field", func(t *testing.T) {
		req := TestRequest{
			Email: "john@example.com",
			Age:   30,
		}

		err := middleware.ValidateRequest(req)
		assert.Error(t, err)
	})

	t.Run("field too short", func(t *testing.T) {
		req := TestRequest{
			Name:  "Jo",
			Email: "john@example.com",
			Age:   30,
		}

		err := middleware.ValidateRequest(req)
		assert.Error(t, err)
	})

	t.Run("invalid email", func(t *testing.T) {
		req := TestRequest{
			Name:  "John Doe",
			Email: "invalid-email",
			Age:   30,
		}

		err := middleware.ValidateRequest(req)
		assert.Error(t, err)
	})

	t.Run("value out of range", func(t *testing.T) {
		req := TestRequest{
			Name:  "John Doe",
			Email: "john@example.com",
			Age:   200,
		}

		err := middleware.ValidateRequest(req)
		assert.Error(t, err)
	})
}

func TestBindAndValidate(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("valid request", func(t *testing.T) {
		req := TestRequest{
			Name:  "John Doe",
			Email: "john@example.com",
			Age:   30,
		}

		body, _ := json.Marshal(req)
		httpReq := httptest.NewRequest(http.MethodPost, "/test", bytes.NewReader(body))
		httpReq.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httpReq

		var boundReq TestRequest
		result := middleware.BindAndValidate(c, &boundReq)

		assert.True(t, result)
		assert.Equal(t, "John Doe", boundReq.Name)
	})

	t.Run("invalid JSON", func(t *testing.T) {
		httpReq := httptest.NewRequest(http.MethodPost, "/test", bytes.NewReader([]byte("invalid json")))
		httpReq.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httpReq

		var boundReq TestRequest
		result := middleware.BindAndValidate(c, &boundReq)

		assert.False(t, result)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("validation failure", func(t *testing.T) {
		req := TestRequest{
			Name:  "Jo", // Too short
			Email: "john@example.com",
			Age:   30,
		}

		body, _ := json.Marshal(req)
		httpReq := httptest.NewRequest(http.MethodPost, "/test", bytes.NewReader(body))
		httpReq.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httpReq

		var boundReq TestRequest
		result := middleware.BindAndValidate(c, &boundReq)

		assert.False(t, result)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestValidationErrorResponse(t *testing.T) {
	t.Run("formats validation errors", func(t *testing.T) {
		req := TestRequest{
			Name: "", // Required but empty
			Age:  200, // Out of range
		}

		err := middleware.ValidateRequest(req)
		assert.Error(t, err)

		errors := middleware.ValidationErrorResponse(err)
		assert.NotNil(t, errors)
		assert.Contains(t, errors, "Name")
	})
}
