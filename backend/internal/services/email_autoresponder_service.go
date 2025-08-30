package services

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"formhub/internal/models"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

type EmailAutoresponderService struct {
	db                    *sql.DB
	templateService       *EmailTemplateService
	providerService       *EmailProviderService
	queueService          *EmailQueueService
}

type AutoresponderEvaluation struct {
	ShouldTrigger bool                   `json:"should_trigger"`
	Conditions    []ConditionResult      `json:"conditions"`
	Variables     map[string]interface{} `json:"variables"`
	DelayMinutes  int                    `json:"delay_minutes"`
	ScheduledAt   time.Time              `json:"scheduled_at"`
}

type ConditionResult struct {
	Type      string `json:"type"` // field, time
	Field     string `json:"field,omitempty"`
	Operator  string `json:"operator"`
	Expected  string `json:"expected"`
	Actual    string `json:"actual"`
	Satisfied bool   `json:"satisfied"`
}

func NewEmailAutoresponderService(db *sql.DB, templateService *EmailTemplateService, providerService *EmailProviderService, queueService *EmailQueueService) *EmailAutoresponderService {
	return &EmailAutoresponderService{
		db:              db,
		templateService: templateService,
		providerService: providerService,
		queueService:    queueService,
	}
}

// CreateAutoresponder creates a new autoresponder configuration
func (s *EmailAutoresponderService) CreateAutoresponder(userID uuid.UUID, req models.CreateAutoresponderRequest) (*models.EmailAutoresponder, error) {
	// Validate template exists and belongs to user
	_, err := s.templateService.GetTemplate(userID, req.TemplateID)
	if err != nil {
		return nil, fmt.Errorf("template not found or access denied: %w", err)
	}

	// Validate provider if specified
	if req.ProviderID != nil {
		_, err := s.providerService.GetProvider(userID, *req.ProviderID)
		if err != nil {
			return nil, fmt.Errorf("provider not found or access denied: %w", err)
		}
	}

	// Validate conditions
	if err := s.validateConditions(req.Conditions); err != nil {
		return nil, fmt.Errorf("invalid conditions: %w", err)
	}

	autoresponder := &models.EmailAutoresponder{
		ID:           uuid.New(),
		UserID:       userID,
		FormID:       req.FormID,
		Name:         req.Name,
		TemplateID:   req.TemplateID,
		ProviderID:   req.ProviderID,
		IsEnabled:    true,
		DelayMinutes: req.DelayMinutes,
		Conditions:   req.Conditions,
		SendToField:  req.SendToField,
		CCEmails:     req.CCEmails,
		BCCEmails:    req.BCCEmails,
		ReplyTo:      req.ReplyTo,
		TrackOpens:   req.TrackOpens,
		TrackClicks:  req.TrackClicks,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	// Insert into database
	conditionsJSON, _ := json.Marshal(autoresponder.Conditions)
	ccEmailsJSON, _ := json.Marshal(autoresponder.CCEmails)
	bccEmailsJSON, _ := json.Marshal(autoresponder.BCCEmails)

	query := `
		INSERT INTO email_autoresponders (
			id, user_id, form_id, name, template_id, provider_id, is_enabled,
			delay_minutes, conditions, send_to_field, cc_emails, bcc_emails,
			reply_to, track_opens, track_clicks, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err = s.db.Exec(query,
		autoresponder.ID, autoresponder.UserID, autoresponder.FormID,
		autoresponder.Name, autoresponder.TemplateID, autoresponder.ProviderID,
		autoresponder.IsEnabled, autoresponder.DelayMinutes, conditionsJSON,
		autoresponder.SendToField, ccEmailsJSON, bccEmailsJSON,
		autoresponder.ReplyTo, autoresponder.TrackOpens, autoresponder.TrackClicks,
		autoresponder.CreatedAt, autoresponder.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create autoresponder: %w", err)
	}

	return autoresponder, nil
}

// GetAutoresponder retrieves an autoresponder by ID
func (s *EmailAutoresponderService) GetAutoresponder(userID, autoresponderID uuid.UUID) (*models.EmailAutoresponder, error) {
	query := `
		SELECT id, user_id, form_id, name, template_id, provider_id, is_enabled,
		       delay_minutes, conditions, send_to_field, cc_emails, bcc_emails,
		       reply_to, track_opens, track_clicks, created_at, updated_at
		FROM email_autoresponders 
		WHERE id = ? AND user_id = ?`

	var autoresponder models.EmailAutoresponder
	var providerID sql.NullString
	var conditionsJSON, ccEmailsJSON, bccEmailsJSON []byte

	err := s.db.QueryRow(query, autoresponderID, userID).Scan(
		&autoresponder.ID, &autoresponder.UserID, &autoresponder.FormID,
		&autoresponder.Name, &autoresponder.TemplateID, &providerID,
		&autoresponder.IsEnabled, &autoresponder.DelayMinutes, &conditionsJSON,
		&autoresponder.SendToField, &ccEmailsJSON, &bccEmailsJSON,
		&autoresponder.ReplyTo, &autoresponder.TrackOpens, &autoresponder.TrackClicks,
		&autoresponder.CreatedAt, &autoresponder.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get autoresponder: %w", err)
	}

	// Parse optional fields
	if providerID.Valid {
		if pid, err := uuid.Parse(providerID.String); err == nil {
			autoresponder.ProviderID = &pid
		}
	}

	// Parse JSON fields
	if len(conditionsJSON) > 0 {
		json.Unmarshal(conditionsJSON, &autoresponder.Conditions)
	}
	if len(ccEmailsJSON) > 0 {
		json.Unmarshal(ccEmailsJSON, &autoresponder.CCEmails)
	}
	if len(bccEmailsJSON) > 0 {
		json.Unmarshal(bccEmailsJSON, &autoresponder.BCCEmails)
	}

	return &autoresponder, nil
}

// ListAutoresponders retrieves autoresponders for a user or form
func (s *EmailAutoresponderService) ListAutoresponders(userID uuid.UUID, formID *uuid.UUID) ([]models.EmailAutoresponder, error) {
	query := `
		SELECT id, user_id, form_id, name, template_id, provider_id, is_enabled,
		       delay_minutes, conditions, send_to_field, cc_emails, bcc_emails,
		       reply_to, track_opens, track_clicks, created_at, updated_at
		FROM email_autoresponders 
		WHERE user_id = ?`
	
	args := []interface{}{userID}

	if formID != nil {
		query += " AND form_id = ?"
		args = append(args, *formID)
	}

	query += " ORDER BY created_at DESC"

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list autoresponders: %w", err)
	}
	defer rows.Close()

	var autoresponders []models.EmailAutoresponder
	for rows.Next() {
		var autoresponder models.EmailAutoresponder
		var providerID sql.NullString
		var conditionsJSON, ccEmailsJSON, bccEmailsJSON []byte

		err := rows.Scan(
			&autoresponder.ID, &autoresponder.UserID, &autoresponder.FormID,
			&autoresponder.Name, &autoresponder.TemplateID, &providerID,
			&autoresponder.IsEnabled, &autoresponder.DelayMinutes, &conditionsJSON,
			&autoresponder.SendToField, &ccEmailsJSON, &bccEmailsJSON,
			&autoresponder.ReplyTo, &autoresponder.TrackOpens, &autoresponder.TrackClicks,
			&autoresponder.CreatedAt, &autoresponder.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan autoresponder: %w", err)
		}

		// Parse optional fields
		if providerID.Valid {
			if pid, err := uuid.Parse(providerID.String); err == nil {
				autoresponder.ProviderID = &pid
			}
		}

		// Parse JSON fields
		if len(conditionsJSON) > 0 {
			json.Unmarshal(conditionsJSON, &autoresponder.Conditions)
		}
		if len(ccEmailsJSON) > 0 {
			json.Unmarshal(ccEmailsJSON, &autoresponder.CCEmails)
		}
		if len(bccEmailsJSON) > 0 {
			json.Unmarshal(bccEmailsJSON, &autoresponder.BCCEmails)
		}

		autoresponders = append(autoresponders, autoresponder)
	}

	return autoresponders, nil
}

// UpdateAutoresponder updates an existing autoresponder
func (s *EmailAutoresponderService) UpdateAutoresponder(userID, autoresponderID uuid.UUID, req models.CreateAutoresponderRequest) (*models.EmailAutoresponder, error) {
	// Validate template exists and belongs to user
	_, err := s.templateService.GetTemplate(userID, req.TemplateID)
	if err != nil {
		return nil, fmt.Errorf("template not found or access denied: %w", err)
	}

	// Validate provider if specified
	if req.ProviderID != nil {
		_, err := s.providerService.GetProvider(userID, *req.ProviderID)
		if err != nil {
			return nil, fmt.Errorf("provider not found or access denied: %w", err)
		}
	}

	// Validate conditions
	if err := s.validateConditions(req.Conditions); err != nil {
		return nil, fmt.Errorf("invalid conditions: %w", err)
	}

	// Update autoresponder
	conditionsJSON, _ := json.Marshal(req.Conditions)
	ccEmailsJSON, _ := json.Marshal(req.CCEmails)
	bccEmailsJSON, _ := json.Marshal(req.BCCEmails)

	query := `
		UPDATE email_autoresponders SET
			name = ?, template_id = ?, provider_id = ?, delay_minutes = ?,
			conditions = ?, send_to_field = ?, cc_emails = ?, bcc_emails = ?,
			reply_to = ?, track_opens = ?, track_clicks = ?, updated_at = ?
		WHERE id = ? AND user_id = ?`

	_, err = s.db.Exec(query,
		req.Name, req.TemplateID, req.ProviderID, req.DelayMinutes,
		conditionsJSON, req.SendToField, ccEmailsJSON, bccEmailsJSON,
		req.ReplyTo, req.TrackOpens, req.TrackClicks, time.Now(),
		autoresponderID, userID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to update autoresponder: %w", err)
	}

	return s.GetAutoresponder(userID, autoresponderID)
}

// ToggleAutoresponder enables or disables an autoresponder
func (s *EmailAutoresponderService) ToggleAutoresponder(userID, autoresponderID uuid.UUID, enabled bool) error {
	query := `UPDATE email_autoresponders SET is_enabled = ?, updated_at = ? WHERE id = ? AND user_id = ?`
	
	_, err := s.db.Exec(query, enabled, time.Now(), autoresponderID, userID)
	if err != nil {
		return fmt.Errorf("failed to toggle autoresponder: %w", err)
	}

	return nil
}

// DeleteAutoresponder deletes an autoresponder
func (s *EmailAutoresponderService) DeleteAutoresponder(userID, autoresponderID uuid.UUID) error {
	query := `DELETE FROM email_autoresponders WHERE id = ? AND user_id = ?`
	
	_, err := s.db.Exec(query, autoresponderID, userID)
	if err != nil {
		return fmt.Errorf("failed to delete autoresponder: %w", err)
	}

	return nil
}

// EvaluateAutoresponders evaluates which autoresponders should trigger for a form submission
func (s *EmailAutoresponderService) EvaluateAutoresponders(formID uuid.UUID, submissionData map[string]interface{}, submissionTime time.Time) ([]*AutoresponderEvaluation, error) {
	// Get all enabled autoresponders for the form
	query := `
		SELECT id, user_id, name, template_id, provider_id, delay_minutes,
		       conditions, send_to_field, cc_emails, bcc_emails, reply_to,
		       track_opens, track_clicks
		FROM email_autoresponders 
		WHERE form_id = ? AND is_enabled = true`

	rows, err := s.db.Query(query, formID)
	if err != nil {
		return nil, fmt.Errorf("failed to get autoresponders: %w", err)
	}
	defer rows.Close()

	var evaluations []*AutoresponderEvaluation
	
	for rows.Next() {
		var id, userID, templateID uuid.UUID
		var providerID sql.NullString
		var name, sendToField, replyTo string
		var delayMinutes int
		var conditionsJSON, ccEmailsJSON, bccEmailsJSON []byte
		var trackOpens, trackClicks bool

		err := rows.Scan(
			&id, &userID, &name, &templateID, &providerID, &delayMinutes,
			&conditionsJSON, &sendToField, &ccEmailsJSON, &bccEmailsJSON,
			&replyTo, &trackOpens, &trackClicks,
		)
		if err != nil {
			continue // Skip invalid entries
		}

		// Parse conditions
		var conditions models.AutoresponderConditions
		if len(conditionsJSON) > 0 {
			json.Unmarshal(conditionsJSON, &conditions)
		}

		// Evaluate conditions
		evaluation := s.evaluateConditions(conditions, submissionData, submissionTime)
		evaluation.DelayMinutes = delayMinutes
		evaluation.ScheduledAt = submissionTime.Add(time.Duration(delayMinutes) * time.Minute)

		// Add submission data as variables
		evaluation.Variables = make(map[string]interface{})
		for k, v := range submissionData {
			evaluation.Variables[k] = v
		}

		evaluations = append(evaluations, evaluation)
	}

	return evaluations, nil
}

// ProcessAutoresponders processes autoresponders for a form submission
func (s *EmailAutoresponderService) ProcessAutoresponders(submission *models.Submission, form *models.Form, user *models.User) error {
	// Get evaluations
	evaluations, err := s.EvaluateAutoresponders(submission.FormID, submission.Data, submission.CreatedAt)
	if err != nil {
		return fmt.Errorf("failed to evaluate autoresponders: %w", err)
	}

	// Process each triggered autoresponder
	for _, evaluation := range evaluations {
		if !evaluation.ShouldTrigger {
			continue
		}

		// Get the autoresponder details again (we need all fields)
		autoresponders, err := s.ListAutoresponders(user.ID, &submission.FormID)
		if err != nil {
			continue
		}

		for _, autoresponder := range autoresponders {
			if !autoresponder.IsEnabled {
				continue
			}

			// Create render context
			context := TemplateRenderContext{
				Variables:  evaluation.Variables,
				FormData:   submission.Data,
				Submission: submission,
				Form:       form,
				User:       user,
				Timestamp:  submission.CreatedAt,
				IPAddress:  submission.IPAddress,
				UserAgent:  submission.UserAgent,
				Referrer:   submission.Referrer,
			}

			// Get recipient email from form data
			recipientEmail, ok := submission.Data[autoresponder.SendToField].(string)
			if !ok || recipientEmail == "" {
				continue // Skip if no valid recipient email
			}

			// Determine provider
			var providerID uuid.UUID
			if autoresponder.ProviderID != nil {
				providerID = *autoresponder.ProviderID
			} else {
				// Use default provider
				defaultProvider, err := s.providerService.GetDefaultProvider(user.ID)
				if err != nil {
					continue // Skip if no provider available
				}
				providerID = defaultProvider.ID
			}

			// Queue the email
			queueItem := &models.EmailQueue{
				ID:           uuid.New(),
				UserID:       user.ID,
				FormID:       &submission.FormID,
				SubmissionID: &submission.ID,
				TemplateID:   autoresponder.TemplateID,
				ProviderID:   &providerID,
				ToEmails:     []string{recipientEmail},
				CCEmails:     autoresponder.CCEmails,
				BCCEmails:    autoresponder.BCCEmails,
				Variables:    evaluation.Variables,
				ScheduledAt:  evaluation.ScheduledAt,
				Status:       models.EmailStatusScheduled,
				Attempts:     0,
				Priority:     1, // Normal priority
				CreatedAt:    time.Now(),
				UpdatedAt:    time.Now(),
			}

			// Add reply-to if specified
			if autoresponder.ReplyTo != "" {
				queueItem.Variables["reply_to"] = autoresponder.ReplyTo
			}

			// Render template to get subject and content
			rendered, err := s.templateService.RenderTemplate(autoresponder.TemplateID, context)
			if err != nil {
				continue // Skip if template rendering fails
			}

			queueItem.Subject = rendered.Subject
			queueItem.HTMLContent = rendered.HTMLContent
			queueItem.TextContent = rendered.TextContent

			// Queue the email
			err = s.queueService.QueueEmail(queueItem)
			if err != nil {
				// Log error but continue processing other autoresponders
				continue
			}
		}
	}

	return nil
}

// Helper methods

func (s *EmailAutoresponderService) validateConditions(conditions models.AutoresponderConditions) error {
	// Validate field conditions
	for _, fieldCondition := range conditions.FieldConditions {
		if fieldCondition.FieldName == "" {
			return fmt.Errorf("field name is required for field conditions")
		}
		
		if fieldCondition.Operator == "" {
			return fmt.Errorf("operator is required for field conditions")
		}

		validOperators := []string{"equals", "not_equals", "contains", "not_contains", "starts_with", "ends_with", "in", "not_in", "exists", "not_exists", "greater_than", "less_than", "regex"}
		if !s.isValidOperator(fieldCondition.Operator, validOperators) {
			return fmt.Errorf("invalid operator: %s", fieldCondition.Operator)
		}

		if fieldCondition.Operator == "in" || fieldCondition.Operator == "not_in" {
			if len(fieldCondition.Values) == 0 {
				return fmt.Errorf("values array is required for 'in' and 'not_in' operators")
			}
		} else if fieldCondition.Operator != "exists" && fieldCondition.Operator != "not_exists" {
			if fieldCondition.Value == "" {
				return fmt.Errorf("value is required for operator: %s", fieldCondition.Operator)
			}
		}
	}

	// Validate time conditions
	if conditions.TimeConditions != nil {
		timeCondition := conditions.TimeConditions
		
		// Validate time format (HH:MM)
		if timeCondition.StartTime != "" {
			if !s.isValidTimeFormat(timeCondition.StartTime) {
				return fmt.Errorf("invalid start time format (should be HH:MM): %s", timeCondition.StartTime)
			}
		}
		
		if timeCondition.EndTime != "" {
			if !s.isValidTimeFormat(timeCondition.EndTime) {
				return fmt.Errorf("invalid end time format (should be HH:MM): %s", timeCondition.EndTime)
			}
		}

		// Validate days
		if len(timeCondition.Days) > 0 {
			validDays := []string{"monday", "tuesday", "wednesday", "thursday", "friday", "saturday", "sunday"}
			for _, day := range timeCondition.Days {
				if !s.contains(validDays, strings.ToLower(day)) {
					return fmt.Errorf("invalid day: %s", day)
				}
			}
		}
	}

	// Validate logical operator
	if conditions.LogicalOperator != "" && conditions.LogicalOperator != "AND" && conditions.LogicalOperator != "OR" {
		return fmt.Errorf("logical operator must be 'AND' or 'OR'")
	}

	return nil
}

func (s *EmailAutoresponderService) evaluateConditions(conditions models.AutoresponderConditions, submissionData map[string]interface{}, submissionTime time.Time) *AutoresponderEvaluation {
	var conditionResults []ConditionResult
	satisfiedCount := 0

	// Evaluate field conditions
	for _, fieldCondition := range conditions.FieldConditions {
		result := s.evaluateFieldCondition(fieldCondition, submissionData)
		conditionResults = append(conditionResults, result)
		if result.Satisfied {
			satisfiedCount++
		}
	}

	// Evaluate time conditions
	if conditions.TimeConditions != nil {
		result := s.evaluateTimeCondition(*conditions.TimeConditions, submissionTime)
		conditionResults = append(conditionResults, result)
		if result.Satisfied {
			satisfiedCount++
		}
	}

	// Determine if autoresponder should trigger
	shouldTrigger := false
	totalConditions := len(conditionResults)

	if totalConditions == 0 {
		// No conditions means always trigger
		shouldTrigger = true
	} else {
		logicalOperator := conditions.LogicalOperator
		if logicalOperator == "" {
			logicalOperator = "AND" // Default to AND
		}

		if logicalOperator == "AND" {
			shouldTrigger = satisfiedCount == totalConditions
		} else { // OR
			shouldTrigger = satisfiedCount > 0
		}
	}

	return &AutoresponderEvaluation{
		ShouldTrigger: shouldTrigger,
		Conditions:    conditionResults,
	}
}

func (s *EmailAutoresponderService) evaluateFieldCondition(condition models.FieldCondition, submissionData map[string]interface{}) ConditionResult {
	result := ConditionResult{
		Type:     "field",
		Field:    condition.FieldName,
		Operator: condition.Operator,
		Expected: condition.Value,
	}

	// Get actual value from submission data
	actualValue, exists := submissionData[condition.FieldName]
	if !exists {
		actualValue = ""
	}

	actualStr := fmt.Sprintf("%v", actualValue)
	result.Actual = actualStr

	// Evaluate condition based on operator
	switch condition.Operator {
	case "equals":
		result.Satisfied = actualStr == condition.Value
	case "not_equals":
		result.Satisfied = actualStr != condition.Value
	case "contains":
		result.Satisfied = strings.Contains(strings.ToLower(actualStr), strings.ToLower(condition.Value))
	case "not_contains":
		result.Satisfied = !strings.Contains(strings.ToLower(actualStr), strings.ToLower(condition.Value))
	case "starts_with":
		result.Satisfied = strings.HasPrefix(strings.ToLower(actualStr), strings.ToLower(condition.Value))
	case "ends_with":
		result.Satisfied = strings.HasSuffix(strings.ToLower(actualStr), strings.ToLower(condition.Value))
	case "in":
		result.Satisfied = s.contains(condition.Values, actualStr)
	case "not_in":
		result.Satisfied = !s.contains(condition.Values, actualStr)
	case "exists":
		result.Satisfied = exists && actualStr != ""
	case "not_exists":
		result.Satisfied = !exists || actualStr == ""
	case "greater_than":
		if actualNum, err := strconv.ParseFloat(actualStr, 64); err == nil {
			if expectedNum, err := strconv.ParseFloat(condition.Value, 64); err == nil {
				result.Satisfied = actualNum > expectedNum
			}
		}
	case "less_than":
		if actualNum, err := strconv.ParseFloat(actualStr, 64); err == nil {
			if expectedNum, err := strconv.ParseFloat(condition.Value, 64); err == nil {
				result.Satisfied = actualNum < expectedNum
			}
		}
	case "regex":
		if re, err := regexp.Compile(condition.Value); err == nil {
			result.Satisfied = re.MatchString(actualStr)
		}
	default:
		result.Satisfied = false
	}

	return result
}

func (s *EmailAutoresponderService) evaluateTimeCondition(condition models.TimeCondition, submissionTime time.Time) ConditionResult {
	result := ConditionResult{
		Type:      "time",
		Operator:  "time_window",
		Satisfied: true,
	}

	// Check day of week
	if len(condition.Days) > 0 {
		dayName := strings.ToLower(submissionTime.Weekday().String())
		dayMatches := false
		for _, allowedDay := range condition.Days {
			if strings.ToLower(allowedDay) == dayName {
				dayMatches = true
				break
			}
		}
		if !dayMatches {
			result.Satisfied = false
			result.Actual = dayName
			result.Expected = strings.Join(condition.Days, ", ")
			return result
		}
	}

	// Check time window
	if condition.StartTime != "" && condition.EndTime != "" {
		currentTime := submissionTime.Format("15:04")
		
		// Simple time comparison (doesn't handle midnight crossover)
		if currentTime < condition.StartTime || currentTime > condition.EndTime {
			result.Satisfied = false
			result.Actual = currentTime
			result.Expected = fmt.Sprintf("%s - %s", condition.StartTime, condition.EndTime)
		}
	}

	return result
}

// Utility methods

func (s *EmailAutoresponderService) isValidOperator(operator string, validOperators []string) bool {
	return s.contains(validOperators, operator)
}

func (s *EmailAutoresponderService) isValidTimeFormat(timeStr string) bool {
	// Check if time matches HH:MM format
	re := regexp.MustCompile(`^([0-1]?[0-9]|2[0-3]):[0-5][0-9]$`)
	return re.MatchString(timeStr)
}

func (s *EmailAutoresponderService) contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}