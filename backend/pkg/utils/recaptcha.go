package utils

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// CaptchaProvider represents different CAPTCHA service providers
type CaptchaProvider string

const (
	ProviderRecaptchaV2  CaptchaProvider = "recaptcha_v2"
	ProviderRecaptchaV3  CaptchaProvider = "recaptcha_v3"
	ProviderHCaptcha     CaptchaProvider = "hcaptcha"
	ProviderTurnstile    CaptchaProvider = "turnstile"
	ProviderFallback     CaptchaProvider = "fallback"
)

// CaptchaConfig holds configuration for all CAPTCHA providers
type CaptchaConfig struct {
	RecaptchaV2Secret  string  `json:"recaptcha_v2_secret"`
	RecaptchaV3Secret  string  `json:"recaptcha_v3_secret"`
	RecaptchaV3MinScore float64 `json:"recaptcha_v3_min_score"`
	HCaptchaSecret     string  `json:"hcaptcha_secret"`
	TurnstileSecret    string  `json:"turnstile_secret"`
	FallbackEnabled    bool    `json:"fallback_enabled"`
	Timeout            int     `json:"timeout"` // seconds
}

// CaptchaResponse represents the unified response from any CAPTCHA provider
type CaptchaResponse struct {
	Success     bool                   `json:"success"`
	Score       float64               `json:"score,omitempty"`
	Action      string                `json:"action,omitempty"`
	ChallengeTS time.Time             `json:"challenge_ts,omitempty"`
	Hostname    string                `json:"hostname,omitempty"`
	ErrorCodes  []string              `json:"error-codes,omitempty"`
	Provider    CaptchaProvider       `json:"provider"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// RecaptchaResponse represents Google reCAPTCHA API response
type RecaptchaResponse struct {
	Success     bool      `json:"success"`
	Score       float64   `json:"score"`
	Action      string    `json:"action"`
	ChallengeTS time.Time `json:"challenge_ts"`
	Hostname    string    `json:"hostname"`
	ErrorCodes  []string  `json:"error-codes"`
}

// HCaptchaResponse represents hCaptcha API response
type HCaptchaResponse struct {
	Success     bool     `json:"success"`
	ChallengeTS string   `json:"challenge_ts"`
	Hostname    string   `json:"hostname"`
	ErrorCodes  []string `json:"error-codes"`
	Credit      bool     `json:"credit,omitempty"`
}

// TurnstileResponse represents Cloudflare Turnstile API response
type TurnstileResponse struct {
	Success     bool     `json:"success"`
	ChallengeTS string   `json:"challenge_ts"`
	Hostname    string   `json:"hostname"`
	ErrorCodes  []string `json:"error-codes"`
	Action      string   `json:"action,omitempty"`
	CData       string   `json:"cdata,omitempty"`
}

// FallbackCaptcha represents our simple fallback CAPTCHA
type FallbackCaptcha struct {
	Question string `json:"question"`
	Answer   string `json:"answer"`
	Token    string `json:"token"`
}

// CaptchaService provides unified CAPTCHA verification across multiple providers
type CaptchaService struct {
	config *CaptchaConfig
	client *http.Client
}

// NewCaptchaService creates a new CAPTCHA service with the provided configuration
func NewCaptchaService(config *CaptchaConfig) *CaptchaService {
	timeout := 10 * time.Second
	if config.Timeout > 0 {
		timeout = time.Duration(config.Timeout) * time.Second
	}
	
	return &CaptchaService{
		config: config,
		client: &http.Client{
			Timeout: timeout,
		},
	}
}

// VerifyCaptcha verifies a CAPTCHA response using the specified provider
func (cs *CaptchaService) VerifyCaptcha(provider CaptchaProvider, token string, remoteIP string) (*CaptchaResponse, error) {
	switch provider {
	case ProviderRecaptchaV2:
		return cs.verifyRecaptchaV2(token, remoteIP)
	case ProviderRecaptchaV3:
		return cs.verifyRecaptchaV3(token, remoteIP)
	case ProviderHCaptcha:
		return cs.verifyHCaptcha(token, remoteIP)
	case ProviderTurnstile:
		return cs.verifyTurnstile(token, remoteIP)
	case ProviderFallback:
		return cs.verifyFallbackCaptcha(token, remoteIP)
	default:
		return nil, fmt.Errorf("unsupported CAPTCHA provider: %s", provider)
	}
}

// VerifyWithFallback attempts to verify using the primary provider, falling back to others if needed
func (cs *CaptchaService) VerifyWithFallback(primaryProvider CaptchaProvider, token string, remoteIP string) (*CaptchaResponse, error) {
	// Try primary provider first
	response, err := cs.VerifyCaptcha(primaryProvider, token, remoteIP)
	if err == nil && response.Success {
		return response, nil
	}
	
	// If fallback is enabled, try other providers
	if cs.config.FallbackEnabled {
		providers := []CaptchaProvider{ProviderRecaptchaV3, ProviderRecaptchaV2, ProviderHCaptcha, ProviderTurnstile}
		
		for _, provider := range providers {
			if provider == primaryProvider {
				continue // Skip the primary provider we already tried
			}
			
			if fallbackResponse, fallbackErr := cs.VerifyCaptcha(provider, token, remoteIP); fallbackErr == nil && fallbackResponse.Success {
				fallbackResponse.Metadata = map[string]interface{}{
					"fallback_used": true,
					"primary_provider": primaryProvider,
					"primary_error": err.Error(),
				}
				return fallbackResponse, nil
			}
		}
	}
	
	// Return the original error if no fallback worked
	return response, err
}

// verifyRecaptchaV2 verifies Google reCAPTCHA v2 response
func (cs *CaptchaService) verifyRecaptchaV2(token, remoteIP string) (*CaptchaResponse, error) {
	if cs.config.RecaptchaV2Secret == "" {
		return nil, fmt.Errorf("reCAPTCHA v2 secret not configured")
	}
	
	data := url.Values{}
	data.Set("secret", cs.config.RecaptchaV2Secret)
	data.Set("response", token)
	data.Set("remoteip", remoteIP)
	
	resp, err := cs.client.Post("https://www.google.com/recaptcha/api/siteverify",
		"application/x-www-form-urlencoded",
		strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to verify reCAPTCHA v2: %w", err)
	}
	defer resp.Body.Close()
	
	var result RecaptchaResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode reCAPTCHA v2 response: %w", err)
	}
	
	return &CaptchaResponse{
		Success:     result.Success,
		ChallengeTS: result.ChallengeTS,
		Hostname:    result.Hostname,
		ErrorCodes:  result.ErrorCodes,
		Provider:    ProviderRecaptchaV2,
	}, nil
}

// verifyRecaptchaV3 verifies Google reCAPTCHA v3 response
func (cs *CaptchaService) verifyRecaptchaV3(token, remoteIP string) (*CaptchaResponse, error) {
	if cs.config.RecaptchaV3Secret == "" {
		return nil, fmt.Errorf("reCAPTCHA v3 secret not configured")
	}
	
	data := url.Values{}
	data.Set("secret", cs.config.RecaptchaV3Secret)
	data.Set("response", token)
	data.Set("remoteip", remoteIP)
	
	resp, err := cs.client.Post("https://www.google.com/recaptcha/api/siteverify",
		"application/x-www-form-urlencoded",
		strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to verify reCAPTCHA v3: %w", err)
	}
	defer resp.Body.Close()
	
	var result RecaptchaResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode reCAPTCHA v3 response: %w", err)
	}
	
	// Check minimum score for v3
	minScore := cs.config.RecaptchaV3MinScore
	if minScore == 0 {
		minScore = 0.5 // Default threshold
	}
	
	success := result.Success && result.Score >= minScore
	
	return &CaptchaResponse{
		Success:     success,
		Score:       result.Score,
		Action:      result.Action,
		ChallengeTS: result.ChallengeTS,
		Hostname:    result.Hostname,
		ErrorCodes:  result.ErrorCodes,
		Provider:    ProviderRecaptchaV3,
		Metadata: map[string]interface{}{
			"min_score_required": minScore,
		},
	}, nil
}

// verifyHCaptcha verifies hCaptcha response
func (cs *CaptchaService) verifyHCaptcha(token, remoteIP string) (*CaptchaResponse, error) {
	if cs.config.HCaptchaSecret == "" {
		return nil, fmt.Errorf("hCaptcha secret not configured")
	}
	
	data := url.Values{}
	data.Set("secret", cs.config.HCaptchaSecret)
	data.Set("response", token)
	data.Set("remoteip", remoteIP)
	
	resp, err := cs.client.Post("https://hcaptcha.com/siteverify",
		"application/x-www-form-urlencoded",
		strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to verify hCaptcha: %w", err)
	}
	defer resp.Body.Close()
	
	var result HCaptchaResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode hCaptcha response: %w", err)
	}
	
	// Parse challenge timestamp
	var challengeTS time.Time
	if result.ChallengeTS != "" {
		challengeTS, _ = time.Parse(time.RFC3339, result.ChallengeTS)
	}
	
	return &CaptchaResponse{
		Success:     result.Success,
		ChallengeTS: challengeTS,
		Hostname:    result.Hostname,
		ErrorCodes:  result.ErrorCodes,
		Provider:    ProviderHCaptcha,
		Metadata: map[string]interface{}{
			"credit": result.Credit,
		},
	}, nil
}

// verifyTurnstile verifies Cloudflare Turnstile response
func (cs *CaptchaService) verifyTurnstile(token, remoteIP string) (*CaptchaResponse, error) {
	if cs.config.TurnstileSecret == "" {
		return nil, fmt.Errorf("Turnstile secret not configured")
	}
	
	data := url.Values{}
	data.Set("secret", cs.config.TurnstileSecret)
	data.Set("response", token)
	data.Set("remoteip", remoteIP)
	
	resp, err := cs.client.Post("https://challenges.cloudflare.com/turnstile/v0/siteverify",
		"application/x-www-form-urlencoded",
		strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to verify Turnstile: %w", err)
	}
	defer resp.Body.Close()
	
	var result TurnstileResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode Turnstile response: %w", err)
	}
	
	// Parse challenge timestamp
	var challengeTS time.Time
	if result.ChallengeTS != "" {
		challengeTS, _ = time.Parse(time.RFC3339, result.ChallengeTS)
	}
	
	return &CaptchaResponse{
		Success:     result.Success,
		ChallengeTS: challengeTS,
		Hostname:    result.Hostname,
		ErrorCodes:  result.ErrorCodes,
		Action:      result.Action,
		Provider:    ProviderTurnstile,
		Metadata: map[string]interface{}{
			"cdata": result.CData,
		},
	}, nil
}

// verifyFallbackCaptcha verifies our simple fallback CAPTCHA
func (cs *CaptchaService) verifyFallbackCaptcha(token, remoteIP string) (*CaptchaResponse, error) {
	// Parse the token which should contain the answer and validation hash
	parts := strings.Split(token, ":")
	if len(parts) != 3 {
		return &CaptchaResponse{
			Success:  false,
			Provider: ProviderFallback,
			ErrorCodes: []string{"invalid-token-format"},
		}, nil
	}
	
	answer, timestamp, hash := parts[0], parts[1], parts[2]
	
	// Verify the token hasn't expired (5 minutes limit)
	ts, err := time.Parse(time.RFC3339, timestamp)
	if err != nil || time.Since(ts) > 5*time.Minute {
		return &CaptchaResponse{
			Success:  false,
			Provider: ProviderFallback,
			ErrorCodes: []string{"token-expired"},
		}, nil
	}
	
	// Verify the hash
	expectedHash := cs.generateFallbackHash(answer, timestamp, remoteIP)
	if hash != expectedHash {
		return &CaptchaResponse{
			Success:  false,
			Provider: ProviderFallback,
			ErrorCodes: []string{"invalid-token"},
		}, nil
	}
	
	return &CaptchaResponse{
		Success:     true,
		ChallengeTS: ts,
		Provider:    ProviderFallback,
	}, nil
}

// GenerateFallbackCaptcha generates a simple math-based CAPTCHA
func (cs *CaptchaService) GenerateFallbackCaptcha(remoteIP string) (*FallbackCaptcha, error) {
	// Generate a simple math question
	a := rand.Intn(10) + 1
	b := rand.Intn(10) + 1
	operation := []string{"+", "-", "×"}[rand.Intn(3)]
	
	var question string
	var answer int
	
	switch operation {
	case "+":
		question = fmt.Sprintf("What is %d + %d?", a, b)
		answer = a + b
	case "-":
		if a < b {
			a, b = b, a // Ensure positive result
		}
		question = fmt.Sprintf("What is %d - %d?", a, b)
		answer = a - b
	case "×":
		question = fmt.Sprintf("What is %d × %d?", a, b)
		answer = a * b
	}
	
	timestamp := time.Now().Format(time.RFC3339)
	answerStr := fmt.Sprintf("%d", answer)
	hash := cs.generateFallbackHash(answerStr, timestamp, remoteIP)
	token := fmt.Sprintf("%s:%s:%s", answerStr, timestamp, hash)
	
	return &FallbackCaptcha{
		Question: question,
		Answer:   answerStr,
		Token:    token,
	}, nil
}

// generateFallbackHash generates a hash for fallback CAPTCHA validation
func (cs *CaptchaService) generateFallbackHash(answer, timestamp, remoteIP string) string {
	// Use a secret key for HMAC (in production, this should be from config)
	secretKey := "fallback-captcha-secret-key"
	
	h := hmac.New(sha256.New, []byte(secretKey))
	h.Write([]byte(answer + timestamp + remoteIP))
	return hex.EncodeToString(h.Sum(nil))[:16] // Use first 16 characters
}

// Legacy functions for backward compatibility

// VerifyRecaptcha verifies a reCAPTCHA v2/v3 response (legacy function)
func VerifyRecaptcha(response, secret string) (*RecaptchaResponse, error) {
	if response == "" || secret == "" {
		return nil, fmt.Errorf("recaptcha response or secret is empty")
	}

	data := url.Values{}
	data.Set("secret", secret)
	data.Set("response", response)

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

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

// IsRecaptchaValid checks if a reCAPTCHA response is valid (legacy function)
func IsRecaptchaValid(response *RecaptchaResponse, minScore float64) bool {
	if response == nil {
		return false
	}
	
	if !response.Success {
		return false
	}
	
	if response.Score > 0 && response.Score < minScore {
		return false
	}
	
	return true
}