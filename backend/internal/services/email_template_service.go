package services

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"formhub/internal/models"
	"html/template"
	"regexp"
	"strings"
	"text/template/parse"
	"time"

	"github.com/google/uuid"
)

type EmailTemplateService struct {
	db *sql.DB
}

type TemplateVariable struct {
	Name        string      `json:"name"`
	Type        string      `json:"type"` // string, number, boolean, array, object
	Description string      `json:"description"`
	Required    bool        `json:"required"`
	DefaultValue interface{} `json:"default_value,omitempty"`
	Example     interface{} `json:"example,omitempty"`
}

type TemplateRenderContext struct {
	Variables   map[string]interface{} `json:"variables"`
	FormData    map[string]interface{} `json:"form_data"`
	Submission  *models.Submission     `json:"submission,omitempty"`
	Form        *models.Form           `json:"form,omitempty"`
	User        *models.User           `json:"user,omitempty"`
	Timestamp   time.Time              `json:"timestamp"`
	IPAddress   string                 `json:"ip_address"`
	UserAgent   string                 `json:"user_agent"`
	Referrer    string                 `json:"referrer"`
}

type RenderedTemplate struct {
	Subject     string `json:"subject"`
	HTMLContent string `json:"html_content"`
	TextContent string `json:"text_content"`
	Variables   map[string]interface{} `json:"variables_used"`
}

type TemplateValidationResult struct {
	IsValid     bool     `json:"is_valid"`
	Errors      []string `json:"errors,omitempty"`
	Warnings    []string `json:"warnings,omitempty"`
	Variables   []TemplateVariable `json:"variables"`
	MissingVars []string `json:"missing_variables,omitempty"`
}

func NewEmailTemplateService(db *sql.DB) *EmailTemplateService {
	return &EmailTemplateService{
		db: db,
	}
}

// CreateTemplate creates a new email template
func (s *EmailTemplateService) CreateTemplate(userID uuid.UUID, req models.CreateEmailTemplateRequest) (*models.EmailTemplate, error) {
	// Validate template content
	validation := s.ValidateTemplate(req.HTMLContent, req.TextContent, req.Subject)
	if !validation.IsValid {
		return nil, fmt.Errorf("template validation failed: %s", strings.Join(validation.Errors, ", "))
	}

	template := &models.EmailTemplate{
		ID:          uuid.New(),
		UserID:      userID,
		FormID:      req.FormID,
		Name:        req.Name,
		Description: req.Description,
		Type:        req.Type,
		Language:    req.Language,
		Subject:     req.Subject,
		HTMLContent: req.HTMLContent,
		TextContent: req.TextContent,
		Variables:   req.Variables,
		ParentID:    req.ParentID,
		IsActive:    true,
		IsDefault:   false,
		Version:     1,
		Tags:        req.Tags,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Set default language if not provided
	if template.Language == "" {
		template.Language = "en"
	}

	// Auto-detect variables if not provided
	if len(template.Variables) == 0 {
		template.Variables = s.ExtractVariables(req.HTMLContent, req.TextContent, req.Subject)
	}

	// Generate text content if not provided
	if template.TextContent == "" && req.HTMLContent != "" {
		template.TextContent = s.GenerateTextFromHTML(req.HTMLContent)
	}

	// Insert into database
	query := `
		INSERT INTO email_templates (
			id, user_id, form_id, name, description, type, language,
			subject, html_content, text_content, variables, parent_id,
			is_active, is_default, version, tags, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	variablesJSON, _ := json.Marshal(template.Variables)
	tagsJSON, _ := json.Marshal(template.Tags)

	_, err := s.db.Exec(query,
		template.ID, template.UserID, template.FormID, template.Name,
		template.Description, template.Type, template.Language,
		template.Subject, template.HTMLContent, template.TextContent,
		variablesJSON, template.ParentID, template.IsActive,
		template.IsDefault, template.Version, tagsJSON,
		template.CreatedAt, template.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create template: %w", err)
	}

	return template, nil
}

// GetTemplate retrieves a template by ID
func (s *EmailTemplateService) GetTemplate(userID, templateID uuid.UUID) (*models.EmailTemplate, error) {
	query := `
		SELECT id, user_id, form_id, name, description, type, language,
		       subject, html_content, text_content, variables, parent_id,
		       is_active, is_default, version, tags, created_at, updated_at
		FROM email_templates 
		WHERE id = ? AND user_id = ?`

	var template models.EmailTemplate
	var formID, parentID sql.NullString
	var variablesJSON, tagsJSON []byte

	err := s.db.QueryRow(query, templateID, userID).Scan(
		&template.ID, &template.UserID, &formID, &template.Name,
		&template.Description, &template.Type, &template.Language,
		&template.Subject, &template.HTMLContent, &template.TextContent,
		&variablesJSON, &parentID, &template.IsActive,
		&template.IsDefault, &template.Version, &tagsJSON,
		&template.CreatedAt, &template.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get template: %w", err)
	}

	// Parse optional fields
	if formID.Valid {
		if fid, err := uuid.Parse(formID.String); err == nil {
			template.FormID = &fid
		}
	}
	if parentID.Valid {
		if pid, err := uuid.Parse(parentID.String); err == nil {
			template.ParentID = &pid
		}
	}

	// Parse JSON fields
	if len(variablesJSON) > 0 {
		json.Unmarshal(variablesJSON, &template.Variables)
	}
	if len(tagsJSON) > 0 {
		json.Unmarshal(tagsJSON, &template.Tags)
	}

	return &template, nil
}

// ListTemplates retrieves templates for a user with filtering
func (s *EmailTemplateService) ListTemplates(userID uuid.UUID, templateType *models.EmailTemplateType, formID *uuid.UUID, language *string) ([]models.EmailTemplate, error) {
	query := `
		SELECT id, user_id, form_id, name, description, type, language,
		       subject, html_content, text_content, variables, parent_id,
		       is_active, is_default, version, tags, created_at, updated_at
		FROM email_templates 
		WHERE user_id = ? AND is_active = true`
	
	args := []interface{}{userID}

	if templateType != nil {
		query += " AND type = ?"
		args = append(args, *templateType)
	}

	if formID != nil {
		query += " AND (form_id = ? OR form_id IS NULL)"
		args = append(args, *formID)
	}

	if language != nil {
		query += " AND language = ?"
		args = append(args, *language)
	}

	query += " ORDER BY created_at DESC"

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list templates: %w", err)
	}
	defer rows.Close()

	var templates []models.EmailTemplate
	for rows.Next() {
		var template models.EmailTemplate
		var formID, parentID sql.NullString
		var variablesJSON, tagsJSON []byte

		err := rows.Scan(
			&template.ID, &template.UserID, &formID, &template.Name,
			&template.Description, &template.Type, &template.Language,
			&template.Subject, &template.HTMLContent, &template.TextContent,
			&variablesJSON, &parentID, &template.IsActive,
			&template.IsDefault, &template.Version, &tagsJSON,
			&template.CreatedAt, &template.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan template: %w", err)
		}

		// Parse optional fields
		if formID.Valid {
			if fid, err := uuid.Parse(formID.String); err == nil {
				template.FormID = &fid
			}
		}
		if parentID.Valid {
			if pid, err := uuid.Parse(parentID.String); err == nil {
				template.ParentID = &pid
			}
		}

		// Parse JSON fields
		if len(variablesJSON) > 0 {
			json.Unmarshal(variablesJSON, &template.Variables)
		}
		if len(tagsJSON) > 0 {
			json.Unmarshal(tagsJSON, &template.Tags)
		}

		templates = append(templates, template)
	}

	return templates, nil
}

// RenderTemplate renders a template with the provided context
func (s *EmailTemplateService) RenderTemplate(templateID uuid.UUID, context TemplateRenderContext) (*RenderedTemplate, error) {
	// Get template (assuming we have access to userID through context)
	// For now, we'll get it without user restriction - in production you'd want proper access control
	query := `SELECT subject, html_content, text_content, variables FROM email_templates WHERE id = ? AND is_active = true`
	
	var subject, htmlContent, textContent string
	var variablesJSON []byte

	err := s.db.QueryRow(query, templateID).Scan(&subject, &htmlContent, &textContent, &variablesJSON)
	if err != nil {
		return nil, fmt.Errorf("failed to get template for rendering: %w", err)
	}

	// Parse template variables
	var templateVars []string
	if len(variablesJSON) > 0 {
		json.Unmarshal(variablesJSON, &templateVars)
	}

	// Build comprehensive variable map
	renderVars := s.buildRenderVariables(context)

	// Render subject
	renderedSubject, err := s.renderString(subject, renderVars)
	if err != nil {
		return nil, fmt.Errorf("failed to render subject: %w", err)
	}

	// Render HTML content
	renderedHTML, err := s.renderString(htmlContent, renderVars)
	if err != nil {
		return nil, fmt.Errorf("failed to render HTML content: %w", err)
	}

	// Render text content
	renderedText, err := s.renderString(textContent, renderVars)
	if err != nil {
		return nil, fmt.Errorf("failed to render text content: %w", err)
	}

	return &RenderedTemplate{
		Subject:     renderedSubject,
		HTMLContent: renderedHTML,
		TextContent: renderedText,
		Variables:   renderVars,
	}, nil
}

// ValidateTemplate validates template syntax and content
func (s *EmailTemplateService) ValidateTemplate(htmlContent, textContent, subject string) TemplateValidationResult {
	var errors []string
	var warnings []string

	// Validate HTML template syntax
	if htmlContent != "" {
		if _, err := template.New("html").Parse(htmlContent); err != nil {
			errors = append(errors, fmt.Sprintf("HTML template syntax error: %s", err.Error()))
		}
	}

	// Validate text template syntax
	if textContent != "" {
		if _, err := template.New("text").Parse(textContent); err != nil {
			errors = append(errors, fmt.Sprintf("Text template syntax error: %s", err.Error()))
		}
	}

	// Validate subject syntax
	if subject != "" {
		if _, err := template.New("subject").Parse(subject); err != nil {
			errors = append(errors, fmt.Sprintf("Subject template syntax error: %s", err.Error()))
		}
	}

	// Extract variables from all content
	variables := s.ExtractTemplateVariables(htmlContent, textContent, subject)

	// Basic content validation
	if htmlContent == "" && textContent == "" {
		errors = append(errors, "Either HTML content or text content must be provided")
	}

	if subject == "" {
		warnings = append(warnings, "Subject is empty - consider adding a subject line")
	}

	// Check for potentially unsafe content
	if strings.Contains(htmlContent, "<script") {
		warnings = append(warnings, "HTML content contains script tags - ensure content is safe")
	}

	return TemplateValidationResult{
		IsValid:   len(errors) == 0,
		Errors:    errors,
		Warnings:  warnings,
		Variables: variables,
	}
}

// ExtractVariables extracts variable names from template content
func (s *EmailTemplateService) ExtractVariables(htmlContent, textContent, subject string) []string {
	variables := make(map[string]bool)

	// Extract from HTML content
	s.extractVarsFromString(htmlContent, variables)

	// Extract from text content
	s.extractVarsFromString(textContent, variables)

	// Extract from subject
	s.extractVarsFromString(subject, variables)

	// Convert map to slice
	var varList []string
	for v := range variables {
		varList = append(varList, v)
	}

	return varList
}

// ExtractTemplateVariables extracts detailed variable information
func (s *EmailTemplateService) ExtractTemplateVariables(htmlContent, textContent, subject string) []TemplateVariable {
	variables := make(map[string]TemplateVariable)

	// Extract variables and infer types
	s.extractDetailedVarsFromString(htmlContent, variables)
	s.extractDetailedVarsFromString(textContent, variables)
	s.extractDetailedVarsFromString(subject, variables)

	// Convert map to slice
	var varList []TemplateVariable
	for _, v := range variables {
		varList = append(varList, v)
	}

	return varList
}

// GenerateTextFromHTML converts HTML content to plain text
func (s *EmailTemplateService) GenerateTextFromHTML(htmlContent string) string {
	// Remove HTML tags and clean up
	re := regexp.MustCompile(`<[^>]*>`)
	text := re.ReplaceAllString(htmlContent, "")
	
	// Clean up whitespace
	text = regexp.MustCompile(`\s+`).ReplaceAllString(text, " ")
	text = strings.TrimSpace(text)

	// Convert common HTML entities
	text = strings.ReplaceAll(text, "&nbsp;", " ")
	text = strings.ReplaceAll(text, "&lt;", "<")
	text = strings.ReplaceAll(text, "&gt;", ">")
	text = strings.ReplaceAll(text, "&amp;", "&")
	text = strings.ReplaceAll(text, "&quot;", `"`)

	return text
}

// Helper methods

func (s *EmailTemplateService) buildRenderVariables(context TemplateRenderContext) map[string]interface{} {
	vars := make(map[string]interface{})

	// Add context variables
	for k, v := range context.Variables {
		vars[k] = v
	}

	// Add form data
	for k, v := range context.FormData {
		vars[k] = v
	}

	// Add submission data if available
	if context.Submission != nil {
		for k, v := range context.Submission.Data {
			vars[k] = v
		}
		vars["submission_id"] = context.Submission.ID.String()
		vars["ip_address"] = context.Submission.IPAddress
		vars["user_agent"] = context.Submission.UserAgent
		vars["referrer"] = context.Submission.Referrer
		vars["timestamp"] = context.Submission.CreatedAt.Format("2006-01-02 15:04:05")
	}

	// Add form information if available
	if context.Form != nil {
		vars["form_name"] = context.Form.Name
		vars["form_description"] = context.Form.Description
		vars["form_id"] = context.Form.ID.String()
	}

	// Add user information if available
	if context.User != nil {
		vars["user_name"] = context.User.FirstName + " " + context.User.LastName
		vars["user_email"] = context.User.Email
		vars["user_company"] = context.User.Company
	}

	// Add standard variables
	vars["current_date"] = time.Now().Format("2006-01-02")
	vars["current_datetime"] = time.Now().Format("2006-01-02 15:04:05")
	vars["current_year"] = time.Now().Year()

	return vars
}

func (s *EmailTemplateService) renderString(content string, variables map[string]interface{}) (string, error) {
	tmpl, err := template.New("template").Funcs(s.getTemplateFunctions()).Parse(content)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, variables)
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}

func (s *EmailTemplateService) getTemplateFunctions() template.FuncMap {
	return template.FuncMap{
		"upper":    strings.ToUpper,
		"lower":    strings.ToLower,
		"title":    strings.Title,
		"trim":     strings.TrimSpace,
		"contains": strings.Contains,
		"replace":  strings.ReplaceAll,
		"split":    strings.Split,
		"join":     strings.Join,
		"now":      time.Now,
		"format_date": func(format string, date interface{}) string {
			if t, ok := date.(time.Time); ok {
				return t.Format(format)
			}
			return ""
		},
		"default": func(defaultVal interface{}, val interface{}) interface{} {
			if val == nil || val == "" {
				return defaultVal
			}
			return val
		},
	}
}

func (s *EmailTemplateService) extractVarsFromString(content string, variables map[string]bool) {
	// Extract {{variable}} patterns
	re := regexp.MustCompile(`\{\{\.?(\w+)\}\}`)
	matches := re.FindAllStringSubmatch(content, -1)
	for _, match := range matches {
		if len(match) > 1 {
			variables[match[1]] = true
		}
	}

	// Extract {{#each variable}} patterns
	re = regexp.MustCompile(`\{\{#each\s+\.?(\w+)\}\}`)
	matches = re.FindAllStringSubmatch(content, -1)
	for _, match := range matches {
		if len(match) > 1 {
			variables[match[1]] = true
		}
	}

	// Extract {{#if variable}} patterns
	re = regexp.MustCompile(`\{\{#if\s+\.?(\w+)\}\}`)
	matches = re.FindAllStringSubmatch(content, -1)
	for _, match := range matches {
		if len(match) > 1 {
			variables[match[1]] = true
		}
	}
}

func (s *EmailTemplateService) extractDetailedVarsFromString(content string, variables map[string]TemplateVariable) {
	// Extract {{variable}} patterns and infer types
	re := regexp.MustCompile(`\{\{\.?(\w+)\}\}`)
	matches := re.FindAllStringSubmatch(content, -1)
	for _, match := range matches {
		if len(match) > 1 {
			varName := match[1]
			if _, exists := variables[varName]; !exists {
				variables[varName] = TemplateVariable{
					Name:        varName,
					Type:        s.inferVariableType(varName),
					Description: s.generateVariableDescription(varName),
					Required:    true,
				}
			}
		}
	}

	// Extract collection variables from {{#each}}
	re = regexp.MustCompile(`\{\{#each\s+\.?(\w+)\}\}`)
	matches = re.FindAllStringSubmatch(content, -1)
	for _, match := range matches {
		if len(match) > 1 {
			varName := match[1]
			if _, exists := variables[varName]; !exists {
				variables[varName] = TemplateVariable{
					Name:        varName,
					Type:        "array",
					Description: s.generateVariableDescription(varName),
					Required:    true,
				}
			}
		}
	}
}

func (s *EmailTemplateService) inferVariableType(varName string) string {
	// Simple type inference based on variable name patterns
	switch {
	case strings.HasSuffix(varName, "_count") || strings.HasSuffix(varName, "_number"):
		return "number"
	case strings.HasSuffix(varName, "_date") || strings.HasSuffix(varName, "_time"):
		return "string" // We'll format dates as strings
	case strings.HasSuffix(varName, "_list") || strings.HasSuffix(varName, "_data") || strings.Contains(varName, "submission"):
		return "array"
	case strings.HasSuffix(varName, "_enabled") || strings.HasSuffix(varName, "_active"):
		return "boolean"
	default:
		return "string"
	}
}

func (s *EmailTemplateService) generateVariableDescription(varName string) string {
	// Generate helpful descriptions based on variable names
	descriptions := map[string]string{
		"name":             "User's full name",
		"email":            "User's email address",
		"message":          "Message content from the form",
		"subject":          "Email subject line",
		"form_name":        "Name of the form",
		"form_description": "Description of the form",
		"timestamp":        "Submission timestamp",
		"ip_address":       "User's IP address",
		"user_agent":       "User's browser information",
		"referrer":         "Page that referred the user",
		"submission_data":  "All form submission data",
		"company":          "User's company name",
		"phone":            "User's phone number",
		"website":          "User's website URL",
	}

	if desc, exists := descriptions[varName]; exists {
		return desc
	}

	// Generate description from variable name
	return strings.Title(strings.ReplaceAll(varName, "_", " "))
}

// UpdateTemplate updates an existing template
func (s *EmailTemplateService) UpdateTemplate(userID, templateID uuid.UUID, req models.CreateEmailTemplateRequest) (*models.EmailTemplate, error) {
	// Validate template content
	validation := s.ValidateTemplate(req.HTMLContent, req.TextContent, req.Subject)
	if !validation.IsValid {
		return nil, fmt.Errorf("template validation failed: %s", strings.Join(validation.Errors, ", "))
	}

	// Auto-detect variables if not provided
	variables := req.Variables
	if len(variables) == 0 {
		variables = s.ExtractVariables(req.HTMLContent, req.TextContent, req.Subject)
	}

	// Generate text content if not provided
	textContent := req.TextContent
	if textContent == "" && req.HTMLContent != "" {
		textContent = s.GenerateTextFromHTML(req.HTMLContent)
	}

	// Update template
	query := `
		UPDATE email_templates SET
			name = ?, description = ?, type = ?, language = ?,
			subject = ?, html_content = ?, text_content = ?, variables = ?,
			parent_id = ?, tags = ?, updated_at = ?
		WHERE id = ? AND user_id = ?`

	variablesJSON, _ := json.Marshal(variables)
	tagsJSON, _ := json.Marshal(req.Tags)

	_, err := s.db.Exec(query,
		req.Name, req.Description, req.Type, req.Language,
		req.Subject, req.HTMLContent, textContent, variablesJSON,
		req.ParentID, tagsJSON, time.Now(),
		templateID, userID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to update template: %w", err)
	}

	// Return updated template
	return s.GetTemplate(userID, templateID)
}

// DeleteTemplate soft deletes a template
func (s *EmailTemplateService) DeleteTemplate(userID, templateID uuid.UUID) error {
	query := `UPDATE email_templates SET is_active = false, updated_at = ? WHERE id = ? AND user_id = ?`
	
	_, err := s.db.Exec(query, time.Now(), templateID, userID)
	if err != nil {
		return fmt.Errorf("failed to delete template: %w", err)
	}

	return nil
}

// CloneTemplate creates a copy of an existing template
func (s *EmailTemplateService) CloneTemplate(userID, templateID uuid.UUID, newName string) (*models.EmailTemplate, error) {
	// Get original template
	original, err := s.GetTemplate(userID, templateID)
	if err != nil {
		return nil, fmt.Errorf("failed to get original template: %w", err)
	}

	// Create clone request
	cloneReq := models.CreateEmailTemplateRequest{
		FormID:      original.FormID,
		Name:        newName,
		Description: "Clone of " + original.Name,
		Type:        original.Type,
		Language:    original.Language,
		Subject:     original.Subject,
		HTMLContent: original.HTMLContent,
		TextContent: original.TextContent,
		Variables:   original.Variables,
		Tags:        original.Tags,
	}

	return s.CreateTemplate(userID, cloneReq)
}