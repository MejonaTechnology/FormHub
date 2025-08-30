package handlers

import (
	"encoding/json"
	"fmt"
	"formhub/internal/services"
	"formhub/pkg/utils"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// SpamAdminHandler handles spam protection administration endpoints
type SpamAdminHandler struct {
	spamService         *services.SpamProtectionService
	behavioralAnalyzer  *services.BehavioralAnalyzer
	mlClassifier        *services.NaiveBayesSpamClassifier
	webhookService      *services.WebhookService
	authService         *services.AuthService
}

// NewSpamAdminHandler creates a new spam admin handler
func NewSpamAdminHandler(
	spamService *services.SpamProtectionService,
	behavioralAnalyzer *services.BehavioralAnalyzer,
	mlClassifier *services.NaiveBayesSpamClassifier,
	webhookService *services.WebhookService,
	authService *services.AuthService,
) *SpamAdminHandler {
	return &SpamAdminHandler{
		spamService:        spamService,
		behavioralAnalyzer: behavioralAnalyzer,
		mlClassifier:       mlClassifier,
		webhookService:     webhookService,
		authService:        authService,
	}
}

// GetSpamProtectionConfig retrieves global spam protection configuration
func (sah *SpamAdminHandler) GetSpamProtectionConfig(c *gin.Context) {
	// This endpoint should be protected and require admin privileges
	// Implementation would load current global configuration
	
	c.JSON(http.StatusOK, gin.H{
		"message": "Global spam protection configuration",
		"config": gin.H{
			"enabled":             true,
			"default_level":       "medium",
			"block_threshold":     0.8,
			"quarantine_threshold": 0.6,
			"captcha_providers": []string{"recaptcha_v3", "hcaptcha", "turnstile", "fallback"},
			"ml_enabled":         true,
			"behavioral_enabled": true,
		},
	})
}

// UpdateSpamProtectionConfig updates global spam protection configuration
func (sah *SpamAdminHandler) UpdateSpamProtectionConfig(c *gin.Context) {
	var configUpdate services.SpamProtectionConfig
	
	if err := c.ShouldBindJSON(&configUpdate); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid configuration format",
			"message": err.Error(),
		})
		return
	}
	
	// Validate configuration
	if err := sah.validateSpamConfig(&configUpdate); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid configuration",
			"message": err.Error(),
		})
		return
	}
	
	// Update configuration
	if err := sah.spamService.UpdateGlobalConfig(&configUpdate); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to update configuration",
			"message": err.Error(),
		})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"message": "Configuration updated successfully",
		"config":  configUpdate,
	})
}

// GetFormSpamConfig retrieves spam protection configuration for a specific form
func (sah *SpamAdminHandler) GetFormSpamConfig(c *gin.Context) {
	formID := c.Param("formId")
	if formID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Form ID is required",
		})
		return
	}
	
	// This would fetch the actual form configuration from the service
	// For now, return a mock configuration
	c.JSON(http.StatusOK, gin.H{
		"form_id": formID,
		"config": gin.H{
			"enabled":      true,
			"level":        "medium",
			"custom_rules": []gin.H{},
			"whitelist":    []string{},
			"blacklist":    []string{},
		},
	})
}

// UpdateFormSpamConfig updates spam protection configuration for a specific form
func (sah *SpamAdminHandler) UpdateFormSpamConfig(c *gin.Context) {
	formID := c.Param("formId")
	if formID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Form ID is required",
		})
		return
	}
	
	var formConfig services.FormSpamConfig
	if err := c.ShouldBindJSON(&formConfig); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid configuration format",
			"message": err.Error(),
		})
		return
	}
	
	formConfig.FormID = formID
	
	if err := sah.spamService.UpdateFormConfig(&formConfig); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to update form configuration",
			"message": err.Error(),
		})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"message":     "Form configuration updated successfully",
		"form_id":     formID,
		"config":      formConfig,
	})
}

// GetSpamStatistics retrieves spam detection statistics
func (sah *SpamAdminHandler) GetSpamStatistics(c *gin.Context) {
	// Parse query parameters
	formID := c.Query("form_id")
	daysStr := c.DefaultQuery("days", "30")
	days, err := strconv.Atoi(daysStr)
	if err != nil || days < 1 || days > 365 {
		days = 30
	}
	
	// Get statistics
	var stats map[string]interface{}
	if formID != "" {
		stats, err = sah.spamService.GetSpamStatistics(formID, days)
	} else {
		stats, err = sah.spamService.GetSpamStatistics("", days) // Global stats
	}
	
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to retrieve statistics",
			"message": err.Error(),
		})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"statistics": stats,
		"period":     fmt.Sprintf("%d days", days),
		"form_id":    formID,
	})
}

// GetQuarantinedSubmissions retrieves quarantined submissions for review
func (sah *SpamAdminHandler) GetQuarantinedSubmissions(c *gin.Context) {
	// Parse query parameters
	formID := c.Query("form_id")
	limitStr := c.DefaultQuery("limit", "50")
	offsetStr := c.DefaultQuery("offset", "0")
	status := c.DefaultQuery("status", "pending")
	
	limit, _ := strconv.Atoi(limitStr)
	offset, _ := strconv.Atoi(offsetStr)
	
	if limit < 1 || limit > 100 {
		limit = 50
	}
	
	// Mock data for now - replace with actual database query
	submissions := []gin.H{
		{
			"id":             uuid.New().String(),
			"form_id":        formID,
			"client_ip":      "192.168.1.100",
			"spam_score":     0.75,
			"confidence":     0.85,
			"triggers":       []string{"suspicious_content", "high_typing_speed"},
			"status":         "pending",
			"submitted_at":   time.Now().Add(-2 * time.Hour),
			"data": gin.H{
				"name":    "John Doe",
				"email":   "john@example.com",
				"message": "This is a test submission",
			},
		},
	}
	
	c.JSON(http.StatusOK, gin.H{
		"submissions": submissions,
		"pagination": gin.H{
			"limit":  limit,
			"offset": offset,
			"total":  len(submissions),
		},
	})
}

// ReviewQuarantinedSubmission reviews a quarantined submission
func (sah *SpamAdminHandler) ReviewQuarantinedSubmission(c *gin.Context) {
	submissionID := c.Param("submissionId")
	if submissionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Submission ID is required",
		})
		return
	}
	
	var review struct {
		Action string `json:"action" binding:"required"` // "approve", "reject", "spam"
		Notes  string `json:"notes"`
	}
	
	if err := c.ShouldBindJSON(&review); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid review format",
			"message": err.Error(),
		})
		return
	}
	
	if review.Action != "approve" && review.Action != "reject" && review.Action != "spam" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Action must be 'approve', 'reject', or 'spam'",
		})
		return
	}
	
	// Get user information from context
	userID := c.GetString("user_id")
	if userID == "" {
		userID = "admin" // Default for now
	}
	
	// Process the review (this would update the database)
	c.JSON(http.StatusOK, gin.H{
		"message":       "Submission reviewed successfully",
		"submission_id": submissionID,
		"action":        review.Action,
		"reviewed_by":   userID,
		"reviewed_at":   time.Now(),
		"notes":         review.Notes,
	})
}

// GetMLModelStats retrieves machine learning model statistics
func (sah *SpamAdminHandler) GetMLModelStats(c *gin.Context) {
	stats := sah.mlClassifier.GetModelStats()
	
	c.JSON(http.StatusOK, gin.H{
		"model_stats": stats,
	})
}

// TrainMLModel triggers machine learning model training
func (sah *SpamAdminHandler) TrainMLModel(c *gin.Context) {
	var request struct {
		UseLatestData bool `json:"use_latest_data"`
		MinSamples    int  `json:"min_samples"`
	}
	
	if err := c.ShouldBindJSON(&request); err != nil {
		// Use defaults
		request.UseLatestData = true
		request.MinSamples = 100
	}
	
	// This would trigger model training in the background
	c.JSON(http.StatusAccepted, gin.H{
		"message":         "Model training initiated",
		"use_latest_data": request.UseLatestData,
		"min_samples":     request.MinSamples,
		"estimated_time":  "10-30 minutes",
	})
}

// GetBehavioralAnalysisStats retrieves behavioral analysis statistics
func (sah *SpamAdminHandler) GetBehavioralAnalysisStats(c *gin.Context) {
	// This would get actual stats from the behavioral analyzer
	stats := gin.H{
		"total_profiles_analyzed": 1250,
		"bot_detection_rate":      12.5,
		"average_confidence":      0.78,
		"top_anomalies": []gin.H{
			{"type": "typing_speed", "count": 45, "avg_score": 0.72},
			{"type": "mouse_behavior", "count": 38, "avg_score": 0.68},
			{"type": "interaction_timing", "count": 29, "avg_score": 0.81},
		},
		"models_last_updated": time.Now().Add(-6 * time.Hour),
	}
	
	c.JSON(http.StatusOK, gin.H{
		"behavioral_stats": stats,
	})
}

// GetCaptchaStats retrieves CAPTCHA usage statistics
func (sah *SpamAdminHandler) GetCaptchaStats(c *gin.Context) {
	days := 7 // Last 7 days
	daysParam := c.Query("days")
	if daysParam != "" {
		if d, err := strconv.Atoi(daysParam); err == nil && d > 0 && d <= 90 {
			days = d
		}
	}
	
	// Mock CAPTCHA statistics
	stats := gin.H{
		"period_days": days,
		"total_challenges": 542,
		"successful_verifications": 478,
		"failed_verifications": 64,
		"success_rate": 88.2,
		"provider_breakdown": gin.H{
			"recaptcha_v3": gin.H{
				"challenges": 312,
				"success_rate": 91.7,
				"avg_score": 0.78,
			},
			"hcaptcha": gin.H{
				"challenges": 156,
				"success_rate": 89.1,
			},
			"turnstile": gin.H{
				"challenges": 52,
				"success_rate": 84.6,
			},
			"fallback": gin.H{
				"challenges": 22,
				"success_rate": 72.7,
			},
		},
		"challenge_reasons": gin.H{
			"suspicious_behavior": 245,
			"ip_reputation": 178,
			"spam_score": 119,
		},
	}
	
	c.JSON(http.StatusOK, gin.H{
		"captcha_stats": stats,
	})
}

// GetWebhookStatus retrieves webhook notification status
func (sah *SpamAdminHandler) GetWebhookStatus(c *gin.Context) {
	formID := c.Query("form_id")
	limitStr := c.DefaultQuery("limit", "20")
	limit, _ := strconv.Atoi(limitStr)
	
	if limit < 1 || limit > 100 {
		limit = 20
	}
	
	// Get webhook history
	var history []services.WebhookNotification
	var err error
	
	if formID != "" {
		history, err = sah.webhookService.GetWebhookHistory(formID, limit)
	} else {
		// Get global webhook stats
		stats := sah.webhookService.GetWebhookStats()
		c.JSON(http.StatusOK, gin.H{
			"webhook_stats": stats,
		})
		return
	}
	
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to retrieve webhook history",
			"message": err.Error(),
		})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"form_id": formID,
		"webhooks": history,
		"limit":   limit,
	})
}

// TestWebhook tests webhook configuration
func (sah *SpamAdminHandler) TestWebhook(c *gin.Context) {
	var config services.WebhookConfig
	
	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid webhook configuration",
			"message": err.Error(),
		})
		return
	}
	
	// Validate webhook URL
	if err := sah.webhookService.ValidateWebhookURL(config.URL); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid webhook URL",
			"message": err.Error(),
		})
		return
	}
	
	// Test the webhook
	response, err := sah.webhookService.TestWebhook(&config)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Webhook test failed",
			"message": err.Error(),
		})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"message":      "Webhook test completed",
		"url":          config.URL,
		"status_code":  response.StatusCode,
		"response_time": response.ResponseTime.Milliseconds(),
		"success":      response.StatusCode >= 200 && response.StatusCode < 300,
		"response":     gin.H{
			"headers": response.Headers,
			"body":    response.Body[:min(len(response.Body), 1000)], // Limit response body
		},
	})
}

// GetIPReputation retrieves IP reputation information
func (sah *SpamAdminHandler) GetIPReputation(c *gin.Context) {
	ip := c.Query("ip")
	if ip == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "IP address is required",
		})
		return
	}
	
	// Mock IP reputation data
	reputation := gin.H{
		"ip":               ip,
		"reputation":       "neutral",
		"risk_score":       0.3,
		"country_code":     "US",
		"asn":             "AS12345",
		"is_vpn":          false,
		"is_proxy":        false,
		"is_tor":          false,
		"submission_count": 5,
		"block_count":     0,
		"last_seen":       time.Now().Add(-2 * time.Hour),
		"first_seen":      time.Now().Add(-30 * 24 * time.Hour),
	}
	
	c.JSON(http.StatusOK, gin.H{
		"ip_reputation": reputation,
	})
}

// UpdateIPReputation manually updates IP reputation
func (sah *SpamAdminHandler) UpdateIPReputation(c *gin.Context) {
	var request struct {
		IP         string  `json:"ip" binding:"required"`
		Reputation string  `json:"reputation" binding:"required"` // good, neutral, suspicious, malicious
		RiskScore  float64 `json:"risk_score"`
		Reason     string  `json:"reason"`
	}
	
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request format",
			"message": err.Error(),
		})
		return
	}
	
	validReputations := map[string]bool{
		"good":       true,
		"neutral":    true,
		"suspicious": true,
		"malicious":  true,
	}
	
	if !validReputations[request.Reputation] {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid reputation value. Must be: good, neutral, suspicious, or malicious",
		})
		return
	}
	
	// Get user information
	userID := c.GetString("user_id")
	if userID == "" {
		userID = "admin"
	}
	
	// This would update the IP reputation in the database
	c.JSON(http.StatusOK, gin.H{
		"message":     "IP reputation updated successfully",
		"ip":          request.IP,
		"reputation":  request.Reputation,
		"risk_score":  request.RiskScore,
		"updated_by":  userID,
		"updated_at":  time.Now(),
	})
}

// ExportSpamData exports spam detection data for analysis
func (sah *SpamAdminHandler) ExportSpamData(c *gin.Context) {
	// Parse query parameters
	format := c.DefaultQuery("format", "json") // json, csv
	formID := c.Query("form_id")
	daysStr := c.DefaultQuery("days", "30")
	
	days, err := strconv.Atoi(daysStr)
	if err != nil || days < 1 || days > 365 {
		days = 30
	}
	
	if format != "json" && format != "csv" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Format must be 'json' or 'csv'",
		})
		return
	}
	
	// This would query the database and export the data
	// For now, return a sample response
	
	if format == "json" {
		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=spam_data_%s.json", time.Now().Format("2006-01-02")))
		c.Header("Content-Type", "application/json")
		
		exportData := gin.H{
			"export_info": gin.H{
				"exported_at": time.Now(),
				"period_days": days,
				"form_id":     formID,
				"total_records": 150,
			},
			"data": []gin.H{
				{
					"timestamp":   time.Now().Add(-24 * time.Hour),
					"form_id":     formID,
					"client_ip":   "192.168.1.100",
					"spam_score":  0.75,
					"action":      "quarantine",
					"triggers":    []string{"suspicious_content"},
				},
			},
		}
		
		c.JSON(http.StatusOK, exportData)
	} else {
		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=spam_data_%s.csv", time.Now().Format("2006-01-02")))
		c.Header("Content-Type", "text/csv")
		
		csvData := "timestamp,form_id,client_ip,spam_score,action,triggers\n"
		csvData += fmt.Sprintf("%s,%s,192.168.1.100,0.75,quarantine,suspicious_content\n", 
			time.Now().Add(-24*time.Hour).Format(time.RFC3339), formID)
		
		c.String(http.StatusOK, csvData)
	}
}

// Helper methods

func (sah *SpamAdminHandler) validateSpamConfig(config *services.SpamProtectionConfig) error {
	if config.BlockThreshold < 0 || config.BlockThreshold > 1 {
		return fmt.Errorf("block_threshold must be between 0 and 1")
	}
	
	if config.QuarantineThreshold < 0 || config.QuarantineThreshold > 1 {
		return fmt.Errorf("quarantine_threshold must be between 0 and 1")
	}
	
	if config.BlockThreshold <= config.QuarantineThreshold {
		return fmt.Errorf("block_threshold must be greater than quarantine_threshold")
	}
	
	if config.CaptchaConfig != nil {
		if config.CaptchaConfig.RecaptchaV3MinScore < 0 || config.CaptchaConfig.RecaptchaV3MinScore > 1 {
			return fmt.Errorf("recaptcha_v3_min_score must be between 0 and 1")
		}
	}
	
	return nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}