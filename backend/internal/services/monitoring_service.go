package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"formhub/internal/models"
	"formhub/pkg/database"
	"log"
	"math"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type MonitoringService struct {
	db               *sqlx.DB
	redis            *database.RedisClient
	analyticsService *AnalyticsService
	realTimeService  *RealTimeService
}

type AlertCondition struct {
	Metric      string      `json:"metric"`
	Operator    string      `json:"operator"` // gt, lt, gte, lte, eq, ne
	Threshold   float64     `json:"threshold"`
	Period      string      `json:"period"` // 5m, 15m, 1h, 24h
	Aggregation string      `json:"aggregation"` // avg, sum, count, min, max
}

type NotificationChannel struct {
	Type   string                 `json:"type"` // email, webhook, slack, sms
	Config map[string]interface{} `json:"config"`
}

type AlertEvaluation struct {
	AlertID      uuid.UUID `json:"alert_id"`
	Triggered    bool      `json:"triggered"`
	CurrentValue float64   `json:"current_value"`
	Threshold    float64   `json:"threshold"`
	Message      string    `json:"message"`
	EvaluatedAt  time.Time `json:"evaluated_at"`
}

func NewMonitoringService(
	db *sqlx.DB,
	redis *database.RedisClient,
	analyticsService *AnalyticsService,
	realTimeService *RealTimeService,
) *MonitoringService {
	return &MonitoringService{
		db:               db,
		redis:            redis,
		analyticsService: analyticsService,
		realTimeService:  realTimeService,
	}
}

// CreateAlert creates a new monitoring alert
func (m *MonitoringService) CreateAlert(ctx context.Context, userID uuid.UUID, req *models.CreateMonitoringAlertRequest) (*models.MonitoringAlert, error) {
	alert := &models.MonitoringAlert{
		ID:                  uuid.New(),
		UserID:              userID,
		AlertName:           req.AlertName,
		AlertType:           req.AlertType,
		FormIDs:             req.FormIDs,
		Conditions:          req.Conditions,
		NotificationMethods: req.NotificationMethods,
		NotificationConfig:  req.NotificationConfig,
		CooldownMinutes:     req.CooldownMinutes,
		IsActive:            true,
		CreatedAt:           time.Now().UTC(),
		UpdatedAt:           time.Now().UTC(),
	}

	if alert.CooldownMinutes == 0 {
		alert.CooldownMinutes = 60 // Default 1 hour cooldown
	}

	query := `
		INSERT INTO monitoring_alerts (
			id, user_id, alert_name, alert_type, form_ids, conditions,
			notification_methods, notification_config, cooldown_minutes,
			is_active, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	formIDsJSON, _ := json.Marshal(alert.FormIDs)
	conditionsJSON, _ := json.Marshal(alert.Conditions)
	methodsJSON, _ := json.Marshal(alert.NotificationMethods)
	configJSON, _ := json.Marshal(alert.NotificationConfig)

	_, err := m.db.ExecContext(ctx, query,
		alert.ID, alert.UserID, alert.AlertName, alert.AlertType,
		string(formIDsJSON), string(conditionsJSON), string(methodsJSON),
		string(configJSON), alert.CooldownMinutes, alert.IsActive,
		alert.CreatedAt, alert.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create monitoring alert: %w", err)
	}

	log.Printf("Created monitoring alert: %s for user %s", alert.AlertName, userID)
	return alert, nil
}

// EvaluateAlerts evaluates all active alerts
func (m *MonitoringService) EvaluateAlerts(ctx context.Context) error {
	query := `
		SELECT id, user_id, alert_name, alert_type, form_ids, conditions,
		       notification_methods, notification_config, cooldown_minutes,
		       last_triggered_at, trigger_count, created_at, updated_at
		FROM monitoring_alerts
		WHERE is_active = TRUE
	`

	rows, err := m.db.QueryContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to get active alerts: %w", err)
	}
	defer rows.Close()

	var alerts []models.MonitoringAlert
	for rows.Next() {
		var alert models.MonitoringAlert
		var formIDsJSON, conditionsJSON, methodsJSON, configJSON string

		err := rows.Scan(
			&alert.ID, &alert.UserID, &alert.AlertName, &alert.AlertType,
			&formIDsJSON, &conditionsJSON, &methodsJSON, &configJSON,
			&alert.CooldownMinutes, &alert.LastTriggeredAt, &alert.TriggerCount,
			&alert.CreatedAt, &alert.UpdatedAt,
		)
		if err != nil {
			continue
		}

		// Parse JSON fields
		json.Unmarshal([]byte(formIDsJSON), &alert.FormIDs)
		json.Unmarshal([]byte(conditionsJSON), &alert.Conditions)
		json.Unmarshal([]byte(methodsJSON), &alert.NotificationMethods)
		json.Unmarshal([]byte(configJSON), &alert.NotificationConfig)

		alerts = append(alerts, alert)
	}

	// Evaluate each alert
	for _, alert := range alerts {
		if err := m.evaluateAlert(ctx, &alert); err != nil {
			log.Printf("Failed to evaluate alert %s: %v", alert.AlertName, err)
		}
	}

	return nil
}

// evaluateAlert evaluates a single alert
func (m *MonitoringService) evaluateAlert(ctx context.Context, alert *models.MonitoringAlert) error {
	// Check cooldown period
	if alert.LastTriggeredAt != nil {
		cooldownUntil := alert.LastTriggeredAt.Add(time.Duration(alert.CooldownMinutes) * time.Minute)
		if time.Now().UTC().Before(cooldownUntil) {
			return nil // Still in cooldown
		}
	}

	// Evaluate based on alert type
	evaluation, err := m.evaluateAlertConditions(ctx, alert)
	if err != nil {
		return fmt.Errorf("failed to evaluate conditions: %w", err)
	}

	if evaluation.Triggered {
		// Trigger alert
		err := m.triggerAlert(ctx, alert, evaluation)
		if err != nil {
			return fmt.Errorf("failed to trigger alert: %w", err)
		}

		// Update alert statistics
		now := time.Now().UTC()
		updateQuery := `
			UPDATE monitoring_alerts 
			SET last_triggered_at = ?, trigger_count = trigger_count + 1, updated_at = ?
			WHERE id = ?
		`
		m.db.ExecContext(ctx, updateQuery, now, now, alert.ID)

		// Record in alert history
		m.recordAlertTrigger(ctx, alert, evaluation)

		log.Printf("Alert triggered: %s (value: %.2f, threshold: %.2f)", 
			alert.AlertName, evaluation.CurrentValue, evaluation.Threshold)
	}

	return nil
}

// evaluateAlertConditions evaluates the conditions for an alert
func (m *MonitoringService) evaluateAlertConditions(ctx context.Context, alert *models.MonitoringAlert) (*AlertEvaluation, error) {
	evaluation := &AlertEvaluation{
		AlertID:     alert.ID,
		EvaluatedAt: time.Now().UTC(),
	}

	switch alert.AlertType {
	case models.AlertTypeHighSpamRate:
		return m.evaluateSpamRate(ctx, alert, evaluation)
	case models.AlertTypeLowConversionRate:
		return m.evaluateConversionRate(ctx, alert, evaluation)
	case models.AlertTypeHighAbandonmentRate:
		return m.evaluateAbandonmentRate(ctx, alert, evaluation)
	case models.AlertTypeUnusualTraffic:
		return m.evaluateTrafficAnomaly(ctx, alert, evaluation)
	case models.AlertTypeFormErrors:
		return m.evaluateFormErrors(ctx, alert, evaluation)
	case models.AlertTypeWebhookFailures:
		return m.evaluateWebhookFailures(ctx, alert, evaluation)
	case models.AlertTypeEmailDeliveryIssues:
		return m.evaluateEmailDeliveryIssues(ctx, alert, evaluation)
	case models.AlertTypeSuspiciousActivity:
		return m.evaluateSuspiciousActivity(ctx, alert, evaluation)
	case models.AlertTypeFormDowntime:
		return m.evaluateFormDowntime(ctx, alert, evaluation)
	case models.AlertTypeCustom:
		return m.evaluateCustomAlert(ctx, alert, evaluation)
	default:
		return evaluation, fmt.Errorf("unknown alert type: %s", alert.AlertType)
	}
}

// evaluateSpamRate evaluates spam rate alert
func (m *MonitoringService) evaluateSpamRate(ctx context.Context, alert *models.MonitoringAlert, eval *AlertEvaluation) (*AlertEvaluation, error) {
	threshold := 10.0 // Default 10% spam rate threshold
	period := time.Hour // Default 1 hour period

	// Extract threshold and period from conditions
	if conditions, ok := alert.Conditions["spam_rate_threshold"].(float64); ok {
		threshold = conditions
	}
	if periodStr, ok := alert.Conditions["period"].(string); ok {
		if p, err := time.ParseDuration(periodStr); err == nil {
			period = p
		}
	}

	// Calculate current spam rate
	startTime := time.Now().UTC().Add(-period)
	var totalSubmissions, spamSubmissions int

	if len(alert.FormIDs) > 0 {
		// Specific forms
		for _, formID := range alert.FormIDs {
			var formTotal, formSpam int
			m.db.GetContext(ctx, &formTotal, 
				"SELECT COUNT(*) FROM submissions WHERE form_id = ? AND created_at >= ?",
				formID, startTime)
			m.db.GetContext(ctx, &formSpam,
				"SELECT COUNT(*) FROM submissions WHERE form_id = ? AND is_spam = TRUE AND created_at >= ?",
				formID, startTime)
			
			totalSubmissions += formTotal
			spamSubmissions += formSpam
		}
	} else {
		// All user forms
		query := `
			SELECT 
				COUNT(*) as total,
				COUNT(CASE WHEN s.is_spam = TRUE THEN 1 END) as spam
			FROM submissions s
			INNER JOIN forms f ON s.form_id = f.id
			WHERE f.user_id = ? AND s.created_at >= ?
		`
		m.db.QueryRowContext(ctx, query, alert.UserID, startTime).Scan(&totalSubmissions, &spamSubmissions)
	}

	spamRate := 0.0
	if totalSubmissions > 0 {
		spamRate = float64(spamSubmissions) / float64(totalSubmissions) * 100
	}

	eval.CurrentValue = spamRate
	eval.Threshold = threshold
	eval.Triggered = spamRate > threshold
	eval.Message = fmt.Sprintf("Spam rate is %.2f%% (threshold: %.2f%%)", spamRate, threshold)

	return eval, nil
}

// evaluateConversionRate evaluates conversion rate alert
func (m *MonitoringService) evaluateConversionRate(ctx context.Context, alert *models.MonitoringAlert, eval *AlertEvaluation) (*AlertEvaluation, error) {
	threshold := 1.0 // Default 1% minimum conversion rate
	period := time.Hour

	if conditions, ok := alert.Conditions["conversion_rate_threshold"].(float64); ok {
		threshold = conditions
	}
	if periodStr, ok := alert.Conditions["period"].(string); ok {
		if p, err := time.ParseDuration(periodStr); err == nil {
			period = p
		}
	}

	startTime := time.Now().UTC().Add(-period)
	var totalViews, totalSubmissions int

	// Get views and submissions for the period
	if len(alert.FormIDs) > 0 {
		for _, formID := range alert.FormIDs {
			var views, submissions int
			m.db.GetContext(ctx, &views,
				"SELECT COUNT(*) FROM form_analytics_events WHERE form_id = ? AND event_type = 'form_view' AND created_at >= ?",
				formID, startTime)
			m.db.GetContext(ctx, &submissions,
				"SELECT COUNT(*) FROM submissions WHERE form_id = ? AND created_at >= ?",
				formID, startTime)

			totalViews += views
			totalSubmissions += submissions
		}
	} else {
		query := `
			SELECT 
				COUNT(CASE WHEN ae.event_type = 'form_view' THEN 1 END) as views,
				COUNT(CASE WHEN s.id IS NOT NULL THEN 1 END) as submissions
			FROM form_analytics_events ae
			LEFT JOIN submissions s ON ae.form_id = s.form_id AND s.created_at >= ?
			INNER JOIN forms f ON ae.form_id = f.id
			WHERE f.user_id = ? AND ae.created_at >= ?
		`
		m.db.QueryRowContext(ctx, query, startTime, alert.UserID, startTime).Scan(&totalViews, &totalSubmissions)
	}

	conversionRate := 0.0
	if totalViews > 0 {
		conversionRate = float64(totalSubmissions) / float64(totalViews) * 100
	}

	eval.CurrentValue = conversionRate
	eval.Threshold = threshold
	eval.Triggered = conversionRate < threshold && totalViews > 10 // Only trigger if there's meaningful traffic
	eval.Message = fmt.Sprintf("Conversion rate is %.2f%% (threshold: %.2f%%)", conversionRate, threshold)

	return eval, nil
}

// evaluateAbandonmentRate evaluates form abandonment rate
func (m *MonitoringService) evaluateAbandonmentRate(ctx context.Context, alert *models.MonitoringAlert, eval *AlertEvaluation) (*AlertEvaluation, error) {
	threshold := 80.0 // Default 80% abandonment rate threshold
	period := time.Hour

	if conditions, ok := alert.Conditions["abandonment_rate_threshold"].(float64); ok {
		threshold = conditions
	}

	startTime := time.Now().UTC().Add(-period)
	var totalStarts, totalCompletions int

	// This is a simplified calculation - in practice you'd track form start/completion events
	query := `
		SELECT 
			COUNT(CASE WHEN event_type = 'form_start' THEN 1 END) as starts,
			COUNT(CASE WHEN event_type = 'form_complete' THEN 1 END) as completions
		FROM form_analytics_events ae
		INNER JOIN forms f ON ae.form_id = f.id
		WHERE f.user_id = ? AND ae.created_at >= ?
	`

	if len(alert.FormIDs) > 0 {
		// Add form filter
		formPlaceholders := make([]string, len(alert.FormIDs))
		args := make([]interface{}, len(alert.FormIDs)+2)
		args[0] = alert.UserID
		args[1] = startTime

		for i, formID := range alert.FormIDs {
			formPlaceholders[i] = "?"
			args[i+2] = formID
		}

		query += fmt.Sprintf(" AND ae.form_id IN (%s)", strings.Join(formPlaceholders, ","))
		m.db.QueryRowContext(ctx, query, args...).Scan(&totalStarts, &totalCompletions)
	} else {
		m.db.QueryRowContext(ctx, query, alert.UserID, startTime).Scan(&totalStarts, &totalCompletions)
	}

	abandonmentRate := 0.0
	if totalStarts > 0 {
		abandonmentRate = (1.0 - float64(totalCompletions)/float64(totalStarts)) * 100
	}

	eval.CurrentValue = abandonmentRate
	eval.Threshold = threshold
	eval.Triggered = abandonmentRate > threshold && totalStarts > 10
	eval.Message = fmt.Sprintf("Abandonment rate is %.2f%% (threshold: %.2f%%)", abandonmentRate, threshold)

	return eval, nil
}

// evaluateTrafficAnomaly evaluates unusual traffic patterns
func (m *MonitoringService) evaluateTrafficAnomaly(ctx context.Context, alert *models.MonitoringAlert, eval *AlertEvaluation) (*AlertEvaluation, error) {
	// Compare current hour traffic to average of last 24 hours
	now := time.Now().UTC()
	currentHourStart := now.Truncate(time.Hour)
	last24HoursStart := now.Add(-24 * time.Hour)

	var currentHourTraffic, avgHourlyTraffic float64

	// Get current hour traffic
	query := `
		SELECT COUNT(*) 
		FROM form_analytics_events ae
		INNER JOIN forms f ON ae.form_id = f.id
		WHERE f.user_id = ? AND ae.event_type = 'form_view' AND ae.created_at >= ?
	`

	m.db.GetContext(ctx, &currentHourTraffic, query, alert.UserID, currentHourStart)

	// Get average hourly traffic over last 24 hours
	query = `
		SELECT COUNT(*) / 24.0
		FROM form_analytics_events ae
		INNER JOIN forms f ON ae.form_id = f.id
		WHERE f.user_id = ? AND ae.event_type = 'form_view' AND ae.created_at BETWEEN ? AND ?
	`

	m.db.GetContext(ctx, &avgHourlyTraffic, query, alert.UserID, last24HoursStart, currentHourStart)

	// Calculate percentage change
	percentageChange := 0.0
	if avgHourlyTraffic > 0 {
		percentageChange = ((currentHourTraffic - avgHourlyTraffic) / avgHourlyTraffic) * 100
	}

	threshold := 200.0 // 200% increase
	if conditions, ok := alert.Conditions["traffic_increase_threshold"].(float64); ok {
		threshold = conditions
	}

	eval.CurrentValue = percentageChange
	eval.Threshold = threshold
	eval.Triggered = math.Abs(percentageChange) > threshold
	eval.Message = fmt.Sprintf("Traffic change: %.1f%% (current: %.0f, avg: %.0f)", 
		percentageChange, currentHourTraffic, avgHourlyTraffic)

	return eval, nil
}

// Additional evaluation methods (simplified for brevity)
func (m *MonitoringService) evaluateFormErrors(ctx context.Context, alert *models.MonitoringAlert, eval *AlertEvaluation) (*AlertEvaluation, error) {
	// Implementation for form validation errors
	eval.Message = "Form errors monitoring not yet implemented"
	return eval, nil
}

func (m *MonitoringService) evaluateWebhookFailures(ctx context.Context, alert *models.MonitoringAlert, eval *AlertEvaluation) (*AlertEvaluation, error) {
	// Implementation for webhook delivery failures
	eval.Message = "Webhook failures monitoring not yet implemented"
	return eval, nil
}

func (m *MonitoringService) evaluateEmailDeliveryIssues(ctx context.Context, alert *models.MonitoringAlert, eval *AlertEvaluation) (*AlertEvaluation, error) {
	// Implementation for email delivery issues
	eval.Message = "Email delivery monitoring not yet implemented"
	return eval, nil
}

func (m *MonitoringService) evaluateSuspiciousActivity(ctx context.Context, alert *models.MonitoringAlert, eval *AlertEvaluation) (*AlertEvaluation, error) {
	// Implementation for suspicious activity detection
	eval.Message = "Suspicious activity monitoring not yet implemented"
	return eval, nil
}

func (m *MonitoringService) evaluateFormDowntime(ctx context.Context, alert *models.MonitoringAlert, eval *AlertEvaluation) (*AlertEvaluation, error) {
	// Implementation for form availability monitoring
	eval.Message = "Form downtime monitoring not yet implemented"
	return eval, nil
}

func (m *MonitoringService) evaluateCustomAlert(ctx context.Context, alert *models.MonitoringAlert, eval *AlertEvaluation) (*AlertEvaluation, error) {
	// Implementation for custom alert conditions
	eval.Message = "Custom alert monitoring not yet implemented"
	return eval, nil
}

// triggerAlert sends notifications for a triggered alert
func (m *MonitoringService) triggerAlert(ctx context.Context, alert *models.MonitoringAlert, evaluation *AlertEvaluation) error {
	for _, method := range alert.NotificationMethods {
		switch method {
		case "email":
			if err := m.sendEmailAlert(ctx, alert, evaluation); err != nil {
				log.Printf("Failed to send email alert: %v", err)
			}
		case "webhook":
			if err := m.sendWebhookAlert(ctx, alert, evaluation); err != nil {
				log.Printf("Failed to send webhook alert: %v", err)
			}
		case "slack":
			if err := m.sendSlackAlert(ctx, alert, evaluation); err != nil {
				log.Printf("Failed to send Slack alert: %v", err)
			}
		}
	}

	// Send real-time alert to WebSocket connections
	severity := m.determineSeverity(evaluation)
	m.realTimeService.BroadcastAlert(alert.UserID, string(alert.AlertType), evaluation.Message, severity)

	return nil
}

// sendEmailAlert sends an email alert
func (m *MonitoringService) sendEmailAlert(ctx context.Context, alert *models.MonitoringAlert, evaluation *AlertEvaluation) error {
	// Extract email configuration
	emails, ok := alert.NotificationConfig["emails"].([]interface{})
	if !ok {
		return fmt.Errorf("no email addresses configured")
	}

	subject := fmt.Sprintf("FormHub Alert: %s", alert.AlertName)
	body := fmt.Sprintf(`
Alert: %s
Type: %s
Message: %s
Current Value: %.2f
Threshold: %.2f
Triggered At: %s

Dashboard: https://formhub.io/dashboard
	`, alert.AlertName, alert.AlertType, evaluation.Message, 
		evaluation.CurrentValue, evaluation.Threshold, evaluation.EvaluatedAt.Format(time.RFC3339))

	// This would integrate with your email service
	log.Printf("Would send email alert to %v: %s", emails, subject)
	
	return nil
}

// sendWebhookAlert sends a webhook alert
func (m *MonitoringService) sendWebhookAlert(ctx context.Context, alert *models.MonitoringAlert, evaluation *AlertEvaluation) error {
	webhookURL, ok := alert.NotificationConfig["webhook_url"].(string)
	if !ok {
		return fmt.Errorf("no webhook URL configured")
	}

	payload := map[string]interface{}{
		"alert_id":      alert.ID,
		"alert_name":    alert.AlertName,
		"alert_type":    alert.AlertType,
		"message":       evaluation.Message,
		"current_value": evaluation.CurrentValue,
		"threshold":     evaluation.Threshold,
		"triggered_at":  evaluation.EvaluatedAt,
		"user_id":       alert.UserID,
	}

	jsonData, _ := json.Marshal(payload)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(webhookURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("webhook request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("webhook returned status: %d", resp.StatusCode)
	}

	return nil
}

// sendSlackAlert sends a Slack alert
func (m *MonitoringService) sendSlackAlert(ctx context.Context, alert *models.MonitoringAlert, evaluation *AlertEvaluation) error {
	webhookURL, ok := alert.NotificationConfig["slack_webhook"].(string)
	if !ok {
		return fmt.Errorf("no Slack webhook configured")
	}

	payload := map[string]interface{}{
		"text": fmt.Sprintf("🚨 *FormHub Alert: %s*", alert.AlertName),
		"attachments": []map[string]interface{}{
			{
				"color": m.getSlackColor(evaluation),
				"fields": []map[string]interface{}{
					{"title": "Alert Type", "value": string(alert.AlertType), "short": true},
					{"title": "Current Value", "value": fmt.Sprintf("%.2f", evaluation.CurrentValue), "short": true},
					{"title": "Threshold", "value": fmt.Sprintf("%.2f", evaluation.Threshold), "short": true},
					{"title": "Message", "value": evaluation.Message, "short": false},
				},
				"ts": evaluation.EvaluatedAt.Unix(),
			},
		},
	}

	jsonData, _ := json.Marshal(payload)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(webhookURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("Slack webhook failed: %w", err)
	}
	defer resp.Body.Close()

	return nil
}

// recordAlertTrigger records an alert trigger in the history
func (m *MonitoringService) recordAlertTrigger(ctx context.Context, alert *models.MonitoringAlert, evaluation *AlertEvaluation) {
	severity := m.determineSeverity(evaluation)

	query := `
		INSERT INTO alert_trigger_history (
			id, alert_id, trigger_reason, trigger_data, severity, created_at
		) VALUES (?, ?, ?, ?, ?, ?)
	`

	triggerData := map[string]interface{}{
		"current_value": evaluation.CurrentValue,
		"threshold":     evaluation.Threshold,
		"message":       evaluation.Message,
	}

	triggerDataJSON, _ := json.Marshal(triggerData)

	m.db.ExecContext(ctx, query,
		uuid.New(), alert.ID, evaluation.Message, string(triggerDataJSON),
		severity, evaluation.EvaluatedAt,
	)
}

// Helper methods

func (m *MonitoringService) determineSeverity(evaluation *AlertEvaluation) string {
	// Simple severity determination based on how far the value is from threshold
	ratio := evaluation.CurrentValue / evaluation.Threshold

	switch {
	case ratio >= 3.0:
		return "critical"
	case ratio >= 2.0:
		return "high"
	case ratio >= 1.5:
		return "medium"
	default:
		return "low"
	}
}

func (m *MonitoringService) getSlackColor(evaluation *AlertEvaluation) string {
	severity := m.determineSeverity(evaluation)
	
	switch severity {
	case "critical":
		return "danger"
	case "high":
		return "warning"
	case "medium":
		return "warning"
	default:
		return "good"
	}
}

// StartMonitoring starts the background monitoring service
func (m *MonitoringService) StartMonitoring(ctx context.Context) {
	// Evaluate alerts every 5 minutes
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	log.Println("Starting monitoring service...")

	for {
		select {
		case <-ctx.Done():
			log.Println("Stopping monitoring service...")
			return
		case <-ticker.C:
			if err := m.EvaluateAlerts(ctx); err != nil {
				log.Printf("Failed to evaluate alerts: %v", err)
			}
		}
	}
}

// GetAlertHistory returns alert trigger history
func (m *MonitoringService) GetAlertHistory(ctx context.Context, userID uuid.UUID, limit int) ([]models.AlertTriggerHistory, error) {
	query := `
		SELECT h.id, h.alert_id, h.trigger_reason, h.trigger_data, h.severity,
		       h.is_resolved, h.resolved_at, h.created_at, a.alert_name
		FROM alert_trigger_history h
		INNER JOIN monitoring_alerts a ON h.alert_id = a.id
		WHERE a.user_id = ?
		ORDER BY h.created_at DESC
		LIMIT ?
	`

	rows, err := m.db.QueryContext(ctx, query, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get alert history: %w", err)
	}
	defer rows.Close()

	var history []models.AlertTriggerHistory
	for rows.Next() {
		var item models.AlertTriggerHistory
		var triggerDataJSON string
		var alertName string

		err := rows.Scan(
			&item.ID, &item.AlertID, &item.TriggerReason, &triggerDataJSON,
			&item.Severity, &item.IsResolved, &item.ResolvedAt, &item.CreatedAt,
			&alertName,
		)
		if err != nil {
			continue
		}

		json.Unmarshal([]byte(triggerDataJSON), &item.TriggerData)
		history = append(history, item)
	}

	return history, nil
}

// GetSystemHealth returns overall system health metrics
func (m *MonitoringService) GetSystemHealth(ctx context.Context) (map[string]interface{}, error) {
	health := make(map[string]interface{})

	// Database connection health
	if err := m.db.PingContext(ctx); err != nil {
		health["database"] = map[string]interface{}{
			"status": "unhealthy",
			"error":  err.Error(),
		}
	} else {
		health["database"] = map[string]interface{}{
			"status": "healthy",
		}
	}

	// Redis health
	if err := m.redis.Client.Ping(ctx).Err(); err != nil {
		health["redis"] = map[string]interface{}{
			"status": "unhealthy",
			"error":  err.Error(),
		}
	} else {
		health["redis"] = map[string]interface{}{
			"status": "healthy",
		}
	}

	// API response time (average from last hour)
	var avgResponseTime float64
	query := `
		SELECT AVG(response_time_ms) 
		FROM api_performance_metrics 
		WHERE created_at >= DATE_SUB(NOW(), INTERVAL 1 HOUR)
	`
	m.db.GetContext(ctx, &avgResponseTime, query)
	
	health["api_performance"] = map[string]interface{}{
		"avg_response_time_ms": avgResponseTime,
		"status": func() string {
			if avgResponseTime > 5000 {
				return "unhealthy"
			} else if avgResponseTime > 2000 {
				return "degraded"
			}
			return "healthy"
		}(),
	}

	// Overall status
	overallStatus := "healthy"
	for _, component := range []string{"database", "redis", "api_performance"} {
		if comp, ok := health[component].(map[string]interface{}); ok {
			if status, ok := comp["status"].(string); ok && status != "healthy" {
				overallStatus = "unhealthy"
				break
			}
		}
	}

	health["overall_status"] = overallStatus
	health["timestamp"] = time.Now().UTC()

	return health, nil
}