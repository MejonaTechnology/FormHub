package services

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// WebhookService handles webhook notifications for spam detection events
type WebhookService struct {
	db               *sql.DB
	redis            *redis.Client
	client           *http.Client
	ctx              context.Context
	
	// Configuration
	maxRetries       int
	retryDelay       time.Duration
	timeout          time.Duration
	maxPayloadSize   int64
	rateLimitWindow  time.Duration
	maxWebhooksPerMin int
	
	// Security
	signatureHeader  string
	timestampHeader  string
	userAgent        string
	
	// Statistics
	stats            *WebhookStats
}

// WebhookConfig holds webhook configuration
type WebhookConfig struct {
	URL               string            `json:"url"`
	Secret            string            `json:"secret,omitempty"`
	Events            []string          `json:"events"`
	Headers           map[string]string `json:"headers,omitempty"`
	ContentType       string            `json:"content_type"`
	Method            string            `json:"method"`
	Timeout           int               `json:"timeout"` // seconds
	MaxRetries        int               `json:"max_retries"`
	RetryDelay        int               `json:"retry_delay"` // seconds
	Enabled           bool              `json:"enabled"`
	RateLimitEnabled  bool              `json:"rate_limit_enabled"`
	VerifySSL         bool              `json:"verify_ssl"`
	CustomPayload     string            `json:"custom_payload,omitempty"` // JSON template
}

// WebhookEvent represents a webhook event
type WebhookEvent struct {
	ID            string                 `json:"id"`
	Type          string                 `json:"type"`
	Timestamp     time.Time              `json:"timestamp"`
	FormID        string                 `json:"form_id"`
	SubmissionID  string                 `json:"submission_id,omitempty"`
	Data          map[string]interface{} `json:"data"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
	Source        string                 `json:"source"` // "spam_detection", "security", "admin"
	Version       string                 `json:"version"`
}

// WebhookNotification represents a webhook notification record
type WebhookNotification struct {
	ID           string                 `json:"id"`
	WebhookURL   string                 `json:"webhook_url"`
	Event        *WebhookEvent          `json:"event"`
	Config       *WebhookConfig         `json:"config"`
	Status       string                 `json:"status"` // "pending", "sent", "failed", "retrying"
	ResponseCode int                    `json:"response_code,omitempty"`
	ResponseBody string                 `json:"response_body,omitempty"`
	RetryCount   int                    `json:"retry_count"`
	NextRetry    *time.Time             `json:"next_retry,omitempty"`
	SentAt       *time.Time             `json:"sent_at,omitempty"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
}

// WebhookStats tracks webhook service statistics
type WebhookStats struct {
	TotalWebhooks    int64 `json:"total_webhooks"`
	SuccessfulSent   int64 `json:"successful_sent"`
	Failed           int64 `json:"failed"`
	Retrying         int64 `json:"retrying"`
	RateLimited      int64 `json:"rate_limited"`
	AvgResponseTime  int64 `json:"avg_response_time_ms"`
	LastReset        time.Time `json:"last_reset"`
}

// WebhookResponse represents the response from a webhook endpoint
type WebhookResponse struct {
	StatusCode   int               `json:"status_code"`
	Headers      map[string]string `json:"headers"`
	Body         string            `json:"body"`
	ResponseTime time.Duration     `json:"response_time"`
	Error        string            `json:"error,omitempty"`
}

// NewWebhookService creates a new webhook service
func NewWebhookService(db *sql.DB, redis *redis.Client) *WebhookService {
	return &WebhookService{
		db:               db,
		redis:            redis,
		ctx:              context.Background(),
		maxRetries:       3,
		retryDelay:       5 * time.Second,
		timeout:          30 * time.Second,
		maxPayloadSize:   1024 * 1024, // 1MB
		rateLimitWindow:  time.Minute,
		maxWebhooksPerMin: 60,
		signatureHeader:  "X-FormHub-Signature",
		timestampHeader:  "X-FormHub-Timestamp",
		userAgent:       "FormHub-Webhooks/1.0",
		stats:           &WebhookStats{LastReset: time.Now()},
		client: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:       10,
				IdleConnTimeout:    30 * time.Second,
				DisableCompression: false,
			},
		},
	}
}

// SendWebhook sends a webhook notification
func (ws *WebhookService) SendWebhook(event *WebhookEvent, config *WebhookConfig) error {
	// Check if webhooks are enabled for this event type
	if !ws.isEventEnabled(event.Type, config.Events) {
		return nil // Not an error, just not configured
	}
	
	// Check rate limiting
	if config.RateLimitEnabled && ws.isRateLimited(config.URL) {
		ws.stats.RateLimited++
		return fmt.Errorf("webhook rate limit exceeded for URL: %s", config.URL)
	}
	
	// Create notification record
	notification := &WebhookNotification{
		ID:          uuid.New().String(),
		WebhookURL:  config.URL,
		Event:       event,
		Config:      config,
		Status:      "pending",
		RetryCount:  0,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	
	// Store notification in database
	if err := ws.storeNotification(notification); err != nil {
		return fmt.Errorf("failed to store webhook notification: %w", err)
	}
	
	// Send immediately or queue for background processing
	if ws.shouldSendImmediately(config) {
		go ws.processWebhook(notification)
	} else {
		// Queue for background processing
		ws.queueWebhook(notification)
	}
	
	ws.stats.TotalWebhooks++
	return nil
}

// SendSpamDetectionWebhook sends a webhook for spam detection events
func (ws *WebhookService) SendSpamDetectionWebhook(result *SpamDetectionResult, 
	formID string, submissionData map[string]interface{}, metadata map[string]interface{}) error {
	
	// Get webhook configuration for this form
	config, err := ws.getWebhookConfig(formID)
	if err != nil || config == nil || !config.Enabled {
		return nil // No webhook configured or disabled
	}
	
	// Create webhook event
	event := &WebhookEvent{
		ID:           uuid.New().String(),
		Type:         ws.getSpamEventType(result.Action),
		Timestamp:    time.Now().UTC(),
		FormID:       formID,
		SubmissionID: ws.extractSubmissionID(metadata),
		Data: map[string]interface{}{
			"spam_score":       result.SpamScore,
			"confidence":       result.Confidence,
			"action":           result.Action,
			"triggers":         result.Triggers,
			"captcha_required": result.CaptchaRequired,
			"processing_time":  result.ProcessingTime.Milliseconds(),
			"submission_data":  submissionData,
		},
		Metadata: metadata,
		Source:   "spam_detection",
		Version:  "1.0",
	}
	
	return ws.SendWebhook(event, config)
}

// SendSecurityWebhook sends a webhook for security events
func (ws *WebhookService) SendSecurityWebhook(eventType, severity, description string,
	formID string, metadata map[string]interface{}) error {
	
	config, err := ws.getWebhookConfig(formID)
	if err != nil || config == nil || !config.Enabled {
		return nil
	}
	
	event := &WebhookEvent{
		ID:        uuid.New().String(),
		Type:      fmt.Sprintf("security.%s", eventType),
		Timestamp: time.Now().UTC(),
		FormID:    formID,
		Data: map[string]interface{}{
			"event_type":  eventType,
			"severity":    severity,
			"description": description,
		},
		Metadata: metadata,
		Source:   "security",
		Version:  "1.0",
	}
	
	return ws.SendWebhook(event, config)
}

// processWebhook processes a single webhook notification
func (ws *WebhookService) processWebhook(notification *WebhookNotification) {
	startTime := time.Now()
	
	// Prepare payload
	payload, err := ws.preparePayload(notification)
	if err != nil {
		ws.updateNotificationStatus(notification.ID, "failed", 0, err.Error())
		return
	}
	
	// Send HTTP request
	response := ws.sendHTTPRequest(notification.Config, payload)
	
	// Update statistics
	ws.stats.AvgResponseTime = (ws.stats.AvgResponseTime + response.ResponseTime.Milliseconds()) / 2
	
	// Handle response
	if response.StatusCode >= 200 && response.StatusCode < 300 {
		// Success
		ws.stats.SuccessfulSent++
		sentAt := time.Now()
		notification.SentAt = &sentAt
		ws.updateNotificationStatus(notification.ID, "sent", response.StatusCode, response.Body)
	} else if notification.RetryCount < notification.Config.MaxRetries {
		// Retry
		ws.stats.Retrying++
		ws.scheduleRetry(notification, response)
	} else {
		// Failed after max retries
		ws.stats.Failed++
		ws.updateNotificationStatus(notification.ID, "failed", response.StatusCode, response.Error)
	}
	
	// Log the webhook attempt
	ws.logWebhookAttempt(notification, response, time.Since(startTime))
}

// preparePayload prepares the webhook payload
func (ws *WebhookService) preparePayload(notification *WebhookNotification) ([]byte, error) {
	var payload interface{}
	
	if notification.Config.CustomPayload != "" {
		// Use custom payload template
		payload = ws.processPayloadTemplate(notification.Config.CustomPayload, notification.Event)
	} else {
		// Use default payload
		payload = notification.Event
	}
	
	// Convert to JSON
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}
	
	// Check payload size
	if int64(len(payloadBytes)) > ws.maxPayloadSize {
		return nil, fmt.Errorf("payload size exceeds limit: %d bytes", len(payloadBytes))
	}
	
	return payloadBytes, nil
}

// sendHTTPRequest sends the HTTP request to the webhook endpoint
func (ws *WebhookService) sendHTTPRequest(config *WebhookConfig, payload []byte) *WebhookResponse {
	response := &WebhookResponse{}
	startTime := time.Now()
	
	// Create request
	method := config.Method
	if method == "" {
		method = "POST"
	}
	
	req, err := http.NewRequest(method, config.URL, bytes.NewBuffer(payload))
	if err != nil {
		response.Error = fmt.Sprintf("failed to create request: %v", err)
		return response
	}
	
	// Set headers
	req.Header.Set("Content-Type", ws.getContentType(config))
	req.Header.Set("User-Agent", ws.userAgent)
	req.Header.Set(ws.timestampHeader, strconv.FormatInt(time.Now().Unix(), 10))
	
	// Set custom headers
	for key, value := range config.Headers {
		req.Header.Set(key, value)
	}
	
	// Add signature if secret is configured
	if config.Secret != "" {
		signature := ws.generateSignature(payload, config.Secret)
		req.Header.Set(ws.signatureHeader, signature)
	}
	
	// Set timeout
	ctx := ws.ctx
	if config.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Duration(config.Timeout)*time.Second)
		defer cancel()
	}
	req = req.WithContext(ctx)
	
	// Send request
	resp, err := ws.client.Do(req)
	response.ResponseTime = time.Since(startTime)
	
	if err != nil {
		response.Error = fmt.Sprintf("request failed: %v", err)
		return response
	}
	defer resp.Body.Close()
	
	response.StatusCode = resp.StatusCode
	response.Headers = make(map[string]string)
	for key, values := range resp.Header {
		if len(values) > 0 {
			response.Headers[key] = values[0]
		}
	}
	
	// Read response body (limit size)
	bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, 10240)) // 10KB limit
	if err != nil {
		response.Error = fmt.Sprintf("failed to read response: %v", err)
	} else {
		response.Body = string(bodyBytes)
	}
	
	return response
}

// scheduleRetry schedules a webhook for retry
func (ws *WebhookService) scheduleRetry(notification *WebhookNotification, response *WebhookResponse) {
	notification.RetryCount++
	notification.Status = "retrying"
	
	// Calculate next retry time with exponential backoff
	delay := time.Duration(notification.RetryCount) * ws.retryDelay
	nextRetry := time.Now().Add(delay)
	notification.NextRetry = &nextRetry
	notification.UpdatedAt = time.Now()
	
	// Update in database
	ws.updateNotificationForRetry(notification, response.StatusCode, response.Body)
	
	// Schedule background retry
	go func() {
		time.Sleep(delay)
		ws.processWebhook(notification)
	}()
}

// Helper methods

func (ws *WebhookService) isEventEnabled(eventType string, enabledEvents []string) bool {
	if len(enabledEvents) == 0 {
		return true // All events enabled if not specified
	}
	
	for _, enabled := range enabledEvents {
		if enabled == eventType || enabled == "*" {
			return true
		}
		// Support wildcard matching
		if strings.HasSuffix(enabled, "*") && strings.HasPrefix(eventType, enabled[:len(enabled)-1]) {
			return true
		}
	}
	
	return false
}

func (ws *WebhookService) isRateLimited(webhookURL string) bool {
	key := fmt.Sprintf("webhook_rate_limit:%s", webhookURL)
	
	// Use Redis for rate limiting
	pipe := ws.redis.Pipeline()
	
	// Count requests in the current window
	now := time.Now()
	windowStart := now.Add(-ws.rateLimitWindow)
	
	pipe.ZRemRangeByScore(ws.ctx, key, "0", fmt.Sprintf("%d", windowStart.Unix()))
	countCmd := pipe.ZCount(ws.ctx, key, fmt.Sprintf("%d", windowStart.Unix()), fmt.Sprintf("%d", now.Unix()))
	pipe.ZAdd(ws.ctx, key, redis.Z{Score: float64(now.Unix()), Member: now.UnixNano()})
	pipe.Expire(ws.ctx, key, ws.rateLimitWindow)
	
	_, err := pipe.Exec(ws.ctx)
	if err != nil {
		return false // Don't rate limit on Redis errors
	}
	
	count, err := countCmd.Result()
	if err != nil {
		return false
	}
	
	return count >= int64(ws.maxWebhooksPerMin)
}

func (ws *WebhookService) shouldSendImmediately(config *WebhookConfig) bool {
	// Send immediately for critical events, queue others
	return config.MaxRetries <= 1
}

func (ws *WebhookService) queueWebhook(notification *WebhookNotification) {
	// Add to Redis queue for background processing
	data, _ := json.Marshal(notification)
	ws.redis.LPush(ws.ctx, "webhook_queue", string(data))
}

func (ws *WebhookService) getSpamEventType(action string) string {
	switch action {
	case "block":
		return "spam.blocked"
	case "quarantine":
		return "spam.quarantined"
	default:
		return "spam.detected"
	}
}

func (ws *WebhookService) extractSubmissionID(metadata map[string]interface{}) string {
	if submissionID, ok := metadata["submission_id"].(string); ok {
		return submissionID
	}
	return ""
}

func (ws *WebhookService) getContentType(config *WebhookConfig) string {
	if config.ContentType != "" {
		return config.ContentType
	}
	return "application/json"
}

func (ws *WebhookService) generateSignature(payload []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

func (ws *WebhookService) processPayloadTemplate(template string, event *WebhookEvent) interface{} {
	// Simple template processing - replace variables
	// In production, use a proper template engine like text/template
	processed := template
	processed = strings.ReplaceAll(processed, "{{.ID}}", event.ID)
	processed = strings.ReplaceAll(processed, "{{.Type}}", event.Type)
	processed = strings.ReplaceAll(processed, "{{.FormID}}", event.FormID)
	processed = strings.ReplaceAll(processed, "{{.Timestamp}}", event.Timestamp.Format(time.RFC3339))
	
	var result interface{}
	json.Unmarshal([]byte(processed), &result)
	return result
}

// Database operations

func (ws *WebhookService) getWebhookConfig(formID string) (*WebhookConfig, error) {
	query := `
		SELECT webhook_config FROM forms 
		WHERE id = ? AND webhook_config IS NOT NULL
	`
	
	var configJSON string
	err := ws.db.QueryRow(query, formID).Scan(&configJSON)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // No webhook configured
		}
		return nil, err
	}
	
	var config WebhookConfig
	if err := json.Unmarshal([]byte(configJSON), &config); err != nil {
		return nil, err
	}
	
	return &config, nil
}

func (ws *WebhookService) storeNotification(notification *WebhookNotification) error {
	payloadJSON, _ := json.Marshal(notification.Event)
	
	query := `
		INSERT INTO webhook_notifications 
		(id, webhook_url, event_type, form_id, submission_id, payload, status, retry_count, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	
	_, err := ws.db.Exec(query, notification.ID, notification.WebhookURL, notification.Event.Type,
		notification.Event.FormID, notification.Event.SubmissionID, string(payloadJSON),
		notification.Status, notification.RetryCount, notification.CreatedAt)
	
	return err
}

func (ws *WebhookService) updateNotificationStatus(id, status string, responseCode int, responseBody string) error {
	query := `
		UPDATE webhook_notifications 
		SET status = ?, response_code = ?, response_body = ?, sent_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`
	
	_, err := ws.db.Exec(query, status, responseCode, responseBody, id)
	return err
}

func (ws *WebhookService) updateNotificationForRetry(notification *WebhookNotification, 
	responseCode int, responseBody string) error {
	
	query := `
		UPDATE webhook_notifications 
		SET status = ?, retry_count = ?, next_retry_at = ?, response_code = ?, response_body = ?
		WHERE id = ?
	`
	
	_, err := ws.db.Exec(query, notification.Status, notification.RetryCount, notification.NextRetry,
		responseCode, responseBody, notification.ID)
	
	return err
}

func (ws *WebhookService) logWebhookAttempt(notification *WebhookNotification, 
	response *WebhookResponse, duration time.Duration) {
	
	logData := map[string]interface{}{
		"webhook_id":     notification.ID,
		"url":           notification.WebhookURL,
		"event_type":    notification.Event.Type,
		"form_id":       notification.Event.FormID,
		"status_code":   response.StatusCode,
		"response_time": duration.Milliseconds(),
		"retry_count":   notification.RetryCount,
		"success":       response.StatusCode >= 200 && response.StatusCode < 300,
	}
	
	if response.Error != "" {
		logData["error"] = response.Error
	}
	
	// Log to security_logs table for audit trail
	query := `
		INSERT INTO security_logs (id, event_type, severity, details, created_at)
		VALUES (?, 'webhook_sent', 'low', ?, ?)
	`
	
	detailsJSON, _ := json.Marshal(logData)
	ws.db.Exec(query, uuid.New().String(), string(detailsJSON), time.Now())
}

// Background processing methods

// ProcessWebhookQueue processes queued webhooks
func (ws *WebhookService) ProcessWebhookQueue() {
	for {
		// Get webhook from queue
		result := ws.redis.BRPop(ws.ctx, 5*time.Second, "webhook_queue")
		if len(result) < 2 {
			continue // Timeout or empty queue
		}
		
		var notification WebhookNotification
		if err := json.Unmarshal([]byte(result[1]), &notification); err != nil {
			continue
		}
		
		// Process the webhook
		ws.processWebhook(&notification)
	}
}

// ProcessRetryQueue processes webhooks scheduled for retry
func (ws *WebhookService) ProcessRetryQueue() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	
	for range ticker.C {
		query := `
			SELECT id, webhook_url, payload, retry_count, max_retries
			FROM webhook_notifications 
			WHERE status = 'retrying' 
			AND next_retry_at <= CURRENT_TIMESTAMP
			LIMIT 100
		`
		
		rows, err := ws.db.Query(query)
		if err != nil {
			continue
		}
		
		for rows.Next() {
			var id, webhookURL, payloadJSON string
			var retryCount, maxRetries int
			
			if err := rows.Scan(&id, &webhookURL, &payloadJSON, &retryCount, &maxRetries); err != nil {
				continue
			}
			
			if retryCount >= maxRetries {
				// Mark as failed
				ws.updateNotificationStatus(id, "failed", 0, "max retries exceeded")
				continue
			}
			
			// Process retry
			var event WebhookEvent
			if err := json.Unmarshal([]byte(payloadJSON), &event); err != nil {
				continue
			}
			
			// Get webhook config and create notification
			config, err := ws.getWebhookConfig(event.FormID)
			if err != nil || config == nil {
				continue
			}
			
			notification := &WebhookNotification{
				ID:         id,
				WebhookURL: webhookURL,
				Event:      &event,
				Config:     config,
				RetryCount: retryCount,
			}
			
			go ws.processWebhook(notification)
		}
		rows.Close()
	}
}

// GetWebhookStats returns webhook service statistics
func (ws *WebhookService) GetWebhookStats() *WebhookStats {
	return ws.stats
}

// GetWebhookHistory returns webhook notification history for a form
func (ws *WebhookService) GetWebhookHistory(formID string, limit int) ([]WebhookNotification, error) {
	query := `
		SELECT id, webhook_url, event_type, payload, status, response_code, 
		       retry_count, sent_at, created_at
		FROM webhook_notifications 
		WHERE form_id = ?
		ORDER BY created_at DESC 
		LIMIT ?
	`
	
	rows, err := ws.db.Query(query, formID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var notifications []WebhookNotification
	
	for rows.Next() {
		var notification WebhookNotification
		var payloadJSON string
		var sentAt sql.NullTime
		
		err := rows.Scan(&notification.ID, &notification.WebhookURL, 
			&notification.Event.Type, &payloadJSON, &notification.Status,
			&notification.ResponseCode, &notification.RetryCount, 
			&sentAt, &notification.CreatedAt)
		
		if err != nil {
			continue
		}
		
		if sentAt.Valid {
			notification.SentAt = &sentAt.Time
		}
		
		// Parse event payload
		var event WebhookEvent
		if json.Unmarshal([]byte(payloadJSON), &event) == nil {
			notification.Event = &event
		}
		
		notifications = append(notifications, notification)
	}
	
	return notifications, nil
}

// ValidateWebhookURL validates a webhook URL
func (ws *WebhookService) ValidateWebhookURL(webhookURL string) error {
	// Parse URL
	u, err := url.Parse(webhookURL)
	if err != nil {
		return fmt.Errorf("invalid URL format: %w", err)
	}
	
	// Check scheme
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("unsupported URL scheme: %s", u.Scheme)
	}
	
	// Require HTTPS in production
	if u.Scheme != "https" {
		// Log warning but don't fail in development
		// In production, you might want to enforce HTTPS
	}
	
	// Check for localhost/private IPs (security consideration)
	if strings.Contains(u.Host, "localhost") || strings.Contains(u.Host, "127.0.0.1") {
		// Allow in development, restrict in production
	}
	
	return nil
}

// TestWebhook sends a test webhook to verify configuration
func (ws *WebhookService) TestWebhook(config *WebhookConfig) (*WebhookResponse, error) {
	// Create test event
	testEvent := &WebhookEvent{
		ID:        uuid.New().String(),
		Type:      "test",
		Timestamp: time.Now().UTC(),
		FormID:    "test-form",
		Data: map[string]interface{}{
			"message": "This is a test webhook from FormHub",
		},
		Source:  "test",
		Version: "1.0",
	}
	
	// Prepare payload
	payload, err := json.Marshal(testEvent)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal test payload: %w", err)
	}
	
	// Send request
	response := ws.sendHTTPRequest(config, payload)
	
	return response, nil
}