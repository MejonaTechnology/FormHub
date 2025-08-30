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