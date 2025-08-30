package services

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/google/uuid"
)

// NewWorkerPool creates a new worker pool for webhook processing
func NewWorkerPool(service *EnhancedWebhookService, workerCount int) *WorkerPool {
	ctx, cancel := context.WithCancel(context.Background())
	
	pool := &WorkerPool{
		workerCount: workerCount,
		jobChan:     make(chan *WebhookJob, workerCount*10), // Buffered channel
		workers:     make([]*WebhookWorker, workerCount),
		ctx:         ctx,
		cancel:      cancel,
	}
	
	// Create workers
	for i := 0; i < workerCount; i++ {
		worker := &WebhookWorker{
			id:      i,
			service: service,
			jobChan: pool.jobChan,
			quit:    make(chan bool),
		}
		pool.workers[i] = worker
	}
	
	return pool
}

// Start starts all workers in the pool
func (wp *WorkerPool) Start() {
	log.Printf("Starting webhook worker pool with %d workers", wp.workerCount)
	
	for _, worker := range wp.workers {
		wp.wg.Add(1)
		go worker.Start(&wp.wg)
	}
}

// Stop stops all workers in the pool
func (wp *WorkerPool) Stop() {
	log.Println("Stopping webhook worker pool...")
	
	// Stop accepting new jobs
	wp.cancel()
	close(wp.jobChan)
	
	// Signal all workers to stop
	for _, worker := range wp.workers {
		close(worker.quit)
	}
	
	// Wait for all workers to finish
	wp.wg.Wait()
	log.Println("Webhook worker pool stopped")
}

// AddJob adds a job to the worker pool
func (wp *WorkerPool) AddJob(job *WebhookJob) {
	select {
	case wp.jobChan <- job:
		// Job added successfully
	case <-wp.ctx.Done():
		// Worker pool is shutting down
		log.Printf("Cannot add job %s: worker pool is shutting down", job.ID)
	default:
		// Channel is full, handle overflow
		log.Printf("Worker pool job queue is full, dropping job %s", job.ID)
		// In production, you might want to implement a overflow strategy
		// like storing jobs in Redis or database
	}
}

// Start starts the webhook worker
func (w *WebhookWorker) Start(wg *sync.WaitGroup) {
	defer wg.Done()
	log.Printf("Starting webhook worker %d", w.id)
	
	for {
		select {
		case job := <-w.jobChan:
			if job != nil {
				w.processJob(job)
			}
		case <-w.quit:
			log.Printf("Stopping webhook worker %d", w.id)
			return
		}
	}
}

// processJob processes a single webhook job
func (w *WebhookWorker) processJob(job *WebhookJob) {
	log.Printf("Worker %d processing job %s for form %s", w.id, job.ID, job.FormID)
	
	processedAt := time.Now()
	job.ProcessedAt = &processedAt
	
	// Process each endpoint in the job
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 5) // Limit concurrent requests per job
	
	for _, endpoint := range job.Endpoints {
		wg.Add(1)
		go func(ep WebhookEndpoint) {
			defer wg.Done()
			
			// Acquire semaphore
			semaphore <- struct{}{}
			defer func() { <-semaphore }()
			
			// Send webhook to this endpoint
			result := w.service.sendSingleWebhook(&ep, job.Event)
			
			// Record result
			w.service.analytics.RecordWebhookResult(job.FormID, ep.ID, result)
			
			// Update circuit breaker
			if result.Success {
				w.service.circuitBreaker.RecordSuccess(ep.ID)
			} else {
				w.service.circuitBreaker.RecordFailure(ep.ID)
			}
			
		}(endpoint)
	}
	
	// Wait for all endpoints to complete
	wg.Wait()
	
	completedAt := time.Now()
	job.CompletedAt = &completedAt
	
	log.Printf("Worker %d completed job %s in %v", w.id, job.ID, completedAt.Sub(processedAt))
}

// sendSingleWebhook sends a webhook to a single endpoint
func (ews *EnhancedWebhookService) sendSingleWebhook(endpoint *WebhookEndpoint, event *EnhancedWebhookEvent) *WebhookResult {
	startTime := time.Now()
	
	result := &WebhookResult{
		EndpointID:   endpoint.ID,
		URL:          endpoint.URL,
		StartTime:    startTime,
		Success:      false,
	}
	
	// Check if endpoint is in circuit breaker open state
	if ews.circuitBreaker.IsOpen(endpoint.ID) {
		result.Error = "Circuit breaker is open"
		result.ResponseTime = time.Since(startTime)
		return result
	}
	
	// Check rate limiting
	if endpoint.RateLimitEnabled && ews.isRateLimited(endpoint.URL) {
		result.Error = "Rate limit exceeded"
		result.ResponseTime = time.Since(startTime)
		ews.analytics.RecordRateLimit(event.FormID, endpoint.ID)
		return result
	}
	
	// Transform payload if needed
	payload, err := ews.prepareWebhookPayload(endpoint, event)
	if err != nil {
		result.Error = fmt.Sprintf("Failed to prepare payload: %v", err)
		result.ResponseTime = time.Since(startTime)
		return result
	}
	
	// Attempt delivery with retries
	maxAttempts := endpoint.MaxRetries + 1 // +1 for initial attempt
	var lastErr error
	
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		if attempt > 1 {
			// Calculate exponential backoff delay
			delay := time.Duration(math.Pow(2, float64(attempt-2))) * time.Duration(endpoint.RetryDelay) * time.Second
			if delay > ews.maxRetryDelay {
				delay = ews.maxRetryDelay
			}
			time.Sleep(delay)
		}
		
		attemptResult := ews.sendWebhookHTTPRequest(endpoint, payload, attempt)
		
		if attemptResult.Success {
			result.Success = true
			result.StatusCode = attemptResult.StatusCode
			result.ResponseBody = attemptResult.ResponseBody
			result.ResponseHeaders = attemptResult.ResponseHeaders
			result.ResponseTime = time.Since(startTime)
			result.Attempts = attempt
			break
		} else {
			lastErr = fmt.Errorf(attemptResult.Error)
			result.StatusCode = attemptResult.StatusCode
			result.ResponseBody = attemptResult.ResponseBody
			
			// Don't retry on certain status codes
			if ews.shouldNotRetry(attemptResult.StatusCode) {
				break
			}
		}
	}
	
	if !result.Success {
		result.Error = lastErr.Error()
	}
	
	result.ResponseTime = time.Since(startTime)
	result.Attempts = maxAttempts
	
	// Log the webhook attempt
	ews.logWebhookAttempt(result, event)
	
	return result
}

// prepareWebhookPayload prepares the payload for a webhook endpoint
func (ews *EnhancedWebhookService) prepareWebhookPayload(endpoint *WebhookEndpoint, event *EnhancedWebhookEvent) ([]byte, error) {
	var payload interface{}
	
	// Apply data transformations
	if endpoint.TransformConfig != nil {
		transformedEvent, err := ews.applyTransformations(event, endpoint.TransformConfig)
		if err != nil {
			return nil, fmt.Errorf("transformation failed: %w", err)
		}
		payload = transformedEvent
	} else if endpoint.CustomPayload != "" {
		// Use custom payload template
		templatedPayload, err := ews.processPayloadTemplate(endpoint.CustomPayload, event)
		if err != nil {
			return nil, fmt.Errorf("template processing failed: %w", err)
		}
		payload = templatedPayload
	} else {
		// Use default payload (the event itself)
		payload = event
	}
	
	// Convert to appropriate format
	var payloadBytes []byte
	var err error
	
	switch strings.ToLower(endpoint.ContentType) {
	case "application/json", "":
		payloadBytes, err = json.Marshal(payload)
	case "application/x-www-form-urlencoded":
		payloadBytes, err = ews.marshalFormData(payload)
	case "application/xml":
		payloadBytes, err = ews.marshalXML(payload)
	case "application/yaml":
		payloadBytes, err = ews.marshalYAML(payload)
	default:
		payloadBytes, err = json.Marshal(payload)
	}
	
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}
	
	// Check payload size
	if int64(len(payloadBytes)) > ews.maxPayloadSize {
		return nil, fmt.Errorf("payload size %d exceeds limit %d", len(payloadBytes), ews.maxPayloadSize)
	}
	
	return payloadBytes, nil
}

// applyTransformations applies field mappings and data filters
func (ews *EnhancedWebhookService) applyTransformations(event *EnhancedWebhookEvent, config *TransformConfig) (map[string]interface{}, error) {
	result := make(map[string]interface{})
	
	// Start with event data
	eventData := map[string]interface{}{
		"id":              event.ID,
		"type":            event.Type,
		"timestamp":       event.Timestamp,
		"form_id":         event.FormID,
		"submission_id":   event.SubmissionID,
		"user_id":         event.UserID,
		"source":          event.Source,
		"version":         event.Version,
		"event_sequence":  event.EventSequence,
		"correlation_id":  event.CorrelationID,
		"environment":     event.Environment,
		"ip_address":      event.IPAddress,
		"user_agent":      event.UserAgent,
	}
	
	// Add form data
	for key, value := range event.Data {
		eventData[key] = value
	}
	
	// Add metadata
	if event.Metadata != nil {
		for key, value := range event.Metadata {
			eventData[fmt.Sprintf("meta_%s", key)] = value
		}
	}
	
	// Apply field mappings
	if len(config.FieldMappings) > 0 {
		for originalField, mappedField := range config.FieldMappings {
			if value, exists := eventData[originalField]; exists {
				result[mappedField] = value
			}
		}
	} else {
		// No mapping, copy all fields
		result = eventData
	}
	
	// Apply data filters
	for _, filter := range config.DataFilters {
		if err := ews.applyDataFilter(&result, filter); err != nil {
			return nil, fmt.Errorf("filter error: %w", err)
		}
	}
	
	return result, nil
}

// applyDataFilter applies a single data filter
func (ews *EnhancedWebhookService) applyDataFilter(data *map[string]interface{}, filter DataFilter) error {
	switch filter.Type {
	case "include":
		// Only keep specified fields
		if fields, ok := filter.Value.([]interface{}); ok {
			newData := make(map[string]interface{})
			for _, field := range fields {
				if fieldStr, ok := field.(string); ok {
					if value, exists := (*data)[fieldStr]; exists {
						newData[fieldStr] = value
					}
				}
			}
			*data = newData
		}
		
	case "exclude":
		// Remove specified fields
		if fields, ok := filter.Value.([]interface{}); ok {
			for _, field := range fields {
				if fieldStr, ok := field.(string); ok {
					delete(*data, fieldStr)
				}
			}
		}
		
	case "transform":
		// Transform field value
		if value, exists := (*data)[filter.Field]; exists {
			switch filter.Action {
			case "uppercase":
				if str, ok := value.(string); ok {
					(*data)[filter.Field] = strings.ToUpper(str)
				}
			case "lowercase":
				if str, ok := value.(string); ok {
					(*data)[filter.Field] = strings.ToLower(str)
				}
			case "trim":
				if str, ok := value.(string); ok {
					(*data)[filter.Field] = strings.TrimSpace(str)
				}
			case "replace":
				if str, ok := value.(string); ok {
					if replaceConfig, ok := filter.Value.(map[string]interface{}); ok {
						if from, ok := replaceConfig["from"].(string); ok {
							if to, ok := replaceConfig["to"].(string); ok {
								(*data)[filter.Field] = strings.ReplaceAll(str, from, to)
							}
						}
					}
				}
			}
		}
		
	case "validate":
		// Validate field value
		if value, exists := (*data)[filter.Field]; exists {
			if !ews.validateFieldValue(value, filter) {
				return fmt.Errorf("validation failed for field %s", filter.Field)
			}
		}
	}
	
	return nil
}

// validateFieldValue validates a field value against filter rules
func (ews *EnhancedWebhookService) validateFieldValue(value interface{}, filter DataFilter) bool {
	if validationRules, ok := filter.Value.(map[string]interface{}); ok {
		// Check required
		if required, ok := validationRules["required"].(bool); ok && required {
			if value == nil || value == "" {
				return false
			}
		}
		
		// Check type
		if expectedType, ok := validationRules["type"].(string); ok {
			if !ews.checkValueType(value, expectedType) {
				return false
			}
		}
		
		// Check string length
		if str, ok := value.(string); ok {
			if minLength, ok := validationRules["min_length"].(float64); ok {
				if len(str) < int(minLength) {
					return false
				}
			}
			if maxLength, ok := validationRules["max_length"].(float64); ok {
				if len(str) > int(maxLength) {
					return false
				}
			}
		}
		
		// Check numeric range
		if num, ok := ews.toFloat64(value); ok {
			if min, ok := validationRules["min"].(float64); ok {
				if num < min {
					return false
				}
			}
			if max, ok := validationRules["max"].(float64); ok {
				if num > max {
					return false
				}
			}
		}
	}
	
	return true
}

// checkValueType checks if a value matches expected type
func (ews *EnhancedWebhookService) checkValueType(value interface{}, expectedType string) bool {
	switch expectedType {
	case "string":
		_, ok := value.(string)
		return ok
	case "number":
		_, ok := ews.toFloat64(value)
		return ok
	case "boolean":
		_, ok := value.(bool)
		return ok
	case "array":
		_, ok := value.([]interface{})
		return ok
	case "object":
		_, ok := value.(map[string]interface{})
		return ok
	default:
		return true // Unknown type, assume valid
	}
}

// processPayloadTemplate processes a payload template
func (ews *EnhancedWebhookService) processPayloadTemplate(templateStr string, event *EnhancedWebhookEvent) (interface{}, error) {
	// Create template with custom functions
	tmpl, err := template.New("webhook").Funcs(template.FuncMap{
		"now":       time.Now,
		"formatTime": func(t time.Time, format string) string { return t.Format(format) },
		"upper":     strings.ToUpper,
		"lower":     strings.ToLower,
		"trim":      strings.TrimSpace,
	}).Parse(templateStr)
	
	if err != nil {
		return nil, fmt.Errorf("template parse error: %w", err)
	}
	
	// Execute template
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, event); err != nil {
		return nil, fmt.Errorf("template execution error: %w", err)
	}
	
	// Parse result as JSON
	var result interface{}
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		// If JSON parsing fails, return as string
		return buf.String(), nil
	}
	
	return result, nil
}

// sendWebhookHTTPRequest sends HTTP request to webhook endpoint
func (ews *EnhancedWebhookService) sendWebhookHTTPRequest(endpoint *WebhookEndpoint, payload []byte, attempt int) *WebhookAttemptResult {
	result := &WebhookAttemptResult{
		Attempt: attempt,
		Success: false,
	}
	
	// Determine HTTP method
	method := strings.ToUpper(endpoint.Method)
	if method == "" {
		method = "POST"
	}
	
	// Create request
	req, err := http.NewRequest(method, endpoint.URL, bytes.NewBuffer(payload))
	if err != nil {
		result.Error = fmt.Sprintf("failed to create request: %v", err)
		return result
	}
	
	// Set headers
	req.Header.Set("Content-Type", ews.getContentType(endpoint))
	req.Header.Set("User-Agent", ews.userAgent)
	req.Header.Set(ews.timestampHeader, strconv.FormatInt(time.Now().Unix(), 10))
	req.Header.Set("X-FormHub-Attempt", strconv.Itoa(attempt))
	req.Header.Set("X-FormHub-Endpoint-ID", endpoint.ID)
	
	// Set custom headers
	for key, value := range endpoint.Headers {
		req.Header.Set(key, value)
	}
	
	// Add signature if secret is configured
	if endpoint.Secret != "" {
		signature := ews.generateSignature(payload, endpoint.Secret)
		req.Header.Set(ews.signatureHeader, signature)
	}
	
	// Set timeout
	ctx := ews.ctx
	if endpoint.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Duration(endpoint.Timeout)*time.Second)
		defer cancel()
	}
	req = req.WithContext(ctx)
	
	// Choose appropriate HTTP client
	client := ews.httpClient
	if endpoint.VerifySSL {
		client = ews.secureClient
	}
	
	// Send request
	startTime := time.Now()
	resp, err := client.Do(req)
	responseTime := time.Since(startTime)
	
	result.ResponseTime = responseTime
	
	if err != nil {
		result.Error = fmt.Sprintf("request failed: %v", err)
		return result
	}
	defer resp.Body.Close()
	
	result.StatusCode = resp.StatusCode
	result.ResponseHeaders = make(map[string]string)
	for key, values := range resp.Header {
		if len(values) > 0 {
			result.ResponseHeaders[key] = values[0]
		}
	}
	
	// Read response body with limit
	bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, 10240)) // 10KB limit
	if err != nil {
		result.Error = fmt.Sprintf("failed to read response: %v", err)
	} else {
		result.ResponseBody = string(bodyBytes)
	}
	
	// Check if request was successful
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		result.Success = true
	} else {
		result.Error = fmt.Sprintf("HTTP %d: %s", resp.StatusCode, result.ResponseBody)
	}
	
	return result
}

// Helper methods

func (ews *EnhancedWebhookService) getContentType(endpoint *WebhookEndpoint) string {
	if endpoint.ContentType != "" {
		return endpoint.ContentType
	}
	return "application/json"
}

func (ews *EnhancedWebhookService) generateSignature(payload []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

func (ews *EnhancedWebhookService) shouldNotRetry(statusCode int) bool {
	// Don't retry on client errors (4xx) except for some specific cases
	if statusCode >= 400 && statusCode < 500 {
		switch statusCode {
		case 408, 429: // Request Timeout, Too Many Requests
			return false // These can be retried
		default:
			return true // Other 4xx errors should not be retried
		}
	}
	return false
}

func (ews *EnhancedWebhookService) isRateLimited(webhookURL string) bool {
	key := fmt.Sprintf("webhook_rate_limit:%s", webhookURL)
	
	// Use Redis sliding window rate limiting
	now := time.Now()
	windowStart := now.Add(-ews.rateLimitWindow)
	
	pipe := ews.redis.Pipeline()
	
	// Remove old entries
	pipe.ZRemRangeByScore(ews.ctx, key, "0", fmt.Sprintf("%d", windowStart.Unix()))
	
	// Count current requests
	countCmd := pipe.ZCount(ews.ctx, key, fmt.Sprintf("%d", windowStart.Unix()), fmt.Sprintf("%d", now.Unix()))
	
	// Add current request
	pipe.ZAdd(ews.ctx, key, redis.Z{Score: float64(now.Unix()), Member: uuid.New().String()})
	
	// Set expiration
	pipe.Expire(ews.ctx, key, ews.rateLimitWindow)
	
	_, err := pipe.Exec(ews.ctx)
	if err != nil {
		log.Printf("Rate limiting check failed: %v", err)
		return false // Allow request on Redis errors
	}
	
	count, err := countCmd.Result()
	if err != nil {
		log.Printf("Rate limiting count failed: %v", err)
		return false
	}
	
	return count >= int64(ews.maxWebhooksPerMin)
}

func (ews *EnhancedWebhookService) logWebhookAttempt(result *WebhookResult, event *EnhancedWebhookEvent) {
	logData := map[string]interface{}{
		"webhook_id":     uuid.New().String(),
		"endpoint_id":    result.EndpointID,
		"url":           result.URL,
		"event_type":    event.Type,
		"form_id":       event.FormID,
		"submission_id": event.SubmissionID,
		"status_code":   result.StatusCode,
		"response_time": result.ResponseTime.Milliseconds(),
		"attempts":      result.Attempts,
		"success":       result.Success,
		"timestamp":     time.Now(),
	}
	
	if result.Error != "" {
		logData["error"] = result.Error
	}
	
	// Store in webhook_logs table
	query := `
		INSERT INTO webhook_logs 
		(id, endpoint_id, form_id, event_type, url, status_code, response_time_ms, attempts, success, error_message, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	
	_, err := ews.db.Exec(query,
		uuid.New().String(),
		result.EndpointID,
		event.FormID,
		event.Type,
		result.URL,
		result.StatusCode,
		result.ResponseTime.Milliseconds(),
		result.Attempts,
		result.Success,
		result.Error,
		time.Now(),
	)
	
	if err != nil {
		log.Printf("Failed to log webhook attempt: %v", err)
	}
}

// Data marshalling helpers

func (ews *EnhancedWebhookService) marshalFormData(payload interface{}) ([]byte, error) {
	data, ok := payload.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("payload must be a map for form data encoding")
	}
	
	values := make([]string, 0)
	for key, value := range data {
		values = append(values, fmt.Sprintf("%s=%s", key, fmt.Sprintf("%v", value)))
	}
	
	return []byte(strings.Join(values, "&")), nil
}

func (ews *EnhancedWebhookService) marshalXML(payload interface{}) ([]byte, error) {
	// Simple XML marshalling - for production use encoding/xml
	data, ok := payload.(map[string]interface{})
	if !ok {
		return json.Marshal(payload) // Fallback to JSON
	}
	
	var buf bytes.Buffer
	buf.WriteString("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n<webhook>\n")
	
	for key, value := range data {
		buf.WriteString(fmt.Sprintf("  <%s>%v</%s>\n", key, value, key))
	}
	
	buf.WriteString("</webhook>")
	return buf.Bytes(), nil
}

func (ews *EnhancedWebhookService) marshalYAML(payload interface{}) ([]byte, error) {
	return yaml.Marshal(payload)
}

// Result types

// WebhookResult represents the result of a webhook delivery
type WebhookResult struct {
	EndpointID      string            `json:"endpoint_id"`
	URL             string            `json:"url"`
	Success         bool              `json:"success"`
	StatusCode      int               `json:"status_code"`
	ResponseBody    string            `json:"response_body"`
	ResponseHeaders map[string]string `json:"response_headers"`
	ResponseTime    time.Duration     `json:"response_time"`
	Attempts        int               `json:"attempts"`
	Error           string            `json:"error,omitempty"`
	StartTime       time.Time         `json:"start_time"`
}

// WebhookAttemptResult represents a single attempt result
type WebhookAttemptResult struct {
	Attempt         int               `json:"attempt"`
	Success         bool              `json:"success"`
	StatusCode      int               `json:"status_code"`
	ResponseBody    string            `json:"response_body"`
	ResponseHeaders map[string]string `json:"response_headers"`
	ResponseTime    time.Duration     `json:"response_time"`
	Error           string            `json:"error,omitempty"`
}