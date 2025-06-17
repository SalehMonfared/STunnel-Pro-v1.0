package middleware

import (
	"fmt"
	"net/http"
	"strings"

	"utunnel-pro/internal/models"
	"utunnel-pro/internal/services"
	"utunnel-pro/internal/utils"

	"github.com/gin-gonic/gin"
)

// AuthMiddleware creates authentication middleware
func AuthMiddleware(authService *services.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get token from Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			utils.UnauthorizedResponse(c, "Authorization header is required")
			c.Abort()
			return
		}

		// Check if token has Bearer prefix
		tokenParts := strings.Split(authHeader, " ")
		if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
			utils.UnauthorizedResponse(c, "Invalid authorization header format")
			c.Abort()
			return
		}

		token := tokenParts[1]

		// Validate token
		user, err := authService.ValidateToken(token)
		if err != nil {
			utils.UnauthorizedResponse(c, "Invalid or expired token")
			c.Abort()
			return
		}

		// Set user in context
		c.Set("user", user)
		c.Set("token", token)

		c.Next()
	}
}

// OptionalAuthMiddleware creates optional authentication middleware
func OptionalAuthMiddleware(authService *services.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get token from Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.Next()
			return
		}

		// Check if token has Bearer prefix
		tokenParts := strings.Split(authHeader, " ")
		if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
			c.Next()
			return
		}

		token := tokenParts[1]

		// Validate token
		user, err := authService.ValidateToken(token)
		if err != nil {
			c.Next()
			return
		}

		// Set user in context
		c.Set("user", user)
		c.Set("token", token)

		c.Next()
	}
}

// RequireRoleMiddleware creates role-based authorization middleware
func RequireRoleMiddleware(requiredRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get user from context
		userInterface, exists := c.Get("user")
		if !exists {
			utils.UnauthorizedResponse(c, "Authentication required")
			c.Abort()
			return
		}

		user, ok := userInterface.(*models.User)
		if !ok {
			utils.InternalServerErrorResponse(c, fmt.Errorf("invalid user type in context"))
			c.Abort()
			return
		}

		// Check if user has required role
		userRole := string(user.Role)
		hasRole := false
		for _, role := range requiredRoles {
			if userRole == role {
				hasRole = true
				break
			}
		}

		if !hasRole {
			utils.ForbiddenResponse(c, "Insufficient permissions")
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequirePermissionMiddleware creates permission-based authorization middleware
func RequirePermissionMiddleware(permission string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get user from context
		userInterface, exists := c.Get("user")
		if !exists {
			utils.UnauthorizedResponse(c, "Authentication required")
			c.Abort()
			return
		}

		user, ok := userInterface.(*models.User)
		if !ok {
			utils.InternalServerErrorResponse(c, fmt.Errorf("invalid user type in context"))
			c.Abort()
			return
		}

		// Check if user can perform action
		if !user.CanPerformAction(permission) {
			utils.ForbiddenResponse(c, "Insufficient permissions")
			c.Abort()
			return
		}

		c.Next()
	}
}

// APIKeyMiddleware creates API key authentication middleware
func APIKeyMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get API key from header
		apiKey := c.GetHeader("X-API-Key")
		if apiKey == "" {
			utils.UnauthorizedResponse(c, "API key is required")
			c.Abort()
			return
		}

		// TODO: Validate API key against database
		// For now, we'll just check if it's not empty
		if len(apiKey) < 32 {
			utils.UnauthorizedResponse(c, "Invalid API key")
			c.Abort()
			return
		}

		c.Next()
	}
}

// AdminOnlyMiddleware restricts access to admin users only
func AdminOnlyMiddleware() gin.HandlerFunc {
	return RequireRoleMiddleware("admin")
}

// ModeratorOrAdminMiddleware restricts access to moderator or admin users
func ModeratorOrAdminMiddleware() gin.HandlerFunc {
	return RequireRoleMiddleware("admin", "moderator")
}
