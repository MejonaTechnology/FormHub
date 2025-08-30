package main

import (
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gin-contrib/cors"
	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	// Initialize Gin router
	r := gin.Default()

	// CORS configuration
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"*"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
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

	// Form submission endpoint
	r.POST("/api/v1/submit", func(c *gin.Context) {
		var submission map[string]interface{}
		if err := c.ShouldBindJSON(&submission); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error": "Invalid JSON format",
			})
			return
		}

		// Check for access_key
		accessKey, exists := submission["access_key"]
		if !exists {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error": "Access key is required",
			})
			return
		}

		// Simple form submission response
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"statusCode": 200,
			"message": "Thank you for your submission!",
			"data": gin.H{
				"access_key": accessKey,
				"submission_id": "FH-20250830-" + generateID(),
			},
		})
	})

	// Start server
	port := "9000"
	log.Printf("FormHub Enterprise API starting on port %s", port)
	log.Fatal(r.Run(":" + port))
}

func generateID() string {
	return "123456" // Simple static ID for now
}