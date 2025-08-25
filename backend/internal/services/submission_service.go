package services

import (
	"crypto/md5"
	"database/sql"
	"encoding/json"
	"fmt"
	"formhub/internal/models"
	"formhub/pkg/email"
	"formhub/pkg/utils"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type SubmissionService struct {
	db           *sql.DB
	redis        *redis.Client
	emailService *email.SMTPService
	formService  *FormService
}

func NewSubmissionService(db *sql.DB, redis *redis.Client, emailService *email.SMTPService) *SubmissionService {
	return &SubmissionService{
		db:           db,
		redis:        redis,
		emailService: emailService,
	}
}

func (s *SubmissionService) SetFormService(formService *FormService) {
	s.formService = formService
}

func (s *SubmissionService) HandleSubmission(req models.SubmissionRequest, ipAddress, userAgent, referrer string) (*models.SubmissionResponse, error) {
	// Find form by access key (API key)
	form, apiKey, err := s.getFormByAccessKey(req.AccessKey)
	if err != nil {
		return &models.SubmissionResponse{
			Success:    false,
			StatusCode: 401,
			Message:    "Invalid access key",
		}, nil
	}

	// Check form is active
	if !form.IsActive {
		return &models.SubmissionResponse{
			Success:    false,
			StatusCode: 403,
			Message:    "Form is not active",
		}, nil
	}

	// Check rate limits
	if err := s.checkRateLimit(apiKey, ipAddress); err != nil {
		return &models.SubmissionResponse{
			Success:    false,
			StatusCode: 429,
			Message:    "Rate limit exceeded",
		}, nil
	}

	// Extract all form data
	formData := make(map[string]interface{})
	if req.Email != "" {
		formData["email"] = req.Email
	}
	if req.Subject != "" {
		formData["subject"] = req.Subject
	}
	if req.Message != "" {
		formData["message"] = req.Message
	}

	// Merge any additional data from req.Data
	for key, value := range req.Data {
		formData[key] = value
	}

	// Basic spam detection
	isSpam, spamScore := s.detectSpam(formData, ipAddress)

	// Create submission
	submission := &models.Submission{
		ID:          uuid.New(),
		FormID:      form.ID,
		Data:        formData,
		IPAddress:   ipAddress,
		UserAgent:   userAgent,
		Referrer:    referrer,
		IsSpam:      isSpam,
		SpamScore:   spamScore,
		EmailSent:   false,
		WebhookSent: false,
		CreatedAt:   time.Now(),
	}

	// Save submission to database
	if err := s.saveSubmission(submission); err != nil {
		log.Printf("Failed to save submission: %v", err)
		return &models.SubmissionResponse{
			Success:    false,
			StatusCode: 500,
			Message:    "Failed to save submission",
		}, nil
	}

	// Send email notification if not spam
	if !isSpam {
		if err := s.sendEmailNotification(form, submission); err != nil {
			log.Printf("Failed to send email notification: %v", err)
		} else {
			s.markEmailSent(submission.ID)
		}

		// Send webhook if configured
		if form.WebhookURL != "" {
			if err := s.sendWebhook(form, submission); err != nil {
				log.Printf("Failed to send webhook: %v", err)
			} else {
				s.markWebhookSent(submission.ID)
			}
		}

		// Increment form submission count
		if err := s.formService.IncrementSubmissionCount(form.ID); err != nil {
			log.Printf("Failed to increment submission count: %v", err)
		}
	}

	// Prepare response
	response := &models.SubmissionResponse{
		Success:    true,
		StatusCode: 200,
		Message:    getSuccessMessage(form),
		Data:       formData,
	}

	// Add redirect URL if specified
	if req.RedirectURL != "" {
		response.RedirectURL = req.RedirectURL
	} else if form.RedirectURL != "" {
		response.RedirectURL = form.RedirectURL
	}

	return response, nil
}

func (s *SubmissionService) getFormByAccessKey(accessKey string) (*models.Form, *models.APIKey, error) {
	// Get API key details
	apiKeyQuery := `
		SELECT id, user_id, name, permissions, rate_limit, is_active, last_used_at
		FROM api_keys 
		WHERE key_hash = ? AND is_active = true
	`

	var apiKey models.APIKey
	keyHash := fmt.Sprintf("%x", md5.Sum([]byte(accessKey)))
	
	err := s.db.QueryRow(apiKeyQuery, keyHash).Scan(
		&apiKey.ID, &apiKey.UserID, &apiKey.Name, &apiKey.Permissions,
		&apiKey.RateLimit, &apiKey.IsActive, &apiKey.LastUsedAt,
	)
	
	if err != nil {
		return nil, nil, fmt.Errorf("api key not found")
	}

	// Update last used timestamp
	s.db.Exec("UPDATE api_keys SET last_used_at = ? WHERE id = ?", time.Now(), apiKey.ID)

	// Get any active form for this user (for simplicity, we'll use the first active form)
	// In a real implementation, you might want to link API keys to specific forms
	formQuery := `
		SELECT id, user_id, name, description, target_email, cc_emails, subject,
			success_message, redirect_url, webhook_url, spam_protection, recaptcha_secret,
			file_uploads, max_file_size, allowed_origins, is_active, submission_count,
			created_at, updated_at
		FROM forms 
		WHERE user_id = ? AND is_active = true 
		ORDER BY created_at DESC 
		LIMIT 1
	`

	var form models.Form
	err = s.db.QueryRow(formQuery, apiKey.UserID).Scan(
		&form.ID, &form.UserID, &form.Name, &form.Description, &form.TargetEmail,
		&form.CCEmails, &form.Subject, &form.SuccessMessage, &form.RedirectURL,
		&form.WebhookURL, &form.SpamProtection, &form.RecaptchaSecret,
		&form.FileUploads, &form.MaxFileSize, &form.AllowedOrigins,
		&form.IsActive, &form.SubmissionCount, &form.CreatedAt, &form.UpdatedAt,
	)

	if err != nil {
		return nil, nil, fmt.Errorf("no active form found for this access key")
	}

	return &form, &apiKey, nil
}

func (s *SubmissionService) checkRateLimit(apiKey *models.APIKey, ipAddress string) error {
	// Implement rate limiting using Redis
	// This is a simple implementation - you might want to use a more sophisticated approach
	key := fmt.Sprintf("rate_limit:%s:%s", apiKey.ID.String(), ipAddress)
	
	// For now, just return nil (no rate limiting)
	// TODO: Implement proper rate limiting
	_ = key
	return nil
}

func (s *SubmissionService) detectSpam(data map[string]interface{}, ipAddress string) (bool, float64) {
	spamScore := 0.0
	
	// Simple spam detection rules
	for key, value := range data {
		valueStr := fmt.Sprintf("%v", value)
		lowerKey := strings.ToLower(key)
		lowerValue := strings.ToLower(valueStr)
		
		// Check for common spam patterns
		spamKeywords := []string{"viagra", "casino", "loan", "bitcoin", "crypto", "seo services"}
		for _, keyword := range spamKeywords {
			if strings.Contains(lowerValue, keyword) {
				spamScore += 0.3
			}
		}
		
		// Check for excessive links
		if strings.Count(lowerValue, "http://") + strings.Count(lowerValue, "https://") > 2 {
			spamScore += 0.2
		}
		
		// Check for excessive capital letters
		if len(valueStr) > 10 {
			upperCount := 0
			for _, char := range valueStr {
				if char >= 'A' && char <= 'Z' {
					upperCount++
				}
			}
			if float64(upperCount)/float64(len(valueStr)) > 0.7 {
				spamScore += 0.2
			}
		}
		
		// Check for honeypot fields
		if lowerKey == "honeypot" || lowerKey == "_gotcha" {
			if valueStr != "" {
				spamScore += 1.0 // Definite spam
			}
		}
	}
	
	return spamScore >= 0.5, spamScore
}

func (s *SubmissionService) saveSubmission(submission *models.Submission) error {
	dataJSON, err := json.Marshal(submission.Data)
	if err != nil {
		return fmt.Errorf("failed to marshal submission data: %w", err)
	}

	query := `
		INSERT INTO submissions (id, form_id, data, ip_address, user_agent, referrer,
			is_spam, spam_score, email_sent, webhook_sent, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err = s.db.Exec(query,
		submission.ID, submission.FormID, string(dataJSON), submission.IPAddress,
		submission.UserAgent, submission.Referrer, submission.IsSpam, submission.SpamScore,
		submission.EmailSent, submission.WebhookSent, submission.CreatedAt,
	)

	return err
}

func (s *SubmissionService) sendEmailNotification(form *models.Form, submission *models.Submission) error {
	// Parse CC emails
	var ccEmails []string
	if form.CCEmails != "" {
		json.Unmarshal([]byte(form.CCEmails), &ccEmails)
	}

	// Prepare email data
	emailData := email.EmailData{
		FormName:       form.Name,
		Subject:        form.Subject,
		ToEmails:       []string{form.TargetEmail},
		CCEmails:       ccEmails,
		SubmissionData: submission.Data,
		IPAddress:      submission.IPAddress,
		Timestamp:      submission.CreatedAt.Format("2006-01-02 15:04:05 UTC"),
	}

	return s.emailService.SendFormSubmission(emailData)
}

func (s *SubmissionService) sendWebhook(form *models.Form, submission *models.Submission) error {
	if form.WebhookURL == "" {
		return fmt.Errorf("webhook URL is not configured")
	}

	// Import the webhook utility
	// Note: You'll need to add this import at the top of the file
	// "formhub/pkg/utils"

	payload := utils.WebhookPayload{
		Event:     "form.submission",
		FormID:    form.ID.String(),
		FormName:  form.Name,
		Timestamp: submission.CreatedAt.Format(time.RFC3339),
		UserAgent: submission.UserAgent,
		IPAddress: submission.IPAddress,
		Submission: utils.WebhookSubmission{
			ID:        submission.ID.String(),
			Data:      submission.Data,
			IsSpam:    submission.IsSpam,
			SpamScore: submission.SpamScore,
			CreatedAt: submission.CreatedAt.Format(time.RFC3339),
		},
		Metadata: map[string]interface{}{
			"form_name": form.Name,
			"user_id":   form.UserID.String(),
		},
	}

	response, err := utils.SendWebhook(form.WebhookURL, payload)
	if err != nil {
		return fmt.Errorf("failed to send webhook: %w", err)
	}

	if !response.Success {
		return fmt.Errorf("webhook returned non-success status: %d", response.StatusCode)
	}

	log.Printf("Webhook sent successfully to %s (status: %d, duration: %s)", 
		form.WebhookURL, response.StatusCode, response.Duration)

	return nil
}

func (s *SubmissionService) markEmailSent(submissionID uuid.UUID) {
	query := `UPDATE submissions SET email_sent = true WHERE id = ?`
	s.db.Exec(query, submissionID)
}

func (s *SubmissionService) markWebhookSent(submissionID uuid.UUID) {
	query := `UPDATE submissions SET webhook_sent = true WHERE id = ?`
	s.db.Exec(query, submissionID)
}

func getSuccessMessage(form *models.Form) string {
	if form.SuccessMessage != "" {
		return form.SuccessMessage
	}
	return "Thank you for your submission! We'll get back to you soon."
}