package handlers

import (
	"encoding/json"
	"formhub/internal/models"
	"formhub/internal/services"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type AnalyticsHandler struct {
	analyticsService           *services.AnalyticsService
	submissionLifecycleService *services.SubmissionLifecycleService
	authService                *services.AuthService
}

func NewAnalyticsHandler(
	analyticsService *services.AnalyticsService,
	submissionLifecycleService *services.SubmissionLifecycleService,
	authService *services.AuthService,
) *AnalyticsHandler {
	return &AnalyticsHandler{
		analyticsService:           analyticsService,
		submissionLifecycleService: submissionLifecycleService,
		authService:                authService,
	}
}

// RecordAnalyticsEvent records an analytics event
func (h *AnalyticsHandler) RecordAnalyticsEvent(c *gin.Context) {
	var req models.AnalyticsEventRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	// Get user ID from context (if authenticated)
	userID := h.getUserIDFromContext(c)
	if userID == uuid.Nil {
		// For public form analytics, we need to get the user ID from the form
		userID = h.getUserIDFromForm(c, req.FormID)
		if userID == uuid.Nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Unable to determine form owner"})
			return
		}
	}

	// Create analytics event
	event := &models.FormAnalyticsEvent{
		FormID:               req.FormID,
		UserID:               userID,
		SessionID:            req.SessionID,
		EventType:            req.EventType,
		FieldName:            req.FieldName,
		FieldValueLength:     req.FieldValueLength,
		FieldValidationError: req.FieldValidationError,
		PageURL:              req.PageURL,
		Referrer:             req.Referrer,
		IPAddress:            c.ClientIP(),
		UserAgent:            &c.Request.Header.Get("User-Agent"),
		EventData:            req.EventData,
	}

	// Extract UTM parameters
	if req.UTMData != nil {
		event.UTMSource = req.UTMData.Source
		event.UTMMedium = req.UTMData.Medium
		event.UTMCampaign = req.UTMData.Campaign
		event.UTMTerm = req.UTMData.Term
		event.UTMContent = req.UTMData.Content
	}

	// Parse User-Agent for device/browser info
	h.parseUserAgent(event, c.Request.Header.Get("User-Agent"))

	// Get geographic info (this would typically use a GeoIP service)
	h.populateGeographicInfo(event, c.ClientIP())

	// Record the event
	err := h.analyticsService.RecordEvent(c.Request.Context(), event)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to record analytics event"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// GetFormAnalyticsDashboard returns comprehensive analytics for a form
func (h *AnalyticsHandler) GetFormAnalyticsDashboard(c *gin.Context) {
	formIDStr := c.Param("id")
	formID, err := uuid.Parse(formIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid form ID"})
		return
	}

	userID := h.getUserIDFromContext(c)
	if userID == uuid.Nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Parse query parameters
	params := &models.AnalyticsQueryParams{}
	
	if startDateStr := c.Query("start_date"); startDateStr != "" {
		if startDate, err := time.Parse("2006-01-02", startDateStr); err == nil {
			params.StartDate = &startDate
		}
	}
	
	if endDateStr := c.Query("end_date"); endDateStr != "" {
		if endDate, err := time.Parse("2006-01-02", endDateStr); err == nil {
			params.EndDate = &endDate
		}
	}

	if timezone := c.Query("timezone"); timezone != "" {
		params.Timezone = &timezone
	}

	params.Granularity = c.DefaultQuery("granularity", "day")

	// Get dashboard data
	dashboard, err := h.analyticsService.GetFormAnalyticsDashboard(c.Request.Context(), formID, userID, params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get analytics data"})
		return
	}

	c.JSON(http.StatusOK, dashboard)
}

// GetRealTimeStats returns real-time analytics
func (h *AnalyticsHandler) GetRealTimeStats(c *gin.Context) {
	userID := h.getUserIDFromContext(c)
	if userID == uuid.Nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	stats, err := h.analyticsService.GetRealTimeStats(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get real-time stats"})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// GetFormConversionFunnel returns conversion funnel data
func (h *AnalyticsHandler) GetFormConversionFunnel(c *gin.Context) {
	formIDStr := c.Param("id")
	formID, err := uuid.Parse(formIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid form ID"})
		return
	}

	userID := h.getUserIDFromContext(c)
	if userID == uuid.Nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Parse date range
	startDate := time.Now().AddDate(0, 0, -30) // Default: last 30 days
	endDate := time.Now()

	if startDateStr := c.Query("start_date"); startDateStr != "" {
		if parsed, err := time.Parse("2006-01-02", startDateStr); err == nil {
			startDate = parsed
		}
	}

	if endDateStr := c.Query("end_date"); endDateStr != "" {
		if parsed, err := time.Parse("2006-01-02", endDateStr); err == nil {
			endDate = parsed
		}
	}

	funnels, err := h.analyticsService.GetFormConversionFunnel(c.Request.Context(), formID, userID, startDate, endDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get conversion funnel data"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"funnels": funnels})
}

// GetSubmissionLifecycle returns submission lifecycle information
func (h *AnalyticsHandler) GetSubmissionLifecycle(c *gin.Context) {
	submissionIDStr := c.Param("id")
	submissionID, err := uuid.Parse(submissionIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid submission ID"})
		return
	}

	lifecycle, err := h.submissionLifecycleService.GetSubmissionLifecycle(c.Request.Context(), submissionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get submission lifecycle"})
		return
	}

	if lifecycle == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Submission not found"})
		return
	}

	c.JSON(http.StatusOK, lifecycle)
}

// GetSubmissionByTrackingID returns submission info by tracking ID (public endpoint)
func (h *AnalyticsHandler) GetSubmissionByTrackingID(c *gin.Context) {
	trackingID := c.Param("tracking_id")
	if trackingID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Tracking ID required"})
		return
	}

	lifecycle, err := h.submissionLifecycleService.GetSubmissionLifecycleByTrackingID(c.Request.Context(), trackingID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Submission not found"})
		return
	}

	// Return limited public information
	publicInfo := gin.H{
		"tracking_id": lifecycle.TrackingID,
		"status":      lifecycle.Status,
		"created_at":  lifecycle.CreatedAt,
		"updated_at":  lifecycle.UpdatedAt,
	}

	// Add response info if available
	if lifecycle.ResponseTime != nil {
		publicInfo["response_time"] = lifecycle.ResponseTime
		publicInfo["response_method"] = lifecycle.ResponseMethod
	}

	c.JSON(http.StatusOK, publicInfo)
}

// UpdateSubmissionLifecycle updates submission lifecycle status
func (h *AnalyticsHandler) UpdateSubmissionLifecycle(c *gin.Context) {
	submissionIDStr := c.Param("id")
	submissionID, err := uuid.Parse(submissionIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid submission ID"})
		return
	}

	userID := h.getUserIDFromContext(c)
	if userID == uuid.Nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req models.UpdateSubmissionLifecycleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	// Update status
	err = h.submissionLifecycleService.UpdateSubmissionStatus(c.Request.Context(), submissionID, req.Status, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update submission status"})
		return
	}

	// Record response if provided
	if req.ResponseMethod != nil {
		notes := ""
		if req.Notes != nil {
			notes = *req.Notes
		}
		err = h.submissionLifecycleService.RecordResponse(c.Request.Context(), submissionID, *req.ResponseMethod, notes)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to record response"})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// GetLifecycleStats returns lifecycle statistics
func (h *AnalyticsHandler) GetLifecycleStats(c *gin.Context) {
	userID := h.getUserIDFromContext(c)
	if userID == uuid.Nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Parse optional form ID
	var formID *uuid.UUID
	if formIDStr := c.Query("form_id"); formIDStr != "" {
		if parsed, err := uuid.Parse(formIDStr); err == nil {
			formID = &parsed
		}
	}

	// Parse date range
	startDate := time.Now().AddDate(0, 0, -30) // Default: last 30 days
	endDate := time.Now()

	if startDateStr := c.Query("start_date"); startDateStr != "" {
		if parsed, err := time.Parse("2006-01-02", startDateStr); err == nil {
			startDate = parsed
		}
	}

	if endDateStr := c.Query("end_date"); endDateStr != "" {
		if parsed, err := time.Parse("2006-01-02", endDateStr); err == nil {
			endDate = parsed
		}
	}

	stats, err := h.submissionLifecycleService.GetLifecycleStats(c.Request.Context(), userID, formID, startDate, endDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get lifecycle stats"})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// GetPendingActions returns pending actions requiring attention
func (h *AnalyticsHandler) GetPendingActions(c *gin.Context) {
	userID := h.getUserIDFromContext(c)
	if userID == uuid.Nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	actions, err := h.submissionLifecycleService.GetPendingActions(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get pending actions"})
		return
	}

	c.JSON(http.StatusOK, actions)
}

// ExportAnalyticsData exports analytics data in specified format
func (h *AnalyticsHandler) ExportAnalyticsData(c *gin.Context) {
	formIDStr := c.Param("id")
	formID, err := uuid.Parse(formIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid form ID"})
		return
	}

	userID := h.getUserIDFromContext(c)
	if userID == uuid.Nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	format := c.DefaultQuery("format", "json")
	
	// Parse query parameters
	params := &models.AnalyticsQueryParams{}
	
	if startDateStr := c.Query("start_date"); startDateStr != "" {
		if startDate, err := time.Parse("2006-01-02", startDateStr); err == nil {
			params.StartDate = &startDate
		}
	}
	
	if endDateStr := c.Query("end_date"); endDateStr != "" {
		if endDate, err := time.Parse("2006-01-02", endDateStr); err == nil {
			params.EndDate = &endDate
		}
	}

	data, contentType, err := h.analyticsService.ExportAnalyticsData(c.Request.Context(), formID, userID, params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to export data"})
		return
	}

	filename := "form_analytics_" + formID.String() + "_" + time.Now().Format("20060102") + "." + format
	
	c.Header("Content-Disposition", "attachment; filename="+filename)
	c.Header("Content-Type", contentType)
	c.Data(http.StatusOK, contentType, data)
}

// RecordPerformanceMetrics records API performance metrics (middleware would typically call this)
func (h *AnalyticsHandler) RecordPerformanceMetrics(c *gin.Context) {
	// This is typically called by middleware, not directly as an endpoint
	var metrics models.APIPerformanceMetrics
	if err := c.ShouldBindJSON(&metrics); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid metrics format"})
		return
	}

	err := h.analyticsService.RecordAPIPerformance(c.Request.Context(), &metrics)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to record performance metrics"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// Helper functions

func (h *AnalyticsHandler) getUserIDFromContext(c *gin.Context) uuid.UUID {
	if userInterface, exists := c.Get("user_id"); exists {
		if userID, ok := userInterface.(uuid.UUID); ok {
			return userID
		}
	}
	return uuid.Nil
}

func (h *AnalyticsHandler) getUserIDFromForm(c *gin.Context, formID uuid.UUID) uuid.UUID {
	// Query the database to get the user ID for this form
	var userID uuid.UUID
	err := h.authService.DB.Get(&userID, "SELECT user_id FROM forms WHERE id = ?", formID)
	if err != nil {
		return uuid.Nil
	}
	return userID
}

func (h *AnalyticsHandler) parseUserAgent(event *models.FormAnalyticsEvent, userAgent string) {
	// This would typically use a User-Agent parsing library
	// For now, we'll do basic parsing
	
	event.UserAgent = &userAgent
	
	// Basic device type detection
	ua := userAgent
	switch {
	case contains(ua, "Mobile") || contains(ua, "iPhone") || contains(ua, "Android"):
		event.DeviceType = models.DeviceTypeMobile
	case contains(ua, "Tablet") || contains(ua, "iPad"):
		event.DeviceType = models.DeviceTypeTablet
	default:
		event.DeviceType = models.DeviceTypeDesktop
	}

	// Basic browser detection
	switch {
	case contains(ua, "Chrome"):
		event.BrowserName = stringPtr("Chrome")
	case contains(ua, "Firefox"):
		event.BrowserName = stringPtr("Firefox")
	case contains(ua, "Safari"):
		event.BrowserName = stringPtr("Safari")
	case contains(ua, "Edge"):
		event.BrowserName = stringPtr("Edge")
	default:
		event.BrowserName = stringPtr("Other")
	}

	// Basic OS detection
	switch {
	case contains(ua, "Windows"):
		event.OSName = stringPtr("Windows")
	case contains(ua, "Macintosh") || contains(ua, "Mac OS"):
		event.OSName = stringPtr("macOS")
	case contains(ua, "Linux"):
		event.OSName = stringPtr("Linux")
	case contains(ua, "Android"):
		event.OSName = stringPtr("Android")
	case contains(ua, "iOS") || contains(ua, "iPhone") || contains(ua, "iPad"):
		event.OSName = stringPtr("iOS")
	default:
		event.OSName = stringPtr("Other")
	}
}

func (h *AnalyticsHandler) populateGeographicInfo(event *models.FormAnalyticsEvent, ipAddress string) {
	// This would typically use a GeoIP service like MaxMind or IP2Location
	// For now, we'll set default values
	
	// In a real implementation, you would:
	// 1. Use a GeoIP service to get location data
	// 2. Handle IPv6 addresses
	// 3. Cache results to improve performance
	
	if ipAddress == "127.0.0.1" || ipAddress == "::1" {
		// Localhost
		event.CountryCode = stringPtr("US")
		event.CountryName = stringPtr("United States")
		event.Region = stringPtr("California")
		event.City = stringPtr("San Francisco")
		event.Timezone = stringPtr("America/Los_Angeles")
		return
	}

	// For demo purposes, set some default values
	// In production, replace this with actual GeoIP lookup
	event.CountryCode = stringPtr("US")
	event.CountryName = stringPtr("United States")
	event.Timezone = stringPtr("America/New_York")
}

// Utility functions
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || 
		(len(s) > len(substr) && 
			(s[:len(substr)] == substr || 
			 s[len(s)-len(substr):] == substr || 
			 len(s) > len(substr) && s[1:len(substr)+1] == substr)))
}

func stringPtr(s string) *string {
	return &s
}

func intPtr(i int) *int {
	return &i
}

func float64Ptr(f float64) *float64 {
	return &f
}