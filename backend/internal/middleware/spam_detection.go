package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"formhub/internal/services"
	"formhub/pkg/utils"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// SpamDetectionMiddleware provides comprehensive spam protection middleware
type SpamDetectionMiddleware struct {
	spamService       *services.SpamProtectionService
	securityService   *services.SecurityService
	behavioralAnalyzer *services.BehavioralAnalyzer
	mlClassifier      *services.NaiveBayesSpamClassifier
	
	// Configuration
	enableRealTimeAnalysis  bool
	enableBehavioralAnalysis bool
	enableMLClassification   bool
	logAllRequests          bool
	blockingEnabled         bool
	
	// Rate limiting and caching
	ipAnalysisCache      map[string]*IPAnalysisResult
	recentSubmissions    map[string][]SubmissionRecord
	honeypotTracker      map[string]int // IP -> honeypot violation count
	
	// Statistics
	stats *SpamDetectionStats
}

// IPAnalysisResult contains cached IP analysis results
type IPAnalysisResult struct {
	IP             string                 `json:"ip"`
	RiskScore      float64                `json:"risk_score"`
	Reputation     string                 `json:"reputation"` // "good", "neutral", "suspicious", "malicious"
	CountryCode    string                 `json:"country_code"`
	ASN            string                 `json:"asn"`
	IsVPN          bool                   `json:"is_vpn"`
	IsProxy        bool                   `json:"is_proxy"`
	IsTor          bool                   `json:"is_tor"`
	LastSeen       time.Time              `json:"last_seen"`
	SubmissionCount int                   `json:"submission_count"`
	BlockCount     int                    `json:"block_count"`
	Metadata       map[string]interface{} `json:"metadata"`
}

// SubmissionRecord tracks recent submissions from an IP
type SubmissionRecord struct {
	Timestamp   time.Time                  `json:"timestamp"`
	FormID      string                     `json:"form_id"`
	UserAgent   string                     `json:"user_agent"`
	Result      string                     `json:"result"` // "allowed", "blocked", "quarantined"
	SpamScore   float64                    `json:"spam_score"`
	Triggers    []string                   `json:"triggers"`
	Metadata    map[string]interface{}     `json:"metadata"`
}

// SpamDetectionStats tracks middleware statistics
type SpamDetectionStats struct {
	TotalRequests       int64 `json:"total_requests"`
	SpamDetected        int64 `json:"spam_detected"`
	Blocked             int64 `json:"blocked"`
	Quarantined         int64 `json:"quarantined"`
	CaptchaChallenges   int64 `json:"captcha_challenges"`
	FalsePositives      int64 `json:"false_positives"`
	TruePositives       int64 `json:"true_positives"`
	ProcessingTimeMs    int64 `json:"avg_processing_time_ms"`
	LastReset           time.Time `json:"last_reset"`
}

// SpamDetectionConfig holds middleware configuration
type SpamDetectionConfig struct {
	EnableRealTimeAnalysis   bool    `json:"enable_real_time_analysis"`
	EnableBehavioralAnalysis bool    `json:"enable_behavioral_analysis"`
	EnableMLClassification   bool    `json:"enable_ml_classification"`
	LogAllRequests          bool    `json:"log_all_requests"`
	BlockingEnabled         bool    `json:"blocking_enabled"`
	MaxSubmissionsPerIP     int     `json:"max_submissions_per_ip"`
	RateLimitWindowMinutes  int     `json:"rate_limit_window_minutes"`
	IPCacheExpiryMinutes    int     `json:"ip_cache_expiry_minutes"`
	AutoBlockThreshold      float64 `json:"auto_block_threshold"`
	QuarantineThreshold     float64 `json:"quarantine_threshold"`
	ChallengeThreshold      float64 `json:"challenge_threshold"`
}

// NewSpamDetectionMiddleware creates a new spam detection middleware
func NewSpamDetectionMiddleware(spamService *services.SpamProtectionService,
	securityService *services.SecurityService,
	behavioralAnalyzer *services.BehavioralAnalyzer,
	mlClassifier *services.NaiveBayesSpamClassifier) *SpamDetectionMiddleware {
	
	return &SpamDetectionMiddleware{
		spamService:              spamService,
		securityService:          securityService,
		behavioralAnalyzer:       behavioralAnalyzer,
		mlClassifier:            mlClassifier,
		enableRealTimeAnalysis:   true,
		enableBehavioralAnalysis: true,
		enableMLClassification:   true,
		logAllRequests:          true,
		blockingEnabled:         true,
		ipAnalysisCache:         make(map[string]*IPAnalysisResult),
		recentSubmissions:       make(map[string][]SubmissionRecord),
		honeypotTracker:         make(map[string]int),
		stats:                   &SpamDetectionStats{LastReset: time.Now()},
	}
}

// SpamProtection is the main middleware function for form submissions
func (sdm *SpamDetectionMiddleware) SpamProtection() gin.HandlerFunc {
	return func(c *gin.Context) {
		startTime := time.Now()
		
		// Only apply to form submission endpoints
		if !sdm.isFormSubmissionEndpoint(c.Request.URL.Path) {
			c.Next()
			return
		}
		
		// Increment request counter
		sdm.stats.TotalRequests++
		
		// Extract request information
		clientIP := c.ClientIP()
		userAgent := c.GetHeader("User-Agent")
		referer := c.GetHeader("Referer")
		formID := sdm.extractFormID(c)
		
		// Create request context with metadata
		requestMetadata := map[string]interface{}{
			"client_ip":    clientIP,
			"user_agent":   userAgent,
			"referer":      referer,
			"form_id":      formID,
			"method":       c.Request.Method,
			"endpoint":     c.Request.URL.Path,
			"query_params": c.Request.URL.Query(),
			"headers":      sdm.sanitizeHeaders(c.Request.Header),
			"timestamp":    time.Now().UTC(),
		}
		
		// Check if IP is already flagged
		if sdm.isIPFlagged(clientIP) {
			sdm.handleBlockedRequest(c, "ip_flagged", "IP address is flagged for suspicious activity")
			sdm.logRequest(requestMetadata, "blocked", "IP flagged", time.Since(startTime))
			return
		}
		
		// Pre-analysis checks
		if blocked, reason := sdm.preAnalysisChecks(c, requestMetadata); blocked {
			sdm.handleBlockedRequest(c, "pre_analysis", reason)
			sdm.logRequest(requestMetadata, "blocked", reason, time.Since(startTime))
			return
		}
		
		// Extract form data for analysis
		var formData map[string]interface{}
		if c.Request.Method == "POST" || c.Request.Method == "PUT" {
			// Read and restore request body for downstream handlers
			if bodyData, err := sdm.extractFormData(c); err == nil {
				formData = bodyData
				requestMetadata["form_data"] = formData
			}
		}
		
		// Perform comprehensive spam analysis if enabled
		if sdm.enableRealTimeAnalysis && formData != nil {
			analysisResult, err := sdm.spamService.AnalyzeSubmission(formID, formData, requestMetadata)
			if err != nil {
				log.Printf("Spam analysis error: %v", err)
				// Continue with request on analysis error
			} else {
				// Handle analysis result
				action := sdm.handleAnalysisResult(c, analysisResult, requestMetadata)
				if action == "block" || action == "quarantine" {
					sdm.logRequest(requestMetadata, action, "spam_detected", time.Since(startTime))
					return
				}
				
				// Add analysis result to context for downstream handlers
				c.Set("spam_analysis", analysisResult)
				requestMetadata["spam_analysis"] = analysisResult
			}
		}
		
		// Update IP tracking
		sdm.updateIPTracking(clientIP, formID, userAgent, "allowed", 0.0, []string{})
		
		// Log successful request
		if sdm.logAllRequests {
			sdm.logRequest(requestMetadata, "allowed", "passed_all_checks", time.Since(startTime))
		}
		
		// Continue to next middleware/handler
		c.Next()
	}
}

// HoneypotProtection middleware specifically for honeypot field detection
func (sdm *SpamDetectionMiddleware) HoneypotProtection(honeypotFields []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method != "POST" && c.Request.Method != "PUT" {
			c.Next()
			return
		}
		
		// Check honeypot fields
		for _, field := range honeypotFields {
			if value := c.PostForm(field); value != "" {
				clientIP := c.ClientIP()
				
				// Track honeypot violations
				sdm.honeypotTracker[clientIP]++
				
				// Auto-flag IP after multiple violations
				if sdm.honeypotTracker[clientIP] >= 3 {
					sdm.flagIP(clientIP, "multiple_honeypot_violations")
				}
				
				sdm.stats.SpamDetected++
				sdm.stats.Blocked++
				
				sdm.handleBlockedRequest(c, "honeypot", fmt.Sprintf("Honeypot field '%s' filled", field))
				return
			}
		}
		
		c.Next()
	}
}

// BehavioralAnalysis middleware for analyzing user behavior patterns
func (sdm *SpamDetectionMiddleware) BehavioralAnalysis() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !sdm.enableBehavioralAnalysis {
			c.Next()
			return
		}
		
		// Extract behavioral data from request headers or form data
		behavioralDataStr := c.GetHeader("X-Behavioral-Data")
		if behavioralDataStr == "" {
			behavioralDataStr = c.PostForm("_behavioral_data")
		}
		
		if behavioralDataStr != "" {
			var behavioralProfile services.BehavioralProfile
			if err := json.Unmarshal([]byte(behavioralDataStr), &behavioralProfile); err == nil {
				// Analyze behavior
				result, err := sdm.behavioralAnalyzer.AnalyzeBehavior(&behavioralProfile)
				if err == nil {
					// Add result to context
					c.Set("behavioral_analysis", result)
					
					// Handle high bot scores
					if result.BotScore >= 0.8 && result.Confidence >= 0.7 {
						sdm.stats.SpamDetected++
						
						if result.Recommendation == "block" {
							sdm.stats.Blocked++
							sdm.handleBlockedRequest(c, "behavioral", 
								fmt.Sprintf("High bot score: %.2f", result.BotScore))
							return
						} else if result.Recommendation == "challenge" {
							// Require CAPTCHA challenge
							c.Set("require_captcha", true)
						}
					}
				}
			}
		}
		
		c.Next()
	}
}

// RateLimitingEnhanced provides enhanced rate limiting with different thresholds per endpoint
func (sdm *SpamDetectionMiddleware) RateLimitingEnhanced() gin.HandlerFunc {
	return func(c *gin.Context) {
		clientIP := c.ClientIP()
		endpoint := c.Request.URL.Path
		
		// Get rate limit configuration for this endpoint
		limit, window := sdm.getRateLimitConfig(endpoint)
		
		// Check current rate
		if sdm.isRateLimited(clientIP, endpoint, limit, window) {
			sdm.stats.SpamDetected++
			sdm.stats.Blocked++
			
			// Track repeated rate limit violations
			sdm.updateIPTracking(clientIP, "", c.GetHeader("User-Agent"), "blocked", 0.8, []string{"rate_limit"})
			
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":       "Rate limit exceeded",
				"message":     fmt.Sprintf("Too many requests. Limit: %d per %v", limit, window),
				"retry_after": int(window.Seconds()),
				"blocked_by":  "rate_limiter",
			})
			c.Abort()
			return
		}
		
		c.Next()
	}
}

// IPReputationCheck middleware for checking IP reputation
func (sdm *SpamDetectionMiddleware) IPReputationCheck() gin.HandlerFunc {
	return func(c *gin.Context) {
		clientIP := c.ClientIP()
		
		// Get or calculate IP reputation
		reputation := sdm.getIPReputation(clientIP)
		c.Set("ip_reputation", reputation)
		
		// Block malicious IPs
		if reputation.Reputation == "malicious" {
			sdm.stats.SpamDetected++
			sdm.stats.Blocked++
			
			sdm.handleBlockedRequest(c, "ip_reputation", 
				fmt.Sprintf("Malicious IP detected (risk score: %.2f)", reputation.RiskScore))
			return
		}
		
		// Challenge suspicious IPs
		if reputation.Reputation == "suspicious" {
			c.Set("require_captcha", true)
			c.Set("challenge_reason", "suspicious_ip")
		}
		
		c.Next()
	}
}

// CaptchaValidation middleware for CAPTCHA verification
func (sdm *SpamDetectionMiddleware) CaptchaValidation() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check if CAPTCHA is required
		requireCaptcha := false
		if val, exists := c.Get("require_captcha"); exists {
			requireCaptcha = val.(bool)
		}
		
		// Also check spam analysis result
		if analysis, exists := c.Get("spam_analysis"); exists {
			if spamResult, ok := analysis.(*services.SpamDetectionResult); ok {
				requireCaptcha = requireCaptcha || spamResult.CaptchaRequired
			}
		}
		
		if !requireCaptcha {
			c.Next()
			return
		}
		
		// Extract CAPTCHA token
		captchaToken := c.PostForm("captcha_token")
		captchaProvider := c.PostForm("captcha_provider")
		
		if captchaToken == "" {
			// No CAPTCHA token provided, send challenge
			sdm.stats.CaptchaChallenges++
			
			c.JSON(http.StatusPreconditionRequired, gin.H{
				"error":             "CAPTCHA required",
				"message":           "Please complete the CAPTCHA challenge",
				"captcha_required":  true,
				"captcha_providers": []string{"recaptcha_v3", "hcaptcha", "turnstile", "fallback"},
				"challenge_reason":  c.GetString("challenge_reason"),
			})
			c.Abort()
			return
		}
		
		// Verify CAPTCHA
		captchaService := utils.NewCaptchaService(&utils.CaptchaConfig{
			RecaptchaV3MinScore: 0.5,
			FallbackEnabled:     true,
			Timeout:             10,
		})
		
		provider := utils.CaptchaProvider(captchaProvider)
		if provider == "" {
			provider = utils.ProviderRecaptchaV3
		}
		
		result, err := captchaService.VerifyWithFallback(provider, captchaToken, c.ClientIP())
		if err != nil || !result.Success {
			sdm.stats.SpamDetected++
			sdm.stats.Blocked++
			
			c.JSON(http.StatusForbidden, gin.H{
				"error":           "CAPTCHA verification failed",
				"message":         "Invalid or expired CAPTCHA token",
				"captcha_result":  result,
				"blocked_by":      "captcha_verification",
			})
			c.Abort()
			return
		}
		
		// CAPTCHA verified successfully
		c.Set("captcha_verified", true)
		c.Set("captcha_result", result)
		
		c.Next()
	}
}

// Helper methods

func (sdm *SpamDetectionMiddleware) isFormSubmissionEndpoint(path string) bool {
	submissionPaths := []string{
		"/api/v1/submit",
		"/api/v1/forms/",
		"/submit",
		"/contact",
		"/feedback",
	}
	
	for _, submissionPath := range submissionPaths {
		if strings.Contains(path, submissionPath) {
			return true
		}
	}
	
	return false
}

func (sdm *SpamDetectionMiddleware) extractFormID(c *gin.Context) string {
	// Try to extract form ID from various sources
	if formID := c.PostForm("form_id"); formID != "" {
		return formID
	}
	if formID := c.GetHeader("X-Form-ID"); formID != "" {
		return formID
	}
	if formID := c.Query("form_id"); formID != "" {
		return formID
	}
	if formID := c.Param("id"); formID != "" {
		return formID
	}
	
	// Generate default form ID based on path
	return fmt.Sprintf("form_%s", strings.ReplaceAll(c.Request.URL.Path, "/", "_"))
}

func (sdm *SpamDetectionMiddleware) sanitizeHeaders(headers http.Header) map[string]string {
	sanitized := make(map[string]string)
	
	// Only include safe headers
	safeHeaders := []string{
		"User-Agent", "Referer", "Accept", "Accept-Language", 
		"Accept-Encoding", "Content-Type", "Origin",
	}
	
	for _, header := range safeHeaders {
		if value := headers.Get(header); value != "" {
			sanitized[header] = value
		}
	}
	
	return sanitized
}

func (sdm *SpamDetectionMiddleware) isIPFlagged(ip string) bool {
	// Check if IP is in our local blacklist
	if analysis, exists := sdm.ipAnalysisCache[ip]; exists {
		return analysis.Reputation == "malicious" || analysis.BlockCount >= 5
	}
	
	return false
}

func (sdm *SpamDetectionMiddleware) preAnalysisChecks(c *gin.Context, metadata map[string]interface{}) (bool, string) {
	// Check for obvious bot indicators in User-Agent
	userAgent := strings.ToLower(c.GetHeader("User-Agent"))
	botIndicators := []string{
		"bot", "crawler", "spider", "scraper", "automated", 
		"curl", "wget", "python-requests", "python-urllib", "go-http-client",
	}
	
	for _, indicator := range botIndicators {
		if strings.Contains(userAgent, indicator) {
			return true, fmt.Sprintf("Bot user agent detected: %s", indicator)
		}
	}
	
	// Check for missing or suspicious User-Agent
	if userAgent == "" || len(userAgent) < 10 {
		return true, "Missing or suspicious user agent"
	}
	
	// Check for suspicious request patterns
	if c.Request.ContentLength > 10*1024*1024 { // 10MB limit
		return true, "Request size exceeds limit"
	}
	
	// Check referer for basic validation (if required)
	referer := c.GetHeader("Referer")
	if referer == "" && c.Request.Method == "POST" {
		// This might be suspicious but not always (could be direct API access)
		metadata["no_referer"] = true
	}
	
	return false, ""
}

func (sdm *SpamDetectionMiddleware) extractFormData(c *gin.Context) (map[string]interface{}, error) {
	data := make(map[string]interface{})
	
	// Handle different content types
	contentType := c.GetHeader("Content-Type")
	
	if strings.Contains(contentType, "application/json") {
		// JSON data
		var jsonData map[string]interface{}
		if err := c.ShouldBindJSON(&jsonData); err == nil {
			data = jsonData
		}
	} else if strings.Contains(contentType, "application/x-www-form-urlencoded") || 
			  strings.Contains(contentType, "multipart/form-data") {
		// Form data
		c.Request.ParseForm()
		c.Request.ParseMultipartForm(32 << 20) // 32MB limit
		
		for key, values := range c.Request.Form {
			if len(values) == 1 {
				data[key] = values[0]
			} else {
				data[key] = values
			}
		}
	}
	
	return data, nil
}

func (sdm *SpamDetectionMiddleware) handleAnalysisResult(c *gin.Context, result *services.SpamDetectionResult, 
	metadata map[string]interface{}) string {
	
	sdm.stats.ProcessingTimeMs += result.ProcessingTime.Milliseconds()
	
	if result.IsSpam {
		sdm.stats.SpamDetected++
	}
	
	switch result.Action {
	case "block":
		sdm.stats.Blocked++
		sdm.handleBlockedRequest(c, "spam_analysis", 
			fmt.Sprintf("Spam detected (score: %.2f)", result.SpamScore))
		return "block"
		
	case "quarantine":
		sdm.stats.Quarantined++
		// Log for manual review but allow the request
		sdm.logQuarantinedRequest(metadata, result)
		return "quarantine"
		
	default:
		return "allow"
	}
}

func (sdm *SpamDetectionMiddleware) handleBlockedRequest(c *gin.Context, blockType, reason string) {
	// Update IP tracking
	clientIP := c.ClientIP()
	formID := sdm.extractFormID(c)
	userAgent := c.GetHeader("User-Agent")
	
	sdm.updateIPTracking(clientIP, formID, userAgent, "blocked", 1.0, []string{blockType})
	
	// Return appropriate response
	c.JSON(http.StatusForbidden, gin.H{
		"error":      "Request blocked",
		"message":    reason,
		"blocked_by": blockType,
		"timestamp":  time.Now().UTC(),
		"request_id": uuid.New().String(),
	})
	c.Abort()
}

func (sdm *SpamDetectionMiddleware) updateIPTracking(ip, formID, userAgent, result string, 
	spamScore float64, triggers []string) {
	
	// Update IP analysis cache
	if analysis, exists := sdm.ipAnalysisCache[ip]; exists {
		analysis.LastSeen = time.Now()
		analysis.SubmissionCount++
		if result == "blocked" {
			analysis.BlockCount++
			analysis.RiskScore = (analysis.RiskScore + spamScore) / 2 // Moving average
		}
	} else {
		// Create new analysis entry
		sdm.ipAnalysisCache[ip] = &IPAnalysisResult{
			IP:              ip,
			RiskScore:       spamScore,
			Reputation:      sdm.calculateReputation(spamScore),
			LastSeen:        time.Now(),
			SubmissionCount: 1,
			BlockCount:      0,
			Metadata:        make(map[string]interface{}),
		}
		if result == "blocked" {
			sdm.ipAnalysisCache[ip].BlockCount = 1
		}
	}
	
	// Update recent submissions
	record := SubmissionRecord{
		Timestamp: time.Now(),
		FormID:    formID,
		UserAgent: userAgent,
		Result:    result,
		SpamScore: spamScore,
		Triggers:  triggers,
		Metadata:  map[string]interface{}{"ip": ip},
	}
	
	sdm.recentSubmissions[ip] = append(sdm.recentSubmissions[ip], record)
	
	// Keep only recent submissions (last 100 or last 24 hours)
	cutoff := time.Now().Add(-24 * time.Hour)
	filtered := []SubmissionRecord{}
	for _, sub := range sdm.recentSubmissions[ip] {
		if sub.Timestamp.After(cutoff) {
			filtered = append(filtered, sub)
		}
	}
	
	if len(filtered) > 100 {
		filtered = filtered[len(filtered)-100:]
	}
	
	sdm.recentSubmissions[ip] = filtered
}

func (sdm *SpamDetectionMiddleware) calculateReputation(riskScore float64) string {
	if riskScore >= 0.8 {
		return "malicious"
	} else if riskScore >= 0.6 {
		return "suspicious"
	} else if riskScore >= 0.3 {
		return "neutral"
	} else {
		return "good"
	}
}

func (sdm *SpamDetectionMiddleware) flagIP(ip, reason string) {
	if analysis, exists := sdm.ipAnalysisCache[ip]; exists {
		analysis.Reputation = "malicious"
		analysis.RiskScore = 1.0
		analysis.Metadata["flag_reason"] = reason
		analysis.Metadata["flagged_at"] = time.Now()
	}
}

func (sdm *SpamDetectionMiddleware) getRateLimitConfig(endpoint string) (int, time.Duration) {
	// Different rate limits for different endpoints
	switch {
	case strings.Contains(endpoint, "/submit"):
		return 10, 15 * time.Minute // 10 submissions per 15 minutes
	case strings.Contains(endpoint, "/contact"):
		return 5, 10 * time.Minute  // 5 contacts per 10 minutes
	case strings.Contains(endpoint, "/api/"):
		return 100, 1 * time.Hour   // 100 API calls per hour
	default:
		return 50, 1 * time.Hour    // Default: 50 requests per hour
	}
}

func (sdm *SpamDetectionMiddleware) isRateLimited(ip, endpoint string, limit int, window time.Duration) bool {
	// Simple rate limiting implementation
	// In production, this should use Redis for distributed rate limiting
	key := fmt.Sprintf("%s:%s", ip, endpoint)
	
	submissions, exists := sdm.recentSubmissions[ip]
	if !exists {
		return false
	}
	
	// Count submissions in the time window for this endpoint
	cutoff := time.Now().Add(-window)
	count := 0
	
	for _, sub := range submissions {
		if sub.Timestamp.After(cutoff) && strings.Contains(sub.FormID, endpoint) {
			count++
		}
	}
	
	return count >= limit
}

func (sdm *SpamDetectionMiddleware) getIPReputation(ip string) *IPAnalysisResult {
	// Check cache first
	if analysis, exists := sdm.ipAnalysisCache[ip]; exists {
		// Update last seen
		analysis.LastSeen = time.Now()
		return analysis
	}
	
	// Create new analysis (in production, this would query external services)
	analysis := &IPAnalysisResult{
		IP:              ip,
		RiskScore:       0.1, // Default low risk
		Reputation:      "neutral",
		LastSeen:        time.Now(),
		SubmissionCount: 0,
		BlockCount:      0,
		Metadata:        make(map[string]interface{}),
	}
	
	// Cache the result
	sdm.ipAnalysisCache[ip] = analysis
	
	return analysis
}

func (sdm *SpamDetectionMiddleware) logRequest(metadata map[string]interface{}, action, reason string, 
	processingTime time.Duration) {
	
	if !sdm.logAllRequests && action == "allowed" {
		return
	}
	
	logEntry := map[string]interface{}{
		"timestamp":       time.Now().UTC(),
		"action":          action,
		"reason":          reason,
		"processing_time": processingTime.Milliseconds(),
		"metadata":        metadata,
	}
	
	// Log to appropriate destination (file, database, external service)
	logJSON, _ := json.Marshal(logEntry)
	log.Printf("SpamDetection: %s", string(logJSON))
}

func (sdm *SpamDetectionMiddleware) logQuarantinedRequest(metadata map[string]interface{}, 
	result *services.SpamDetectionResult) {
	
	logEntry := map[string]interface{}{
		"timestamp":      time.Now().UTC(),
		"action":         "quarantined",
		"spam_score":     result.SpamScore,
		"confidence":     result.Confidence,
		"triggers":       result.Triggers,
		"metadata":       metadata,
		"requires_review": true,
	}
	
	logJSON, _ := json.Marshal(logEntry)
	log.Printf("SpamQuarantine: %s", string(logJSON))
}

// GetStats returns current middleware statistics
func (sdm *SpamDetectionMiddleware) GetStats() *SpamDetectionStats {
	// Calculate average processing time
	if sdm.stats.TotalRequests > 0 {
		sdm.stats.ProcessingTimeMs = sdm.stats.ProcessingTimeMs / sdm.stats.TotalRequests
	}
	
	return sdm.stats
}

// ResetStats resets the middleware statistics
func (sdm *SpamDetectionMiddleware) ResetStats() {
	sdm.stats = &SpamDetectionStats{
		LastReset: time.Now(),
	}
}

// CleanupCache removes old entries from in-memory caches
func (sdm *SpamDetectionMiddleware) CleanupCache() {
	cutoff := time.Now().Add(-24 * time.Hour)
	
	// Clean IP analysis cache
	for ip, analysis := range sdm.ipAnalysisCache {
		if analysis.LastSeen.Before(cutoff) {
			delete(sdm.ipAnalysisCache, ip)
		}
	}
	
	// Clean recent submissions
	for ip, submissions := range sdm.recentSubmissions {
		filtered := []SubmissionRecord{}
		for _, sub := range submissions {
			if sub.Timestamp.After(cutoff) {
				filtered = append(filtered, sub)
			}
		}
		
		if len(filtered) > 0 {
			sdm.recentSubmissions[ip] = filtered
		} else {
			delete(sdm.recentSubmissions, ip)
		}
	}
	
	// Clean honeypot tracker
	for ip, count := range sdm.honeypotTracker {
		if count < 3 { // Reset counts periodically
			delete(sdm.honeypotTracker, ip)
		}
	}
}