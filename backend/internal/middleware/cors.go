package middleware

import (
	"utunnel-pro/internal/config"

	"github.com/gin-gonic/gin"
)

// CORSMiddleware creates CORS middleware
func CORSMiddleware(cfg *config.Config) gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		// Set CORS headers
		for _, origin := range cfg.Security.CORSAllowedOrigins {
			c.Header("Access-Control-Allow-Origin", origin)
		}

		c.Header("Access-Control-Allow-Credentials", "true")
		
		// Set allowed methods
		methods := ""
		for i, method := range cfg.Security.CORSAllowedMethods {
			if i > 0 {
				methods += ", "
			}
			methods += method
		}
		c.Header("Access-Control-Allow-Methods", methods)

		// Set allowed headers
		headers := ""
		for i, header := range cfg.Security.CORSAllowedHeaders {
			if i > 0 {
				headers += ", "
			}
			headers += header
		}
		c.Header("Access-Control-Allow-Headers", headers)

		// Handle preflight requests
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})
}
