package middleware

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

func RateLimit(redis *redis.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		if redis == nil {
			// If Redis is not available, skip rate limiting
			c.Next()
			return
		}

		// Get client IP
		clientIP := c.ClientIP()
		
		// Create rate limit key
		key := "rate_limit:ip:" + clientIP
		
		ctx := context.Background()
		
		// Check current count
		current, err := redis.Get(ctx, key).Int()
		if err != nil && !errors.Is(err, redis.Nil) {
			// If Redis fails, allow the request
			c.Next()
			return
		}
		
		// Rate limit: 100 requests per minute per IP
		limit := 100
		window := time.Minute
		
		if current >= limit {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "Rate limit exceeded. Please try again later.",
				"retry_after": "60 seconds",
			})
			c.Abort()
			return
		}
		
		// Increment counter
		pipe := redis.Pipeline()
		pipe.Incr(ctx, key)
		pipe.Expire(ctx, key, window)
		_, err = pipe.Exec(ctx)
		
		if err != nil {
			// If Redis fails, allow the request
			c.Next()
			return
		}
		
		c.Next()
	}
}