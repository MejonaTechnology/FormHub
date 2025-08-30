package middleware

import (
	"fmt"
	"formhub/internal/services"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

type SecurityMiddleware struct {
	securityService *services.SecurityService
	rateLimiters    map[string]*rate.Limiter
}

func NewSecurityMiddleware(securityService *services.SecurityService) *SecurityMiddleware {
	return &SecurityMiddleware{
		securityService: securityService,
		rateLimiters:    make(map[string]*rate.Limiter),
	}
}

// SecurityHeaders adds essential security headers to all responses
func (sm *SecurityMiddleware) SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Content Security Policy
		c.Header("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline' 'unsafe-eval'; style-src 'self' 'unsafe-inline'; img-src 'self' data: https:; font-src 'self' data:; connect-src 'self'; media-src 'self'; object-src 'none'; child-src 'none'; worker-src 'none'; frame-ancestors 'none'; form-action 'self'; base-uri 'self'")
		
		// Prevent MIME type sniffing
		c.Header("X-Content-Type-Options", "nosniff")
		
		// Enable XSS protection
		c.Header("X-XSS-Protection", "1; mode=block")
		
		// Prevent clickjacking
		c.Header("X-Frame-Options", "DENY")
		
		// Force HTTPS in production
		if gin.Mode() == gin.ReleaseMode {
			c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains; preload")
		}
		
		// Referrer policy
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		
		// Permissions policy
		c.Header("Permissions-Policy", "geolocation=(), microphone=(), camera=(), payment=()")
		
		// Hide server info
		c.Header("Server", "FormHub")
		
		c.Next()
	}
}

// RateLimiter implements IP-based rate limiting
func (sm *SecurityMiddleware) RateLimiter(requestsPerMinute int) gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		
		// Get or create rate limiter for this IP
		limiter, exists := sm.rateLimiters[ip]
		if !exists {
			limiter = rate.NewLimiter(rate.Every(time.Minute/time.Duration(requestsPerMinute)), requestsPerMinute)
			sm.rateLimiters[ip] = limiter
		}
		
		if !limiter.Allow() {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":   "Rate limit exceeded",
				"message": "Too many requests from your IP address. Please try again later.",
				"retry_after": 60,
			})
			c.Abort()
			return
		}
		
		c.Next()
	}
}

// InputSanitization validates and sanitizes incoming request data
func (sm *SecurityMiddleware) InputSanitization() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check for suspicious patterns in URL
		if sm.containsSuspiciousPatterns(c.Request.URL.Path) {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "Invalid request",
				"message": "Request contains suspicious patterns",
			})
			c.Abort()
			return
		}
		
		// Check for suspicious headers
		userAgent := c.GetHeader("User-Agent")
		if sm.isSuspiciousUserAgent(userAgent) {
			c.JSON(http.StatusForbidden, gin.H{
				"error":   "Access denied",
				"message": "Suspicious user agent detected",
			})
			c.Abort()
			return
		}
		
		// Check for SQL injection in query parameters
		for key, values := range c.Request.URL.Query() {
			for _, value := range values {
				if sm.detectSQLInjection(value) {
					c.JSON(http.StatusBadRequest, gin.H{
						"error":   "Invalid request",
						"message": "Malicious content detected in query parameters",
					})
					c.Abort()
					return
				}
			}
			
			// Also check parameter names
			if sm.detectSQLInjection(key) {
				c.JSON(http.StatusBadRequest, gin.H{
					"error":   "Invalid request",
					"message": "Malicious content detected in parameter names",
				})
				c.Abort()
				return
			}
		}
		
		c.Next()
	}
}

// FileUploadSecurity applies security measures specifically for file uploads
func (sm *SecurityMiddleware) FileUploadSecurity() gin.HandlerFunc {
	return func(c *gin.Context) {
		contentType := c.GetHeader("Content-Type")
		
		// Only apply to multipart uploads
		if strings.Contains(contentType, "multipart/form-data") {
			// Limit upload size (32MB for regular uploads, 100MB for bulk)
			maxSize := int64(32 << 20) // 32MB
			if strings.Contains(c.Request.URL.Path, "bulk") {
				maxSize = int64(100 << 20) // 100MB
			}
			
			c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxSize)
			
			// Set timeout for uploads
			c.Request = c.Request.WithContext(c.Request.Context())
		}
		
		c.Next()
	}
}

// AntiCSRF provides CSRF protection for state-changing operations
func (sm *SecurityMiddleware) AntiCSRF() gin.HandlerFunc {
	return func(c *gin.Context) {
		method := c.Request.Method
		
		// Only check CSRF for state-changing methods
		if method == "POST" || method == "PUT" || method == "DELETE" || method == "PATCH" {
			// Check for CSRF token in header or form
			token := c.GetHeader("X-CSRF-Token")
			if token == "" {
				token = c.PostForm("_token")
			}
			
			// For now, we'll skip CSRF validation if no token is provided
			// In production, implement proper CSRF token validation
			if token == "" && gin.Mode() == gin.ReleaseMode {
				c.JSON(http.StatusForbidden, gin.H{
					"error":   "CSRF token missing",
					"message": "CSRF token is required for this operation",
				})
				c.Abort()
				return
			}
		}
		
		c.Next()
	}
}

// IPWhitelist restricts access to specific IP addresses (for admin endpoints)
func (sm *SecurityMiddleware) IPWhitelist(allowedIPs []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		clientIP := c.ClientIP()
		
		allowed := false
		for _, ip := range allowedIPs {
			if clientIP == ip {
				allowed = true
				break
			}
		}
		
		if !allowed {
			c.JSON(http.StatusForbidden, gin.H{
				"error":   "Access denied",
				"message": "Your IP address is not authorized to access this resource",
			})
			c.Abort()
			return
		}
		
		c.Next()
	}
}

// GeoblockingMiddleware blocks requests from specific countries/regions
func (sm *SecurityMiddleware) GeoblockingMiddleware(blockedCountries []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// This would integrate with a geolocation service in production
		// For now, it's a placeholder that allows all requests
		c.Next()
	}
}

// BotDetection detects and blocks obvious bot traffic
func (sm *SecurityMiddleware) BotDetection() gin.HandlerFunc {
	return func(c *gin.Context) {
		userAgent := c.GetHeader("User-Agent")
		
		// Block obvious bots
		botPatterns := []string{
			"bot", "crawler", "spider", "scraper", "automated", 
			"curl/", "wget/", "python-requests", "python-urllib",
		}
		
		lowerUA := strings.ToLower(userAgent)
		for _, pattern := range botPatterns {
			if strings.Contains(lowerUA, pattern) {
				c.JSON(http.StatusForbidden, gin.H{
					"error":   "Access denied",
					"message": "Automated requests are not allowed",
				})
				c.Abort()
				return
			}
		}
		
		// Check for missing or suspicious user agents
		if userAgent == "" || len(userAgent) < 10 {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "Invalid request",
				"message": "Valid user agent is required",
			})
			c.Abort()
			return
		}
		
		c.Next()
	}
}

// Helper methods

func (sm *SecurityMiddleware) containsSuspiciousPatterns(input string) bool {
	patterns := []string{
		"../", "..\\", "/etc/passwd", "/proc/", "cmd.exe", "powershell",
		"<script", "javascript:", "vbscript:", "data:text/html",
	}
	
	lowerInput := strings.ToLower(input)
	for _, pattern := range patterns {
		if strings.Contains(lowerInput, pattern) {
			return true
		}
	}
	return false
}

func (sm *SecurityMiddleware) isSuspiciousUserAgent(userAgent string) bool {
	if userAgent == "" || len(userAgent) < 5 {
		return true
	}
	
	// Check for null bytes or control characters
	for _, char := range userAgent {
		if char < 32 && char != 9 && char != 10 && char != 13 {
			return true
		}
	}
	
	return false
}

func (sm *SecurityMiddleware) detectSQLInjection(input string) bool {
	patterns := []string{
		"'", "\"", ";--", "/*", "*/", " or ", " and ", " union ", " select ",
		" insert ", " update ", " delete ", " drop ", " create ", " alter ",
		"xp_", "sp_", "exec ", "execute ", "script", "javascript:",
	}
	
	lowerInput := strings.ToLower(input)
	for _, pattern := range patterns {
		if strings.Contains(lowerInput, pattern) {
			return true
		}
	}
	return false
}

// CleanupOldRateLimiters removes old rate limiters to prevent memory leaks
func (sm *SecurityMiddleware) CleanupOldRateLimiters() {
	// This should be called periodically (e.g., every hour)
	// Remove rate limiters that haven't been used recently
	// Implementation would track last use time and clean up accordingly
}

// RequestLogging logs all requests for security monitoring
func (sm *SecurityMiddleware) RequestLogging() gin.HandlerFunc {
	return gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		// Custom log format that includes security-relevant information
		return fmt.Sprintf("%s - [%s] \"%s %s %s %d %s\" %s \"%s\" \"%s\"\n",
			param.ClientIP,
			param.TimeStamp.Format("02/Jan/2006:15:04:05 -0700"),
			param.Method,
			param.Path,
			param.Request.Proto,
			param.StatusCode,
			param.Latency,
			param.Request.UserAgent(),
			param.Request.Referer(),
			param.ErrorMessage,
		)
	})
}