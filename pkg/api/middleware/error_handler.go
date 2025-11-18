package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/therealutkarshpriyadarshi/dag/pkg/api/dto"
)

// ErrorHandler is a middleware that handles errors and panics
func ErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
					Error:   "Internal Server Error",
					Message: "An unexpected error occurred",
					Code:    "INTERNAL_ERROR",
				})
				c.Abort()
			}
		}()

		c.Next()

		// Check if there were any errors
		if len(c.Errors) > 0 {
			err := c.Errors.Last()

			// Determine status code if not already set
			statusCode := c.Writer.Status()
			if statusCode == http.StatusOK {
				statusCode = http.StatusInternalServerError
			}

			c.JSON(statusCode, dto.ErrorResponse{
				Error:   http.StatusText(statusCode),
				Message: err.Error(),
			})
		}
	}
}

// AbortWithError is a helper function to abort with a specific error
func AbortWithError(c *gin.Context, statusCode int, code, message string) {
	c.JSON(statusCode, dto.ErrorResponse{
		Error:   http.StatusText(statusCode),
		Message: message,
		Code:    code,
	})
	c.Abort()
}

// AbortWithErrorDetails is a helper function to abort with error details
func AbortWithErrorDetails(c *gin.Context, statusCode int, code, message string, details map[string]interface{}) {
	c.JSON(statusCode, dto.ErrorResponse{
		Error:   http.StatusText(statusCode),
		Message: message,
		Code:    code,
		Details: details,
	})
	c.Abort()
}
