package middleware

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// JWTConfig holds JWT configuration
type JWTConfig struct {
	SecretKey     []byte
	Expiration    time.Duration
	RefreshWindow time.Duration
}

// Claims represents JWT claims
type Claims struct {
	UserID   string   `json:"user_id"`
	Username string   `json:"username"`
	Roles    []string `json:"roles"`
	jwt.RegisteredClaims
}

// DefaultJWTConfig returns default JWT configuration
func DefaultJWTConfig() *JWTConfig {
	return &JWTConfig{
		SecretKey:     []byte("your-secret-key-change-in-production"), // TODO: Load from env
		Expiration:    24 * time.Hour,
		RefreshWindow: 1 * time.Hour,
	}
}

// GenerateToken generates a new JWT token
func GenerateToken(config *JWTConfig, userID, username string, roles []string) (string, error) {
	claims := Claims{
		UserID:   userID,
		Username: username,
		Roles:    roles,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(config.Expiration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "dag-orchestrator",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(config.SecretKey)
}

// ValidateToken validates a JWT token
func ValidateToken(config *JWTConfig, tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return config.SecretKey, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("invalid token")
}

// JWTAuth returns a middleware that validates JWT tokens
func JWTAuth(config *JWTConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			AbortWithError(c, http.StatusUnauthorized, "NO_TOKEN", "Authorization header required")
			return
		}

		// Extract token from "Bearer <token>"
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			AbortWithError(c, http.StatusUnauthorized, "INVALID_TOKEN_FORMAT", "Authorization header format must be 'Bearer {token}'")
			return
		}

		tokenString := parts[1]
		claims, err := ValidateToken(config, tokenString)
		if err != nil {
			AbortWithError(c, http.StatusUnauthorized, "INVALID_TOKEN", err.Error())
			return
		}

		// Store claims in context
		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("roles", claims.Roles)

		c.Next()
	}
}

// RequireRole returns a middleware that checks for specific roles
func RequireRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userRoles, exists := c.Get("roles")
		if !exists {
			AbortWithError(c, http.StatusForbidden, "NO_ROLES", "User roles not found")
			return
		}

		rolesList, ok := userRoles.([]string)
		if !ok {
			AbortWithError(c, http.StatusForbidden, "INVALID_ROLES", "Invalid user roles format")
			return
		}

		// Check if user has any of the required roles
		hasRole := false
		for _, requiredRole := range roles {
			for _, userRole := range rolesList {
				if userRole == requiredRole {
					hasRole = true
					break
				}
			}
			if hasRole {
				break
			}
		}

		if !hasRole {
			AbortWithError(c, http.StatusForbidden, "INSUFFICIENT_PERMISSIONS",
				fmt.Sprintf("Required roles: %v", roles))
			return
		}

		c.Next()
	}
}

// OptionalAuth is a middleware that validates JWT if present but doesn't require it
func OptionalAuth(config *JWTConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.Next()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) == 2 && parts[0] == "Bearer" {
			tokenString := parts[1]
			claims, err := ValidateToken(config, tokenString)
			if err == nil {
				c.Set("user_id", claims.UserID)
				c.Set("username", claims.Username)
				c.Set("roles", claims.Roles)
			}
		}

		c.Next()
	}
}
