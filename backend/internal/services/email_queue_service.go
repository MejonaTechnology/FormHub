package services

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"formhub/internal/models"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
)

type EmailQueueService struct {
	db              *sql.DB
	providerService *EmailProviderService
	analyticsService *EmailAnalyticsService
	isProcessing    bool
	processingMux   sync.RWMutex
	stopChan        chan bool
	config          QueueConfig
}

type QueueConfig struct {
	BatchSize       int           `json:"batch_size"`       // Number of emails to process at once
	RetryAttempts   int           `json:"retry_attempts"`   // Maximum retry attempts
	RetryDelay      time.Duration `json:"retry_delay"`      // Base retry delay
	ProcessInterval time.Duration `json:"process_interval"` // How often to process queue
	MaxWorkers      int           `json:"max_workers"`      // Maximum concurrent workers
}

type QueueStats struct {
	Pending   int `json:"pending"`
	Scheduled int `json:"scheduled"`
	Sending   int `json:"sending"`
	Sent      int `json:"sent"`
	Failed    int `json:"failed"`
	Total     int `json:"total"`
}

type ProcessingResult struct {
	Processed int `json:"processed"`
	Sent      int `json:"sent"`
	Failed    int `json:"failed"`
	Errors    []string `json:"errors,omitempty"`
}

func NewEmailQueueService(db *sql.DB, providerService *EmailProviderService, analyticsService *EmailAnalyticsService) *EmailQueueService {
	config := QueueConfig{
		BatchSize:       50,
		RetryAttempts:   5,
		RetryDelay:      time.Minute * 5,
		ProcessInterval: time.Minute * 1,
		MaxWorkers:      5,
	}

	return &EmailQueueService{
		db:               db,
		providerService:  providerService,
		analyticsService: analyticsService,
		isProcessing:     false,
		stopChan:         make(chan bool),
		config:           config,
	}
}

// QueueEmail adds an email to the queue
func (s *EmailQueueService) QueueEmail(email *models.EmailQueue) error {
	// Set defaults
	if email.ID == uuid.Nil {
		email.ID = uuid.New()
	}
	if email.CreatedAt.IsZero() {
		email.CreatedAt = time.Now()
	}
	if email.UpdatedAt.IsZero() {
		email.UpdatedAt = time.Now()
	}
	if email.ScheduledAt.IsZero() {
		email.ScheduledAt = time.Now()
	}
	if email.Status == "" {
		if email.ScheduledAt.After(time.Now()) {
			email.Status = models.EmailStatusScheduled
		} else {
			email.Status = models.EmailStatusPending
		}
	}

	// Insert into database
	toEmailsJSON, _ := json.Marshal(email.ToEmails)
	ccEmailsJSON, _ := json.Marshal(email.CCEmails)
	bccEmailsJSON, _ := json.Marshal(email.BCCEmails)
	variablesJSON, _ := json.Marshal(email.Variables)

	query := `
		INSERT INTO email_queue (
			id, user_id, form_id, submission_id, template_id, provider_id,
			to_emails, cc_emails, bcc_emails, subject, html_content, text_content,
			variables, scheduled_at, status, attempts, priority, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := s.db.Exec(query,
		email.ID, email.UserID, email.FormID, email.SubmissionID,
		email.TemplateID, email.ProviderID, toEmailsJSON, ccEmailsJSON,
		bccEmailsJSON, email.Subject, email.HTMLContent, email.TextContent,
		variablesJSON, email.ScheduledAt, email.Status, email.Attempts,
		email.Priority, email.CreatedAt, email.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to queue email: %w", err)
	}

	return nil
}

// GetQueuedEmail retrieves a queued email by ID
func (s *EmailQueueService) GetQueuedEmail(queueID uuid.UUID) (*models.EmailQueue, error) {
	query := `
		SELECT id, user_id, form_id, submission_id, template_id, provider_id,
		       to_emails, cc_emails, bcc_emails, subject, html_content, text_content,
		       variables, scheduled_at, sent_at, status, attempts, last_error,
		       priority, created_at, updated_at
		FROM email_queue WHERE id = ?`

	var email models.EmailQueue
	var formID, submissionID, providerID sql.NullString
	var sentAt sql.NullTime
	var toEmailsJSON, ccEmailsJSON, bccEmailsJSON, variablesJSON []byte

	err := s.db.QueryRow(query, queueID).Scan(
		&email.ID, &email.UserID, &formID, &submissionID,
		&email.TemplateID, &providerID, &toEmailsJSON, &ccEmailsJSON,
		&bccEmailsJSON, &email.Subject, &email.HTMLContent, &email.TextContent,
		&variablesJSON, &email.ScheduledAt, &sentAt, &email.Status,
		&email.Attempts, &email.LastError, &email.Priority,
		&email.CreatedAt, &email.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get queued email: %w", err)
	}

	// Parse optional fields
	if formID.Valid {
		if fid, err := uuid.Parse(formID.String); err == nil {
			email.FormID = &fid
		}
	}
	if submissionID.Valid {
		if sid, err := uuid.Parse(submissionID.String); err == nil {
			email.SubmissionID = &sid
		}
	}
	if providerID.Valid {
		if pid, err := uuid.Parse(providerID.String); err == nil {
			email.ProviderID = &pid
		}
	}
	if sentAt.Valid {
		email.SentAt = &sentAt.Time
	}

	// Parse JSON fields
	if len(toEmailsJSON) > 0 {
		json.Unmarshal(toEmailsJSON, &email.ToEmails)
	}
	if len(ccEmailsJSON) > 0 {
		json.Unmarshal(ccEmailsJSON, &email.CCEmails)
	}
	if len(bccEmailsJSON) > 0 {
		json.Unmarshal(bccEmailsJSON, &email.BCCEmails)
	}
	if len(variablesJSON) > 0 {
		json.Unmarshal(variablesJSON, &email.Variables)
	}

	return &email, nil
}

// ListQueuedEmails retrieves queued emails with filtering and pagination
func (s *EmailQueueService) ListQueuedEmails(userID *uuid.UUID, status *models.EmailStatus, limit, offset int) ([]models.EmailQueue, error) {
	query := `
		SELECT id, user_id, form_id, submission_id, template_id, provider_id,
		       to_emails, cc_emails, bcc_emails, subject, html_content, text_content,
		       variables, scheduled_at, sent_at, status, attempts, last_error,
		       priority, created_at, updated_at
		FROM email_queue WHERE 1=1`
	
	var args []interface{}

	if userID != nil {
		query += " AND user_id = ?"
		args = append(args, *userID)
	}

	if status != nil {
		query += " AND status = ?"
		args = append(args, *status)
	}

	query += " ORDER BY priority DESC, scheduled_at ASC"
	
	if limit > 0 {
		query += " LIMIT ?"
		args = append(args, limit)
		
		if offset > 0 {
			query += " OFFSET ?"
			args = append(args, offset)
		}
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list queued emails: %w", err)
	}
	defer rows.Close()

	var emails []models.EmailQueue
	for rows.Next() {
		var email models.EmailQueue
		var formID, submissionID, providerID sql.NullString
		var sentAt sql.NullTime
		var toEmailsJSON, ccEmailsJSON, bccEmailsJSON, variablesJSON []byte

		err := rows.Scan(
			&email.ID, &email.UserID, &formID, &submissionID,
			&email.TemplateID, &providerID, &toEmailsJSON, &ccEmailsJSON,
			&bccEmailsJSON, &email.Subject, &email.HTMLContent, &email.TextContent,
			&variablesJSON, &email.ScheduledAt, &sentAt, &email.Status,
			&email.Attempts, &email.LastError, &email.Priority,
			&email.CreatedAt, &email.UpdatedAt,
		)
		if err != nil {
			continue // Skip invalid entries
		}

		// Parse optional fields
		if formID.Valid {
			if fid, err := uuid.Parse(formID.String); err == nil {
				email.FormID = &fid
			}
		}
		if submissionID.Valid {
			if sid, err := uuid.Parse(submissionID.String); err == nil {
				email.SubmissionID = &sid
			}
		}
		if providerID.Valid {
			if pid, err := uuid.Parse(providerID.String); err == nil {
				email.ProviderID = &pid
			}
		}
		if sentAt.Valid {
			email.SentAt = &sentAt.Time
		}

		// Parse JSON fields
		if len(toEmailsJSON) > 0 {
			json.Unmarshal(toEmailsJSON, &email.ToEmails)
		}
		if len(ccEmailsJSON) > 0 {
			json.Unmarshal(ccEmailsJSON, &email.CCEmails)
		}
		if len(bccEmailsJSON) > 0 {
			json.Unmarshal(bccEmailsJSON, &email.BCCEmails)
		}
		if len(variablesJSON) > 0 {
			json.Unmarshal(variablesJSON, &email.Variables)
		}

		emails = append(emails, email)
	}

	return emails, nil
}

// GetQueueStats returns statistics about the email queue
func (s *EmailQueueService) GetQueueStats(userID *uuid.UUID) (*QueueStats, error) {
	query := `
		SELECT 
			COUNT(*) as total,
			SUM(CASE WHEN status = 'pending' THEN 1 ELSE 0 END) as pending,
			SUM(CASE WHEN status = 'scheduled' THEN 1 ELSE 0 END) as scheduled,
			SUM(CASE WHEN status = 'sending' THEN 1 ELSE 0 END) as sending,
			SUM(CASE WHEN status = 'sent' THEN 1 ELSE 0 END) as sent,
			SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END) as failed
		FROM email_queue`
	
	var args []interface{}
	if userID != nil {
		query += " WHERE user_id = ?"
		args = append(args, *userID)
	}

	var stats QueueStats
	err := s.db.QueryRow(query, args...).Scan(
		&stats.Total, &stats.Pending, &stats.Scheduled,
		&stats.Sending, &stats.Sent, &stats.Failed,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get queue stats: %w", err)
	}

	return &stats, nil
}

// StartProcessor starts the email queue processor
func (s *EmailQueueService) StartProcessor() error {
	s.processingMux.Lock()
	defer s.processingMux.Unlock()

	if s.isProcessing {
		return fmt.Errorf("processor is already running")
	}

	s.isProcessing = true
	
	go s.processQueueLoop()
	
	log.Println("Email queue processor started")
	return nil
}

// StopProcessor stops the email queue processor
func (s *EmailQueueService) StopProcessor() error {
	s.processingMux.Lock()
	defer s.processingMux.Unlock()

	if !s.isProcessing {
		return fmt.Errorf("processor is not running")
	}

	s.stopChan <- true
	s.isProcessing = false
	
	log.Println("Email queue processor stopped")
	return nil
}

// IsProcessing returns true if the processor is currently running
func (s *EmailQueueService) IsProcessing() bool {
	s.processingMux.RLock()
	defer s.processingMux.RUnlock()
	return s.isProcessing
}

// ProcessPendingEmails manually processes pending emails
func (s *EmailQueueService) ProcessPendingEmails() (*ProcessingResult, error) {
	return s.processPendingEmails()
}

// UpdateEmailStatus updates the status of a queued email
func (s *EmailQueueService) UpdateEmailStatus(queueID uuid.UUID, status models.EmailStatus, error string) error {
	now := time.Now()
	
	query := `UPDATE email_queue SET status = ?, last_error = ?, updated_at = ?`
	args := []interface{}{status, error, now}
	
	// Set sent_at if status is sent
	if status == models.EmailStatusSent {
		query += `, sent_at = ?`
		args = append(args, now)
	}
	
	query += ` WHERE id = ?`
	args = append(args, queueID)

	_, err := s.db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("failed to update email status: %w", err)
	}

	return nil
}

// IncrementAttempts increments the retry attempts for an email
func (s *EmailQueueService) IncrementAttempts(queueID uuid.UUID) error {
	query := `UPDATE email_queue SET attempts = attempts + 1, updated_at = ? WHERE id = ?`
	
	_, err := s.db.Exec(query, time.Now(), queueID)
	if err != nil {
		return fmt.Errorf("failed to increment attempts: %w", err)
	}

	return nil
}

// RetryFailedEmail retries a failed email with exponential backoff
func (s *EmailQueueService) RetryFailedEmail(queueID uuid.UUID) error {
	email, err := s.GetQueuedEmail(queueID)
	if err != nil {
		return fmt.Errorf("failed to get email for retry: %w", err)
	}

	// Check if we haven't exceeded max attempts
	if email.Attempts >= s.config.RetryAttempts {
		return fmt.Errorf("maximum retry attempts exceeded")
	}

	// Calculate next retry time with exponential backoff
	backoffDuration := s.config.RetryDelay * time.Duration(1<<email.Attempts) // 2^attempts
	nextRetry := time.Now().Add(backoffDuration)

	// Reset status and schedule retry
	query := `UPDATE email_queue SET status = ?, scheduled_at = ?, last_error = '', updated_at = ? WHERE id = ?`
	
	_, err = s.db.Exec(query, models.EmailStatusScheduled, nextRetry, time.Now(), queueID)
	if err != nil {
		return fmt.Errorf("failed to schedule retry: %w", err)
	}

	return nil
}

// CancelEmail cancels a queued email
func (s *EmailQueueService) CancelEmail(queueID uuid.UUID) error {
	query := `UPDATE email_queue SET status = ?, updated_at = ? WHERE id = ? AND status IN (?, ?)`
	
	_, err := s.db.Exec(query, models.EmailStatusCancelled, time.Now(), queueID, models.EmailStatusPending, models.EmailStatusScheduled)
	if err != nil {
		return fmt.Errorf("failed to cancel email: %w", err)
	}

	return nil
}

// CleanupOldEmails removes old email records based on retention policy
func (s *EmailQueueService) CleanupOldEmails(retentionDays int) error {
	cutoffDate := time.Now().AddDate(0, 0, -retentionDays)
	
	query := `DELETE FROM email_queue WHERE created_at < ? AND status IN (?, ?, ?)`
	
	result, err := s.db.Exec(query, cutoffDate, models.EmailStatusSent, models.EmailStatusFailed, models.EmailStatusCancelled)
	if err != nil {
		return fmt.Errorf("failed to cleanup old emails: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	log.Printf("Cleaned up %d old email records", rowsAffected)

	return nil
}

// Private methods

func (s *EmailQueueService) processQueueLoop() {
	ticker := time.NewTicker(s.config.ProcessInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if result, err := s.processPendingEmails(); err != nil {
				log.Printf("Error processing email queue: %v", err)
			} else if result.Processed > 0 {
				log.Printf("Processed %d emails: %d sent, %d failed", result.Processed, result.Sent, result.Failed)
			}
		case <-s.stopChan:
			return
		}
	}
}

func (s *EmailQueueService) processPendingEmails() (*ProcessingResult, error) {
	// Get pending and scheduled emails that are ready to be sent
	query := `
		SELECT id FROM email_queue 
		WHERE (status = ? OR (status = ? AND scheduled_at <= ?))
		ORDER BY priority DESC, scheduled_at ASC
		LIMIT ?`

	now := time.Now()
	rows, err := s.db.Query(query, models.EmailStatusPending, models.EmailStatusScheduled, now, s.config.BatchSize)
	if err != nil {
		return nil, fmt.Errorf("failed to get pending emails: %w", err)
	}
	defer rows.Close()

	var emailIDs []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err == nil {
			emailIDs = append(emailIDs, id)
		}
	}

	if len(emailIDs) == 0 {
		return &ProcessingResult{}, nil
	}

	// Process emails concurrently
	resultChan := make(chan bool, len(emailIDs))
	errorChan := make(chan string, len(emailIDs))
	
	// Create worker pool
	workerCount := s.config.MaxWorkers
	if len(emailIDs) < workerCount {
		workerCount = len(emailIDs)
	}

	jobs := make(chan uuid.UUID, len(emailIDs))
	
	// Start workers
	for i := 0; i < workerCount; i++ {
		go s.emailWorker(jobs, resultChan, errorChan)
	}

	// Send jobs
	for _, id := range emailIDs {
		jobs <- id
	}
	close(jobs)

	// Collect results
	var sent, failed int
	var errors []string
	
	for i := 0; i < len(emailIDs); i++ {
		select {
		case success := <-resultChan:
			if success {
				sent++
			} else {
				failed++
			}
		case errMsg := <-errorChan:
			errors = append(errors, errMsg)
			failed++
		case <-time.After(time.Minute * 5): // Timeout after 5 minutes
			errors = append(errors, "Processing timeout")
			failed++
		}
	}

	return &ProcessingResult{
		Processed: len(emailIDs),
		Sent:      sent,
		Failed:    failed,
		Errors:    errors,
	}, nil
}

func (s *EmailQueueService) emailWorker(jobs <-chan uuid.UUID, results chan<- bool, errors chan<- string) {
	for emailID := range jobs {
		success := s.processEmail(emailID)
		if success {
			results <- true
		} else {
			results <- false
		}
	}
}

func (s *EmailQueueService) processEmail(emailID uuid.UUID) bool {
	// Mark as sending
	if err := s.UpdateEmailStatus(emailID, models.EmailStatusSending, ""); err != nil {
		return false
	}

	// Get email details
	email, err := s.GetQueuedEmail(emailID)
	if err != nil {
		s.UpdateEmailStatus(emailID, models.EmailStatusFailed, err.Error())
		return false
	}

	// Determine provider
	var providerID uuid.UUID
	if email.ProviderID != nil {
		providerID = *email.ProviderID
	} else {
		// Try to get default provider for user
		defaultProvider, err := s.providerService.GetDefaultProvider(email.UserID)
		if err != nil {
			s.UpdateEmailStatus(emailID, models.EmailStatusFailed, "No email provider configured")
			return false
		}
		providerID = defaultProvider.ID
	}

	// Build email message
	message := EmailMessage{
		To:          email.ToEmails,
		CC:          email.CCEmails,
		BCC:         email.BCCEmails,
		Subject:     email.Subject,
		HTMLContent: email.HTMLContent,
		TextContent: email.TextContent,
		TrackOpens:  true,  // Could be configured per email
		TrackClicks: true,  // Could be configured per email
	}

	// Add reply-to if specified in variables
	if replyTo, ok := email.Variables["reply_to"].(string); ok && replyTo != "" {
		message.ReplyTo = replyTo
	}

	// Send email
	result, err := s.providerService.SendEmail(providerID, message)
	if err != nil {
		s.IncrementAttempts(emailID)
		s.UpdateEmailStatus(emailID, models.EmailStatusFailed, err.Error())
		
		// Schedule retry if we haven't exceeded max attempts
		if email.Attempts < s.config.RetryAttempts {
			s.RetryFailedEmail(emailID)
		}
		
		return false
	}

	if !result.Success {
		s.IncrementAttempts(emailID)
		s.UpdateEmailStatus(emailID, models.EmailStatusFailed, result.Error)
		
		// Schedule retry if we haven't exceeded max attempts
		if email.Attempts < s.config.RetryAttempts {
			s.RetryFailedEmail(emailID)
		}
		
		return false
	}

	// Mark as sent
	s.UpdateEmailStatus(emailID, models.EmailStatusSent, "")

	// Create analytics entries for tracking
	if s.analyticsService != nil {
		for _, recipient := range email.ToEmails {
			analytics := &models.EmailAnalytics{
				ID:           uuid.New(),
				QueueID:      email.ID,
				UserID:       email.UserID,
				FormID:       email.FormID,
				TemplateID:   email.TemplateID,
				EmailAddress: recipient,
				DeliveredAt:  timePtr(time.Now()),
				CreatedAt:    time.Now(),
				UpdatedAt:    time.Now(),
			}
			s.analyticsService.CreateAnalytics(analytics)
		}
	}

	return true
}

// Helper function
func timePtr(t time.Time) *time.Time {
	return &t
}