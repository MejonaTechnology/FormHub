package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"formhub/internal/models"
	"log"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type AnalyticsService struct {
	db    *sqlx.DB
	redis *redis.Client
}

func NewAnalyticsService(db *sqlx.DB, redis *redis.Client) *AnalyticsService {
	return &AnalyticsService{
		db:    db,
		redis: redis,
	}
}

// RecordEvent records an analytics event
func (s *AnalyticsService) RecordEvent(ctx context.Context, event *models.FormAnalyticsEvent) error {
	// Generate UUID if not provided
	if event.ID == uuid.Nil {
		event.ID = uuid.New()
	}

	// Set timestamp if not provided
	if event.CreatedAt.IsZero() {
		event.CreatedAt = time.Now().UTC()
	}

	// Insert the event into database
	query := `
		INSERT INTO form_analytics_events (
			id, form_id, user_id, session_id, event_type, field_name, field_value_length,
			field_validation_error, page_url, referrer, utm_source, utm_medium, utm_campaign,
			utm_term, utm_content, device_type, browser_name, browser_version, os_name,
			os_version, screen_resolution, viewport_size, ip_address, country_code,
			country_name, region, city, latitude, longitude, timezone, user_agent,
			event_data, created_at
		) VALUES (
			?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?,
			?, ?, ?, ?, ?, ?, ?, ?, ?
		)
	`

	eventDataJSON, _ := json.Marshal(event.EventData)

	_, err := s.db.ExecContext(ctx, query,
		event.ID, event.FormID, event.UserID, event.SessionID, event.EventType,
		event.FieldName, event.FieldValueLength, event.FieldValidationError,
		event.PageURL, event.Referrer, event.UTMSource, event.UTMMedium,
		event.UTMCampaign, event.UTMTerm, event.UTMContent, event.DeviceType,
		event.BrowserName, event.BrowserVersion, event.OSName, event.OSVersion,
		event.ScreenResolution, event.ViewportSize, event.IPAddress,
		event.CountryCode, event.CountryName, event.Region, event.City,
		event.Latitude, event.Longitude, event.Timezone, event.UserAgent,
		string(eventDataJSON), event.CreatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to record analytics event: %w", err)
	}

	// Update real-time counters in Redis
	s.updateRealTimeCounters(ctx, event)

	// Trigger aggregation for certain events
	go s.triggerAggregation(event)

	return nil
}

// updateRealTimeCounters updates Redis counters for real-time analytics
func (s *AnalyticsService) updateRealTimeCounters(ctx context.Context, event *models.FormAnalyticsEvent) {
	pipe := s.redis.Client.Pipeline()

	date := event.CreatedAt.Format("2006-01-02")
	hour := event.CreatedAt.Format("2006-01-02:15")

	// Form-specific counters
	formKey := fmt.Sprintf("analytics:form:%s", event.FormID)
	pipe.HIncrBy(ctx, formKey+":daily:"+date, string(event.EventType), 1)
	pipe.HIncrBy(ctx, formKey+":hourly:"+hour, string(event.EventType), 1)
	pipe.Expire(ctx, formKey+":daily:"+date, 31*24*time.Hour)
	pipe.Expire(ctx, formKey+":hourly:"+hour, 25*time.Hour)

	// User-specific counters
	userKey := fmt.Sprintf("analytics:user:%s", event.UserID)
	pipe.HIncrBy(ctx, userKey+":daily:"+date, string(event.EventType), 1)
	pipe.Expire(ctx, userKey+":daily:"+date, 31*24*time.Hour)

	// Global counters
	globalKey := "analytics:global"
	pipe.HIncrBy(ctx, globalKey+":daily:"+date, string(event.EventType), 1)
	pipe.HIncrBy(ctx, globalKey+":hourly:"+hour, string(event.EventType), 1)
	pipe.Expire(ctx, globalKey+":daily:"+date, 31*24*time.Hour)
	pipe.Expire(ctx, globalKey+":hourly:"+hour, 25*time.Hour)

	// Device type counters
	if event.DeviceType != "" {
		pipe.HIncrBy(ctx, formKey+":devices:"+date, string(event.DeviceType), 1)
		pipe.Expire(ctx, formKey+":devices:"+date, 31*24*time.Hour)
	}

	// Geographic counters
	if event.CountryCode != nil && *event.CountryCode != "" {
		pipe.HIncrBy(ctx, formKey+":countries:"+date, *event.CountryCode, 1)
		pipe.Expire(ctx, formKey+":countries:"+date, 31*24*time.Hour)
	}

	_, err := pipe.Exec(ctx)
	if err != nil {
		log.Printf("Failed to update real-time counters: %v", err)
	}
}

// triggerAggregation triggers background aggregation for certain events
func (s *AnalyticsService) triggerAggregation(event *models.FormAnalyticsEvent) {
	// For high-impact events, trigger immediate aggregation
	switch event.EventType {
	case models.EventTypeFormSubmit, models.EventTypeFormComplete, models.EventTypeFormAbandon:
		s.aggregateFormFunnelData(event.FormID, event.CreatedAt)
	}
}

// GetFormAnalyticsDashboard returns comprehensive analytics for a form
func (s *AnalyticsService) GetFormAnalyticsDashboard(ctx context.Context, formID uuid.UUID, userID uuid.UUID, params *models.AnalyticsQueryParams) (*models.FormAnalyticsDashboard, error) {
	// Set default date range if not provided
	endDate := time.Now().UTC()
	startDate := endDate.AddDate(0, 0, -30) // Last 30 days

	if params.StartDate != nil {
		startDate = *params.StartDate
	}
	if params.EndDate != nil {
		endDate = *params.EndDate
	}

	dashboard := &models.FormAnalyticsDashboard{
		FormID: formID,
	}

	// Get form name
	err := s.db.GetContext(ctx, &dashboard.FormName, 
		"SELECT name FROM forms WHERE id = ? AND user_id = ?", formID, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get form name: %w", err)
	}

	// Get basic stats
	err = s.getBasicFormStats(ctx, dashboard, formID, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to get basic stats: %w", err)
	}

	// Get top countries
	dashboard.TopCountries, err = s.getTopCountries(ctx, formID, startDate, endDate, 10)
	if err != nil {
		log.Printf("Failed to get top countries: %v", err)
		dashboard.TopCountries = []models.CountryStats{}
	}

	// Get device breakdown
	dashboard.DeviceBreakdown, err = s.getDeviceBreakdown(ctx, formID, startDate, endDate)
	if err != nil {
		log.Printf("Failed to get device breakdown: %v", err)
		dashboard.DeviceBreakdown = []models.DeviceStats{}
	}

	// Get hourly stats
	dashboard.HourlyStats, err = s.getHourlyStats(ctx, formID, startDate, endDate)
	if err != nil {
		log.Printf("Failed to get hourly stats: %v", err)
		dashboard.HourlyStats = []models.HourlyStats{}
	}

	// Get field analytics
	dashboard.FieldAnalytics, err = s.getFieldAnalytics(ctx, formID, startDate, endDate)
	if err != nil {
		log.Printf("Failed to get field analytics: %v", err)
		dashboard.FieldAnalytics = []models.FieldAnalyticsStats{}
	}

	// Get recent submissions
	dashboard.RecentSubmissions, err = s.getRecentSubmissions(ctx, formID, 10)
	if err != nil {
		log.Printf("Failed to get recent submissions: %v", err)
		dashboard.RecentSubmissions = []models.RecentSubmissionStat{}
	}

	return dashboard, nil
}

// getBasicFormStats gets basic form statistics
func (s *AnalyticsService) getBasicFormStats(ctx context.Context, dashboard *models.FormAnalyticsDashboard, formID uuid.UUID, startDate, endDate time.Time) error {
	query := `
		SELECT 
			COUNT(CASE WHEN event_type = 'form_view' THEN 1 END) as total_views,
			COUNT(CASE WHEN event_type = 'form_submit' THEN 1 END) as total_submissions,
			AVG(CASE WHEN event_type = 'form_complete' AND JSON_EXTRACT(event_data, '$.completion_time') IS NOT NULL 
				THEN JSON_EXTRACT(event_data, '$.completion_time') END) as avg_completion_time
		FROM form_analytics_events 
		WHERE form_id = ? AND created_at BETWEEN ? AND ?
	`

	row := s.db.QueryRowContext(ctx, query, formID, startDate, endDate)
	
	var avgCompletionTime sql.NullFloat64
	err := row.Scan(&dashboard.TotalViews, &dashboard.TotalSubmissions, &avgCompletionTime)
	if err != nil {
		return err
	}

	if avgCompletionTime.Valid {
		dashboard.AverageCompletionTime = int(avgCompletionTime.Float64)
	}

	// Calculate conversion rate
	if dashboard.TotalViews > 0 {
		dashboard.ConversionRate = float64(dashboard.TotalSubmissions) / float64(dashboard.TotalViews) * 100
	}

	// Get spam rate from submissions
	var spamCount int
	err = s.db.GetContext(ctx, &spamCount, 
		"SELECT COUNT(*) FROM submissions WHERE form_id = ? AND is_spam = TRUE AND created_at BETWEEN ? AND ?", 
		formID, startDate, endDate)
	if err == nil && dashboard.TotalSubmissions > 0 {
		dashboard.SpamRate = float64(spamCount) / float64(dashboard.TotalSubmissions) * 100
	}

	return nil
}

// getTopCountries gets top countries by traffic
func (s *AnalyticsService) getTopCountries(ctx context.Context, formID uuid.UUID, startDate, endDate time.Time, limit int) ([]models.CountryStats, error) {
	query := `
		SELECT 
			country_code,
			country_name,
			COUNT(CASE WHEN event_type = 'form_view' THEN 1 END) as views,
			COUNT(CASE WHEN event_type = 'form_submit' THEN 1 END) as submissions,
			CASE 
				WHEN COUNT(CASE WHEN event_type = 'form_view' THEN 1 END) > 0 
				THEN (COUNT(CASE WHEN event_type = 'form_submit' THEN 1 END) * 100.0 / COUNT(CASE WHEN event_type = 'form_view' THEN 1 END))
				ELSE 0 
			END as conversion_rate
		FROM form_analytics_events 
		WHERE form_id = ? AND created_at BETWEEN ? AND ? AND country_code IS NOT NULL
		GROUP BY country_code, country_name 
		ORDER BY views DESC 
		LIMIT ?
	`

	rows, err := s.db.QueryContext(ctx, query, formID, startDate, endDate, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var countries []models.CountryStats
	for rows.Next() {
		var country models.CountryStats
		err := rows.Scan(&country.CountryCode, &country.CountryName, 
			&country.Views, &country.Submissions, &country.ConversionRate)
		if err != nil {
			continue
		}
		countries = append(countries, country)
	}

	return countries, nil
}

// getDeviceBreakdown gets device type breakdown
func (s *AnalyticsService) getDeviceBreakdown(ctx context.Context, formID uuid.UUID, startDate, endDate time.Time) ([]models.DeviceStats, error) {
	query := `
		SELECT 
			device_type,
			COUNT(CASE WHEN event_type = 'form_view' THEN 1 END) as views,
			COUNT(CASE WHEN event_type = 'form_submit' THEN 1 END) as submissions,
			CASE 
				WHEN COUNT(CASE WHEN event_type = 'form_view' THEN 1 END) > 0 
				THEN (COUNT(CASE WHEN event_type = 'form_submit' THEN 1 END) * 100.0 / COUNT(CASE WHEN event_type = 'form_view' THEN 1 END))
				ELSE 0 
			END as conversion_rate
		FROM form_analytics_events 
		WHERE form_id = ? AND created_at BETWEEN ? AND ?
		GROUP BY device_type 
		ORDER BY views DESC
	`

	rows, err := s.db.QueryContext(ctx, query, formID, startDate, endDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var devices []models.DeviceStats
	for rows.Next() {
		var device models.DeviceStats
		err := rows.Scan(&device.DeviceType, &device.Views, &device.Submissions, &device.ConversionRate)
		if err != nil {
			continue
		}
		devices = append(devices, device)
	}

	return devices, nil
}

// getHourlyStats gets hourly statistics
func (s *AnalyticsService) getHourlyStats(ctx context.Context, formID uuid.UUID, startDate, endDate time.Time) ([]models.HourlyStats, error) {
	query := `
		SELECT 
			HOUR(created_at) as hour,
			COUNT(CASE WHEN event_type = 'form_view' THEN 1 END) as views,
			COUNT(CASE WHEN event_type = 'form_submit' THEN 1 END) as submissions,
			CASE 
				WHEN COUNT(CASE WHEN event_type = 'form_view' THEN 1 END) > 0 
				THEN (COUNT(CASE WHEN event_type = 'form_submit' THEN 1 END) * 100.0 / COUNT(CASE WHEN event_type = 'form_view' THEN 1 END))
				ELSE 0 
			END as conversion_rate
		FROM form_analytics_events 
		WHERE form_id = ? AND created_at BETWEEN ? AND ?
		GROUP BY HOUR(created_at) 
		ORDER BY hour
	`

	rows, err := s.db.QueryContext(ctx, query, formID, startDate, endDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var hourlyStats []models.HourlyStats
	for rows.Next() {
		var stats models.HourlyStats
		err := rows.Scan(&stats.Hour, &stats.Views, &stats.Submissions, &stats.ConversionRate)
		if err != nil {
			continue
		}
		hourlyStats = append(hourlyStats, stats)
	}

	return hourlyStats, nil
}

// getFieldAnalytics gets field-level analytics
func (s *AnalyticsService) getFieldAnalytics(ctx context.Context, formID uuid.UUID, startDate, endDate time.Time) ([]models.FieldAnalyticsStats, error) {
	query := `
		SELECT 
			field_name,
			COUNT(*) as interactions,
			COUNT(CASE WHEN event_type = 'field_focus' THEN 1 END) as focus_events,
			COUNT(CASE WHEN event_type = 'validation_error' THEN 1 END) as validation_errors,
			AVG(CASE WHEN event_type = 'field_blur' AND JSON_EXTRACT(event_data, '$.time_spent') IS NOT NULL 
				THEN JSON_EXTRACT(event_data, '$.time_spent') END) as avg_time_to_fill,
			CASE 
				WHEN COUNT(CASE WHEN event_type = 'field_focus' THEN 1 END) > 0 
				THEN (COUNT(CASE WHEN event_type = 'validation_error' THEN 1 END) * 100.0 / COUNT(CASE WHEN event_type = 'field_focus' THEN 1 END))
				ELSE 0 
			END as error_rate
		FROM form_analytics_events 
		WHERE form_id = ? AND created_at BETWEEN ? AND ? AND field_name IS NOT NULL
		GROUP BY field_name 
		ORDER BY interactions DESC
	`

	rows, err := s.db.QueryContext(ctx, query, formID, startDate, endDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var fieldStats []models.FieldAnalyticsStats
	for rows.Next() {
		var stats models.FieldAnalyticsStats
		var avgTimeToFill sql.NullFloat64
		err := rows.Scan(&stats.FieldName, &stats.Interactions, &stats.FocusEvents, 
			&stats.ValidationErrors, &avgTimeToFill, &stats.ErrorRate)
		if err != nil {
			continue
		}
		
		if avgTimeToFill.Valid {
			stats.AvgTimeToFill = int(avgTimeToFill.Float64)
		}

		fieldStats = append(fieldStats, stats)
	}

	return fieldStats, nil
}

// getRecentSubmissions gets recent submissions with tracking info
func (s *AnalyticsService) getRecentSubmissions(ctx context.Context, formID uuid.UUID, limit int) ([]models.RecentSubmissionStat, error) {
	query := `
		SELECT 
			s.id,
			COALESCE(sl.tracking_id, ''),
			COALESCE(sl.status, 'received'),
			ae.country_name,
			ae.device_type,
			s.created_at
		FROM submissions s
		LEFT JOIN submission_lifecycle sl ON s.id = sl.submission_id
		LEFT JOIN (
			SELECT DISTINCT form_id, session_id, country_name, device_type,
			ROW_NUMBER() OVER (PARTITION BY session_id ORDER BY created_at DESC) as rn
			FROM form_analytics_events 
			WHERE event_type = 'form_submit'
		) ae ON s.form_id = ae.form_id AND ae.rn = 1
		WHERE s.form_id = ? 
		ORDER BY s.created_at DESC 
		LIMIT ?
	`

	rows, err := s.db.QueryContext(ctx, query, formID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var submissions []models.RecentSubmissionStat
	for rows.Next() {
		var submission models.RecentSubmissionStat
		err := rows.Scan(&submission.SubmissionID, &submission.TrackingID, 
			&submission.Status, &submission.CountryName, &submission.DeviceType, 
			&submission.CreatedAt)
		if err != nil {
			continue
		}
		submissions = append(submissions, submission)
	}

	return submissions, nil
}

// GetRealTimeStats returns real-time analytics
func (s *AnalyticsService) GetRealTimeStats(ctx context.Context, userID uuid.UUID) (*models.RealTimeStats, error) {
	stats := &models.RealTimeStats{}

	// Get active sessions from Redis
	activeSessionsKey := fmt.Sprintf("analytics:active_sessions:%s", userID)
	activeSessions, _ := s.redis.Client.SCard(ctx, activeSessionsKey).Result()
	stats.ActiveSessions = int(activeSessions)

	// Get submissions from the last hour and 24 hours
	now := time.Now().UTC()
	oneHourAgo := now.Add(-1 * time.Hour)
	twentyFourHoursAgo := now.Add(-24 * time.Hour)

	// Get submissions last hour
	err := s.db.GetContext(ctx, &stats.SubmissionsLastHour, `
		SELECT COUNT(*) FROM submissions s 
		INNER JOIN forms f ON s.form_id = f.id 
		WHERE f.user_id = ? AND s.created_at >= ?
	`, userID, oneHourAgo)
	if err != nil {
		log.Printf("Failed to get submissions last hour: %v", err)
	}

	// Get submissions last 24 hours
	err = s.db.GetContext(ctx, &stats.SubmissionsLast24h, `
		SELECT COUNT(*) FROM submissions s 
		INNER JOIN forms f ON s.form_id = f.id 
		WHERE f.user_id = ? AND s.created_at >= ?
	`, userID, twentyFourHoursAgo)
	if err != nil {
		log.Printf("Failed to get submissions last 24h: %v", err)
	}

	// Get spam blocked last hour
	err = s.db.GetContext(ctx, &stats.SpamBlockedLastHour, `
		SELECT COUNT(*) FROM submissions s 
		INNER JOIN forms f ON s.form_id = f.id 
		WHERE f.user_id = ? AND s.is_spam = TRUE AND s.created_at >= ?
	`, userID, oneHourAgo)
	if err != nil {
		log.Printf("Failed to get spam blocked: %v", err)
	}

	// Get top forms last hour
	stats.TopFormsLastHour, err = s.getTopFormsLastHour(ctx, userID)
	if err != nil {
		log.Printf("Failed to get top forms: %v", err)
		stats.TopFormsLastHour = []models.FormActivityStat{}
	}

	// Get live submissions (last 10)
	stats.LiveSubmissions, err = s.getLiveSubmissions(ctx, userID, 10)
	if err != nil {
		log.Printf("Failed to get live submissions: %v", err)
		stats.LiveSubmissions = []models.LiveSubmission{}
	}

	// Get system health stats
	stats.SystemHealth = s.getSystemHealthStats(ctx)

	return stats, nil
}

// getTopFormsLastHour gets most active forms in the last hour
func (s *AnalyticsService) getTopFormsLastHour(ctx context.Context, userID uuid.UUID) ([]models.FormActivityStat, error) {
	query := `
		SELECT 
			f.id,
			f.name,
			COUNT(CASE WHEN s.created_at >= ? THEN 1 END) as submissions,
			COUNT(CASE WHEN ae.event_type = 'form_view' AND ae.created_at >= ? THEN 1 END) as views
		FROM forms f
		LEFT JOIN submissions s ON f.id = s.form_id
		LEFT JOIN form_analytics_events ae ON f.id = ae.form_id
		WHERE f.user_id = ?
		GROUP BY f.id, f.name
		HAVING submissions > 0 OR views > 0
		ORDER BY submissions DESC, views DESC
		LIMIT 5
	`

	oneHourAgo := time.Now().UTC().Add(-1 * time.Hour)
	rows, err := s.db.QueryContext(ctx, query, oneHourAgo, oneHourAgo, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var forms []models.FormActivityStat
	for rows.Next() {
		var form models.FormActivityStat
		err := rows.Scan(&form.FormID, &form.FormName, &form.Submissions, &form.Views)
		if err != nil {
			continue
		}
		forms = append(forms, form)
	}

	return forms, nil
}

// getLiveSubmissions gets recent live submissions
func (s *AnalyticsService) getLiveSubmissions(ctx context.Context, userID uuid.UUID, limit int) ([]models.LiveSubmission, error) {
	query := `
		SELECT 
			s.id,
			f.name,
			ae.country_name,
			s.created_at,
			s.is_spam
		FROM submissions s
		INNER JOIN forms f ON s.form_id = f.id
		LEFT JOIN (
			SELECT DISTINCT form_id, session_id, country_name,
			ROW_NUMBER() OVER (PARTITION BY session_id ORDER BY created_at DESC) as rn
			FROM form_analytics_events 
			WHERE event_type = 'form_submit'
		) ae ON s.form_id = ae.form_id AND ae.rn = 1
		WHERE f.user_id = ?
		ORDER BY s.created_at DESC
		LIMIT ?
	`

	rows, err := s.db.QueryContext(ctx, query, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var submissions []models.LiveSubmission
	for rows.Next() {
		var submission models.LiveSubmission
		err := rows.Scan(&submission.SubmissionID, &submission.FormName, 
			&submission.CountryName, &submission.CreatedAt, &submission.IsSpam)
		if err != nil {
			continue
		}
		submissions = append(submissions, submission)
	}

	return submissions, nil
}

// getSystemHealthStats gets system health statistics
func (s *AnalyticsService) getSystemHealthStats(ctx context.Context) models.SystemHealthStats {
	stats := models.SystemHealthStats{}

	// Get average API response time from the last hour
	oneHourAgo := time.Now().UTC().Add(-1 * time.Hour)
	var avgResponseTime sql.NullFloat64
	err := s.db.Get(&avgResponseTime, 
		"SELECT AVG(response_time_ms) FROM api_performance_metrics WHERE created_at >= ?", 
		oneHourAgo)
	if err == nil && avgResponseTime.Valid {
		stats.APIResponseTime = avgResponseTime.Float64
	}

	// Get email delivery rate
	var totalEmails, deliveredEmails int
	s.db.Get(&totalEmails, 
		"SELECT COUNT(*) FROM email_queue WHERE created_at >= ?", oneHourAgo)
	s.db.Get(&deliveredEmails, 
		"SELECT COUNT(*) FROM email_queue WHERE status = 'sent' AND created_at >= ?", oneHourAgo)
	
	if totalEmails > 0 {
		stats.EmailDeliveryRate = float64(deliveredEmails) / float64(totalEmails) * 100
	}

	// Get webhook success rate
	var totalWebhooks, successfulWebhooks int
	s.db.Get(&totalWebhooks, 
		"SELECT COUNT(*) FROM submissions WHERE webhook_sent = TRUE AND created_at >= ?", oneHourAgo)
	// This would need to be implemented based on webhook tracking
	stats.WebhookSuccessRate = 95.0 // Default value

	// Get spam detection rate
	var totalSubmissions, spamSubmissions int
	s.db.Get(&totalSubmissions, 
		"SELECT COUNT(*) FROM submissions WHERE created_at >= ?", oneHourAgo)
	s.db.Get(&spamSubmissions, 
		"SELECT COUNT(*) FROM submissions WHERE is_spam = TRUE AND created_at >= ?", oneHourAgo)
	
	if totalSubmissions > 0 {
		stats.SpamDetectionRate = float64(spamSubmissions) / float64(totalSubmissions) * 100
	}

	// Get database and Redis connection stats (these would be implementation-specific)
	stats.DatabaseConnections = 10  // Would come from database pool stats
	stats.RedisConnections = 5      // Would come from Redis pool stats

	return stats
}

// aggregateFormFunnelData aggregates funnel data for a form
func (s *AnalyticsService) aggregateFormFunnelData(formID uuid.UUID, eventTime time.Time) {
	ctx := context.Background()
	date := eventTime.Format("2006-01-02")

	// This would typically be run as a background job
	query := `
		INSERT INTO form_conversion_funnels (
			id, form_id, user_id, date, total_views, total_starts, total_submits, 
			total_completes, total_abandons, conversion_rate, completion_rate, 
			created_at, updated_at
		) 
		SELECT 
			UUID(),
			?,
			(SELECT user_id FROM forms WHERE id = ?),
			DATE(?),
			COUNT(CASE WHEN event_type = 'form_view' THEN 1 END),
			COUNT(CASE WHEN event_type = 'form_start' THEN 1 END),
			COUNT(CASE WHEN event_type = 'form_submit' THEN 1 END),
			COUNT(CASE WHEN event_type = 'form_complete' THEN 1 END),
			COUNT(CASE WHEN event_type = 'form_abandon' THEN 1 END),
			CASE 
				WHEN COUNT(CASE WHEN event_type = 'form_view' THEN 1 END) > 0 
				THEN (COUNT(CASE WHEN event_type = 'form_submit' THEN 1 END) * 100.0 / COUNT(CASE WHEN event_type = 'form_view' THEN 1 END))
				ELSE 0 
			END,
			CASE 
				WHEN COUNT(CASE WHEN event_type = 'form_submit' THEN 1 END) > 0 
				THEN (COUNT(CASE WHEN event_type = 'form_complete' THEN 1 END) * 100.0 / COUNT(CASE WHEN event_type = 'form_submit' THEN 1 END))
				ELSE 0 
			END,
			NOW(),
			NOW()
		FROM form_analytics_events 
		WHERE form_id = ? AND DATE(created_at) = ?
		ON DUPLICATE KEY UPDATE
			total_views = VALUES(total_views),
			total_starts = VALUES(total_starts),
			total_submits = VALUES(total_submits),
			total_completes = VALUES(total_completes),
			total_abandons = VALUES(total_abandons),
			conversion_rate = VALUES(conversion_rate),
			completion_rate = VALUES(completion_rate),
			updated_at = NOW()
	`

	_, err := s.db.ExecContext(ctx, query, formID, formID, eventTime, formID, date)
	if err != nil {
		log.Printf("Failed to aggregate funnel data: %v", err)
	}
}

// RecordAPIPerformance records API performance metrics
func (s *AnalyticsService) RecordAPIPerformance(ctx context.Context, metrics *models.APIPerformanceMetrics) error {
	if metrics.ID == uuid.Nil {
		metrics.ID = uuid.New()
	}

	if metrics.CreatedAt.IsZero() {
		metrics.CreatedAt = time.Now().UTC()
	}

	query := `
		INSERT INTO api_performance_metrics (
			id, endpoint_path, http_method, response_time_ms, status_code,
			user_id, form_id, ip_address, user_agent, request_size_bytes,
			response_size_bytes, error_message, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := s.db.ExecContext(ctx, query,
		metrics.ID, metrics.EndpointPath, metrics.HTTPMethod, metrics.ResponseTimeMs,
		metrics.StatusCode, metrics.UserID, metrics.FormID, metrics.IPAddress,
		metrics.UserAgent, metrics.RequestSizeBytes, metrics.ResponseSizeBytes,
		metrics.ErrorMessage, metrics.CreatedAt,
	)

	return err
}

// TrackSession tracks or updates a user session
func (s *AnalyticsService) TrackSession(ctx context.Context, session *models.UserSession) error {
	// Try to update existing session first
	query := `
		UPDATE user_sessions 
		SET last_activity_at = ?, total_session_time = ?, 
		    total_forms_viewed = ?, total_forms_started = ?, total_forms_submitted = ?
		WHERE session_id = ?
	`

	result, err := s.db.ExecContext(ctx, query, 
		session.LastActivityAt, session.TotalSessionTime,
		session.TotalFormsViewed, session.TotalFormsStarted, session.TotalFormsSubmitted,
		session.SessionID)

	if err != nil {
		return err
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		// Insert new session
		if session.ID == uuid.Nil {
			session.ID = uuid.New()
		}

		insertQuery := `
			INSERT INTO user_sessions (
				id, session_id, user_id, ip_address, user_agent, device_type,
				browser_name, browser_version, os_name, os_version, country_code,
				country_name, region, city, timezone, referrer, landing_page,
				utm_source, utm_medium, utm_campaign, utm_term, utm_content,
				total_forms_viewed, total_forms_started, total_forms_submitted,
				total_session_time, is_bot, bot_detection_score, started_at,
				last_activity_at
			) VALUES (
				?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?,
				?, ?, ?, ?, ?, ?, ?, ?
			)
		`

		_, err = s.db.ExecContext(ctx, insertQuery,
			session.ID, session.SessionID, session.UserID, session.IPAddress,
			session.UserAgent, session.DeviceType, session.BrowserName, session.BrowserVersion,
			session.OSName, session.OSVersion, session.CountryCode, session.CountryName,
			session.Region, session.City, session.Timezone, session.Referrer, session.LandingPage,
			session.UTMSource, session.UTMMedium, session.UTMCampaign, session.UTMTerm,
			session.UTMContent, session.TotalFormsViewed, session.TotalFormsStarted,
			session.TotalFormsSubmitted, session.TotalSessionTime, session.IsBot,
			session.BotDetectionScore, session.StartedAt, session.LastActivityAt,
		)
	}

	// Update active sessions in Redis
	if err == nil {
		activeSessionsKey := "analytics:active_sessions"
		if session.UserID != nil {
			activeSessionsKey = fmt.Sprintf("analytics:active_sessions:%s", *session.UserID)
		}
		s.redis.Client.SAdd(ctx, activeSessionsKey, session.SessionID)
		s.redis.Client.Expire(ctx, activeSessionsKey, 2*time.Hour)
	}

	return err
}

// GenerateTrackingID generates a unique tracking ID for submissions
func (s *AnalyticsService) GenerateTrackingID() string {
	// Generate a random tracking ID (format: FH-YYYYMMDD-XXXXXX)
	now := time.Now()
	dateStr := now.Format("20060102")
	randomStr := s.generateRandomString(6)
	return fmt.Sprintf("FH-%s-%s", dateStr, randomStr)
}

// generateRandomString generates a random alphanumeric string
func (s *AnalyticsService) generateRandomString(length int) string {
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}

// GetFormConversionFunnel gets conversion funnel data for a form
func (s *AnalyticsService) GetFormConversionFunnel(ctx context.Context, formID uuid.UUID, userID uuid.UUID, startDate, endDate time.Time) ([]models.FormConversionFunnel, error) {
	query := `
		SELECT id, form_id, user_id, date, total_views, total_starts, total_submits,
		       total_completes, total_abandons, abandonment_points, conversion_rate,
		       completion_rate, average_time_to_submit, average_time_to_abandon,
		       created_at, updated_at
		FROM form_conversion_funnels
		WHERE form_id = ? AND user_id = ? AND date BETWEEN ? AND ?
		ORDER BY date
	`

	rows, err := s.db.QueryContext(ctx, query, formID, userID, startDate, endDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var funnels []models.FormConversionFunnel
	for rows.Next() {
		var funnel models.FormConversionFunnel
		var abandonmentPointsJSON sql.NullString

		err := rows.Scan(
			&funnel.ID, &funnel.FormID, &funnel.UserID, &funnel.Date,
			&funnel.TotalViews, &funnel.TotalStarts, &funnel.TotalSubmits,
			&funnel.TotalCompletes, &funnel.TotalAbandons, &abandonmentPointsJSON,
			&funnel.ConversionRate, &funnel.CompletionRate, &funnel.AvgTimeToSubmit,
			&funnel.AvgTimeToAbandon, &funnel.CreatedAt, &funnel.UpdatedAt,
		)
		if err != nil {
			continue
		}

		// Parse abandonment points JSON
		if abandonmentPointsJSON.Valid {
			json.Unmarshal([]byte(abandonmentPointsJSON.String), &funnel.AbandonmentPoints)
		}

		funnels = append(funnels, funnel)
	}

	return funnels, nil
}

// ExportAnalyticsData exports analytics data in specified format
func (s *AnalyticsService) ExportAnalyticsData(ctx context.Context, formID uuid.UUID, userID uuid.UUID, params *models.AnalyticsQueryParams) ([]byte, string, error) {
	// This would implement CSV, JSON, or other export formats
	// For now, returning JSON format

	dashboard, err := s.GetFormAnalyticsDashboard(ctx, formID, userID, params)
	if err != nil {
		return nil, "", err
	}

	jsonData, err := json.MarshalIndent(dashboard, "", "  ")
	if err != nil {
		return nil, "", err
	}

	return jsonData, "application/json", nil
}