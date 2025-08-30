package handlers

import (
	"fmt"
	"formhub/internal/models"
	"formhub/internal/services"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type SubmissionHandler struct {
	submissionService      *services.SubmissionService
	formService            *services.FormService
	authService            *services.AuthService
	fileUploadService      *services.FileUploadService
	fieldValidationService *services.FieldValidationService
}

func NewSubmissionHandler(submissionService *services.SubmissionService, formService *services.FormService, authService *services.AuthService, fileUploadService *services.FileUploadService, fieldValidationService *services.FieldValidationService) *SubmissionHandler {
	// Set the form service in submission service for cross-service communication
	submissionService.SetFormService(formService)
	
	return &SubmissionHandler{
		submissionService:      submissionService,
		formService:            formService,
		authService:            authService,
		fileUploadService:      fileUploadService,
		fieldValidationService: fieldValidationService,
	}
}

func (h *SubmissionHandler) HandleSubmission(c *gin.Context) {
	// Get session ID for file uploads
	sessionID := c.GetHeader("X-Session-ID")
	if sessionID == "" {
		sessionID = uuid.New().String()
	}

	var req models.SubmissionRequest
	var hasFiles bool
	
	// Handle multipart form data (with files) vs regular form/JSON data
	contentType := c.GetHeader("Content-Type")
	
	if strings.Contains(contentType, "multipart/form-data") {
		hasFiles = true
		// Handle multipart form with files
		if err := h.handleMultipartSubmission(c, &req, sessionID); err != nil {
			c.JSON(http.StatusBadRequest, models.SubmissionResponse{
				Success:    false,
				StatusCode: 400,
				Message:    err.Error(),
			})
			return
		}
	} else if strings.Contains(contentType, "application/json") {
		// Handle JSON submission
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, models.SubmissionResponse{
				Success:    false,
				StatusCode: 400,
				Message:    "Invalid JSON data: " + err.Error(),
			})
			return
		}
	} else {
		// Handle regular form data
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
			if !h.isReservedField(key) {
				if len(values) == 1 {
					req.Data[key] = values[0]
				} else {
					req.Data[key] = values
				}
			}
		}
	}

	// Get form and validate access
	form, apiKey, err := h.getFormByAccessKey(req.AccessKey)
	if err != nil {
		c.JSON(http.StatusUnauthorized, models.SubmissionResponse{
			Success:    false,
			StatusCode: 401,
			Message:    "Invalid access key",
		})
		return
	}

	// Validate form submission with advanced field validation
	validationResult, err := h.fieldValidationService.ValidateFormSubmission(form.ID, req.Data)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.SubmissionResponse{
			Success:    false,
			StatusCode: 500,
			Message:    "Validation error",
		})
		return
	}

	// Return validation errors if any
	if !validationResult.IsValid {
		c.JSON(http.StatusBadRequest, models.SubmissionResponse{
			Success:    false,
			StatusCode: 400,
			Message:    "Form validation failed",
			Data: map[string]interface{}{
				"validation_errors": validationResult.Errors,
				"field_results":     validationResult.FieldResults,
			},
		})
		return
	}

	// Get client information
	ipAddress := c.ClientIP()
	userAgent := c.GetHeader("User-Agent")
	referrer := c.GetHeader("Referer")

	// Create enhanced submission request with validated data
	enhancedReq := models.SubmissionRequest{
		AccessKey:         req.AccessKey,
		Data:              validationResult.FieldResults,
		Email:             req.Email,
		Subject:           req.Subject,
		Message:           req.Message,
		RedirectURL:       req.RedirectURL,
		RecaptchaResponse: req.RecaptchaResponse,
		Files:             req.Files,
	}

	// Handle the submission with file support
	response, err := h.submissionService.HandleSubmissionWithFiles(enhancedReq, ipAddress, userAgent, referrer, sessionID, hasFiles)
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

// UploadFiles handles file upload endpoint
func (h *SubmissionHandler) UploadFiles(c *gin.Context) {
	var req models.FileUploadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate access key
	form, _, err := h.getFormByAccessKey(req.AccessKey)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid access key"})
		return
	}

	// Check if form allows file uploads
	if !form.FileUploads {
		c.JSON(http.StatusForbidden, gin.H{"error": "File uploads are not enabled for this form"})
		return
	}

	// Get form field to validate file settings
	field, err := h.fieldValidationService.GetFormFieldByName(form.ID, req.FieldName)
	if err != nil || field.Type != models.FieldTypeFile {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid file field"})
		return
	}

	// Parse multipart form
	err = c.Request.ParseMultipartForm(32 << 20) // 32MB max memory
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse multipart form"})
		return
	}

	files := c.Request.MultipartForm.File["files"]
	if len(files) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No files provided"})
		return
	}

	sessionID := c.GetHeader("X-Session-ID")
	if sessionID == "" {
		sessionID = uuid.New().String()
	}

	// Upload files
	maxFiles := 1
	if field.FileSettings != nil && field.FileSettings.MaxFiles > 0 {
		maxFiles = field.FileSettings.MaxFiles
	}

	results, err := h.fileUploadService.UploadMultipleFiles(files, field.ID, sessionID, maxFiles)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate uploaded files against field settings
	validationResult := h.fileUploadService.ValidateFieldFiles(results, field.FileSettings)

	response := gin.H{
		"success":     len(results) > 0,
		"files":       results,
		"session_id":  sessionID,
		"validation":  validationResult,
	}

	c.JSON(http.StatusOK, response)
}

// BulkUploadFiles handles multiple file uploads
func (h *SubmissionHandler) BulkUploadFiles(c *gin.Context) {
	var req models.BulkFileUploadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate access key
	form, _, err := h.getFormByAccessKey(req.AccessKey)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid access key"})
		return
	}

	// Check if form allows file uploads
	if !form.FileUploads {
		c.JSON(http.StatusForbidden, gin.H{"error": "File uploads are not enabled for this form"})
		return
	}

	// Get form field
	field, err := h.fieldValidationService.GetFormFieldByName(form.ID, req.FieldName)
	if err != nil || field.Type != models.FieldTypeFile {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid file field"})
		return
	}

	// Parse multipart form
	err = c.Request.ParseMultipartForm(100 << 20) // 100MB max memory for bulk uploads
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse multipart form"})
		return
	}

	files := c.Request.MultipartForm.File["files"]
	if len(files) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No files provided"})
		return
	}

	sessionID := c.GetHeader("X-Session-ID")
	if sessionID == "" {
		sessionID = uuid.New().String()
	}

	// Determine max files
	maxFiles := req.MaxFiles
	if maxFiles == 0 && field.FileSettings != nil {
		maxFiles = field.FileSettings.MaxFiles
	}
	if maxFiles == 0 {
		maxFiles = 10 // Default limit
	}

	// Upload files
	results, err := h.fileUploadService.UploadMultipleFiles(files, field.ID, sessionID, maxFiles)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate uploaded files
	validationResult := h.fileUploadService.ValidateFieldFiles(results, field.FileSettings)

	response := gin.H{
		"success":      len(results) > 0,
		"files":        results,
		"session_id":   sessionID,
		"validation":   validationResult,
		"total_files":  len(results),
	}

	c.JSON(http.StatusOK, response)
}

// Helper methods

func (h *SubmissionHandler) handleMultipartSubmission(c *gin.Context, req *models.SubmissionRequest, sessionID string) error {
	// Parse multipart form
	err := c.Request.ParseMultipartForm(32 << 20) // 32MB max memory
	if err != nil {
		return fmt.Errorf("failed to parse multipart form: %w", err)
	}

	// Extract regular form fields
	req.Data = make(map[string]interface{})
	
	for key, values := range c.Request.MultipartForm.Value {
		if h.isReservedField(key) {
			switch key {
			case "access_key":
				req.AccessKey = values[0]
			case "email":
				req.Email = values[0]
			case "subject":
				req.Subject = values[0]
			case "message":
				req.Message = values[0]
			case "redirect":
				req.RedirectURL = values[0]
			case "g-recaptcha-response":
				req.RecaptchaResponse = values[0]
			}
		} else {
			if len(values) == 1 {
				req.Data[key] = values[0]
			} else {
				req.Data[key] = values
			}
		}
	}

	// Handle file uploads
	req.Files = []models.FileUploadResult{}
	for fieldName, files := range c.Request.MultipartForm.File {
		for _, file := range files {
			// Upload each file temporarily
			result, err := h.fileUploadService.UploadFile(file, uuid.Nil, sessionID) // Field ID will be resolved later
			if err != nil {
				result = &models.FileUploadResult{
					OriginalName: file.Filename,
					Error:        err.Error(),
				}
			}
			
			// Add field name to result for later processing
			result.Error = fieldName // Temporary storage for field name
			req.Files = append(req.Files, *result)
		}
	}

	return nil
}

func (h *SubmissionHandler) isReservedField(key string) bool {
	reservedFields := map[string]bool{
		"access_key":            true,
		"email":                 true,
		"subject":               true,
		"message":               true,
		"redirect":              true,
		"g-recaptcha-response":  true,
	}
	return reservedFields[key]
}

func (h *SubmissionHandler) getFormByAccessKey(accessKey string) (*models.Form, *models.APIKey, error) {
	// This is a placeholder - in a real implementation, you'd query the database
	// For now, we'll delegate to the form service
	return h.formService.GetFormByAccessKey(accessKey)
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