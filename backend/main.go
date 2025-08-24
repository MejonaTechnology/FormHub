package main

import (
	"context"
	"formhub/internal/config"
	"formhub/internal/handlers"
	"formhub/internal/middleware"
	"formhub/internal/services"
	"formhub/pkg/database"
	"formhub/pkg/email"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize database
	db, err := database.NewPostgresDB(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Initialize Redis
	redis, err := database.NewRedisClient(cfg.RedisURL)
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer redis.Close()

	// Initialize email service
	emailService, err := email.NewSMTPService(cfg.SMTPConfig)
	if err != nil {
		log.Fatalf("Failed to initialize email service: %v", err)
	}

	// Initialize services
	userService := services.NewUserService(db, redis)
	formService := services.NewFormService(db, redis)
	submissionService := services.NewSubmissionService(db, redis, emailService)
	authService := services.NewAuthService(db, redis, cfg.JWTSecret)

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(authService)
	formHandler := handlers.NewFormHandler(formService, authService)
	submissionHandler := handlers.NewSubmissionHandler(submissionService, formService, authService)

	// Setup Gin router
	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	// CORS middleware
	router.Use(cors.New(cors.Config{
		AllowOrigins:     cfg.AllowedOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-API-Key"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// Rate limiting middleware
	router.Use(middleware.RateLimit(redis))

	// API routes
	api := router.Group("/api/v1")
	{
		// Public endpoints
		api.POST("/submit", submissionHandler.HandleSubmission)
		
		// Authentication
		auth := api.Group("/auth")
		{
			auth.POST("/register", authHandler.Register)
			auth.POST("/login", authHandler.Login)
			auth.POST("/refresh", authHandler.RefreshToken)
		}

		// Protected endpoints
		protected := api.Group("/")
		protected.Use(middleware.AuthRequired(authService))
		{
			// User management
			protected.GET("/profile", authHandler.GetProfile)
			protected.PUT("/profile", authHandler.UpdateProfile)

			// Forms
			protected.GET("/forms", formHandler.GetForms)
			protected.POST("/forms", formHandler.CreateForm)
			protected.GET("/forms/:id", formHandler.GetForm)
			protected.PUT("/forms/:id", formHandler.UpdateForm)
			protected.DELETE("/forms/:id", formHandler.DeleteForm)

			// Submissions
			protected.GET("/forms/:id/submissions", submissionHandler.GetSubmissions)
			protected.GET("/submissions/:id", submissionHandler.GetSubmission)
			protected.DELETE("/submissions/:id", submissionHandler.DeleteSubmission)

			// API Keys
			protected.GET("/api-keys", authHandler.GetAPIKeys)
			protected.POST("/api-keys", authHandler.CreateAPIKey)
			protected.DELETE("/api-keys/:id", authHandler.DeleteAPIKey)
		}
	}

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "healthy",
			"version": "1.0.0",
			"time":    time.Now().UTC(),
		})
	})

	// Start server
	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: router,
	}

	// Graceful server shutdown
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	log.Printf("FormHub API server started on port %s", cfg.Port)

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	log.Println("Server exited")
}