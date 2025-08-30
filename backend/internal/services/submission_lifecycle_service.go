package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"formhub/internal/models"
	"formhub/pkg/database"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type SubmissionLifecycleService struct {
	db               *sqlx.DB
	redis            *database.RedisClient
	analyticsService *AnalyticsService
}

func NewSubmissionLifecycleService(db *sqlx.DB, redis *database.RedisClient, analyticsService *AnalyticsService) *SubmissionLifecycleService {
	return &SubmissionLifecycleService{
		db:               db,
		redis:            redis,
		analyticsService: analyticsService,
	}
}

// CreateSubmissionLifecycle creates a new submission lifecycle entry
func (s *SubmissionLifecycleService) CreateSubmissionLifecycle(ctx context.Context, submissionID, formID, userID uuid.UUID) (*models.SubmissionLifecycle, error) {
	trackingID := s.analyticsService.GenerateTrackingID()
	
	lifecycle := &models.SubmissionLifecycle{
		ID:           uuid.New(),
		SubmissionID: submissionID,
		FormID:       formID,
		UserID:       userID,
		TrackingID:   trackingID,
		Status:       models.SubmissionStatusReceived,
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}

	query := `
		INSERT INTO submission_lifecycle (
			id, submission_id, form_id, user_id, tracking_id, status, 
			created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := s.db.ExecContext(ctx, query,
		lifecycle.ID, lifecycle.SubmissionID, lifecycle.FormID, lifecycle.UserID,
		lifecycle.TrackingID, lifecycle.Status, lifecycle.CreatedAt, lifecycle.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create submission lifecycle: %w", err)
	}

	// Cache in Redis for quick access
	s.cacheLifecycleData(ctx, lifecycle)

	return lifecycle, nil
}

// UpdateSubmissionStatus updates the status of a submission lifecycle
func (s *SubmissionLifecycleService) UpdateSubmissionStatus(ctx context.Context, submissionID uuid.UUID, status models.SubmissionStatus, processingTimeMs *int) error {
	updates := []string{"status = ?", "updated_at = ?"}
	args := []interface{}{status, time.Now().UTC()}

	if processingTimeMs != nil {
		updates = append(updates, "processing_time_ms = ?")
		args = append(args, *processingTimeMs)
	}

	query := fmt.Sprintf(`
		UPDATE submission_lifecycle 
		SET %s
		WHERE submission_id = ?
	`, strings.Join(updates, ", "))

	args = append(args, submissionID)

	_, err := s.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to update submission status: %w", err)
	}

	// Update cache
	s.updateLifecycleCache(ctx, submissionID, map[string]interface{}{
		"status":             status,
		"processing_time_ms": processingTimeMs,
		"updated_at":         time.Now().UTC(),
	})

	// Record status change event for analytics
	s.recordStatusChangeEvent(ctx, submissionID, status)

	return nil
}

// UpdateValidationErrors updates validation errors for a submission
func (s *SubmissionLifecycleService) UpdateValidationErrors(ctx context.Context, submissionID uuid.UUID, errors []string) error {
	errorsJSON, _ := json.Marshal(errors)

	query := `
		UPDATE submission_lifecycle 
		SET validation_errors = ?, updated_at = ?
		WHERE submission_id = ?
	`

	_, err := s.db.ExecContext(ctx, query, string(errorsJSON), time.Now().UTC(), submissionID)
	if err != nil {
		return fmt.Errorf("failed to update validation errors: %w", err)
	}

	s.updateLifecycleCache(ctx, submissionID, map[string]interface{}{
		"validation_errors": errors,
		"updated_at":        time.Now().UTC(),
	})

	return nil
}

// UpdateSpamDetection updates spam detection results
func (s *SubmissionLifecycleService) UpdateSpamDetection(ctx context.Context, submissionID uuid.UUID, score float64, reasons []string) error {
	reasonsJSON, _ := json.Marshal(reasons)

	query := `
		UPDATE submission_lifecycle 
		SET spam_detection_score = ?, spam_detection_reasons = ?, updated_at = ?
		WHERE submission_id = ?
	`

	_, err := s.db.ExecContext(ctx, query, score, string(reasonsJSON), time.Now().UTC(), submissionID)
	if err != nil {
		return fmt.Errorf("failed to update spam detection: %w", err)
	}

	s.updateLifecycleCache(ctx, submissionID, map[string]interface{}{
		"spam_detection_score":   score,
		"spam_detection_reasons": reasons,
		"updated_at":             time.Now().UTC(),
	})

	return nil
}

// UpdateEmailDelivery updates email delivery status and timing
func (s *SubmissionLifecycleService) UpdateEmailDelivery(ctx context.Context, submissionID uuid.UUID, status models.EmailDeliveryStatus, deliveryTimeMs int) error {
	query := `
		UPDATE submission_lifecycle 
		SET email_delivery_status = ?, email_delivery_time_ms = ?, updated_at = ?
		WHERE submission_id = ?
	`

	_, err := s.db.ExecContext(ctx, query, status, deliveryTimeMs, time.Now().UTC(), submissionID)
	if err != nil {
		return fmt.Errorf("failed to update email delivery: %w", err)
	}

	s.updateLifecycleCache(ctx, submissionID, map[string]interface{}{
		"email_delivery_status":   status,
		"email_delivery_time_ms":  deliveryTimeMs,
		"updated_at":              time.Now().UTC(),
	})

	return nil
}

// UpdateWebhookDelivery updates webhook delivery status and timing
func (s *SubmissionLifecycleService) UpdateWebhookDelivery(ctx context.Context, submissionID uuid.UUID, status models.WebhookDeliveryStatus, deliveryTimeMs int, responseCode *int) error {
	query := `
		UPDATE submission_lifecycle 
		SET webhook_delivery_status = ?, webhook_delivery_time_ms = ?, webhook_response_code = ?, updated_at = ?
		WHERE submission_id = ?
	`

	_, err := s.db.ExecContext(ctx, query, status, deliveryTimeMs, responseCode, time.Now().UTC(), submissionID)
	if err != nil {
		return fmt.Errorf("failed to update webhook delivery: %w", err)
	}

	s.updateLifecycleCache(ctx, submissionID, map[string]interface{}{
		"webhook_delivery_status":   status,
		"webhook_delivery_time_ms":  deliveryTimeMs,
		"webhook_response_code":     responseCode,
		"updated_at":                time.Now().UTC(),
	})

	return nil
}

// RecordResponse records a response to the submission
func (s *SubmissionLifecycleService) RecordResponse(ctx context.Context, submissionID uuid.UUID, method models.ResponseMethod, notes string) error {
	responseTime := time.Now().UTC()

	query := `
		UPDATE submission_lifecycle 
		SET response_time = ?, response_method = ?, notes = ?, status = ?, updated_at = ?
		WHERE submission_id = ?
	`

	_, err := s.db.ExecContext(ctx, query, responseTime, method, notes, models.SubmissionStatusResponded, time.Now().UTC(), submissionID)
	if err != nil {
		return fmt.Errorf("failed to record response: %w", err)
	}

	s.updateLifecycleCache(ctx, submissionID, map[string]interface{}{
		"response_time":   responseTime,
		"response_method": method,
		"notes":           notes,
		"status":          models.SubmissionStatusResponded,
		"updated_at":      time.Now().UTC(),
	})

	// Record response event for analytics
	s.recordStatusChangeEvent(ctx, submissionID, models.SubmissionStatusResponded)

	return nil
}

// GetSubmissionLifecycle retrieves submission lifecycle by submission ID
func (s *SubmissionLifecycleService) GetSubmissionLifecycle(ctx context.Context, submissionID uuid.UUID) (*models.SubmissionLifecycle, error) {
	// Try cache first
	if lifecycle := s.getLifecycleFromCache(ctx, submissionID); lifecycle != nil {
		return lifecycle, nil
	}

	// Query from database
	var lifecycle models.SubmissionLifecycle
	var validationErrorsJSON, spamReasonsJSON sql.NullString

	query := `
		SELECT id, submission_id, form_id, user_id, tracking_id, status, 
		       processing_time_ms, validation_errors, spam_detection_score, 
		       spam_detection_reasons, email_delivery_status, email_delivery_time_ms, 
		       webhook_delivery_status, webhook_delivery_time_ms, webhook_response_code, 
		       response_time, response_method, notes, created_at, updated_at
		FROM submission_lifecycle 
		WHERE submission_id = ?
	`

	err := s.db.QueryRowContext(ctx, query, submissionID).Scan(
		&lifecycle.ID, &lifecycle.SubmissionID, &lifecycle.FormID, &lifecycle.UserID,
		&lifecycle.TrackingID, &lifecycle.Status, &lifecycle.ProcessingTimeMs,
		&validationErrorsJSON, &lifecycle.SpamDetectionScore, &spamReasonsJSON,
		&lifecycle.EmailDeliveryStatus, &lifecycle.EmailDeliveryTimeMs,
		&lifecycle.WebhookDeliveryStatus, &lifecycle.WebhookDeliveryTimeMs,
		&lifecycle.WebhookResponseCode, &lifecycle.ResponseTime, &lifecycle.ResponseMethod,
		&lifecycle.Notes, &lifecycle.CreatedAt, &lifecycle.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get submission lifecycle: %w", err)
	}

	// Parse JSON fields
	if validationErrorsJSON.Valid {
		json.Unmarshal([]byte(validationErrorsJSON.String), &lifecycle.ValidationErrors)
	}
	if spamReasonsJSON.Valid {
		json.Unmarshal([]byte(spamReasonsJSON.String), &lifecycle.SpamDetectionReasons)
	}

	// Cache the result
	s.cacheLifecycleData(ctx, &lifecycle)

	return &lifecycle, nil
}

// GetSubmissionLifecycleByTrackingID retrieves submission lifecycle by tracking ID
func (s *SubmissionLifecycleService) GetSubmissionLifecycleByTrackingID(ctx context.Context, trackingID string) (*models.SubmissionLifecycle, error) {
	var lifecycle models.SubmissionLifecycle
	var validationErrorsJSON, spamReasonsJSON sql.NullString

	query := `
		SELECT id, submission_id, form_id, user_id, tracking_id, status, 
		       processing_time_ms, validation_errors, spam_detection_score, 
		       spam_detection_reasons, email_delivery_status, email_delivery_time_ms, 
		       webhook_delivery_status, webhook_delivery_time_ms, webhook_response_code, 
		       response_time, response_method, notes, created_at, updated_at
		FROM submission_lifecycle 
		WHERE tracking_id = ?
	`

	err := s.db.QueryRowContext(ctx, query, trackingID).Scan(
		&lifecycle.ID, &lifecycle.SubmissionID, &lifecycle.FormID, &lifecycle.UserID,
		&lifecycle.TrackingID, &lifecycle.Status, &lifecycle.ProcessingTimeMs,
		&validationErrorsJSON, &lifecycle.SpamDetectionScore, &spamReasonsJSON,
		&lifecycle.EmailDeliveryStatus, &lifecycle.EmailDeliveryTimeMs,
		&lifecycle.WebhookDeliveryStatus, &lifecycle.WebhookDeliveryTimeMs,
		&lifecycle.WebhookResponseCode, &lifecycle.ResponseTime, &lifecycle.ResponseMethod,
		&lifecycle.Notes, &lifecycle.CreatedAt, &lifecycle.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("submission not found")
		}
		return nil, fmt.Errorf("failed to get submission lifecycle: %w", err)
	}

	// Parse JSON fields
	if validationErrorsJSON.Valid {
		json.Unmarshal([]byte(validationErrorsJSON.String), &lifecycle.ValidationErrors)
	}
	if spamReasonsJSON.Valid {
		json.Unmarshal([]byte(spamReasonsJSON.String), &lifecycle.SpamDetectionReasons)
	}

	return &lifecycle, nil
}

// GetSubmissionsByStatus retrieves submissions by status for a user
func (s *SubmissionLifecycleService) GetSubmissionsByStatus(ctx context.Context, userID uuid.UUID, status models.SubmissionStatus, limit, offset int) ([]models.SubmissionLifecycle, error) {
	query := `
		SELECT sl.id, sl.submission_id, sl.form_id, sl.user_id, sl.tracking_id, sl.status, 
		       sl.processing_time_ms, sl.validation_errors, sl.spam_detection_score, 
		       sl.spam_detection_reasons, sl.email_delivery_status, sl.email_delivery_time_ms, 
		       sl.webhook_delivery_status, sl.webhook_delivery_time_ms, sl.webhook_response_code, 
		       sl.response_time, sl.response_method, sl.notes, sl.created_at, sl.updated_at
		FROM submission_lifecycle sl
		WHERE sl.user_id = ? AND sl.status = ?
		ORDER BY sl.updated_at DESC
		LIMIT ? OFFSET ?
	`

	rows, err := s.db.QueryContext(ctx, query, userID, status, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get submissions by status: %w", err)
	}
	defer rows.Close()

	var lifecycles []models.SubmissionLifecycle
	for rows.Next() {
		var lifecycle models.SubmissionLifecycle
		var validationErrorsJSON, spamReasonsJSON sql.NullString

		err := rows.Scan(
			&lifecycle.ID, &lifecycle.SubmissionID, &lifecycle.FormID, &lifecycle.UserID,
			&lifecycle.TrackingID, &lifecycle.Status, &lifecycle.ProcessingTimeMs,
			&validationErrorsJSON, &lifecycle.SpamDetectionScore, &spamReasonsJSON,
			&lifecycle.EmailDeliveryStatus, &lifecycle.EmailDeliveryTimeMs,
			&lifecycle.WebhookDeliveryStatus, &lifecycle.WebhookDeliveryTimeMs,
			&lifecycle.WebhookResponseCode, &lifecycle.ResponseTime, &lifecycle.ResponseMethod,
			&lifecycle.Notes, &lifecycle.CreatedAt, &lifecycle.UpdatedAt,
		)

		if err != nil {
			continue
		}

		// Parse JSON fields
		if validationErrorsJSON.Valid {
			json.Unmarshal([]byte(validationErrorsJSON.String), &lifecycle.ValidationErrors)
		}
		if spamReasonsJSON.Valid {
			json.Unmarshal([]byte(spamReasonsJSON.String), &lifecycle.SpamDetectionReasons)
		}

		lifecycles = append(lifecycles, lifecycle)
	}

	return lifecycles, nil
}

// GetLifecycleStats gets lifecycle statistics for a user or form
func (s *SubmissionLifecycleService) GetLifecycleStats(ctx context.Context, userID uuid.UUID, formID *uuid.UUID, startDate, endDate time.Time) (map[string]interface{}, error) {
	whereClause := "WHERE user_id = ? AND created_at BETWEEN ? AND ?"
	args := []interface{}{userID, startDate, endDate}

	if formID != nil {
		whereClause += " AND form_id = ?"
		args = append(args, *formID)
	}

	query := fmt.Sprintf(`
		SELECT 
			status,
			COUNT(*) as count,
			AVG(processing_time_ms) as avg_processing_time,
			AVG(email_delivery_time_ms) as avg_email_delivery_time,
			AVG(webhook_delivery_time_ms) as avg_webhook_delivery_time
		FROM submission_lifecycle 
		%s
		GROUP BY status
	`, whereClause)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get lifecycle stats: %w", err)
	}
	defer rows.Close()

	stats := make(map[string]interface{})
	statusCounts := make(map[string]int)
	processingTimes := make(map[string]float64)
	emailDeliveryTimes := make(map[string]float64)
	webhookDeliveryTimes := make(map[string]float64)

	totalSubmissions := 0

	for rows.Next() {
		var status string
		var count int
		var avgProcessingTime, avgEmailDeliveryTime, avgWebhookDeliveryTime sql.NullFloat64

		err := rows.Scan(&status, &count, &avgProcessingTime, &avgEmailDeliveryTime, &avgWebhookDeliveryTime)
		if err != nil {
			continue
		}

		statusCounts[status] = count
		totalSubmissions += count

		if avgProcessingTime.Valid {
			processingTimes[status] = avgProcessingTime.Float64
		}
		if avgEmailDeliveryTime.Valid {
			emailDeliveryTimes[status] = avgEmailDeliveryTime.Float64
		}
		if avgWebhookDeliveryTime.Valid {
			webhookDeliveryTimes[status] = avgWebhookDeliveryTime.Float64
		}
	}

	stats["total_submissions"] = totalSubmissions
	stats["status_counts"] = statusCounts
	stats["average_processing_times"] = processingTimes
	stats["average_email_delivery_times"] = emailDeliveryTimes
	stats["average_webhook_delivery_times"] = webhookDeliveryTimes

	// Calculate percentages
	statusPercentages := make(map[string]float64)
	for status, count := range statusCounts {
		if totalSubmissions > 0 {
			statusPercentages[status] = float64(count) / float64(totalSubmissions) * 100
		}
	}
	stats["status_percentages"] = statusPercentages

	return stats, nil
}

// cacheLifecycleData caches lifecycle data in Redis
func (s *SubmissionLifecycleService) cacheLifecycleData(ctx context.Context, lifecycle *models.SubmissionLifecycle) {
	key := fmt.Sprintf("lifecycle:%s", lifecycle.SubmissionID)
	data, _ := json.Marshal(lifecycle)
	s.redis.Client.Set(ctx, key, data, 1*time.Hour)
}

// getLifecycleFromCache retrieves lifecycle data from cache
func (s *SubmissionLifecycleService) getLifecycleFromCache(ctx context.Context, submissionID uuid.UUID) *models.SubmissionLifecycle {
	key := fmt.Sprintf("lifecycle:%s", submissionID)
	data, err := s.redis.Client.Get(ctx, key).Result()
	if err != nil {
		return nil
	}

	var lifecycle models.SubmissionLifecycle
	if json.Unmarshal([]byte(data), &lifecycle) != nil {
		return nil
	}

	return &lifecycle
}

// updateLifecycleCache updates specific fields in the cache
func (s *SubmissionLifecycleService) updateLifecycleCache(ctx context.Context, submissionID uuid.UUID, updates map[string]interface{}) {
	lifecycle := s.getLifecycleFromCache(ctx, submissionID)
	if lifecycle == nil {
		return
	}

	// Update fields using reflection or manual mapping
	// For simplicity, we'll just refresh the cache by deleting it
	// The next read will fetch from database and cache again
	key := fmt.Sprintf("lifecycle:%s", submissionID)
	s.redis.Client.Del(ctx, key)
}

// recordStatusChangeEvent records a status change event for analytics
func (s *SubmissionLifecycleService) recordStatusChangeEvent(ctx context.Context, submissionID uuid.UUID, status models.SubmissionStatus) {
	// Get submission details for the event
	var formID, userID uuid.UUID
	err := s.db.QueryRowContext(ctx, 
		"SELECT form_id, user_id FROM submission_lifecycle WHERE submission_id = ?", 
		submissionID).Scan(&formID, &userID)
	if err != nil {
		log.Printf("Failed to get submission details for event: %v", err)
		return
	}

	// Create analytics event
	event := &models.FormAnalyticsEvent{
		FormID:    formID,
		UserID:    userID,
		SessionID: fmt.Sprintf("lifecycle-%s", submissionID),
		EventType: models.AnalyticsEventType(fmt.Sprintf("submission_status_%s", status)),
		IPAddress: "127.0.0.1", // System event
		EventData: map[string]interface{}{
			"submission_id": submissionID,
			"status":        status,
			"timestamp":     time.Now().UTC(),
		},
		CreatedAt: time.Now().UTC(),
	}

	// Record the event (fire and forget)
	go s.analyticsService.RecordEvent(context.Background(), event)
}

// ArchiveOldSubmissions archives old submissions based on retention policy
func (s *SubmissionLifecycleService) ArchiveOldSubmissions(ctx context.Context, retentionDays int) error {
	cutoffDate := time.Now().UTC().AddDate(0, 0, -retentionDays)

	query := `
		UPDATE submission_lifecycle 
		SET status = ?, updated_at = ?
		WHERE status IN ('completed', 'responded') AND updated_at < ?
	`

	result, err := s.db.ExecContext(ctx, query, models.SubmissionStatusArchived, time.Now().UTC(), cutoffDate)
	if err != nil {
		return fmt.Errorf("failed to archive old submissions: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	log.Printf("Archived %d old submissions", rowsAffected)

	return nil
}

// GetPendingActions gets submissions that require action
func (s *SubmissionLifecycleService) GetPendingActions(ctx context.Context, userID uuid.UUID) (map[string]interface{}, error) {
	queries := map[string]string{
		"pending_email_delivery": `
			SELECT COUNT(*) FROM submission_lifecycle 
			WHERE user_id = ? AND email_delivery_status = 'pending'
		`,
		"failed_email_delivery": `
			SELECT COUNT(*) FROM submission_lifecycle 
			WHERE user_id = ? AND email_delivery_status IN ('failed', 'bounced')
		`,
		"failed_webhook_delivery": `
			SELECT COUNT(*) FROM submission_lifecycle 
			WHERE user_id = ? AND webhook_delivery_status = 'failed'
		`,
		"high_spam_score": `
			SELECT COUNT(*) FROM submission_lifecycle 
			WHERE user_id = ? AND spam_detection_score > 0.8 AND status != 'spam_flagged'
		`,
		"awaiting_response": `
			SELECT COUNT(*) FROM submission_lifecycle 
			WHERE user_id = ? AND status IN ('email_sent', 'completed') AND response_time IS NULL
		`,
	}

	results := make(map[string]interface{})

	for key, query := range queries {
		var count int
		err := s.db.GetContext(ctx, &count, query, userID)
		if err != nil {
			log.Printf("Failed to get %s count: %v", key, err)
			count = 0
		}
		results[key] = count
	}

	return results, nil
}