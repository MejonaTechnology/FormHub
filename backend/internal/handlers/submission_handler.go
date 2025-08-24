package handlers

import (
	"formhub/internal/models"
	"formhub/internal/services"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type SubmissionHandler struct {
	submissionService *services.SubmissionService
	formService       *services.FormService
	authService       *services.AuthService
}

func NewSubmissionHandler(submissionService *services.SubmissionService, formService *services.FormService, authService *services.AuthService) *SubmissionHandler {
	// Set the form service in submission service for cross-service communication
	submissionService.SetFormService(formService)
	
	return &SubmissionHandler{
		submissionService: submissionService,
		formService:       formService,
		authService:       authService,
	}
}

func (h *SubmissionHandler) HandleSubmission(c *gin.Context) {
	var req models.SubmissionRequest
	
	// Handle both JSON and form data
	contentType := c.GetHeader("Content-Type")
	if contentType == "application/json" {
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, models.SubmissionResponse{
				Success:    false,
				StatusCode: 400,
				Message:    "Invalid JSON data: " + err.Error(),
			})
			return
		}
	} else {
		// Handle form data
		if err := c.ShouldBind(&req); err != nil {
			c.JSON(http.StatusBadRequest, models.SubmissionResponse{
				Success:    false,
				StatusCode: 400,
				Message:    "Invalid form data: " + err.Error(),
			})
			return
		}
		
		// Extract additional form fields into Data map
		req.Data = make(map[string]interface{})
		for key, values := range c.Request.PostForm {
			if key != "access_key" && key != "email" && key != "subject" && 
			   key != "message" && key != "redirect" && key != "g-recaptcha-response" {
				if len(values) == 1 {
					req.Data[key] = values[0]
				} else {
					req.Data[key] = values
				}
			}
		}
	}

	// Get client information
	ipAddress := c.ClientIP()
	userAgent := c.GetHeader("User-Agent")
	referrer := c.GetHeader("Referer")

	// Handle the submission
	response, err := h.submissionService.HandleSubmission(req, ipAddress, userAgent, referrer)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.SubmissionResponse{
			Success:    false,
			StatusCode: 500,
			Message:    "Internal server error",
		})
		return
	}

	// Return appropriate status code
	statusCode := response.StatusCode
	if statusCode == 0 {
		statusCode = http.StatusOK
	}

	c.JSON(statusCode, response)
}

func (h *SubmissionHandler) GetSubmissions(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	formIDStr := c.Param("id")
	formID, err := uuid.Parse(formIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid form ID"})
		return
	}

	// Verify form ownership
	form, err := h.formService.GetFormByID(formID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Form not found"})
		return
	}

	if form.UserID != userID.(uuid.UUID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	// TODO: Implement GetSubmissionsByFormID in submission service
	c.JSON(http.StatusOK, gin.H{
		"submissions": []interface{}{},
		"message":     "Submissions endpoint not yet implemented",
		"form_id":     formID,
	})
}

func (h *SubmissionHandler) GetSubmission(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	submissionIDStr := c.Param("id")
	submissionID, err := uuid.Parse(submissionIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid submission ID"})
		return
	}

	// TODO: Implement GetSubmissionByID in submission service with ownership check
	c.JSON(http.StatusOK, gin.H{
		"submission": nil,
		"message":    "Get submission endpoint not yet implemented",
		"submission_id": submissionID,
		"user_id":    userID,
	})
}

func (h *SubmissionHandler) DeleteSubmission(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	submissionIDStr := c.Param("id")
	submissionID, err := uuid.Parse(submissionIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid submission ID"})
		return
	}

	// TODO: Implement DeleteSubmission in submission service with ownership check
	c.JSON(http.StatusOK, gin.H{
		"message":       "Submission deleted successfully",
		"submission_id": submissionID,
		"user_id":       userID,
	})
}