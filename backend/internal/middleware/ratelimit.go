package middleware

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"utunnel-pro/internal/config"
	"utunnel-pro/internal/utils"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
)

// RateLimitMiddleware creates rate limiting middleware
func RateLimitMiddleware(cfg *config.Config) gin.HandlerFunc {
	// Initialize Redis client for rate limiting
	redisClient := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Redis.Host, cfg.Redis.Port),
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})

	return gin.HandlerFunc(func(c *gin.Context) {
		// Get client IP
		clientIP := c.ClientIP()
		
		// Create rate limit key
		key := fmt.Sprintf("rate_limit:%s", clientIP)
		
		ctx := context.Background()
		
		// Get current count
		current, err := redisClient.Get(ctx, key).Int()
		if err != nil && err != redis.Nil {
			// If Redis is down, allow the request
			c.Next()
			return
		}
		
		// Check if limit exceeded
		if current >= cfg.Security.RateLimitRequests {
			utils.TooManyRequestsResponse(c)
			c.Abort()
			return
		}
		
		// Increment counter
		pipe := redisClient.Pipeline()
		pipe.Incr(ctx, key)
		pipe.Expire(ctx, key, cfg.Security.RateLimitWindow)
		_, err = pipe.Exec(ctx)
		
		if err != nil {
			// If Redis operation fails, allow the request
			c.Next()
			return
		}
		
		// Set rate limit headers
		c.Header("X-RateLimit-Limit", strconv.Itoa(cfg.Security.RateLimitRequests))
		c.Header("X-RateLimit-Remaining", strconv.Itoa(cfg.Security.RateLimitRequests-current-1))
		c.Header("X-RateLimit-Reset", strconv.FormatInt(time.Now().Add(cfg.Security.RateLimitWindow).Unix(), 10))
		
		c.Next()
	})
}
