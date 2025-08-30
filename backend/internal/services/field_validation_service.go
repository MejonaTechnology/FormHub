package services

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"formhub/internal/models"
	"net/mail"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

type FieldValidationService struct {
	db *sql.DB
}

func NewFieldValidationService(db *sql.DB) *FieldValidationService {
	return &FieldValidationService{
		db: db,
	}
}

// ValidateFormSubmission validates all fields in a form submission
func (s *FieldValidationService) ValidateFormSubmission(formID uuid.UUID, data map[string]interface{}) (*models.FormValidationResult, error) {
	// Get form fields
	fields, err := s.getFormFields(formID)
	if err != nil {
		return nil, fmt.Errorf("failed to get form fields: %w", err)
	}

	result := &models.FormValidationResult{
		IsValid:      true,
		Errors:       make(map[string][]string),
		FieldResults: make(map[string]models.FieldValidationResult),
	}

	// Validate each configured field
	for _, field := range fields {
		fieldResult := s.validateField(field, data[field.Name])
		result.FieldResults[field.Name] = fieldResult

		if !fieldResult.IsValid {
			result.IsValid = false
			result.Errors[field.Name] = fieldResult.Errors
		}
	}

	// Check for required fields that are missing
	for _, field := range fields {
		if field.Required {
			value, exists := data[field.Name]
			if !exists || s.isEmpty(value) {
				result.IsValid = false
				if result.Errors[field.Name] == nil {
					result.Errors[field.Name] = []string{}
				}
				result.Errors[field.Name] = append(result.Errors[field.Name], "This field is required")
			}
		}
	}

	return result, nil
}

// ValidateField validates a single form field
func (s *FieldValidationService) validateField(field models.FormField, value interface{}) models.FieldValidationResult {
	result := models.FieldValidationResult{
		IsValid: true,
		Value:   value,
		Errors:  []string{},
	}

	// Skip validation if field is not required and value is empty
	if !field.Required && s.isEmpty(value) {
		return result
	}

	// Convert value to string for validation
	valueStr := s.convertToString(value)

	// Type-specific validation
	switch field.Type {
	case models.FieldTypeEmail:
		s.validateEmail(&result, valueStr)
	case models.FieldTypeNumber:
		s.validateNumber(&result, valueStr, field.Validation)
	case models.FieldTypeDate:
		s.validateDate(&result, valueStr)
	case models.FieldTypeTime:
		s.validateTime(&result, valueStr)
	case models.FieldTypeDateTime:
		s.validateDateTime(&result, valueStr)
	case models.FieldTypeURL:
		s.validateURL(&result, valueStr)
	case models.FieldTypeTel:
		s.validatePhone(&result, valueStr)
	case models.FieldTypeText, models.FieldTypeTextarea, models.FieldTypePassword:
		s.validateText(&result, valueStr, field.Validation)
	case models.FieldTypeSelect, models.FieldTypeRadio:
		s.validateSelection(&result, valueStr, field.Options)
	case models.FieldTypeCheckbox:
		s.validateCheckbox(&result, value, field.Options)
	case models.FieldTypeHidden:
		// Hidden fields generally don't need validation beyond basic checks
		s.validateHidden(&result, valueStr, field.Validation)
	}

	// Apply general validation rules
	s.applyGeneralValidation(&result, valueStr, field.Validation)

	return result
}

// Type-specific validation methods

func (s *FieldValidationService) validateEmail(result *models.FieldValidationResult, value string) {
	if value == "" {
		return
	}

	_, err := mail.ParseAddress(value)
	if err != nil {
		result.IsValid = false
		result.Errors = append(result.Errors, "Please enter a valid email address")
	}
}

func (s *FieldValidationService) validateNumber(result *models.FieldValidationResult, value string, validation models.FormFieldValidation) {
	if value == "" {
		return
	}

	num, err := strconv.ParseFloat(value, 64)
	if err != nil {
		result.IsValid = false
		result.Errors = append(result.Errors, "Please enter a valid number")
		return
	}

	result.Value = num

	// Check min/max values
	if validation.MinValue != nil && num < *validation.MinValue {
		result.IsValid = false
		result.Errors = append(result.Errors, fmt.Sprintf("Value must be at least %.2f", *validation.MinValue))
	}

	if validation.MaxValue != nil && num > *validation.MaxValue {
		result.IsValid = false
		result.Errors = append(result.Errors, fmt.Sprintf("Value must not exceed %.2f", *validation.MaxValue))
	}
}

func (s *FieldValidationService) validateDate(result *models.FieldValidationResult, value string) {
	if value == "" {
		return
	}

	// Try multiple date formats
	formats := []string{
		"2006-01-02",
		"01/02/2006",
		"02/01/2006",
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02 15:04:05",
	}

	var parsedDate time.Time
	var err error

	for _, format := range formats {
		parsedDate, err = time.Parse(format, value)
		if err == nil {
			break
		}
	}

	if err != nil {
		result.IsValid = false
		result.Errors = append(result.Errors, "Please enter a valid date (YYYY-MM-DD)")
		return
	}

	result.Value = parsedDate.Format("2006-01-02")
}

func (s *FieldValidationService) validateTime(result *models.FieldValidationResult, value string) {
	if value == "" {
		return
	}

	// Try multiple time formats
	formats := []string{
		"15:04",
		"15:04:05",
		"3:04 PM",
		"3:04:05 PM",
	}

	var err error
	for _, format := range formats {
		_, err = time.Parse(format, value)
		if err == nil {
			break
		}
	}

	if err != nil {
		result.IsValid = false
		result.Errors = append(result.Errors, "Please enter a valid time (HH:MM)")
	}
}

func (s *FieldValidationService) validateDateTime(result *models.FieldValidationResult, value string) {
	if value == "" {
		return
	}

	// Try multiple datetime formats
	formats := []string{
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
		"01/02/2006 15:04:05",
	}

	var parsedDateTime time.Time
	var err error

	for _, format := range formats {
		parsedDateTime, err = time.Parse(format, value)
		if err == nil {
			break
		}
	}

	if err != nil {
		result.IsValid = false
		result.Errors = append(result.Errors, "Please enter a valid date and time")
		return
	}

	result.Value = parsedDateTime.Format("2006-01-02T15:04:05Z07:00")
}

func (s *FieldValidationService) validateURL(result *models.FieldValidationResult, value string) {
	if value == "" {
		return
	}

	// Add protocol if missing
	if !strings.HasPrefix(value, "http://") && !strings.HasPrefix(value, "https://") {
		value = "https://" + value
	}

	_, err := url.ParseRequestURI(value)
	if err != nil {
		result.IsValid = false
		result.Errors = append(result.Errors, "Please enter a valid URL")
	} else {
		result.Value = value
	}
}

func (s *FieldValidationService) validatePhone(result *models.FieldValidationResult, value string) {
	if value == "" {
		return
	}

	// Remove common phone number formatting
	cleaned := strings.ReplaceAll(value, " ", "")
	cleaned = strings.ReplaceAll(cleaned, "-", "")
	cleaned = strings.ReplaceAll(cleaned, "(", "")
	cleaned = strings.ReplaceAll(cleaned, ")", "")
	cleaned = strings.ReplaceAll(cleaned, ".", "")

	// Basic phone number validation (10-15 digits, optionally starting with +)
	phoneRegex := regexp.MustCompile(`^\+?[1-9]\d{9,14}$`)
	if !phoneRegex.MatchString(cleaned) {
		result.IsValid = false
		result.Errors = append(result.Errors, "Please enter a valid phone number")
	}
}

func (s *FieldValidationService) validateText(result *models.FieldValidationResult, value string, validation models.FormFieldValidation) {
	// Length validation is handled in applyGeneralValidation
	// This method can be extended for text-specific validations
}

func (s *FieldValidationService) validateSelection(result *models.FieldValidationResult, value string, options []models.FormFieldOption) {
	if value == "" {
		return
	}

	// Check if value is in allowed options
	valid := false
	for _, option := range options {
		if option.Value == value {
			valid = true
			break
		}
	}

	if !valid {
		result.IsValid = false
		result.Errors = append(result.Errors, "Please select a valid option")
	}
}

func (s *FieldValidationService) validateCheckbox(result *models.FieldValidationResult, value interface{}, options []models.FormFieldOption) {
	// Handle both single checkbox and checkbox group
	switch v := value.(type) {
	case string:
		// Single checkbox
		if v != "" && v != "true" && v != "false" && v != "1" && v != "0" {
			s.validateSelection(result, v, options)
		}
	case []interface{}:
		// Multiple checkboxes
		for _, item := range v {
			if itemStr, ok := item.(string); ok {
				valid := false
				for _, option := range options {
					if option.Value == itemStr {
						valid = true
						break
					}
				}
				if !valid {
					result.IsValid = false
					result.Errors = append(result.Errors, fmt.Sprintf("'%s' is not a valid option", itemStr))
				}
			}
		}
	}
}

func (s *FieldValidationService) validateHidden(result *models.FieldValidationResult, value string, validation models.FormFieldValidation) {
	// Hidden fields might have specific validation patterns or expected values
	// This is useful for honeypot fields or security tokens
}

// Apply general validation rules
func (s *FieldValidationService) applyGeneralValidation(result *models.FieldValidationResult, value string, validation models.FormFieldValidation) {
	if !result.IsValid {
		return // Skip if already invalid
	}

	// Length validation
	if validation.MinLength != nil && len(value) < *validation.MinLength {
		result.IsValid = false
		errorMsg := fmt.Sprintf("Must be at least %d characters long", *validation.MinLength)
		if validation.CustomError != "" {
			errorMsg = validation.CustomError
		}
		result.Errors = append(result.Errors, errorMsg)
	}

	if validation.MaxLength != nil && len(value) > *validation.MaxLength {
		result.IsValid = false
		errorMsg := fmt.Sprintf("Must not exceed %d characters", *validation.MaxLength)
		if validation.CustomError != "" {
			errorMsg = validation.CustomError
		}
		result.Errors = append(result.Errors, errorMsg)
	}

	// Pattern validation
	if validation.Pattern != "" && value != "" {
		regex, err := regexp.Compile(validation.Pattern)
		if err == nil && !regex.MatchString(value) {
			result.IsValid = false
			errorMsg := "Invalid format"
			if validation.CustomError != "" {
				errorMsg = validation.CustomError
			}
			result.Errors = append(result.Errors, errorMsg)
		}
	}
}

// Helper methods

func (s *FieldValidationService) isEmpty(value interface{}) bool {
	if value == nil {
		return true
	}

	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v) == ""
	case []interface{}:
		return len(v) == 0
	default:
		return false
	}
}

func (s *FieldValidationService) convertToString(value interface{}) string {
	if value == nil {
		return ""
	}

	switch v := value.(type) {
	case string:
		return v
	case int, int64, float64:
		return fmt.Sprintf("%v", v)
	case bool:
		return strconv.FormatBool(v)
	default:
		return fmt.Sprintf("%v", v)
	}
}

func (s *FieldValidationService) getFormFields(formID uuid.UUID) ([]models.FormField, error) {
	query := `
		SELECT id, form_id, name, label, type, required, placeholder, default_value,
			validation, file_settings, field_order, is_active, created_at, updated_at
		FROM form_fields 
		WHERE form_id = ? AND is_active = true 
		ORDER BY field_order, created_at
	`

	rows, err := s.db.Query(query, formID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var fields []models.FormField
	for rows.Next() {
		var field models.FormField
		var validationJSON, fileSettingsJSON sql.NullString

		err := rows.Scan(
			&field.ID, &field.FormID, &field.Name, &field.Label, &field.Type,
			&field.Required, &field.Placeholder, &field.DefaultValue,
			&validationJSON, &fileSettingsJSON, &field.Order, &field.IsActive,
			&field.CreatedAt, &field.UpdatedAt,
		)
		if err != nil {
			continue
		}

		// Parse validation JSON
		if validationJSON.Valid && validationJSON.String != "" {
			json.Unmarshal([]byte(validationJSON.String), &field.Validation)
		}

		// Parse file settings JSON
		if fileSettingsJSON.Valid && fileSettingsJSON.String != "" {
			json.Unmarshal([]byte(fileSettingsJSON.String), &field.FileSettings)
		}

		// Get field options for select, radio, checkbox fields
		if field.Type == models.FieldTypeSelect || field.Type == models.FieldTypeRadio || field.Type == models.FieldTypeCheckbox {
			options, err := s.getFieldOptions(field.ID)
			if err == nil {
				field.Options = options
			}
		}

		fields = append(fields, field)
	}

	return fields, nil
}

func (s *FieldValidationService) getFieldOptions(fieldID uuid.UUID) ([]models.FormFieldOption, error) {
	query := `
		SELECT id, field_id, label, value, selected, option_order
		FROM form_field_options 
		WHERE field_id = ? 
		ORDER BY option_order, label
	`

	rows, err := s.db.Query(query, fieldID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var options []models.FormFieldOption
	for rows.Next() {
		var option models.FormFieldOption
		err := rows.Scan(
			&option.ID, &option.FieldID, &option.Label,
			&option.Value, &option.Selected, &option.Order,
		)
		if err != nil {
			continue
		}
		options = append(options, option)
	}

	return options, nil
}

// GetFormFieldByName retrieves a specific form field by name
func (s *FieldValidationService) GetFormFieldByName(formID uuid.UUID, fieldName string) (*models.FormField, error) {
	query := `
		SELECT id, form_id, name, label, type, required, placeholder, default_value,
			validation, file_settings, field_order, is_active, created_at, updated_at
		FROM form_fields 
		WHERE form_id = ? AND name = ? AND is_active = true
	`

	var field models.FormField
	var validationJSON, fileSettingsJSON sql.NullString

	err := s.db.QueryRow(query, formID, fieldName).Scan(
		&field.ID, &field.FormID, &field.Name, &field.Label, &field.Type,
		&field.Required, &field.Placeholder, &field.DefaultValue,
		&validationJSON, &fileSettingsJSON, &field.Order, &field.IsActive,
		&field.CreatedAt, &field.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	// Parse validation JSON
	if validationJSON.Valid && validationJSON.String != "" {
		json.Unmarshal([]byte(validationJSON.String), &field.Validation)
	}

	// Parse file settings JSON
	if fileSettingsJSON.Valid && fileSettingsJSON.String != "" {
		json.Unmarshal([]byte(fileSettingsJSON.String), &field.FileSettings)
	}

	// Get field options if applicable
	if field.Type == models.FieldTypeSelect || field.Type == models.FieldTypeRadio || field.Type == models.FieldTypeCheckbox {
		options, err := s.getFieldOptions(field.ID)
		if err == nil {
			field.Options = options
		}
	}

	return &field, nil
}

// ValidateIndividualField validates a single field without form context
func (s *FieldValidationService) ValidateIndividualField(fieldType models.FormFieldType, value interface{}, validation models.FormFieldValidation, required bool) models.FieldValidationResult {
	field := models.FormField{
		Type:       fieldType,
		Required:   required,
		Validation: validation,
	}

	return s.validateField(field, value)
}