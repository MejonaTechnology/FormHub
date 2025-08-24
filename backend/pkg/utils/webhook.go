package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type WebhookPayload struct {
	Event       string                 `json:"event"`
	FormID      string                 `json:"form_id"`
	FormName    string                 `json:"form_name"`
	Submission  WebhookSubmission      `json:"submission"`
	Timestamp   string                 `json:"timestamp"`
	UserAgent   string                 `json:"user_agent"`
	IPAddress   string                 `json:"ip_address"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

type WebhookSubmission struct {
	ID        string                 `json:"id"`
	Data      map[string]interface{} `json:"data"`
	IsSpam    bool                   `json:"is_spam"`
	SpamScore float64                `json:"spam_score"`
	CreatedAt string                 `json:"created_at"`
}

type WebhookResponse struct {
	Success     bool   `json:"success"`
	StatusCode  int    `json:"status_code"`
	Response    string `json:"response,omitempty"`
	Error       string `json:"error,omitempty"`
	Duration    string `json:"duration"`
}

func SendWebhook(webhookURL string, payload WebhookPayload) (*WebhookResponse, error) {
	if webhookURL == "" {
		return nil, fmt.Errorf("webhook URL is empty")
	}

	startTime := time.Now()

	// Marshal payload to JSON
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return &WebhookResponse{
			Success:    false,
			StatusCode: 0,
			Error:      fmt.Sprintf("Failed to marshal payload: %v", err),
			Duration:   time.Since(startTime).String(),
		}, err
	}

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// Create request
	req, err := http.NewRequest("POST", webhookURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return &WebhookResponse{
			Success:    false,
			StatusCode: 0,
			Error:      fmt.Sprintf("Failed to create request: %v", err),
			Duration:   time.Since(startTime).String(),
		}, err
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "FormHub/1.0 (+https://formhub.com)")
	req.Header.Set("X-FormHub-Event", payload.Event)
	req.Header.Set("X-FormHub-Delivery", fmt.Sprintf("%d", time.Now().Unix()))

	// Send request
	resp, err := client.Do(req)
	if err != nil {
		return &WebhookResponse{
			Success:    false,
			StatusCode: 0,
			Error:      fmt.Sprintf("Failed to send webhook: %v", err),
			Duration:   time.Since(startTime).String(),
		}, err
	}
	defer resp.Body.Close()

	// Read response body (limit to 1KB to prevent abuse)
	responseBody := make([]byte, 1024)
	n, _ := resp.Body.Read(responseBody)
	responseText := string(responseBody[:n])

	duration := time.Since(startTime)

	// Check if webhook was successful
	success := resp.StatusCode >= 200 && resp.StatusCode < 300

	return &WebhookResponse{
		Success:    success,
		StatusCode: resp.StatusCode,
		Response:   responseText,
		Duration:   duration.String(),
	}, nil
}

func ValidateWebhookURL(url string) error {
	if url == "" {
		return fmt.Errorf("webhook URL is required")
	}

	// Basic URL validation
	req, err := http.NewRequest("HEAD", url, nil)
	if err != nil {
		return fmt.Errorf("invalid webhook URL: %v", err)
	}

	if req.URL.Scheme != "http" && req.URL.Scheme != "https" {
		return fmt.Errorf("webhook URL must use HTTP or HTTPS")
	}

	// For production, you might want to restrict to HTTPS only
	// if req.URL.Scheme != "https" {
	//     return fmt.Errorf("webhook URL must use HTTPS")
	// }

	return nil
}