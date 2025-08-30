package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"formhub/internal/services"

	"github.com/gin-gonic/gin"
)

// EnhancedWebhookHandler handles webhook management API endpoints
type EnhancedWebhookHandler struct {
	webhookService   *services.EnhancedWebhookService
	integrationManager *services.IntegrationManager
	authService      *services.AuthService
}

// NewEnhancedWebhookHandler creates a new enhanced webhook handler
func NewEnhancedWebhookHandler(
	webhookService *services.EnhancedWebhookService,
	integrationManager *services.IntegrationManager,
	authService *services.AuthService,
) *EnhancedWebhookHandler {
	return &EnhancedWebhookHandler{
		webhookService:   webhookService,
		integrationManager: integrationManager,
		authService:      authService,
	}
}

// Webhook Endpoint Management

// CreateWebhookEndpoint creates a new webhook endpoint for a form
func (ewh *EnhancedWebhookHandler) CreateWebhookEndpoint(c *gin.Context) {
	formID := c.Param("formId")
	if formID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Form ID is required"})
		return
	}
	
	var endpoint services.WebhookEndpoint
	if err := c.ShouldBindJSON(&endpoint); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid endpoint data", "details": err.Error()})
		return
	}
	
	// Validate user permissions
	userID, _ := ewh.authService.GetUserIDFromContext(c)
	if !ewh.canManageForm(userID, formID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Permission denied"})
		return
	}
	
	if err := ewh.webhookService.CreateWebhookEndpoint(formID, &endpoint); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create endpoint", "details": err.Error()})
		return
	}
	
	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "Webhook endpoint created successfully",
		"endpoint": endpoint,
	})
}

// UpdateWebhookEndpoint updates an existing webhook endpoint
func (ewh *EnhancedWebhookHandler) UpdateWebhookEndpoint(c *gin.Context) {
	formID := c.Param("formId")
	endpointID := c.Param("endpointId")
	
	if formID == "" || endpointID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Form ID and endpoint ID are required"})
		return
	}
	
	var updates services.WebhookEndpoint
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid endpoint data", "details": err.Error()})
		return
	}
	
	// Validate user permissions
	userID, _ := ewh.authService.GetUserIDFromContext(c)
	if !ewh.canManageForm(userID, formID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Permission denied"})
		return
	}
	
	if err := ewh.webhookService.UpdateWebhookEndpoint(formID, endpointID, &updates); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update endpoint", "details": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Webhook endpoint updated successfully",
	})
}

// DeleteWebhookEndpoint deletes a webhook endpoint
func (ewh *EnhancedWebhookHandler) DeleteWebhookEndpoint(c *gin.Context) {
	formID := c.Param("formId")
	endpointID := c.Param("endpointId")
	
	if formID == "" || endpointID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Form ID and endpoint ID are required"})
		return
	}
	
	// Validate user permissions
	userID, _ := ewh.authService.GetUserIDFromContext(c)
	if !ewh.canManageForm(userID, formID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Permission denied"})
		return
	}
	
	if err := ewh.webhookService.DeleteWebhookEndpoint(formID, endpointID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete endpoint", "details": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Webhook endpoint deleted successfully",
	})
}

// GetWebhookEndpoints returns all webhook endpoints for a form
func (ewh *EnhancedWebhookHandler) GetWebhookEndpoints(c *gin.Context) {
	formID := c.Param("formId")
	if formID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Form ID is required"})
		return
	}
	
	// Validate user permissions
	userID, _ := ewh.authService.GetUserIDFromContext(c)
	if !ewh.canAccessForm(userID, formID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Permission denied"})
		return
	}
	
	config, err := ewh.webhookService.GetFormWebhookConfig(formID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get endpoints", "details": err.Error()})
		return
	}
	
	endpoints := []services.WebhookEndpoint{}
	if config != nil {
		endpoints = config.Endpoints
	}
	
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"endpoints": endpoints,
	})
}

// TestWebhookEndpoint tests a webhook endpoint
func (ewh *EnhancedWebhookHandler) TestWebhookEndpoint(c *gin.Context) {
	formID := c.Param("formId")
	endpointID := c.Param("endpointId")
	
	if formID == "" || endpointID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Form ID and endpoint ID are required"})
		return
	}
	
	// Validate user permissions
	userID, _ := ewh.authService.GetUserIDFromContext(c)
	if !ewh.canManageForm(userID, formID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Permission denied"})
		return
	}
	
	result, err := ewh.webhookService.TestWebhookEndpoint(formID, endpointID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to test endpoint", "details": err.Error()})
		return
	}
	
	statusCode := http.StatusOK
	if !result.Success {
		statusCode = http.StatusBadRequest
	}
	
	c.JSON(statusCode, gin.H{
		"success": result.Success,
		"result": result,
	})
}

// Webhook Analytics

// GetWebhookAnalytics returns comprehensive webhook analytics
func (ewh *EnhancedWebhookHandler) GetWebhookAnalytics(c *gin.Context) {
	formID := c.Param("formId")
	if formID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Form ID is required"})
		return
	}
	
	// Validate user permissions
	userID, _ := ewh.authService.GetUserIDFromContext(c)
	if !ewh.canAccessForm(userID, formID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Permission denied"})
		return
	}
	
	// Parse time range parameters
	timeRange, err := ewh.parseTimeRange(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid time range", "details": err.Error()})
		return
	}
	
	analytics, err := ewh.webhookService.GetWebhookAnalytics(formID, *timeRange)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get analytics", "details": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"analytics": analytics,
	})
}

// GetRealtimeWebhookStats returns real-time webhook statistics
func (ewh *EnhancedWebhookHandler) GetRealtimeWebhookStats(c *gin.Context) {
	formID := c.Param("formId")
	if formID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Form ID is required"})
		return
	}
	
	// Validate user permissions
	userID, _ := ewh.authService.GetUserIDFromContext(c)
	if !ewh.canAccessForm(userID, formID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Permission denied"})
		return
	}
	
	stats, err := ewh.webhookService.GetRealtimeWebhookStats(formID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get realtime stats", "details": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"stats": stats,
	})
}

// GetWebhookMonitoring returns monitoring data
func (ewh *EnhancedWebhookHandler) GetWebhookMonitoring(c *gin.Context) {
	formID := c.Param("formId")
	if formID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Form ID is required"})
		return
	}
	
	// Validate user permissions
	userID, _ := ewh.authService.GetUserIDFromContext(c)
	if !ewh.canAccessForm(userID, formID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Permission denied"})
		return
	}
	
	monitoring, err := ewh.webhookService.GetWebhookMonitoringData(formID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get monitoring data", "details": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"monitoring": monitoring,
	})
}

// Third-party Integrations

// ListIntegrations returns available integrations
func (ewh *EnhancedWebhookHandler) ListIntegrations(c *gin.Context) {
	integrations := ewh.integrationManager.ListIntegrations()
	
	integrationList := make([]map[string]interface{}, 0, len(integrations))
	for _, integration := range integrations {
		schema := integration.GetSchema()
		integrationList = append(integrationList, map[string]interface{}{
			"name":        integration.Name(),
			"schema":      schema,
		})
	}
	
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"integrations": integrationList,
	})
}

// GetIntegrationSchema returns the schema for a specific integration
func (ewh *EnhancedWebhookHandler) GetIntegrationSchema(c *gin.Context) {
	integrationName := c.Param("integration")
	
	integration, exists := ewh.integrationManager.GetIntegration(integrationName)
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Integration not found"})
		return
	}
	
	schema := integration.GetSchema()
	
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"schema": schema,
	})
}

// TestIntegration tests an integration configuration
func (ewh *EnhancedWebhookHandler) TestIntegration(c *gin.Context) {
	integrationName := c.Param("integration")
	
	integration, exists := ewh.integrationManager.GetIntegration(integrationName)
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Integration not found"})
		return
	}
	
	var config map[string]interface{}
	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid configuration", "details": err.Error()})
		return
	}
	
	// Validate configuration
	if err := integration.ValidateConfig(config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Configuration validation failed", "details": err.Error()})
		return
	}
	
	// Test authentication
	if err := integration.Authenticate(config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Authentication failed", "details": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Integration test successful",
	})
}

// SendToIntegration sends a test event to an integration
func (ewh *EnhancedWebhookHandler) SendToIntegration(c *gin.Context) {
	integrationName := c.Param("integration")
	
	var request struct {
		Config map[string]interface{} `json:"config"`
		Event  *services.EnhancedWebhookEvent `json:"event,omitempty"`
	}
	
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data", "details": err.Error()})
		return
	}
	
	// Create test event if not provided
	if request.Event == nil {
		request.Event = &services.EnhancedWebhookEvent{
			ID:        "test-" + strconv.FormatInt(time.Now().UnixNano(), 10),
			Type:      "test",
			Timestamp: time.Now().UTC(),
			FormID:    "test-form",
			Source:    "test",
			Version:   "2.0",
			Environment: "test",
			Data: map[string]interface{}{
				"message": "This is a test event from FormHub",
				"test":    true,
			},
		}
	}
	
	if err := ewh.integrationManager.SendToIntegration(integrationName, request.Event, request.Config); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send to integration", "details": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Event sent to integration successfully",
	})
}

// Integration Marketplace

// GetMarketplaceIntegrations returns marketplace integrations with filtering
func (ewh *EnhancedWebhookHandler) GetMarketplaceIntegrations(c *gin.Context) {
	var filter services.MarketplaceFilter
	
	// Parse query parameters
	if category := c.Query("category"); category != "" {
		filter.Category = category
	}
	if popular := c.Query("popular"); popular == "true" {
		filter.Popular = true
	}
	if featured := c.Query("featured"); featured == "true" {
		filter.Featured = true
	}
	if minRating := c.Query("min_rating"); minRating != "" {
		if rating, err := strconv.ParseFloat(minRating, 64); err == nil {
			filter.MinRating = rating
		}
	}
	if sortBy := c.Query("sort_by"); sortBy != "" {
		filter.SortBy = sortBy
	}
	if sortOrder := c.Query("sort_order"); sortOrder != "" {
		filter.SortOrder = sortOrder
	}
	if limit := c.Query("limit"); limit != "" {
		if l, err := strconv.Atoi(limit); err == nil {
			filter.Limit = l
		}
	}
	
	integrations := ewh.integrationManager.GetMarketplaceIntegrations(&filter)
	
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"integrations": integrations,
		"filter": filter,
	})
}

// GetMarketplaceIntegration returns a specific marketplace integration
func (ewh *EnhancedWebhookHandler) GetMarketplaceIntegration(c *gin.Context) {
	integrationID := c.Param("integrationId")
	
	integration, exists := ewh.integrationManager.GetMarketplaceIntegration(integrationID)
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Integration not found"})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"integration": integration,
	})
}

// InstallMarketplaceIntegration installs a marketplace integration
func (ewh *EnhancedWebhookHandler) InstallMarketplaceIntegration(c *gin.Context) {
	integrationID := c.Param("integrationId")
	formID := c.Param("formId")
	
	// Validate user permissions
	userID, _ := ewh.authService.GetUserIDFromContext(c)
	if !ewh.canManageForm(userID, formID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Permission denied"})
		return
	}
	
	var config map[string]interface{}
	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid configuration", "details": err.Error()})
		return
	}
	
	if err := ewh.integrationManager.InstallMarketplaceIntegration(formID, integrationID, config); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to install integration", "details": err.Error()})
		return
	}
	
	// Increment download counter
	ewh.integrationManager.IncrementDownloads(integrationID)
	
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Integration installed successfully",
	})
}

// GetMarketplaceCategories returns available marketplace categories
func (ewh *EnhancedWebhookHandler) GetMarketplaceCategories(c *gin.Context) {
	categories := ewh.integrationManager.GetMarketplaceCategories()
	
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"categories": categories,
	})
}

// Helper methods

func (ewh *EnhancedWebhookHandler) parseTimeRange(c *gin.Context) (*services.TimeRange, error) {
	defaultStart := time.Now().AddDate(0, 0, -7) // 7 days ago
	defaultEnd := time.Now()
	
	startStr := c.DefaultQuery("start", defaultStart.Format(time.RFC3339))
	endStr := c.DefaultQuery("end", defaultEnd.Format(time.RFC3339))
	
	start, err := time.Parse(time.RFC3339, startStr)
	if err != nil {
		return nil, err
	}
	
	end, err := time.Parse(time.RFC3339, endStr)
	if err != nil {
		return nil, err
	}
	
	return &services.TimeRange{
		Start: start,
		End:   end,
	}, nil
}

func (ewh *EnhancedWebhookHandler) canAccessForm(userID, formID string) bool {
	// Implementation would check if user has read access to the form
	// For this example, we'll assume all authenticated users can access
	return userID != ""
}

func (ewh *EnhancedWebhookHandler) canManageForm(userID, formID string) bool {
	// Implementation would check if user has write access to the form
	// For this example, we'll assume all authenticated users can manage
	return userID != ""
}

// WebSocket endpoint for real-time webhook monitoring
func (ewh *EnhancedWebhookHandler) WebSocketMonitoring(c *gin.Context) {
	formID := c.Param("formId")
	if formID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Form ID is required"})
		return
	}
	
	// Validate user permissions
	userID, _ := ewh.authService.GetUserIDFromContext(c)
	if !ewh.canAccessForm(userID, formID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Permission denied"})
		return
	}
	
	// Upgrade to WebSocket connection
	// Implementation would handle WebSocket upgrade and streaming
	// For now, return error as WebSocket handling requires specific implementation
	c.JSON(http.StatusNotImplemented, gin.H{
		"error": "WebSocket monitoring not implemented in this example",
		"message": "Use GET /api/v1/forms/:formId/webhooks/monitoring for HTTP-based monitoring",
	})
}

// Additional handler methods would be implemented here for:
// - Webhook logs and history
// - Webhook security settings
// - Bulk webhook operations
// - Webhook templates
// - Webhook scheduling
// - Integration OAuth flows
// - Custom integration creation
// - Webhook testing tools