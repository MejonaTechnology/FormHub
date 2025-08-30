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
	db, err := database.NewMySQLDB(cfg.DatabaseURL)
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

	// Initialize core services
	formService := services.NewFormService(db, redis)
	submissionService := services.NewSubmissionService(db, redis, emailService)
	authService := services.NewAuthService(db, redis, cfg.JWTSecret)
	
	// Initialize analytics services
	analyticsService := services.NewAnalyticsService(db, redis)
	submissionLifecycleService := services.NewSubmissionLifecycleService(db, redis, analyticsService)
	geoIPService := services.NewGeoIPService(db, redis, cfg.GeoIPAPIKey) // Add GeoIP API key to config
	abTestingService := services.NewABTestingService(db, redis, analyticsService)
	cacheService := services.NewCacheService(redis)
	realTimeService := services.NewRealTimeService(db, redis, analyticsService)
	monitoringService := services.NewMonitoringService(db, redis, analyticsService, realTimeService)
	
	// Initialize reporting service (requires email service interface)
	reportingService := services.NewReportingService(db, redis, analyticsService, emailService, geoIPService)
	
	// Initialize spam protection services
	securityService := services.NewSecurityService(db, redis, "./quarantine")
	spamService := services.NewSpamProtectionService(db, redis, securityService)
	mlClassifier := services.NewNaiveBayesSpamClassifier(db, redis)
	behavioralAnalyzer := services.NewBehavioralAnalyzer(db, redis)
	
	// Initialize enhanced webhook system
	enhancedWebhookService := services.NewEnhancedWebhookService(db, redis)
	integrationManager := services.NewIntegrationManager(db, redis)
	
	// Keep legacy webhook service for compatibility
	webhookService := services.NewWebhookService(db, redis)

	// Initialize email template services
	emailTemplateService := services.NewEmailTemplateService(db)
	emailProviderService := services.NewEmailProviderService(db)
	emailAnalyticsService := services.NewEmailAnalyticsService(db)
	emailQueueService := services.NewEmailQueueService(db, emailProviderService, emailAnalyticsService)
	emailAutoresponderService := services.NewEmailAutoresponderService(db, emailTemplateService, emailProviderService, emailQueueService)
	templateBuilderService := services.NewTemplateBuilderService(db)
	abTestingService := services.NewEmailABTestingService(db, emailTemplateService, emailAnalyticsService, emailQueueService)

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(authService)
	formHandler := handlers.NewFormHandler(formService, authService)
	submissionHandler := handlers.NewSubmissionHandler(submissionService, formService, authService)
	spamAdminHandler := handlers.NewSpamAdminHandler(spamService, behavioralAnalyzer, mlClassifier, webhookService, authService)
	emailTemplateHandler := handlers.NewEmailTemplateHandler(
		emailTemplateService, 
		emailProviderService, 
		emailAutoresponderService, 
		emailQueueService, 
		emailAnalyticsService, 
		templateBuilderService, 
		abTestingService,
	)
	
	// Initialize analytics handler
	analyticsHandler := handlers.NewAnalyticsHandler(analyticsService, submissionLifecycleService, authService)
	
	// Initialize enhanced webhook handler
	enhancedWebhookHandler := handlers.NewEnhancedWebhookHandler(enhancedWebhookService, integrationManager, authService)

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

	// Initialize spam detection middleware
	spamMiddleware := middleware.NewSpamDetectionMiddleware(spamService, securityService, behavioralAnalyzer, mlClassifier)
	
	// Security and spam protection middleware
	router.Use(middleware.RateLimit(redis))
	router.Use(spamMiddleware.SpamProtection())
	router.Use(spamMiddleware.BehavioralAnalysis())
	router.Use(spamMiddleware.IPReputationCheck())
	
	// Honeypot protection for form submissions
	honeypotFields := []string{"_honeypot", "_hp", "_bot_check", "_email_confirm"}
	router.Use(spamMiddleware.HoneypotProtection(honeypotFields))

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
			
			// Enhanced Webhook System
			webhooks := protected.Group("/forms/:formId/webhooks")
			{
				// Webhook Endpoint Management
				webhooks.POST("/endpoints", enhancedWebhookHandler.CreateWebhookEndpoint)
				webhooks.GET("/endpoints", enhancedWebhookHandler.GetWebhookEndpoints)
				webhooks.PUT("/endpoints/:endpointId", enhancedWebhookHandler.UpdateWebhookEndpoint)
				webhooks.DELETE("/endpoints/:endpointId", enhancedWebhookHandler.DeleteWebhookEndpoint)
				webhooks.POST("/endpoints/:endpointId/test", enhancedWebhookHandler.TestWebhookEndpoint)
				
				// Webhook Analytics
				webhooks.GET("/analytics", enhancedWebhookHandler.GetWebhookAnalytics)
				webhooks.GET("/stats/realtime", enhancedWebhookHandler.GetRealtimeWebhookStats)
				
				// Webhook Monitoring
				webhooks.GET("/monitoring", enhancedWebhookHandler.GetWebhookMonitoring)
				webhooks.GET("/monitoring/ws", enhancedWebhookHandler.WebSocketMonitoring)
			}

			// Submissions
			protected.GET("/forms/:id/submissions", submissionHandler.GetSubmissions)
			protected.GET("/submissions/:id", submissionHandler.GetSubmission)
			protected.DELETE("/submissions/:id", submissionHandler.DeleteSubmission)

			// API Keys
			protected.GET("/api-keys", authHandler.GetAPIKeys)
			protected.POST("/api-keys", authHandler.CreateAPIKey)
			protected.DELETE("/api-keys/:id", authHandler.DeleteAPIKey)
			
			// Third-party Integrations
			integrations := protected.Group("/integrations")
			{
				integrations.GET("", enhancedWebhookHandler.ListIntegrations)
				integrations.GET("/:integration/schema", enhancedWebhookHandler.GetIntegrationSchema)
				integrations.POST("/:integration/test", enhancedWebhookHandler.TestIntegration)
				integrations.POST("/:integration/send", enhancedWebhookHandler.SendToIntegration)
			}
			
			// Integration Marketplace
			marketplace := protected.Group("/marketplace")
			{
				marketplace.GET("/integrations", enhancedWebhookHandler.GetMarketplaceIntegrations)
				marketplace.GET("/integrations/:integrationId", enhancedWebhookHandler.GetMarketplaceIntegration)
				marketplace.POST("/forms/:formId/integrations/:integrationId/install", enhancedWebhookHandler.InstallMarketplaceIntegration)
				marketplace.GET("/categories", enhancedWebhookHandler.GetMarketplaceCategories)
			}

			// Analytics endpoints
			analytics := protected.Group("/analytics")
			{
				// Form Analytics
				analytics.GET("/forms/:id/dashboard", analyticsHandler.GetFormAnalyticsDashboard)
				analytics.GET("/forms/:id/funnel", analyticsHandler.GetFormConversionFunnel)
				analytics.GET("/forms/:id/export", analyticsHandler.ExportAnalyticsData)
				
				// Real-time Analytics
				analytics.GET("/realtime", analyticsHandler.GetRealTimeStats)
				analytics.GET("/ws", func(c *gin.Context) {
					realTimeService.HandleWebSocket(c)
				})
				
				// Submission Lifecycle
				analytics.GET("/submissions/:id/lifecycle", analyticsHandler.GetSubmissionLifecycle)
				analytics.PUT("/submissions/:id/lifecycle", analyticsHandler.UpdateSubmissionLifecycle)
				analytics.GET("/lifecycle/stats", analyticsHandler.GetLifecycleStats)
				analytics.GET("/pending-actions", analyticsHandler.GetPendingActions)
			}
			
			// Public analytics endpoint (tracking ID lookup)
			api.GET("/track/:tracking_id", analyticsHandler.GetSubmissionByTrackingID)
			
			// Analytics event recording (for client-side tracking)
			api.POST("/analytics/events", analyticsHandler.RecordAnalyticsEvent)

			// Email Templates
			emailRoutes := protected.Group("/email")
			{
				// Email Templates
				templates := emailRoutes.Group("/templates")
				{
					templates.POST("", emailTemplateHandler.CreateTemplate)
					templates.GET("", emailTemplateHandler.ListTemplates)
					templates.GET("/:id", emailTemplateHandler.GetTemplate)
					templates.PUT("/:id", emailTemplateHandler.UpdateTemplate)
					templates.DELETE("/:id", emailTemplateHandler.DeleteTemplate)
					templates.POST("/:id/clone", emailTemplateHandler.CloneTemplate)
					templates.POST("/:id/preview", emailTemplateHandler.PreviewTemplate)
					templates.GET("/:id/analytics", emailTemplateHandler.GetTemplateAnalytics)
				}

				// Email Providers
				providers := emailRoutes.Group("/providers")
				{
					providers.POST("", emailTemplateHandler.CreateProvider)
					providers.GET("", emailTemplateHandler.ListProviders)
					providers.POST("/:id/test", emailTemplateHandler.TestProvider)
				}

				// Autoresponders
				autoresponders := emailRoutes.Group("/autoresponders")
				{
					autoresponders.POST("", emailTemplateHandler.CreateAutoresponder)
					autoresponders.GET("", emailTemplateHandler.ListAutoresponders)
					autoresponders.POST("/:id/toggle", emailTemplateHandler.ToggleAutoresponder)
				}

				// Email Queue
				queue := emailRoutes.Group("/queue")
				{
					queue.GET("/stats", emailTemplateHandler.GetQueueStats)
					queue.GET("/emails", emailTemplateHandler.ListQueuedEmails)
					queue.POST("/process", emailTemplateHandler.ProcessQueue)
				}

				// Email Analytics
				analytics := emailRoutes.Group("/analytics")
				{
					analytics.GET("/overview", emailTemplateHandler.GetUserAnalytics)
					analytics.GET("/top-templates", emailTemplateHandler.GetTopPerformingTemplates)
				}

				// Template Builder
				builder := emailRoutes.Group("/builder")
				{
					builder.POST("/designs", emailTemplateHandler.CreateTemplateDesign)
					builder.POST("/preview", emailTemplateHandler.GeneratePreview)
					builder.GET("/components", emailTemplateHandler.GetAvailableComponents)
				}

				// A/B Testing
				abtests := emailRoutes.Group("/ab-tests")
				{
					abtests.POST("", emailTemplateHandler.CreateABTest)
					abtests.POST("/:id/start", emailTemplateHandler.StartABTest)
					abtests.GET("/:id/results", emailTemplateHandler.GetABTestResults)
				}
			}
			
			// Spam Protection Administration (requires admin role)
			admin := protected.Group("/admin")
			// In production, add admin role check middleware here
			{
				// Spam protection configuration
				admin.GET("/spam/config", spamAdminHandler.GetSpamProtectionConfig)
				admin.PUT("/spam/config", spamAdminHandler.UpdateSpamProtectionConfig)
				admin.GET("/spam/forms/:formId/config", spamAdminHandler.GetFormSpamConfig)
				admin.PUT("/spam/forms/:formId/config", spamAdminHandler.UpdateFormSpamConfig)
				
				// Statistics and monitoring
				admin.GET("/spam/statistics", spamAdminHandler.GetSpamStatistics)
				admin.GET("/spam/quarantined", spamAdminHandler.GetQuarantinedSubmissions)
				admin.PUT("/spam/quarantined/:submissionId", spamAdminHandler.ReviewQuarantinedSubmission)
				
				// Machine learning management
				admin.GET("/spam/ml/stats", spamAdminHandler.GetMLModelStats)
				admin.POST("/spam/ml/train", spamAdminHandler.TrainMLModel)
				
				// Behavioral analysis
				admin.GET("/spam/behavioral/stats", spamAdminHandler.GetBehavioralAnalysisStats)
				
				// CAPTCHA management
				admin.GET("/spam/captcha/stats", spamAdminHandler.GetCaptchaStats)
				
				// Webhook management
				admin.GET("/spam/webhooks", spamAdminHandler.GetWebhookStatus)
				admin.POST("/spam/webhooks/test", spamAdminHandler.TestWebhook)
				
				// IP reputation management
				admin.GET("/spam/ip-reputation", spamAdminHandler.GetIPReputation)
				admin.PUT("/spam/ip-reputation", spamAdminHandler.UpdateIPReputation)
				
				// Data export
				admin.GET("/spam/export", spamAdminHandler.ExportSpamData)
			}
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

	// Start background services
	go func() {
		log.Println("Starting webhook queue processor...")
		webhookService.ProcessWebhookQueue()
	}()
	
	go func() {
		log.Println("Starting webhook retry processor...")
		webhookService.ProcessRetryQueue()
	}()

	// Start email queue processor
	go func() {
		log.Println("Starting email queue processor...")
		if err := emailQueueService.StartProcessor(); err != nil {
			log.Printf("Failed to start email queue processor: %v", err)
		}
	}()
	
	// Start analytics and monitoring services
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	
	go func() {
		log.Println("Starting real-time analytics service...")
		realTimeService.StartRealTimeUpdates(ctx)
	}()
	
	go func() {
		log.Println("Starting monitoring service...")
		monitoringService.StartMonitoring(ctx)
	}()
	
	go func() {
		log.Println("Starting cache background tasks...")
		cacheService.StartBackgroundTasks(ctx)
	}()
	
	// Start automated reporting
	go func() {
		ticker := time.NewTicker(15 * time.Minute) // Check for reports every 15 minutes
		defer ticker.Stop()
		
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := reportingService.SendScheduledReports(ctx); err != nil {
					log.Printf("Failed to send scheduled reports: %v", err)
				}
			}
		}
	}()
	
	// Start A/B test optimization
	go func() {
		ticker := time.NewTicker(1 * time.Hour) // Check A/B tests every hour
		defer ticker.Stop()
		
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := abTestingService.AutoOptimizeABTests(ctx); err != nil {
					log.Printf("Failed to optimize A/B tests: %v", err)
				}
			}
		}
	}()
	
	// Start periodic cache cleanup and maintenance
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()
		
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				log.Println("Cleaning up spam detection cache...")
				spamMiddleware.CleanupCache()
				
				// Clean up old email records (older than 90 days)
				log.Println("Cleaning up old email records...")
				if err := emailQueueService.CleanupOldEmails(90); err != nil {
					log.Printf("Failed to cleanup old emails: %v", err)
				}
				
				// Archive old submissions
				if err := submissionLifecycleService.ArchiveOldSubmissions(ctx, 365); err != nil {
					log.Printf("Failed to archive old submissions: %v", err)
				}
			}
		}
	}()
	
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

	// Stop email queue processor
	log.Println("Stopping email queue processor...")
	if err := emailQueueService.StopProcessor(); err != nil {
		log.Printf("Error stopping email queue processor: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	log.Println("Server exited")
}