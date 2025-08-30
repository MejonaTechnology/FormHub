package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// WebhookAnalytics provides comprehensive analytics for webhook operations
func (wa *WebhookAnalytics) RecordWebhookJob(job *WebhookJob) {
	wa.mu.Lock()
	defer wa.mu.Unlock()
	
	// Record job creation in Redis for real-time stats
	key := fmt.Sprintf("webhook_jobs:%s", job.FormID)
	pipe := wa.redis.Pipeline()
	
	// Increment job counter
	pipe.HIncrBy(context.Background(), key, "total_jobs", 1)
	pipe.HIncrBy(context.Background(), key, "pending_jobs", 1)
	
	// Store job timestamp for rate calculation
	pipe.ZAdd(context.Background(), key+":timestamps", redis.Z{
		Score:  float64(time.Now().Unix()),
		Member: job.ID,
	})
	
	// Set expiration for data retention
	pipe.Expire(context.Background(), key, 24*time.Hour)
	pipe.Expire(context.Background(), key+":timestamps", 24*time.Hour)
	
	if _, err := pipe.Exec(context.Background()); err != nil {
		log.Printf("Failed to record webhook job analytics: %v", err)
	}
}

// RecordWebhookResult records the result of a webhook delivery
func (wa *WebhookAnalytics) RecordWebhookResult(formID, endpointID string, result *WebhookResult) {
	wa.mu.Lock()
	defer wa.mu.Unlock()
	
	ctx := context.Background()
	
	// Record in Redis for real-time analytics
	formKey := fmt.Sprintf("webhook_analytics:%s", formID)
	endpointKey := fmt.Sprintf("webhook_endpoint_analytics:%s:%s", formID, endpointID)
	
	pipe := wa.redis.Pipeline()
	
	// Form-level metrics
	pipe.HIncrBy(ctx, formKey, "total_requests", 1)
	pipe.HSet(ctx, formKey, "last_request", time.Now().Unix())
	
	// Endpoint-level metrics
	pipe.HIncrBy(ctx, endpointKey, "total_requests", 1)
	pipe.HSet(ctx, endpointKey, "last_request", time.Now().Unix())
	
	if result.Success {
		pipe.HIncrBy(ctx, formKey, "successful_requests", 1)
		pipe.HIncrBy(ctx, endpointKey, "successful_requests", 1)
		pipe.HSet(ctx, endpointKey, "last_success", time.Now().Unix())
	} else {
		pipe.HIncrBy(ctx, formKey, "failed_requests", 1)
		pipe.HIncrBy(ctx, endpointKey, "failed_requests", 1)
		pipe.HSet(ctx, endpointKey, "last_failure", time.Now().Unix())
		
		// Track error types
		errorKey := fmt.Sprintf("webhook_errors:%s", formID)
		pipe.HIncrBy(ctx, errorKey, wa.categorizeError(result.Error), 1)
	}
	
	// Response time tracking
	responseTimeMs := result.ResponseTime.Milliseconds()
	pipe.LPush(ctx, fmt.Sprintf("webhook_response_times:%s", endpointID), responseTimeMs)
	pipe.LTrim(ctx, fmt.Sprintf("webhook_response_times:%s", endpointID), 0, 999) // Keep last 1000
	
	// Status code tracking
	if result.StatusCode > 0 {
		pipe.HIncrBy(ctx, fmt.Sprintf("webhook_status_codes:%s", formID), 
			strconv.Itoa(result.StatusCode), 1)
	}
	
	// Set expiration
	pipe.Expire(ctx, formKey, 30*24*time.Hour)       // 30 days
	pipe.Expire(ctx, endpointKey, 30*24*time.Hour)   // 30 days
	
	if _, err := pipe.Exec(ctx); err != nil {
		log.Printf("Failed to record webhook result analytics: %v", err)
	}
	
	// Store detailed record in database for long-term analytics
	go wa.storeWebhookRecord(formID, endpointID, result)
}

// RecordRateLimit records rate limiting events
func (wa *WebhookAnalytics) RecordRateLimit(formID, endpointID string) {
	ctx := context.Background()
	key := fmt.Sprintf("webhook_rate_limits:%s", formID)
	
	wa.redis.HIncrBy(ctx, key, endpointID, 1)
	wa.redis.Expire(ctx, key, 24*time.Hour)
}

// GetAnalytics returns comprehensive analytics data
func (wa *WebhookAnalytics) GetAnalytics(formID string, timeRange TimeRange) (*WebhookAnalyticsData, error) {
	ctx := context.Background()
	
	// Get real-time data from Redis
	realtimeData, err := wa.getRealtimeAnalytics(ctx, formID)
	if err != nil {
		log.Printf("Failed to get realtime analytics: %v", err)
		realtimeData = &RealtimeAnalytics{}
	}
	
	// Get historical data from database
	historicalData, err := wa.getHistoricalAnalytics(formID, timeRange)
	if err != nil {
		log.Printf("Failed to get historical analytics: %v", err)
		historicalData = &HistoricalAnalytics{}
	}
	
	// Get endpoint analytics
	endpointStats, err := wa.getEndpointAnalytics(ctx, formID)
	if err != nil {
		log.Printf("Failed to get endpoint analytics: %v", err)
	}
	
	// Calculate success rate
	successRate := 0.0
	if historicalData.TotalWebhooks > 0 {
		successRate = float64(historicalData.SuccessfulSent) / float64(historicalData.TotalWebhooks) * 100
	}
	
	// Get performance metrics
	performanceMetrics, err := wa.getPerformanceMetrics(ctx, formID)
	if err != nil {
		log.Printf("Failed to get performance metrics: %v", err)
	}
	
	return &WebhookAnalyticsData{
		FormID:             formID,
		TimeRange:          timeRange,
		TotalWebhooks:      historicalData.TotalWebhooks,
		SuccessfulSent:     historicalData.SuccessfulSent,
		Failed:             historicalData.Failed,
		SuccessRate:        successRate,
		AvgResponseTime:    performanceMetrics.AvgResponseTime,
		EndpointStats:      endpointStats,
		HourlyStats:        historicalData.HourlyStats,
		DailyStats:         historicalData.DailyStats,
		ErrorBreakdown:     historicalData.ErrorBreakdown,
		ResponseCodes:      historicalData.ResponseCodes,
		TopErrors:          historicalData.TopErrors,
		PerformanceMetrics: performanceMetrics,
	}, nil
}

// GetRealtimeStats returns current real-time statistics
func (wa *WebhookAnalytics) GetRealtimeStats(formID string) (*RealtimeWebhookStats, error) {
	ctx := context.Background()
	
	key := fmt.Sprintf("webhook_analytics:%s", formID)
	data, err := wa.redis.HGetAll(ctx, key).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get realtime stats: %w", err)
	}
	
	stats := &RealtimeWebhookStats{
		FormID:    formID,
		Timestamp: time.Now(),
	}
	
	if val, exists := data["total_requests"]; exists {
		stats.TotalRequests, _ = strconv.ParseInt(val, 10, 64)
	}
	if val, exists := data["successful_requests"]; exists {
		stats.SuccessfulRequests, _ = strconv.ParseInt(val, 10, 64)
	}
	if val, exists := data["failed_requests"]; exists {
		stats.FailedRequests, _ = strconv.ParseInt(val, 10, 64)
	}
	if val, exists := data["last_request"]; exists {
		if timestamp, err := strconv.ParseInt(val, 10, 64); err == nil {
			lastRequest := time.Unix(timestamp, 0)
			stats.LastRequest = &lastRequest
		}
	}
	
	// Calculate success rate
	if stats.TotalRequests > 0 {
		stats.SuccessRate = float64(stats.SuccessfulRequests) / float64(stats.TotalRequests) * 100
	}
	
	// Get current queue size
	queueSize, err := wa.redis.LLen(ctx, "webhook_queue").Result()
	if err == nil {
		stats.QueueSize = int(queueSize)
	}
	
	return stats, nil
}

// Helper methods

func (wa *WebhookAnalytics) getRealtimeAnalytics(ctx context.Context, formID string) (*RealtimeAnalytics, error) {
	key := fmt.Sprintf("webhook_analytics:%s", formID)
	data, err := wa.redis.HGetAll(ctx, key).Result()
	if err != nil {
		return nil, err
	}
	
	analytics := &RealtimeAnalytics{}
	
	if val, exists := data["total_requests"]; exists {
		analytics.TotalRequests, _ = strconv.ParseInt(val, 10, 64)
	}
	if val, exists := data["successful_requests"]; exists {
		analytics.SuccessfulRequests, _ = strconv.ParseInt(val, 10, 64)
	}
	if val, exists := data["failed_requests"]; exists {
		analytics.FailedRequests, _ = strconv.ParseInt(val, 10, 64)
	}
	
	return analytics, nil
}

func (wa *WebhookAnalytics) getHistoricalAnalytics(formID string, timeRange TimeRange) (*HistoricalAnalytics, error) {
	analytics := &HistoricalAnalytics{
		ErrorBreakdown: make(map[string]int64),
		ResponseCodes:  make(map[int]int64),
	}
	
	// Query database for historical data
	query := `
		SELECT 
			COUNT(*) as total_webhooks,
			SUM(CASE WHEN success = 1 THEN 1 ELSE 0 END) as successful_sent,
			SUM(CASE WHEN success = 0 THEN 1 ELSE 0 END) as failed,
			AVG(response_time_ms) as avg_response_time
		FROM webhook_logs 
		WHERE form_id = ? 
		AND created_at BETWEEN ? AND ?
	`
	
	var avgResponseTimeMs sql.NullFloat64
	err := wa.db.QueryRow(query, formID, timeRange.Start, timeRange.End).Scan(
		&analytics.TotalWebhooks,
		&analytics.SuccessfulSent,
		&analytics.Failed,
		&avgResponseTimeMs,
	)
	
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to query historical analytics: %w", err)
	}
	
	if avgResponseTimeMs.Valid {
		analytics.AvgResponseTime = time.Duration(avgResponseTimeMs.Float64) * time.Millisecond
	}
	
	// Get hourly stats
	analytics.HourlyStats, err = wa.getHourlyStats(formID, timeRange)
	if err != nil {
		log.Printf("Failed to get hourly stats: %v", err)
	}
	
	// Get daily stats
	analytics.DailyStats, err = wa.getDailyStats(formID, timeRange)
	if err != nil {
		log.Printf("Failed to get daily stats: %v", err)
	}
	
	// Get error breakdown
	analytics.ErrorBreakdown, err = wa.getErrorBreakdown(formID, timeRange)
	if err != nil {
		log.Printf("Failed to get error breakdown: %v", err)
	}
	
	// Get response codes
	analytics.ResponseCodes, err = wa.getResponseCodeBreakdown(formID, timeRange)
	if err != nil {
		log.Printf("Failed to get response code breakdown: %v", err)
	}
	
	// Get top errors
	analytics.TopErrors, err = wa.getTopErrors(formID, timeRange)
	if err != nil {
		log.Printf("Failed to get top errors: %v", err)
	}
	
	return analytics, nil
}

func (wa *WebhookAnalytics) getEndpointAnalytics(ctx context.Context, formID string) ([]EndpointAnalytics, error) {
	// Get list of endpoints for this form
	endpoints, err := wa.getFormEndpoints(formID)
	if err != nil {
		return nil, err
	}
	
	var stats []EndpointAnalytics
	
	for _, endpoint := range endpoints {
		endpointKey := fmt.Sprintf("webhook_endpoint_analytics:%s:%s", formID, endpoint.ID)
		data, err := wa.redis.HGetAll(ctx, endpointKey).Result()
		if err != nil {
			continue
		}
		
		stat := EndpointAnalytics{
			EndpointID: endpoint.ID,
			Name:       endpoint.Name,
			URL:        endpoint.URL,
		}
		
		if val, exists := data["total_requests"]; exists {
			stat.TotalRequests, _ = strconv.ParseInt(val, 10, 64)
		}
		if val, exists := data["successful_requests"]; exists {
			stat.SuccessfulRequests, _ = strconv.ParseInt(val, 10, 64)
		}
		if val, exists := data["failed_requests"]; exists {
			stat.FailedRequests, _ = strconv.ParseInt(val, 10, 64)
		}
		
		// Calculate success rate
		if stat.TotalRequests > 0 {
			stat.SuccessRate = float64(stat.SuccessfulRequests) / float64(stat.TotalRequests) * 100
		}
		
		// Get average response time
		responseTimesKey := fmt.Sprintf("webhook_response_times:%s", endpoint.ID)
		responseTimes, err := wa.redis.LRange(ctx, responseTimesKey, 0, -1).Result()
		if err == nil && len(responseTimes) > 0 {
			totalTime := int64(0)
			count := int64(0)
			for _, timeStr := range responseTimes {
				if time, err := strconv.ParseInt(timeStr, 10, 64); err == nil {
					totalTime += time
					count++
				}
			}
			if count > 0 {
				stat.AvgResponseTime = time.Duration(totalTime/count) * time.Millisecond
			}
		}
		
		// Get last success/failure times
		if val, exists := data["last_success"]; exists {
			if timestamp, err := strconv.ParseInt(val, 10, 64); err == nil {
				lastSuccess := time.Unix(timestamp, 0)
				stat.LastSuccess = &lastSuccess
			}
		}
		if val, exists := data["last_failure"]; exists {
			if timestamp, err := strconv.ParseInt(val, 10, 64); err == nil {
				lastFailure := time.Unix(timestamp, 0)
				stat.LastFailure = &lastFailure
			}
		}
		
		stats = append(stats, stat)
	}
	
	return stats, nil
}

func (wa *WebhookAnalytics) getPerformanceMetrics(ctx context.Context, formID string) (*PerformanceMetrics, error) {
	// Get all response times for endpoints in this form
	endpoints, err := wa.getFormEndpoints(formID)
	if err != nil {
		return nil, err
	}
	
	var allResponseTimes []int64
	
	for _, endpoint := range endpoints {
		responseTimesKey := fmt.Sprintf("webhook_response_times:%s", endpoint.ID)
		responseTimes, err := wa.redis.LRange(ctx, responseTimesKey, 0, -1).Result()
		if err != nil {
			continue
		}
		
		for _, timeStr := range responseTimes {
			if time, err := strconv.ParseInt(timeStr, 10, 64); err == nil {
				allResponseTimes = append(allResponseTimes, time)
			}
		}
	}
	
	if len(allResponseTimes) == 0 {
		return &PerformanceMetrics{}, nil
	}
	
	// Sort for percentile calculations
	sort.Slice(allResponseTimes, func(i, j int) bool {
		return allResponseTimes[i] < allResponseTimes[j]
	})
	
	metrics := &PerformanceMetrics{
		MinResponseTime: time.Duration(allResponseTimes[0]) * time.Millisecond,
		MaxResponseTime: time.Duration(allResponseTimes[len(allResponseTimes)-1]) * time.Millisecond,
	}
	
	// Calculate average
	total := int64(0)
	for _, responseTime := range allResponseTimes {
		total += responseTime
	}
	metrics.AvgResponseTime = time.Duration(total/int64(len(allResponseTimes))) * time.Millisecond
	
	// Calculate percentiles
	metrics.P50ResponseTime = time.Duration(allResponseTimes[int(float64(len(allResponseTimes))*0.50)]) * time.Millisecond
	metrics.P90ResponseTime = time.Duration(allResponseTimes[int(float64(len(allResponseTimes))*0.90)]) * time.Millisecond
	metrics.P95ResponseTime = time.Duration(allResponseTimes[int(float64(len(allResponseTimes))*0.95)]) * time.Millisecond
	metrics.P99ResponseTime = time.Duration(allResponseTimes[int(float64(len(allResponseTimes))*0.99)]) * time.Millisecond
	
	return metrics, nil
}

func (wa *WebhookAnalytics) getHourlyStats(formID string, timeRange TimeRange) ([]HourlyWebhookStats, error) {
	query := `
		SELECT 
			DATE_FORMAT(created_at, '%Y-%m-%d %H:00:00') as hour,
			COUNT(*) as total_requests,
			SUM(CASE WHEN success = 1 THEN 1 ELSE 0 END) as successful_requests,
			SUM(CASE WHEN success = 0 THEN 1 ELSE 0 END) as failed_requests,
			AVG(response_time_ms) as avg_response_time
		FROM webhook_logs 
		WHERE form_id = ? 
		AND created_at BETWEEN ? AND ?
		GROUP BY DATE_FORMAT(created_at, '%Y-%m-%d %H:00:00')
		ORDER BY hour
	`
	
	rows, err := wa.db.Query(query, formID, timeRange.Start, timeRange.End)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var stats []HourlyWebhookStats
	
	for rows.Next() {
		var stat HourlyWebhookStats
		var hourStr string
		var avgResponseTimeMs sql.NullFloat64
		
		err := rows.Scan(&hourStr, &stat.TotalRequests, &stat.SuccessfulRequests, 
			&stat.FailedRequests, &avgResponseTimeMs)
		if err != nil {
			continue
		}
		
		if hour, err := time.Parse("2006-01-02 15:04:05", hourStr); err == nil {
			stat.Hour = hour
		}
		
		if avgResponseTimeMs.Valid {
			stat.AvgResponseTime = time.Duration(avgResponseTimeMs.Float64) * time.Millisecond
		}
		
		stats = append(stats, stat)
	}
	
	return stats, nil
}

func (wa *WebhookAnalytics) getDailyStats(formID string, timeRange TimeRange) ([]DailyWebhookStats, error) {
	query := `
		SELECT 
			DATE(created_at) as date,
			COUNT(*) as total_requests,
			SUM(CASE WHEN success = 1 THEN 1 ELSE 0 END) as successful_requests,
			SUM(CASE WHEN success = 0 THEN 1 ELSE 0 END) as failed_requests,
			AVG(response_time_ms) as avg_response_time
		FROM webhook_logs 
		WHERE form_id = ? 
		AND created_at BETWEEN ? AND ?
		GROUP BY DATE(created_at)
		ORDER BY date
	`
	
	rows, err := wa.db.Query(query, formID, timeRange.Start, timeRange.End)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var stats []DailyWebhookStats
	
	for rows.Next() {
		var stat DailyWebhookStats
		var dateStr string
		var avgResponseTimeMs sql.NullFloat64
		
		err := rows.Scan(&dateStr, &stat.TotalRequests, &stat.SuccessfulRequests, 
			&stat.FailedRequests, &avgResponseTimeMs)
		if err != nil {
			continue
		}
		
		if date, err := time.Parse("2006-01-02", dateStr); err == nil {
			stat.Date = date
		}
		
		if avgResponseTimeMs.Valid {
			stat.AvgResponseTime = time.Duration(avgResponseTimeMs.Float64) * time.Millisecond
		}
		
		stats = append(stats, stat)
	}
	
	return stats, nil
}

func (wa *WebhookAnalytics) getErrorBreakdown(formID string, timeRange TimeRange) (map[string]int64, error) {
	query := `
		SELECT error_message, COUNT(*) as count
		FROM webhook_logs 
		WHERE form_id = ? 
		AND success = 0
		AND created_at BETWEEN ? AND ?
		AND error_message IS NOT NULL
		GROUP BY error_message
		ORDER BY count DESC
		LIMIT 20
	`
	
	rows, err := wa.db.Query(query, formID, timeRange.Start, timeRange.End)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	breakdown := make(map[string]int64)
	
	for rows.Next() {
		var errorMessage string
		var count int64
		
		if err := rows.Scan(&errorMessage, &count); err == nil {
			// Categorize the error
			category := wa.categorizeError(errorMessage)
			breakdown[category] += count
		}
	}
	
	return breakdown, nil
}

func (wa *WebhookAnalytics) getResponseCodeBreakdown(formID string, timeRange TimeRange) (map[int]int64, error) {
	query := `
		SELECT status_code, COUNT(*) as count
		FROM webhook_logs 
		WHERE form_id = ? 
		AND created_at BETWEEN ? AND ?
		AND status_code IS NOT NULL
		GROUP BY status_code
		ORDER BY count DESC
	`
	
	rows, err := wa.db.Query(query, formID, timeRange.Start, timeRange.End)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	breakdown := make(map[int]int64)
	
	for rows.Next() {
		var statusCode int
		var count int64
		
		if err := rows.Scan(&statusCode, &count); err == nil {
			breakdown[statusCode] = count
		}
	}
	
	return breakdown, nil
}

func (wa *WebhookAnalytics) getTopErrors(formID string, timeRange TimeRange) ([]ErrorStat, error) {
	query := `
		SELECT error_message, COUNT(*) as count,
		       (COUNT(*) * 100.0 / (SELECT COUNT(*) FROM webhook_logs WHERE form_id = ? AND success = 0 AND created_at BETWEEN ? AND ?)) as percentage
		FROM webhook_logs 
		WHERE form_id = ? 
		AND success = 0
		AND created_at BETWEEN ? AND ?
		AND error_message IS NOT NULL
		GROUP BY error_message
		ORDER BY count DESC
		LIMIT 10
	`
	
	rows, err := wa.db.Query(query, formID, timeRange.Start, timeRange.End, formID, timeRange.Start, timeRange.End)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var errors []ErrorStat
	
	for rows.Next() {
		var stat ErrorStat
		var percentage sql.NullFloat64
		
		err := rows.Scan(&stat.Error, &stat.Count, &percentage)
		if err != nil {
			continue
		}
		
		if percentage.Valid {
			stat.Percentage = percentage.Float64
		}
		
		errors = append(errors, stat)
	}
	
	return errors, nil
}

func (wa *WebhookAnalytics) getFormEndpoints(formID string) ([]WebhookEndpoint, error) {
	query := `SELECT webhook_config FROM forms WHERE id = ? AND webhook_config IS NOT NULL`
	
	var configJSON string
	err := wa.db.QueryRow(query, formID).Scan(&configJSON)
	if err != nil {
		if err == sql.ErrNoRows {
			return []WebhookEndpoint{}, nil
		}
		return nil, err
	}
	
	var config FormWebhookConfig
	if err := json.Unmarshal([]byte(configJSON), &config); err != nil {
		return nil, err
	}
	
	return config.Endpoints, nil
}

func (wa *WebhookAnalytics) storeWebhookRecord(formID, endpointID string, result *WebhookResult) {
	query := `
		INSERT INTO webhook_logs 
		(id, endpoint_id, form_id, url, status_code, response_time_ms, attempts, success, error_message, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	
	_, err := wa.db.Exec(query,
		uuid.New().String(),
		endpointID,
		formID,
		result.URL,
		result.StatusCode,
		result.ResponseTime.Milliseconds(),
		result.Attempts,
		result.Success,
		result.Error,
		time.Now(),
	)
	
	if err != nil {
		log.Printf("Failed to store webhook record: %v", err)
	}
}

func (wa *WebhookAnalytics) categorizeError(errorMessage string) string {
	errorMessage = strings.ToLower(errorMessage)
	
	if strings.Contains(errorMessage, "timeout") || strings.Contains(errorMessage, "deadline exceeded") {
		return "timeout"
	}
	if strings.Contains(errorMessage, "connection") || strings.Contains(errorMessage, "network") {
		return "connection"
	}
	if strings.Contains(errorMessage, "dns") || strings.Contains(errorMessage, "resolve") {
		return "dns"
	}
	if strings.Contains(errorMessage, "tls") || strings.Contains(errorMessage, "certificate") {
		return "tls"
	}
	if strings.Contains(errorMessage, "401") || strings.Contains(errorMessage, "unauthorized") {
		return "authentication"
	}
	if strings.Contains(errorMessage, "403") || strings.Contains(errorMessage, "forbidden") {
		return "authorization"
	}
	if strings.Contains(errorMessage, "404") || strings.Contains(errorMessage, "not found") {
		return "not_found"
	}
	if strings.Contains(errorMessage, "429") || strings.Contains(errorMessage, "rate limit") {
		return "rate_limit"
	}
	if strings.Contains(errorMessage, "500") || strings.Contains(errorMessage, "internal server error") {
		return "server_error"
	}
	if strings.Contains(errorMessage, "502") || strings.Contains(errorMessage, "bad gateway") {
		return "bad_gateway"
	}
	if strings.Contains(errorMessage, "503") || strings.Contains(errorMessage, "service unavailable") {
		return "service_unavailable"
	}
	
	return "unknown"
}

// Supporting types

type RealtimeAnalytics struct {
	TotalRequests      int64 `json:"total_requests"`
	SuccessfulRequests int64 `json:"successful_requests"`
	FailedRequests     int64 `json:"failed_requests"`
}

type HistoricalAnalytics struct {
	TotalWebhooks   int64                    `json:"total_webhooks"`
	SuccessfulSent  int64                    `json:"successful_sent"`
	Failed          int64                    `json:"failed"`
	AvgResponseTime time.Duration            `json:"avg_response_time"`
	HourlyStats     []HourlyWebhookStats     `json:"hourly_stats"`
	DailyStats      []DailyWebhookStats      `json:"daily_stats"`
	ErrorBreakdown  map[string]int64         `json:"error_breakdown"`
	ResponseCodes   map[int]int64            `json:"response_codes"`
	TopErrors       []ErrorStat              `json:"top_errors"`
}

type RealtimeWebhookStats struct {
	FormID             string     `json:"form_id"`
	TotalRequests      int64      `json:"total_requests"`
	SuccessfulRequests int64      `json:"successful_requests"`
	FailedRequests     int64      `json:"failed_requests"`
	SuccessRate        float64    `json:"success_rate"`
	LastRequest        *time.Time `json:"last_request"`
	QueueSize          int        `json:"queue_size"`
	Timestamp          time.Time  `json:"timestamp"`
}