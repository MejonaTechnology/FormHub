package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// Start starts the webhook monitoring service
func (wm *WebhookMonitor) Start() {
	log.Println("Starting webhook monitoring service...")
	
	go wm.runHealthChecks()
	go wm.monitorAlerts()
	go wm.processMetrics()
	
	log.Println("Webhook monitoring service started")
}

// Stop stops the webhook monitoring service
func (wm *WebhookMonitor) Stop() {
	log.Println("Stopping webhook monitoring service...")
	// Implementation would handle graceful shutdown
}

// GetMonitoringData returns comprehensive monitoring data for a form
func (wm *WebhookMonitor) GetMonitoringData(formID string) (*WebhookMonitoringData, error) {
	wm.mu.RLock()
	defer wm.mu.RUnlock()
	
	ctx := context.Background()
	
	// Get current status
	status, err := wm.calculateOverallStatus(ctx, formID)
	if err != nil {
		log.Printf("Failed to calculate overall status: %v", err)
		status = "unknown"
	}
	
	// Get endpoint counts
	activeEndpoints, failingEndpoints, err := wm.getEndpointCounts(ctx, formID)
	if err != nil {
		log.Printf("Failed to get endpoint counts: %v", err)
	}
	
	// Get current load and queue size
	currentLoad, queueSize, err := wm.getCurrentLoad(ctx, formID)
	if err != nil {
		log.Printf("Failed to get current load: %v", err)
	}
	
	// Get recent events
	recentEvents, err := wm.getRecentEvents(formID, 50)
	if err != nil {
		log.Printf("Failed to get recent events: %v", err)
	}
	
	// Get health checks
	healthChecks, err := wm.getHealthChecks(ctx, formID)
	if err != nil {
		log.Printf("Failed to get health checks: %v", err)
	}
	
	// Get active alerts
	alerts, err := wm.getActiveAlerts(formID)
	if err != nil {
		log.Printf("Failed to get active alerts: %v", err)
	}
	
	return &WebhookMonitoringData{
		FormID:           formID,
		Status:           status,
		ActiveEndpoints:  activeEndpoints,
		FailingEndpoints: failingEndpoints,
		CurrentLoad:      currentLoad,
		QueueSize:        queueSize,
		RecentEvents:     recentEvents,
		HealthChecks:     healthChecks,
		Alerts:           alerts,
	}, nil
}

// RecordMonitorEvent records a monitoring event
func (wm *WebhookMonitor) RecordMonitorEvent(event *MonitorEvent) {
	wm.mu.Lock()
	defer wm.mu.Unlock()
	
	ctx := context.Background()
	
	// Store in Redis for real-time access
	eventKey := fmt.Sprintf("monitor_events:%s", event.FormID)
	eventJSON, _ := json.Marshal(event)
	
	pipe := wm.redis.Pipeline()
	pipe.LPush(ctx, eventKey, string(eventJSON))
	pipe.LTrim(ctx, eventKey, 0, 999) // Keep last 1000 events
	pipe.Expire(ctx, eventKey, 24*time.Hour)
	
	if _, err := pipe.Exec(ctx); err != nil {
		log.Printf("Failed to record monitor event: %v", err)
	}
	
	// Store in database for long-term retention
	go wm.storeMonitorEvent(event)
	
	// Check if this event should trigger an alert
	if wm.shouldTriggerAlert(event) {
		go wm.processAlert(event)
	}
	
	// Notify subscribers
	wm.notifySubscribers(event)
}

// Subscribe subscribes to monitoring events for a form
func (wm *WebhookMonitor) Subscribe(formID, subscriberID string) <-chan *MonitorEvent {
	wm.mu.Lock()
	defer wm.mu.Unlock()
	
	channel := make(chan *MonitorEvent, 100) // Buffered channel
	key := fmt.Sprintf("%s:%s", formID, subscriberID)
	wm.subscribers[key] = channel
	
	return channel
}

// Unsubscribe unsubscribes from monitoring events
func (wm *WebhookMonitor) Unsubscribe(formID, subscriberID string) {
	wm.mu.Lock()
	defer wm.mu.Unlock()
	
	key := fmt.Sprintf("%s:%s", formID, subscriberID)
	if channel, exists := wm.subscribers[key]; exists {
		close(channel)
		delete(wm.subscribers, key)
	}
}

// CreateAlert creates a new monitoring alert
func (wm *WebhookMonitor) CreateAlert(alert *MonitoringAlert) error {
	alert.ID = uuid.New().String()
	alert.Timestamp = time.Now()
	alert.Acknowledged = false
	
	// Store alert in database
	query := `
		INSERT INTO webhook_alerts 
		(id, form_id, endpoint_id, type, severity, message, data, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`
	
	dataJSON, _ := json.Marshal(alert.Data)
	_, err := wm.db.Exec(query,
		alert.ID,
		"", // form_id will be extracted from context
		alert.EndpointID,
		alert.Type,
		alert.Severity,
		alert.Message,
		string(dataJSON),
		alert.Timestamp,
	)
	
	if err != nil {
		return fmt.Errorf("failed to create alert: %w", err)
	}
	
	// Create monitoring event
	event := &MonitorEvent{
		Type:        "alert_created",
		Timestamp:   time.Now(),
		EndpointID:  alert.EndpointID,
		Severity:    alert.Severity,
		Message:     fmt.Sprintf("Alert created: %s", alert.Message),
		Data: map[string]interface{}{
			"alert_id":   alert.ID,
			"alert_type": alert.Type,
		},
	}
	
	wm.RecordMonitorEvent(event)
	
	return nil
}

// AcknowledgeAlert acknowledges an alert
func (wm *WebhookMonitor) AcknowledgeAlert(alertID string) error {
	query := `UPDATE webhook_alerts SET acknowledged = TRUE, acknowledged_at = ? WHERE id = ?`
	_, err := wm.db.Exec(query, time.Now(), alertID)
	return err
}

// runHealthChecks performs periodic health checks on webhook endpoints
func (wm *WebhookMonitor) runHealthChecks() {
	ticker := time.NewTicker(wm.checkInterval)
	defer ticker.Stop()
	
	for range ticker.C {
		wm.performHealthChecks()
	}
}

// performHealthChecks checks the health of all webhook endpoints
func (wm *WebhookMonitor) performHealthChecks() {
	ctx := context.Background()
	
	// Get all forms with webhook configurations
	forms, err := wm.getFormsWithWebhooks()
	if err != nil {
		log.Printf("Failed to get forms with webhooks: %v", err)
		return
	}
	
	for _, formID := range forms {
		go wm.checkFormEndpoints(ctx, formID)
	}
}

// checkFormEndpoints performs health checks for all endpoints in a form
func (wm *WebhookMonitor) checkFormEndpoints(ctx context.Context, formID string) {
	endpoints, err := wm.getFormEndpoints(formID)
	if err != nil {
		log.Printf("Failed to get endpoints for form %s: %v", formID, err)
		return
	}
	
	for _, endpoint := range endpoints {
		if endpoint.Enabled {
			go wm.performEndpointHealthCheck(ctx, formID, &endpoint)
		}
	}
}

// performEndpointHealthCheck performs a health check on a specific endpoint
func (wm *WebhookMonitor) performEndpointHealthCheck(ctx context.Context, formID string, endpoint *WebhookEndpoint) {
	startTime := time.Now()
	
	healthCheck := &EndpointHealthCheck{
		EndpointID:   endpoint.ID,
		Status:       "unknown",
		LastCheck:    startTime,
		ResponseTime: 0,
		SuccessRate:  0,
	}
	
	// Get recent success rate
	healthCheck.SuccessRate = wm.calculateEndpointSuccessRate(ctx, formID, endpoint.ID)
	
	// Perform simple HTTP check (HEAD request)
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequestWithContext(ctx, "HEAD", endpoint.URL, nil)
	if err != nil {
		healthCheck.Status = "unhealthy"
		healthCheck.ErrorMessage = fmt.Sprintf("Failed to create request: %v", err)
		wm.storeHealthCheck(formID, healthCheck)
		return
	}
	
	resp, err := client.Do(req)
	healthCheck.ResponseTime = time.Since(startTime)
	
	if err != nil {
		healthCheck.Status = "unhealthy"
		healthCheck.ErrorMessage = fmt.Sprintf("Request failed: %v", err)
	} else {
		resp.Body.Close()
		if resp.StatusCode < 400 {
			healthCheck.Status = "healthy"
		} else {
			healthCheck.Status = "unhealthy"
			healthCheck.ErrorMessage = fmt.Sprintf("HTTP %d", resp.StatusCode)
		}
	}
	
	// Store health check result
	wm.storeHealthCheck(formID, healthCheck)
	
	// Create monitoring event if status changed
	wm.checkStatusChange(formID, endpoint.ID, healthCheck)
}

// monitorAlerts monitors for alert conditions
func (wm *WebhookMonitor) monitorAlerts() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	
	for range ticker.C {
		wm.checkAlertConditions()
	}
}

// checkAlertConditions checks various conditions that might trigger alerts
func (wm *WebhookMonitor) checkAlertConditions() {
	ctx := context.Background()
	
	forms, err := wm.getFormsWithWebhooks()
	if err != nil {
		return
	}
	
	for _, formID := range forms {
		wm.checkFormAlertConditions(ctx, formID)
	}
}

// checkFormAlertConditions checks alert conditions for a specific form
func (wm *WebhookMonitor) checkFormAlertConditions(ctx context.Context, formID string) {
	// Check failure rate threshold
	failureRate := wm.calculateFormFailureRate(ctx, formID)
	if failureRate > wm.alertThreshold {
		alert := &MonitoringAlert{
			Type:     "high_failure_rate",
			Severity: "critical",
			Message:  fmt.Sprintf("High failure rate detected: %.2f%%", failureRate*100),
			Data: map[string]interface{}{
				"failure_rate": failureRate,
				"threshold":    wm.alertThreshold,
			},
		}
		wm.CreateAlert(alert)
	}
	
	// Check queue size
	queueSize, _ := wm.redis.LLen(ctx, "webhook_queue").Result()
	if queueSize > 1000 { // Threshold for large queue
		alert := &MonitoringAlert{
			Type:     "large_queue",
			Severity: "warning",
			Message:  fmt.Sprintf("Large webhook queue detected: %d items", queueSize),
			Data: map[string]interface{}{
				"queue_size": queueSize,
			},
		}
		wm.CreateAlert(alert)
	}
	
	// Check endpoint health
	endpoints, err := wm.getFormEndpoints(formID)
	if err != nil {
		return
	}
	
	for _, endpoint := range endpoints {
		healthStatus := wm.getEndpointHealthStatus(ctx, formID, endpoint.ID)
		if healthStatus == "unhealthy" {
			alert := &MonitoringAlert{
				Type:       "endpoint_unhealthy",
				Severity:   "warning",
				Message:    fmt.Sprintf("Endpoint %s is unhealthy", endpoint.Name),
				EndpointID: endpoint.ID,
				Data: map[string]interface{}{
					"endpoint_url":  endpoint.URL,
					"endpoint_name": endpoint.Name,
				},
			}
			wm.CreateAlert(alert)
		}
	}
}

// processMetrics processes and aggregates monitoring metrics
func (wm *WebhookMonitor) processMetrics() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	
	for range ticker.C {
		wm.aggregateMetrics()
	}
}

// aggregateMetrics aggregates monitoring metrics for reporting
func (wm *WebhookMonitor) aggregateMetrics() {
	ctx := context.Background()
	
	forms, err := wm.getFormsWithWebhooks()
	if err != nil {
		return
	}
	
	for _, formID := range forms {
		// Aggregate metrics for this form
		metrics := wm.collectFormMetrics(ctx, formID)
		
		// Store aggregated metrics
		metricsJSON, _ := json.Marshal(metrics)
		key := fmt.Sprintf("webhook_metrics:%s", formID)
		wm.redis.HSet(ctx, key, "latest", string(metricsJSON))
		wm.redis.Expire(ctx, key, 24*time.Hour)
		
		// Store historical metrics
		timestamp := time.Now().Unix()
		historicalKey := fmt.Sprintf("webhook_metrics_historical:%s", formID)
		wm.redis.ZAdd(ctx, historicalKey, redis.Z{
			Score:  float64(timestamp),
			Member: string(metricsJSON),
		})
		wm.redis.ZRemRangeByScore(ctx, historicalKey, "0", fmt.Sprintf("%d", timestamp-86400*7)) // Keep 7 days
		wm.redis.Expire(ctx, historicalKey, 7*24*time.Hour)
	}
}

// Helper methods

func (wm *WebhookMonitor) calculateOverallStatus(ctx context.Context, formID string) (string, error) {
	// Get failure rate
	failureRate := wm.calculateFormFailureRate(ctx, formID)
	
	// Get unhealthy endpoint count
	endpoints, err := wm.getFormEndpoints(formID)
	if err != nil {
		return "unknown", err
	}
	
	unhealthyCount := 0
	for _, endpoint := range endpoints {
		if wm.getEndpointHealthStatus(ctx, formID, endpoint.ID) == "unhealthy" {
			unhealthyCount++
		}
	}
	
	// Determine overall status
	if len(endpoints) == 0 {
		return "unknown", nil
	}
	
	if unhealthyCount == len(endpoints) {
		return "critical", nil
	} else if unhealthyCount > 0 || failureRate > wm.alertThreshold {
		return "warning", nil
	} else {
		return "healthy", nil
	}
}

func (wm *WebhookMonitor) getEndpointCounts(ctx context.Context, formID string) (int, int, error) {
	endpoints, err := wm.getFormEndpoints(formID)
	if err != nil {
		return 0, 0, err
	}
	
	active := 0
	failing := 0
	
	for _, endpoint := range endpoints {
		if endpoint.Enabled {
			active++
			if wm.getEndpointHealthStatus(ctx, formID, endpoint.ID) == "unhealthy" {
				failing++
			}
		}
	}
	
	return active, failing, nil
}

func (wm *WebhookMonitor) getCurrentLoad(ctx context.Context, formID string) (int, int, error) {
	// Get current processing jobs
	currentLoad := 0 // This would be tracked by the worker pool
	
	// Get queue size
	queueSize, err := wm.redis.LLen(ctx, "webhook_queue").Result()
	if err != nil {
		return 0, 0, err
	}
	
	return currentLoad, int(queueSize), nil
}

func (wm *WebhookMonitor) getRecentEvents(formID string, limit int) ([]MonitorEvent, error) {
	ctx := context.Background()
	eventKey := fmt.Sprintf("monitor_events:%s", formID)
	
	events, err := wm.redis.LRange(ctx, eventKey, 0, int64(limit-1)).Result()
	if err != nil {
		return nil, err
	}
	
	var monitorEvents []MonitorEvent
	for _, eventJSON := range events {
		var event MonitorEvent
		if err := json.Unmarshal([]byte(eventJSON), &event); err == nil {
			monitorEvents = append(monitorEvents, event)
		}
	}
	
	return monitorEvents, nil
}

func (wm *WebhookMonitor) getHealthChecks(ctx context.Context, formID string) ([]EndpointHealthCheck, error) {
	endpoints, err := wm.getFormEndpoints(formID)
	if err != nil {
		return nil, err
	}
	
	var healthChecks []EndpointHealthCheck
	
	for _, endpoint := range endpoints {
		healthKey := fmt.Sprintf("webhook_health:%s:%s", formID, endpoint.ID)
		healthJSON, err := wm.redis.Get(ctx, healthKey).Result()
		if err != nil {
			continue
		}
		
		var health EndpointHealthCheck
		if err := json.Unmarshal([]byte(healthJSON), &health); err == nil {
			healthChecks = append(healthChecks, health)
		}
	}
	
	return healthChecks, nil
}

func (wm *WebhookMonitor) getActiveAlerts(formID string) ([]MonitoringAlert, error) {
	query := `
		SELECT id, type, severity, message, endpoint_id, data, created_at
		FROM webhook_alerts 
		WHERE form_id = ? AND acknowledged = FALSE 
		ORDER BY created_at DESC
		LIMIT 50
	`
	
	rows, err := wm.db.Query(query, formID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var alerts []MonitoringAlert
	
	for rows.Next() {
		var alert MonitoringAlert
		var dataJSON sql.NullString
		var endpointID sql.NullString
		
		err := rows.Scan(&alert.ID, &alert.Type, &alert.Severity, 
			&alert.Message, &endpointID, &dataJSON, &alert.Timestamp)
		if err != nil {
			continue
		}
		
		if endpointID.Valid {
			alert.EndpointID = endpointID.String
		}
		
		if dataJSON.Valid {
			json.Unmarshal([]byte(dataJSON.String), &alert.Data)
		}
		
		alerts = append(alerts, alert)
	}
	
	return alerts, nil
}

func (wm *WebhookMonitor) calculateFormFailureRate(ctx context.Context, formID string) float64 {
	key := fmt.Sprintf("webhook_analytics:%s", formID)
	
	data, err := wm.redis.HMGet(ctx, key, "total_requests", "failed_requests").Result()
	if err != nil || len(data) != 2 {
		return 0
	}
	
	totalRequests := wm.parseRedisInt(data[0])
	failedRequests := wm.parseRedisInt(data[1])
	
	if totalRequests == 0 {
		return 0
	}
	
	return float64(failedRequests) / float64(totalRequests)
}

func (wm *WebhookMonitor) calculateEndpointSuccessRate(ctx context.Context, formID, endpointID string) float64 {
	key := fmt.Sprintf("webhook_endpoint_analytics:%s:%s", formID, endpointID)
	
	data, err := wm.redis.HMGet(ctx, key, "total_requests", "successful_requests").Result()
	if err != nil || len(data) != 2 {
		return 0
	}
	
	totalRequests := wm.parseRedisInt(data[0])
	successfulRequests := wm.parseRedisInt(data[1])
	
	if totalRequests == 0 {
		return 100 // No requests yet, assume healthy
	}
	
	return float64(successfulRequests) / float64(totalRequests) * 100
}

func (wm *WebhookMonitor) getEndpointHealthStatus(ctx context.Context, formID, endpointID string) string {
	healthKey := fmt.Sprintf("webhook_health:%s:%s", formID, endpointID)
	healthJSON, err := wm.redis.Get(ctx, healthKey).Result()
	if err != nil {
		return "unknown"
	}
	
	var health EndpointHealthCheck
	if err := json.Unmarshal([]byte(healthJSON), &health); err != nil {
		return "unknown"
	}
	
	return health.Status
}

func (wm *WebhookMonitor) getFormsWithWebhooks() ([]string, error) {
	query := `SELECT id FROM forms WHERE webhook_config IS NOT NULL`
	
	rows, err := wm.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var formIDs []string
	for rows.Next() {
		var formID string
		if err := rows.Scan(&formID); err == nil {
			formIDs = append(formIDs, formID)
		}
	}
	
	return formIDs, nil
}

func (wm *WebhookMonitor) getFormEndpoints(formID string) ([]WebhookEndpoint, error) {
	query := `SELECT webhook_config FROM forms WHERE id = ? AND webhook_config IS NOT NULL`
	
	var configJSON string
	err := wm.db.QueryRow(query, formID).Scan(&configJSON)
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

func (wm *WebhookMonitor) storeHealthCheck(formID string, healthCheck *EndpointHealthCheck) {
	ctx := context.Background()
	
	// Store in Redis for real-time access
	healthKey := fmt.Sprintf("webhook_health:%s:%s", formID, healthCheck.EndpointID)
	healthJSON, _ := json.Marshal(healthCheck)
	wm.redis.Set(ctx, healthKey, string(healthJSON), time.Hour)
	
	// Store in database for historical tracking
	query := `
		INSERT INTO webhook_health_checks 
		(id, form_id, endpoint_id, status, response_time_ms, success_rate, error_message, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE 
		status = VALUES(status), response_time_ms = VALUES(response_time_ms), 
		success_rate = VALUES(success_rate), error_message = VALUES(error_message), 
		updated_at = VALUES(created_at)
	`
	
	_, err := wm.db.Exec(query,
		uuid.New().String(),
		formID,
		healthCheck.EndpointID,
		healthCheck.Status,
		healthCheck.ResponseTime.Milliseconds(),
		healthCheck.SuccessRate,
		healthCheck.ErrorMessage,
		healthCheck.LastCheck,
	)
	
	if err != nil {
		log.Printf("Failed to store health check: %v", err)
	}
}

func (wm *WebhookMonitor) storeMonitorEvent(event *MonitorEvent) {
	query := `
		INSERT INTO webhook_monitor_events 
		(id, form_id, endpoint_id, type, severity, message, data, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`
	
	dataJSON, _ := json.Marshal(event.Data)
	_, err := wm.db.Exec(query,
		uuid.New().String(),
		event.FormID,
		event.EndpointID,
		event.Type,
		event.Severity,
		event.Message,
		string(dataJSON),
		event.Timestamp,
	)
	
	if err != nil {
		log.Printf("Failed to store monitor event: %v", err)
	}
}

func (wm *WebhookMonitor) shouldTriggerAlert(event *MonitorEvent) bool {
	// Define conditions that should trigger alerts
	criticalTypes := map[string]bool{
		"endpoint_down":       true,
		"high_failure_rate":   true,
		"security_breach":     true,
		"service_unavailable": true,
	}
	
	return event.Severity == "critical" || criticalTypes[event.Type]
}

func (wm *WebhookMonitor) processAlert(event *MonitorEvent) {
	alert := &MonitoringAlert{
		Type:       event.Type,
		Severity:   event.Severity,
		Message:    event.Message,
		EndpointID: event.EndpointID,
		Data:       event.Data,
	}
	
	wm.CreateAlert(alert)
}

func (wm *WebhookMonitor) checkStatusChange(formID, endpointID string, currentCheck *EndpointHealthCheck) {
	// Get previous health status
	ctx := context.Background()
	healthKey := fmt.Sprintf("webhook_health:%s:%s", formID, endpointID)
	
	previousHealthJSON, err := wm.redis.Get(ctx, healthKey+"_previous").Result()
	if err != nil {
		// No previous status, this is the first check
		wm.redis.Set(ctx, healthKey+"_previous", currentCheck.Status, time.Hour)
		return
	}
	
	var previousHealth EndpointHealthCheck
	if err := json.Unmarshal([]byte(previousHealthJSON), &previousHealth); err != nil {
		return
	}
	
	// Check if status changed
	if previousHealth.Status != currentCheck.Status {
		event := &MonitorEvent{
			Type:       "status_change",
			Timestamp:  time.Now(),
			FormID:     formID,
			EndpointID: endpointID,
			Severity:   wm.getSeverityForStatusChange(previousHealth.Status, currentCheck.Status),
			Message:    fmt.Sprintf("Endpoint status changed from %s to %s", previousHealth.Status, currentCheck.Status),
			Data: map[string]interface{}{
				"previous_status": previousHealth.Status,
				"current_status":  currentCheck.Status,
				"response_time":   currentCheck.ResponseTime.Milliseconds(),
			},
		}
		
		wm.RecordMonitorEvent(event)
		
		// Update previous status
		currentHealthJSON, _ := json.Marshal(currentCheck)
		wm.redis.Set(ctx, healthKey+"_previous", string(currentHealthJSON), time.Hour)
	}
}

func (wm *WebhookMonitor) getSeverityForStatusChange(previousStatus, currentStatus string) string {
	if currentStatus == "unhealthy" {
		return "warning"
	} else if previousStatus == "unhealthy" && currentStatus == "healthy" {
		return "info"
	}
	return "info"
}

func (wm *WebhookMonitor) notifySubscribers(event *MonitorEvent) {
	wm.mu.RLock()
	defer wm.mu.RUnlock()
	
	// Notify all subscribers for this form
	for key, channel := range wm.subscribers {
		if strings.HasPrefix(key, event.FormID+":") {
			select {
			case channel <- event:
				// Event sent successfully
			default:
				// Channel is full, skip this subscriber
				log.Printf("Subscriber channel full, skipping notification for %s", key)
			}
		}
	}
}

func (wm *WebhookMonitor) collectFormMetrics(ctx context.Context, formID string) map[string]interface{} {
	key := fmt.Sprintf("webhook_analytics:%s", formID)
	data, _ := wm.redis.HGetAll(ctx, key).Result()
	
	metrics := make(map[string]interface{})
	metrics["form_id"] = formID
	metrics["timestamp"] = time.Now().Unix()
	
	for field, value := range data {
		if intVal := wm.parseRedisInt(value); intVal != 0 {
			metrics[field] = intVal
		} else {
			metrics[field] = value
		}
	}
	
	return metrics
}

func (wm *WebhookMonitor) parseRedisInt(value interface{}) int64 {
	if str, ok := value.(string); ok {
		if val, err := strconv.ParseInt(str, 10, 64); err == nil {
			return val
		}
	}
	return 0
}