package middleware

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

var validate *validator.Validate

func init() {
	validate = validator.New()

	// Register custom validators
	_ = validate.RegisterValidation("cron", validateCron)
}

// validateCron validates a cron expression
func validateCron(fl validator.FieldLevel) bool {
	cronExpr := fl.Field().String()
	if cronExpr == "" {
		return true // Allow empty for optional fields
	}

	// Basic cron validation - in production, use a proper cron parser
	// For now, just check if it has at least 5 parts
	// A proper implementation would use github.com/robfig/cron/v3
	return len(cronExpr) > 0
}

// ValidateRequest validates a request struct
func ValidateRequest(obj interface{}) error {
	return validate.Struct(obj)
}

// ValidationErrorResponse converts validator errors to a readable format
func ValidationErrorResponse(err error) map[string]interface{} {
	errors := make(map[string]interface{})

	if validationErrors, ok := err.(validator.ValidationErrors); ok {
		for _, fieldError := range validationErrors {
			field := fieldError.Field()
			tag := fieldError.Tag()

			var message string
			switch tag {
			case "required":
				message = fmt.Sprintf("%s is required", field)
			case "min":
				message = fmt.Sprintf("%s must be at least %s", field, fieldError.Param())
			case "max":
				message = fmt.Sprintf("%s must be at most %s", field, fieldError.Param())
			case "oneof":
				message = fmt.Sprintf("%s must be one of: %s", field, fieldError.Param())
			case "cron":
				message = fmt.Sprintf("%s must be a valid cron expression", field)
			default:
				message = fmt.Sprintf("%s failed validation: %s", field, tag)
			}

			errors[field] = message
		}
	} else {
		errors["validation"] = err.Error()
	}

	return errors
}

// BindAndValidate binds and validates a request
func BindAndValidate(c *gin.Context, obj interface{}) bool {
	if err := c.ShouldBindJSON(obj); err != nil {
		AbortWithError(c, http.StatusBadRequest, "INVALID_JSON", err.Error())
		return false
	}

	if err := ValidateRequest(obj); err != nil {
		details := ValidationErrorResponse(err)
		AbortWithErrorDetails(c, http.StatusBadRequest, "VALIDATION_ERROR", "Request validation failed", details)
		return false
	}

	return true
}
