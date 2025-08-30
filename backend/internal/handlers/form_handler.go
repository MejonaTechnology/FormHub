package handlers

import (
	"formhub/internal/models"
	"formhub/internal/services"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type FormHandler struct {
	formService            *services.FormService
	authService            *services.AuthService
	fieldValidationService *services.FieldValidationService
	fileUploadService      *services.FileUploadService
}

func NewFormHandler(formService *services.FormService, authService *services.AuthService, fieldValidationService *services.FieldValidationService, fileUploadService *services.FileUploadService) *FormHandler {
	return &FormHandler{
		formService:            formService,
		authService:            authService,
		fieldValidationService: fieldValidationService,
		fileUploadService:      fileUploadService,
	}
}

func (h *FormHandler) GetForms(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	forms, err := h.formService.GetUserForms(userID.(uuid.UUID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get forms"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"forms": forms})
}

func (h *FormHandler) CreateForm(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var req models.CreateFormRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	form, err := h.formService.CreateForm(userID.(uuid.UUID), req)
	if err != nil {
		if err.Error() == "form limit reached for plan free" || 
		   err.Error() == "form limit reached for plan starter" {
			c.JSON(http.StatusPaymentRequired, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Form created successfully",
		"form": form,
	})
}

func (h *FormHandler) GetForm(c *gin.Context) {
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

	form, err := h.formService.GetFormByID(formID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Form not found"})
		return
	}

	// Check ownership
	if form.UserID != userID.(uuid.UUID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"form": form})
}

func (h *FormHandler) UpdateForm(c *gin.Context) {
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

	var req models.CreateFormRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	form, err := h.formService.UpdateForm(formID, userID.(uuid.UUID), req)
	if err != nil {
		if err.Error() == "unauthorized" {
			c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Form updated successfully",
		"form": form,
	})
}

func (h *FormHandler) DeleteForm(c *gin.Context) {
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

	if err := h.formService.DeleteForm(formID, userID.(uuid.UUID)); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Form deleted successfully"})
}

// CreateFormWithFields creates a new form with custom field configuration
func (h *FormHandler) CreateFormWithFields(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var req models.CreateFormWithFieldsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Create the base form first
	form, err := h.formService.CreateForm(userID.(uuid.UUID), req.CreateFormRequest)
	if err != nil {
		if err.Error() == "form limit reached for plan free" || 
		   err.Error() == "form limit reached for plan starter" {
			c.JSON(http.StatusPaymentRequired, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Add custom fields if provided
	if len(req.Fields) > 0 {
		createdFields, err := h.formService.CreateFormFields(form.ID, req.Fields)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Form created but failed to add custom fields: " + err.Error(),
				"form": form,
			})
			return
		}

		// Return form with fields
		c.JSON(http.StatusCreated, gin.H{
			"message": "Form with custom fields created successfully",
			"form": models.FormWithFields{
				Form:   *form,
				Fields: createdFields,
			},
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Form created successfully",
		"form": form,
	})
}

// GetFormWithFields retrieves a form with its field configuration
func (h *FormHandler) GetFormWithFields(c *gin.Context) {
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

	// Get form
	form, err := h.formService.GetFormByID(formID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Form not found"})
		return
	}

	// Check ownership
	if form.UserID != userID.(uuid.UUID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	// Get form fields
	fields, err := h.formService.GetFormFields(formID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get form fields"})
		return
	}

	formWithFields := models.FormWithFields{
		Form:   *form,
		Fields: fields,
	}

	c.JSON(http.StatusOK, gin.H{"form": formWithFields})
}

// AddFormField adds a new field to an existing form
func (h *FormHandler) AddFormField(c *gin.Context) {
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

	var req models.CreateFormFieldRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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

	// Create the field
	field, err := h.formService.CreateFormField(formID, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Form field created successfully",
		"field": field,
	})
}

// UpdateFormField updates an existing form field
func (h *FormHandler) UpdateFormField(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	formIDStr := c.Param("id")
	fieldIDStr := c.Param("field_id")
	
	formID, err := uuid.Parse(formIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid form ID"})
		return
	}

	fieldID, err := uuid.Parse(fieldIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid field ID"})
		return
	}

	var req models.CreateFormFieldRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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

	// Update the field
	field, err := h.formService.UpdateFormField(fieldID, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Form field updated successfully",
		"field": field,
	})
}

// DeleteFormField removes a field from a form
func (h *FormHandler) DeleteFormField(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	formIDStr := c.Param("id")
	fieldIDStr := c.Param("field_id")
	
	formID, err := uuid.Parse(formIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid form ID"})
		return
	}

	fieldID, err := uuid.Parse(fieldIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid field ID"})
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

	// Delete the field
	if err := h.formService.DeleteFormField(fieldID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Form field deleted successfully"})
}

// ReorderFormFields updates the order of form fields
func (h *FormHandler) ReorderFormFields(c *gin.Context) {
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

	var req struct {
		FieldOrder []struct {
			ID    uuid.UUID `json:"id" binding:"required"`
			Order int       `json:"order" binding:"required"`
		} `json:"field_order" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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

	// Update field orders
	if err := h.formService.UpdateFieldOrder(formID, req.FieldOrder); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Form field order updated successfully"})
}

// ValidateFormFields validates form field configuration
func (h *FormHandler) ValidateFormFields(c *gin.Context) {
	formIDStr := c.Param("id")
	formID, err := uuid.Parse(formIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid form ID"})
		return
	}

	var req struct {
		Data map[string]interface{} `json:"data" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate the form submission
	validationResult, err := h.fieldValidationService.ValidateFormSubmission(formID, req.Data)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Validation error: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"validation_result": validationResult,
	})
}

// GetFormFieldTypes returns available form field types and their configurations
func (h *FormHandler) GetFormFieldTypes(c *gin.Context) {
	fieldTypes := map[string]interface{}{
		"text": map[string]interface{}{
			"label":        "Text Input",
			"description":  "Single line text input",
			"validations":  []string{"minLength", "maxLength", "pattern"},
			"supports_placeholder": true,
			"supports_default":     true,
		},
		"email": map[string]interface{}{
			"label":        "Email Input",
			"description":  "Email address input with validation",
			"validations":  []string{"email_format"},
			"supports_placeholder": true,
			"supports_default":     true,
		},
		"number": map[string]interface{}{
			"label":        "Number Input",
			"description":  "Numeric input field",
			"validations":  []string{"minValue", "maxValue"},
			"supports_placeholder": true,
			"supports_default":     true,
		},
		"date": map[string]interface{}{
			"label":        "Date Input",
			"description":  "Date picker input",
			"validations":  []string{"date_format"},
			"supports_placeholder": false,
			"supports_default":     true,
		},
		"time": map[string]interface{}{
			"label":        "Time Input",
			"description":  "Time picker input",
			"validations":  []string{"time_format"},
			"supports_placeholder": false,
			"supports_default":     true,
		},
		"datetime": map[string]interface{}{
			"label":        "Date & Time Input",
			"description":  "Combined date and time picker",
			"validations":  []string{"datetime_format"},
			"supports_placeholder": false,
			"supports_default":     true,
		},
		"url": map[string]interface{}{
			"label":        "URL Input",
			"description":  "URL/website address input",
			"validations":  []string{"url_format"},
			"supports_placeholder": true,
			"supports_default":     true,
		},
		"tel": map[string]interface{}{
			"label":        "Phone Input",
			"description":  "Phone number input",
			"validations":  []string{"phone_format"},
			"supports_placeholder": true,
			"supports_default":     true,
		},
		"textarea": map[string]interface{}{
			"label":        "Text Area",
			"description":  "Multi-line text input",
			"validations":  []string{"minLength", "maxLength"},
			"supports_placeholder": true,
			"supports_default":     true,
		},
		"select": map[string]interface{}{
			"label":        "Dropdown Select",
			"description":  "Single selection dropdown",
			"validations":  []string{"valid_option"},
			"requires_options":     true,
			"supports_placeholder": false,
			"supports_default":     true,
		},
		"radio": map[string]interface{}{
			"label":        "Radio Buttons",
			"description":  "Single selection radio group",
			"validations":  []string{"valid_option"},
			"requires_options":     true,
			"supports_placeholder": false,
			"supports_default":     true,
		},
		"checkbox": map[string]interface{}{
			"label":        "Checkboxes",
			"description":  "Multiple selection checkboxes",
			"validations":  []string{"valid_options"},
			"requires_options":     true,
			"supports_placeholder": false,
			"supports_default":     false,
		},
		"file": map[string]interface{}{
			"label":        "File Upload",
			"description":  "File upload field",
			"validations":  []string{"file_type", "file_size", "file_count"},
			"requires_file_settings": true,
			"supports_placeholder":   false,
			"supports_default":       false,
		},
		"hidden": map[string]interface{}{
			"label":        "Hidden Field",
			"description":  "Hidden field for tracking data",
			"validations":  []string{"pattern"},
			"supports_placeholder": false,
			"supports_default":     true,
		},
		"password": map[string]interface{}{
			"label":        "Password Input",
			"description":  "Password field with masking",
			"validations":  []string{"minLength", "maxLength", "pattern"},
			"supports_placeholder": true,
			"supports_default":     false,
		},
	}

	c.JSON(http.StatusOK, gin.H{
		"field_types": fieldTypes,
		"validation_rules": map[string]interface{}{
			"minLength": "Minimum character length",
			"maxLength": "Maximum character length",
			"minValue":  "Minimum numeric value",
			"maxValue":  "Maximum numeric value",
			"pattern":   "Regular expression pattern",
		},
	})
}