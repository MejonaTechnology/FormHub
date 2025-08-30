package handlers

import (
	"fmt"
	"formhub/internal/models"
	"formhub/internal/services"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type EmailTemplateHandler struct {
	templateService       *services.EmailTemplateService
	providerService       *services.EmailProviderService
	autoresponderService  *services.EmailAutoresponderService
	queueService         *services.EmailQueueService
	analyticsService     *services.EmailAnalyticsService
	builderService       *services.TemplateBuilderService
	abTestingService     *services.EmailABTestingService
}

func NewEmailTemplateHandler(
	templateService *services.EmailTemplateService,
	providerService *services.EmailProviderService,
	autoresponderService *services.EmailAutoresponderService,
	queueService *services.EmailQueueService,
	analyticsService *services.EmailAnalyticsService,
	builderService *services.TemplateBuilderService,
	abTestingService *services.EmailABTestingService,
) *EmailTemplateHandler {
	return &EmailTemplateHandler{
		templateService:      templateService,
		providerService:      providerService,
		autoresponderService: autoresponderService,
		queueService:         queueService,
		analyticsService:     analyticsService,
		builderService:       builderService,
		abTestingService:     abTestingService,
	}
}

// Email Template endpoints

func (h *EmailTemplateHandler) CreateTemplate(c *gin.Context) {
	var req models.CreateEmailTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := getUserIDFromContext(c)
	template, err := h.templateService.CreateTemplate(userID, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success":  true,
		"template": template,
	})
}

func (h *EmailTemplateHandler) GetTemplate(c *gin.Context) {
	templateID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid template ID"})
		return
	}

	userID := getUserIDFromContext(c)
	template, err := h.templateService.GetTemplate(userID, templateID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Template not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":  true,
		"template": template,
	})
}

func (h *EmailTemplateHandler) ListTemplates(c *gin.Context) {
	userID := getUserIDFromContext(c)

	// Parse query parameters
	var templateType *models.EmailTemplateType
	if t := c.Query("type"); t != "" {
		tt := models.EmailTemplateType(t)
		templateType = &tt
	}

	var formID *uuid.UUID
	if f := c.Query("form_id"); f != "" {
		if fid, err := uuid.Parse(f); err == nil {
			formID = &fid
		}
	}

	var language *string
	if l := c.Query("language"); l != "" {
		language = &l
	}

	templates, err := h.templateService.ListTemplates(userID, templateType, formID, language)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"templates": templates,
	})
}

func (h *EmailTemplateHandler) UpdateTemplate(c *gin.Context) {
	templateID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid template ID"})
		return
	}

	var req models.CreateEmailTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := getUserIDFromContext(c)
	template, err := h.templateService.UpdateTemplate(userID, templateID, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":  true,
		"template": template,
	})
}

func (h *EmailTemplateHandler) DeleteTemplate(c *gin.Context) {
	templateID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid template ID"})
		return
	}

	userID := getUserIDFromContext(c)
	err = h.templateService.DeleteTemplate(userID, templateID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Template deleted successfully",
	})
}

func (h *EmailTemplateHandler) CloneTemplate(c *gin.Context) {
	templateID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid template ID"})
		return
	}

	var req struct {
		Name string `json:"name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := getUserIDFromContext(c)
	template, err := h.templateService.CloneTemplate(userID, templateID, req.Name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success":  true,
		"template": template,
	})
}

func (h *EmailTemplateHandler) PreviewTemplate(c *gin.Context) {
	templateID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid template ID"})
		return
	}

	var req models.EmailTemplatePreviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	req.TemplateID = templateID

	// Create render context
	context := services.TemplateRenderContext{
		Variables: req.Variables,
		Timestamp: time.Now(),
	}

	rendered, err := h.templateService.RenderTemplate(templateID, context)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":  true,
		"rendered": rendered,
	})
}

// Email Provider endpoints

func (h *EmailTemplateHandler) CreateProvider(c *gin.Context) {
	var req models.CreateEmailProviderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := getUserIDFromContext(c)
	provider, err := h.providerService.CreateProvider(userID, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success":  true,
		"provider": provider,
	})
}

func (h *EmailTemplateHandler) ListProviders(c *gin.Context) {
	userID := getUserIDFromContext(c)
	providers, err := h.providerService.ListProviders(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"providers": providers,
	})
}

func (h *EmailTemplateHandler) TestProvider(c *gin.Context) {
	providerID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid provider ID"})
		return
	}

	userID := getUserIDFromContext(c)
	err = h.providerService.TestProvider(userID, providerID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Provider test successful",
	})
}

// Autoresponder endpoints

func (h *EmailTemplateHandler) CreateAutoresponder(c *gin.Context) {
	var req models.CreateAutoresponderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := getUserIDFromContext(c)
	autoresponder, err := h.autoresponderService.CreateAutoresponder(userID, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success":       true,
		"autoresponder": autoresponder,
	})
}

func (h *EmailTemplateHandler) ListAutoresponders(c *gin.Context) {
	userID := getUserIDFromContext(c)

	var formID *uuid.UUID
	if f := c.Query("form_id"); f != "" {
		if fid, err := uuid.Parse(f); err == nil {
			formID = &fid
		}
	}

	autoresponders, err := h.autoresponderService.ListAutoresponders(userID, formID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":        true,
		"autoresponders": autoresponders,
	})
}

func (h *EmailTemplateHandler) ToggleAutoresponder(c *gin.Context) {
	autoresponderID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid autoresponder ID"})
		return
	}

	var req struct {
		Enabled bool `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := getUserIDFromContext(c)
	err = h.autoresponderService.ToggleAutoresponder(userID, autoresponderID, req.Enabled)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	status := "disabled"
	if req.Enabled {
		status = "enabled"
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": fmt.Sprintf("Autoresponder %s successfully", status),
	})
}

// Email Queue endpoints

func (h *EmailTemplateHandler) GetQueueStats(c *gin.Context) {
	userID := getUserIDFromContext(c)
	stats, err := h.queueService.GetQueueStats(&userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"stats":   stats,
	})
}

func (h *EmailTemplateHandler) ListQueuedEmails(c *gin.Context) {
	userID := getUserIDFromContext(c)

	// Parse query parameters
	var status *models.EmailStatus
	if s := c.Query("status"); s != "" {
		st := models.EmailStatus(s)
		status = &st
	}

	limit := 50
	if l := c.Query("limit"); l != "" {
		if parsed, err := parseInt(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	offset := 0
	if o := c.Query("offset"); o != "" {
		if parsed, err := parseInt(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	emails, err := h.queueService.ListQueuedEmails(&userID, status, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"emails":  emails,
	})
}

func (h *EmailTemplateHandler) ProcessQueue(c *gin.Context) {
	result, err := h.queueService.ProcessPendingEmails()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"result":  result,
	})
}

// Analytics endpoints

func (h *EmailTemplateHandler) GetTemplateAnalytics(c *gin.Context) {
	templateID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid template ID"})
		return
	}

	userID := getUserIDFromContext(c)

	// Parse date range
	startDate, endDate := parseDateRange(c)

	report, err := h.analyticsService.GetTemplateAnalytics(userID, templateID, startDate, endDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"report":  report,
	})
}

func (h *EmailTemplateHandler) GetUserAnalytics(c *gin.Context) {
	userID := getUserIDFromContext(c)
	startDate, endDate := parseDateRange(c)

	report, err := h.analyticsService.GetUserAnalytics(userID, startDate, endDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"report":  report,
	})
}

func (h *EmailTemplateHandler) GetTopPerformingTemplates(c *gin.Context) {
	userID := getUserIDFromContext(c)
	startDate, endDate := parseDateRange(c)

	limit := 10
	if l := c.Query("limit"); l != "" {
		if parsed, err := parseInt(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	reports, err := h.analyticsService.GetTopPerformingTemplates(userID, limit, startDate, endDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"templates": reports,
	})
}

// Template Builder endpoints

func (h *EmailTemplateHandler) CreateTemplateDesign(c *gin.Context) {
	var req services.BuilderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := getUserIDFromContext(c)
	design, err := h.builderService.CreateTemplate(userID, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"design":  design,
	})
}

func (h *EmailTemplateHandler) GeneratePreview(c *gin.Context) {
	var req services.PreviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	html, err := h.builderService.GeneratePreview(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, html)
}

func (h *EmailTemplateHandler) GetAvailableComponents(c *gin.Context) {
	components := h.builderService.GetAvailableComponents()

	c.JSON(http.StatusOK, gin.H{
		"success":    true,
		"components": components,
	})
}

// A/B Testing endpoints

func (h *EmailTemplateHandler) CreateABTest(c *gin.Context) {
	var req services.CreateABTestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := getUserIDFromContext(c)
	test, err := h.abTestingService.CreateABTest(userID, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"test":    test,
	})
}

func (h *EmailTemplateHandler) StartABTest(c *gin.Context) {
	testID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid test ID"})
		return
	}

	userID := getUserIDFromContext(c)
	test, err := h.abTestingService.StartABTest(userID, testID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"test":    test,
	})
}

func (h *EmailTemplateHandler) GetABTestResults(c *gin.Context) {
	testID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid test ID"})
		return
	}

	userID := getUserIDFromContext(c)
	result, err := h.abTestingService.GetABTestResults(userID, testID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"result":  result,
	})
}

// Utility functions

func getUserIDFromContext(c *gin.Context) uuid.UUID {
	// This would typically extract user ID from JWT token or session
	// For now, return a dummy UUID
	userIDStr, exists := c.Get("user_id")
	if !exists {
		return uuid.New() // In production, this should be an error
	}
	
	if userID, ok := userIDStr.(uuid.UUID); ok {
		return userID
	}
	
	return uuid.New() // In production, this should be an error
}

func parseDateRange(c *gin.Context) (time.Time, time.Time) {
	defaultStart := time.Now().AddDate(0, -1, 0) // 1 month ago
	defaultEnd := time.Now()

	startStr := c.Query("start_date")
	endStr := c.Query("end_date")

	startDate := defaultStart
	endDate := defaultEnd

	if startStr != "" {
		if parsed, err := time.Parse("2006-01-02", startStr); err == nil {
			startDate = parsed
		}
	}

	if endStr != "" {
		if parsed, err := time.Parse("2006-01-02", endStr); err == nil {
			endDate = parsed
		}
	}

	return startDate, endDate
}

func parseInt(s string) (int, error) {
	result := 0
	for _, r := range s {
		if r < '0' || r > '9' {
			return 0, fmt.Errorf("invalid integer")
		}
		result = result*10 + int(r-'0')
	}
	return result, nil
}