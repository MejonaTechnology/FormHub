package main

import (
	"log"
	"net/http"

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

	// Forms management endpoint
	r.GET("/api/v1/forms", func(c *gin.Context) {
		// Check for authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error": "Authorization header required",
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
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error": "Authorization header required",
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