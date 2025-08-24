package utils

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type RecaptchaResponse struct {
	Success     bool      `json:"success"`
	Score       float64   `json:"score"`
	Action      string    `json:"action"`
	ChallengeTS time.Time `json:"challenge_ts"`
	Hostname    string    `json:"hostname"`
	ErrorCodes  []string  `json:"error-codes"`
}

func VerifyRecaptcha(response, secret string) (*RecaptchaResponse, error) {
	if response == "" || secret == "" {
		return nil, fmt.Errorf("recaptcha response or secret is empty")
	}

	// Prepare the request
	data := url.Values{}
	data.Set("secret", secret)
	data.Set("response", response)

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	// Make request to Google's recaptcha API
	resp, err := client.Post("https://www.google.com/recaptcha/api/siteverify",
		"application/x-www-form-urlencoded",
		strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to verify recaptcha: %w", err)
	}
	defer resp.Body.Close()

	var result RecaptchaResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode recaptcha response: %w", err)
	}

	return &result, nil
}

func IsRecaptchaValid(response *RecaptchaResponse, minScore float64) bool {
	if response == nil {
		return false
	}
	
	// Check if verification was successful
	if !response.Success {
		return false
	}
	
	// For reCAPTCHA v3, check the score (0.0 to 1.0, where 1.0 is very likely human)
	if response.Score > 0 && response.Score < minScore {
		return false
	}
	
	return true
}