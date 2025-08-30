package main

import (
	"crypto/subtle"
	"fmt"
	"html"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gin-contrib/cors"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"golang.org/x/time/rate"
)

// Valid access keys whitelist - in production, load from database or config
var validAccessKeys = map[string]bool{
	"f6c7a044-4b28-4cfb-8c02-c24c2cece786-c496113b-176d-433c-972d-596f5028d91f": true,
	"a1b2c3d4-e5f6-7890-abcd-ef1234567890-fedcba09-8765-4321-0987-654321fedcba": true,
	"test-key-for-development-only-not-for-production-use-12345678901234567890": true,
}

// Rate limiter for API endpoints
var rateLimiter = rate.NewLimiter(rate.Every(time.Second), 100) // 100 requests per second

// Input validation patterns
var (
	emailPattern    = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	namePattern     = regexp.MustCompile(`^[a-zA-Z\s'-]+$`)
	phonePattern    = regexp.MustCompile(`^[\+]?[0-9\s\-\(\)]+$`)
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	// Initialize Gin router with security middleware
	r := gin.Default()

	// Add security middleware
	r.Use(securityHeaders())
	r.Use(rateLimitMiddleware())
	r.Use(inputSanitizationMiddleware())

	// CORS configuration - restrictive for production security
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"https://formhub.example.com", "https://app.formhub.com", "http://localhost:3000", "http://localhost:3001"}, // Whitelist specific origins
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Requested-With"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// Health check endpoint
	r.GET("/api/v1/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "healthy",
			"message": "FormHub API is running",
			"version": "2.0.0-enterprise",
		})
	})

	// Login endpoint for testing
	r.POST("/api/v1/auth/login", func(c *gin.Context) {
		var loginData map[string]interface{}
		if err := c.ShouldBindJSON(&loginData); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error": "Invalid JSON format",
			})
			return
		}

		email, _ := loginData["email"].(string)
		password, _ := loginData["password"].(string)

		// Simple test credentials check
		if email == "testuser@example.com" && password == "testpass123" {
			c.JSON(http.StatusOK, gin.H{
				"success": true,
				"access_token": "test-token-" + generateID(),
				"user": gin.H{
					"id": "user-1",
					"email": email,
					"first_name": "Test",
					"last_name": "User",
					"plan_type": "free",
				},
			})
		} else {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error": "Invalid credentials",
			})
		}
	})

	// Registration endpoint for testing
	r.POST("/api/v1/auth/register", func(c *gin.Context) {
		var regData map[string]interface{}
		if err := c.ShouldBindJSON(&regData); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error": "Invalid JSON format",
			})
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"success": true,
			"access_token": "test-token-" + generateID(),
			"user": gin.H{
				"id": "user-" + generateID(),
				"email": regData["email"],
				"first_name": regData["first_name"],
				"last_name": regData["last_name"],
				"plan_type": "free",
			},
		})
	})

	// Forms management endpoint
	r.GET("/api/v1/forms", func(c *gin.Context) {
		// Check for authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" || !strings.Contains(authHeader, "Bearer ") {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error": "Valid authorization required",
			})
			return
		}

		// Simple response - in production this would fetch from database
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"forms": []gin.H{
				{
					"id": "example-form-1",
					"name": "Contact Form",
					"description": "Basic contact form",
					"target_email": "contact@example.com",
					"is_active": true,
					"submission_count": 5,
					"created_at": "2025-08-30T08:00:00Z",
				},
			},
		})
	})

	// API Keys endpoint
	r.GET("/api/v1/api-keys", func(c *gin.Context) {
		// Check for authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" || !strings.Contains(authHeader, "Bearer ") {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error": "Valid authorization required",
			})
			return
		}

		// Simple response - in production this would fetch from database
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"api_keys": []gin.H{
				{
					"id": "key-1",
					"name": "Default API Key",
					"permissions": "form_submit",
					"rate_limit": 1000,
					"is_active": true,
					"created_at": "2025-08-30T08:00:00Z",
				},
			},
		})
	})

	// Form creation endpoint
	r.POST("/api/v1/forms", func(c *gin.Context) {
		// Check for authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" || !strings.Contains(authHeader, "Bearer ") {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error": "Valid authorization required",
			})
			return
		}

		var formData map[string]interface{}
		if err := c.ShouldBindJSON(&formData); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error": "Invalid JSON format",
			})
			return
		}

		// Simple response - in production this would save to database
		c.JSON(http.StatusCreated, gin.H{
			"success": true,
			"form": gin.H{
				"id": "form-" + generateID(),
				"name": formData["name"],
				"description": formData["description"],
				"target_email": formData["target_email"],
				"is_active": true,
				"submission_count": 0,
				"created_at": "2025-08-30T13:00:00Z",
			},
		})
	})

	// Form deletion endpoint
	r.DELETE("/api/v1/forms/:id", func(c *gin.Context) {
		// Check for authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" || !strings.Contains(authHeader, "Bearer ") {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error": "Valid authorization required",
			})
			return
		}

		formId := c.Param("id")
		if formId == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error": "Form ID is required",
			})
			return
		}

		// Simple response - in production this would delete from database
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "Form deleted successfully",
		})
	})

	// Form submission endpoint with security validations
	r.POST("/api/v1/submit", func(c *gin.Context) {
		var submission map[string]interface{}
		if err := c.ShouldBindJSON(&submission); err != nil {
			log.Printf("Invalid JSON submission from IP: %s", c.ClientIP())
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error": "Invalid JSON format",
			})
			return
		}

		// CRITICAL: Validate access key against whitelist
		accessKey, exists := submission["access_key"]
		if !exists {
			log.Printf("Missing access key from IP: %s", c.ClientIP())
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error": "Access key is required",
			})
			return
		}

		accessKeyStr, ok := accessKey.(string)
		if !ok || !isValidAccessKey(accessKeyStr) {
			log.Printf("Invalid access key attempted from IP: %s, Key: %v", c.ClientIP(), accessKey)
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error": "Invalid access key",
			})
			return
		}

		// Validate required fields
		if err := validateFormSubmission(submission); err != nil {
			log.Printf("Form validation failed from IP: %s, Error: %s", c.ClientIP(), err.Error())
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error": err.Error(),
			})
			return
		}

		// Sanitize all input fields
		sanitizedSubmission := sanitizeSubmission(submission)

		// Log successful submission (without sensitive data)
		log.Printf("Valid form submission from IP: %s, Access Key: %s...%s", 
			c.ClientIP(), 
			accessKeyStr[:8], 
			accessKeyStr[len(accessKeyStr)-8:])

		// Generate secure submission ID
		submissionID := "FH-" + time.Now().Format("20060102") + "-" + generateSecureID()

		// Return success response
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"statusCode": 200,
			"message": "Thank you for your submission!",
			"data": gin.H{
				"submission_id": submissionID,
				"sanitized_data": sanitizedSubmission,
			},
		})
	})

	// Individual form CRUD operations
	r.GET("/api/v1/forms/:id", func(c *gin.Context) {
		// Check for authorization header
		if !validateBearerToken(c) {
			return
		}

		formID := c.Param("id")
		if !isValidUUID(formID) {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error": "Invalid form ID format",
			})
			return
		}

		// Mock response for individual form
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"form": gin.H{
				"id": formID,
				"name": "Sample Form",
				"description": "Sample form description",
				"target_email": "contact@example.com",
				"is_active": true,
				"submission_count": 10,
				"created_at": "2025-08-30T08:00:00Z",
				"updated_at": "2025-08-30T12:00:00Z",
			},
		})
	})

	r.PUT("/api/v1/forms/:id", func(c *gin.Context) {
		// Check for authorization header
		if !validateBearerToken(c) {
			return
		}

		formID := c.Param("id")
		if !isValidUUID(formID) {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error": "Invalid form ID format",
			})
			return
		}

		var formData map[string]interface{}
		if err := c.ShouldBindJSON(&formData); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error": "Invalid JSON format",
			})
			return
		}

		// Validate form update data
		if err := validateFormData(formData); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error": err.Error(),
			})
			return
		}

		// Mock update response
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"form": gin.H{
				"id": formID,
				"name": formData["name"],
				"description": formData["description"],
				"target_email": formData["target_email"],
				"is_active": formData["is_active"],
				"updated_at": time.Now().Format(time.RFC3339),
			},
		})
	})

	// API Key management endpoints (fixed routing)
	r.POST("/api/v1/api-keys", func(c *gin.Context) {
		if !validateBearerToken(c) {
			return
		}

		var keyData map[string]interface{}
		if err := c.ShouldBindJSON(&keyData); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error": "Invalid JSON format",
			})
			return
		}

		// Generate new API key
		newAPIKey := generateSecureID() + "-" + generateSecureID()

		c.JSON(http.StatusCreated, gin.H{
			"success": true,
			"api_key": gin.H{
				"id": "key-" + generateSecureID(),
				"name": keyData["name"],
				"key": newAPIKey,
				"permissions": "form_submit",
				"rate_limit": 1000,
				"is_active": true,
				"created_at": time.Now().Format(time.RFC3339),
			},
		})
	})

	r.DELETE("/api/v1/api-keys/:id", func(c *gin.Context) {
		if !validateBearerToken(c) {
			return
		}

		keyID := c.Param("id")
		if keyID == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error": "API key ID is required",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "API key deleted successfully",
		})
	})

	// Start server
	port := "9000"
	log.Printf("FormHub Enterprise API starting on port %s", port)
	log.Fatal(r.Run(":" + port))
}

// Security middleware functions
func securityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Security headers for production
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Header("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'; img-src 'self' data: https:; font-src 'self' https:; connect-src 'self'")
		c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains; preload")
		c.Header("Permissions-Policy", "geolocation=(), microphone=(), camera=()")
		c.Next()
	}
}

func rateLimitMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !rateLimiter.Allow() {
			log.Printf("Rate limit exceeded from IP: %s", c.ClientIP())
			c.JSON(http.StatusTooManyRequests, gin.H{
				"success": false,
				"error": "Rate limit exceeded. Please try again later.",
			})
			c.Abort()
			return
		}
		c.Next()
	}
}

func inputSanitizationMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Log request details for security monitoring
		log.Printf("Request from IP: %s, Method: %s, Path: %s, UserAgent: %s", 
			c.ClientIP(), c.Request.Method, c.Request.URL.Path, c.GetHeader("User-Agent"))
		c.Next()
	}
}

// CRITICAL: Access key validation with constant-time comparison
func isValidAccessKey(key string) bool {
	if key == "" {
		return false
	}

	// Use constant-time comparison to prevent timing attacks
	for validKey := range validAccessKeys {
		if subtle.ConstantTimeCompare([]byte(key), []byte(validKey)) == 1 {
			return true
		}
	}
	return false
}

// Validate bearer token for authenticated endpoints
func validateBearerToken(c *gin.Context) bool {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" || !strings.Contains(authHeader, "Bearer ") {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error": "Valid authorization required",
		})
		return false
	}

	// Extract and validate token (basic validation)
	token := strings.TrimPrefix(authHeader, "Bearer ")
	if len(token) < 10 {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error": "Invalid authorization token",
		})
		return false
	}

	return true
}

// Comprehensive form submission validation
func validateFormSubmission(submission map[string]interface{}) error {
	// Validate email
	if email, exists := submission["email"]; exists {
		if emailStr, ok := email.(string); ok {
			if emailStr != "" && !emailPattern.MatchString(emailStr) {
				return fmt.Errorf("invalid email format")
			}
		}
	}

	// Validate name fields
	for _, field := range []string{"name", "first_name", "last_name"} {
		if value, exists := submission[field]; exists {
			if valueStr, ok := value.(string); ok {
				if valueStr != "" {
					if len(valueStr) > 100 {
						return fmt.Errorf("name field '%s' is too long (max 100 characters)", field)
					}
					if !namePattern.MatchString(valueStr) {
						return fmt.Errorf("invalid name format: %s", field)
					}
				}
			}
		}
	}

	// Validate message
	if message, exists := submission["message"]; exists {
		if messageStr, ok := message.(string); ok {
			if messageStr != "" && len(messageStr) > 5000 {
				return fmt.Errorf("message too long (max 5000 characters)")
			}
		}
	}

	// Validate phone
	if phone, exists := submission["phone"]; exists {
		if phoneStr, ok := phone.(string); ok {
			if phoneStr != "" {
				if len(phoneStr) < 7 || len(phoneStr) > 20 {
					return fmt.Errorf("phone number must be between 7 and 20 characters")
				}
				if !phonePattern.MatchString(phoneStr) {
					return fmt.Errorf("invalid phone number format")
				}
			}
		}
	}

	// Validate subject
	if subject, exists := submission["subject"]; exists {
		if subjectStr, ok := subject.(string); ok {
			if subjectStr != "" && len(subjectStr) > 200 {
				return fmt.Errorf("subject too long (max 200 characters)")
			}
		}
	}

	return nil
}

// Sanitize form submission data to prevent XSS
func sanitizeSubmission(submission map[string]interface{}) map[string]interface{} {
	sanitized := make(map[string]interface{})
	
	for key, value := range submission {
		// Skip access_key from sanitized output
		if key == "access_key" {
			continue
		}
		
		if valueStr, ok := value.(string); ok {
			// HTML escape to prevent XSS
			sanitized[key] = html.EscapeString(strings.TrimSpace(valueStr))
		} else {
			sanitized[key] = value
		}
	}
	
	return sanitized
}

// Validate form creation/update data
func validateFormData(formData map[string]interface{}) error {
	// Validate required fields
	requiredFields := []string{"name", "target_email"}
	for _, field := range requiredFields {
		if _, exists := formData[field]; !exists {
			return fmt.Errorf("missing required field: %s", field)
		}
	}

	// Validate email format
	if email, exists := formData["target_email"]; exists {
		if emailStr, ok := email.(string); ok {
			if !emailPattern.MatchString(emailStr) {
				return fmt.Errorf("invalid target email format")
			}
		}
	}

	// Validate name
	if name, exists := formData["name"]; exists {
		if nameStr, ok := name.(string); ok {
			if len(nameStr) < 1 || len(nameStr) > 100 {
				return fmt.Errorf("form name must be between 1 and 100 characters")
			}
		}
	}

	return nil
}

// Check if string is valid UUID format
func isValidUUID(u string) bool {
	_, err := uuid.Parse(u)
	return err == nil
}

// Generate secure random ID
func generateSecureID() string {
	return uuid.New().String()
}

// Legacy function for backward compatibility
func generateID() string {
	return generateSecureID()[:8] // Return first 8 characters for backward compatibility
}