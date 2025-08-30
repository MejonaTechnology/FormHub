package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"formhub/internal/models"
	"formhub/pkg/utils"
	"log"
	"math"
	"net"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// SpamProtectionLevel defines the level of spam protection
type SpamProtectionLevel string

const (
	SpamProtectionLow    SpamProtectionLevel = "low"
	SpamProtectionMedium SpamProtectionLevel = "medium"
	SpamProtectionHigh   SpamProtectionLevel = "high"
	SpamProtectionMax    SpamProtectionLevel = "maximum"
)

// SpamProtectionConfig holds configuration for spam protection
type SpamProtectionConfig struct {
	// Global settings
	Enabled             bool                `json:"enabled"`
	DefaultLevel        SpamProtectionLevel `json:"default_level"`
	BlockThreshold      float64             `json:"block_threshold"`      // 0.0-1.0
	QuarantineThreshold float64             `json:"quarantine_threshold"` // 0.0-1.0
	
	// CAPTCHA settings
	CaptchaConfig       *utils.CaptchaConfig `json:"captcha_config"`
	RequireCaptcha      bool                 `json:"require_captcha"`
	CaptchaProvider     utils.CaptchaProvider `json:"captcha_provider"`
	CaptchaFallback     bool                 `json:"captcha_fallback"`
	
	// Rate limiting
	RateLimitEnabled    bool  `json:"rate_limit_enabled"`
	MaxSubmissionsPerIP int   `json:"max_submissions_per_ip"`
	RateLimitWindowMin  int   `json:"rate_limit_window_minutes"`
	
	// Honeypot
	HoneypotEnabled     bool     `json:"honeypot_enabled"`
	HoneypotFields      []string `json:"honeypot_fields"`
	
	// Content filtering
	ContentFilterEnabled bool     `json:"content_filter_enabled"`
	BlockedKeywords     []string `json:"blocked_keywords"`
	BlockedDomains      []string `json:"blocked_domains"`
	MaxUrlCount         int      `json:"max_url_count"`
	MaxLinkCount        int      `json:"max_link_count"`
	
	// Behavioral analysis
	BehavioralEnabled   bool    `json:"behavioral_enabled"`
	MinTypingTime       float64 `json:"min_typing_time_seconds"`
	MaxTypingSpeed      float64 `json:"max_typing_speed_wpm"`
	RequireInteraction  bool    `json:"require_interaction"`
	
	// IP reputation
	IPReputationEnabled bool     `json:"ip_reputation_enabled"`
	BlockedCountries    []string `json:"blocked_countries"`
	BlockedASNs         []string `json:"blocked_asns"`
	VPNDetectionEnabled bool     `json:"vpn_detection_enabled"`
	
	// Machine learning
	MLEnabled           bool    `json:"ml_enabled"`
	MLThreshold         float64 `json:"ml_threshold"`
	EnableLearning      bool    `json:"enable_learning"`
	
	// Webhook notifications
	WebhookURL          string `json:"webhook_url"`
	NotifyOnBlock       bool   `json:"notify_on_block"`
	NotifyOnQuarantine  bool   `json:"notify_on_quarantine"`
}

// FormSpamConfig holds per-form spam protection settings
type FormSpamConfig struct {
	FormID                string                   `json:"form_id"`
	Enabled               bool                     `json:"enabled"`
	Level                 SpamProtectionLevel      `json:"level"`
	CustomConfig          *SpamProtectionConfig    `json:"custom_config,omitempty"`
	Whitelist             []string                 `json:"whitelist,omitempty"`        // Whitelisted IPs
	Blacklist             []string                 `json:"blacklist,omitempty"`        // Blacklisted IPs
	CustomRules           []CustomSpamRule         `json:"custom_rules,omitempty"`
	HoneypotFieldOverride []string                 `json:"honeypot_field_override,omitempty"`
}

// CustomSpamRule represents a custom spam detection rule
type CustomSpamRule struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Pattern     string  `json:"pattern"`     // Regex pattern
	Field       string  `json:"field"`       // Field to check ("*" for all)
	Action      string  `json:"action"`      // "block", "quarantine", "flag"
	Score       float64 `json:"score"`       // Score to add (0.0-1.0)
	Enabled     bool    `json:"enabled"`
}

// SpamDetectionResult contains the result of spam analysis
type SpamDetectionResult struct {
	IsSpam          bool                   `json:"is_spam"`
	SpamScore       float64                `json:"spam_score"`
	Confidence      float64                `json:"confidence"`
	Action          string                 `json:"action"` // "allow", "block", "quarantine"
	Triggers        []SpamTrigger          `json:"triggers"`
	Metadata        map[string]interface{} `json:"metadata"`
	ProcessingTime  time.Duration          `json:"processing_time"`
	CaptchaRequired bool                   `json:"captcha_required"`
	CaptchaChallenge *utils.FallbackCaptcha `json:"captcha_challenge,omitempty"`
}

// SpamTrigger represents a triggered spam detection rule
type SpamTrigger struct {
	Type        string  `json:"type"`
	Rule        string  `json:"rule"`
	Description string  `json:"description"`
	Score       float64 `json:"score"`
	Severity    string  `json:"severity"` // "low", "medium", "high", "critical"
	Field       string  `json:"field,omitempty"`
	Value       string  `json:"value,omitempty"`
}

// BehavioralData represents behavioral analysis data from the frontend
type BehavioralData struct {
	TypingTime      float64 `json:"typing_time"`      // Total time spent typing
	TypingSpeed     float64 `json:"typing_speed"`     // Words per minute
	MouseMovements  int     `json:"mouse_movements"`  // Number of mouse movements
	Keystrokes      int     `json:"keystrokes"`       // Total keystrokes
	Backspaces      int     `json:"backspaces"`       // Number of backspaces
	CopyPastes      int     `json:"copy_pastes"`      // Number of copy/paste actions
	TabSwitches     int     `json:"tab_switches"`     // Number of tab switches
	TimeOnPage      float64 `json:"time_on_page"`     // Total time on page
	ScrollBehavior  string  `json:"scroll_behavior"`  // "natural", "bot-like", "suspicious"
	InteractionTime float64 `json:"interaction_time"` // Time before first interaction
}

// SpamProtectionService provides comprehensive spam protection
type SpamProtectionService struct {
	db              *sql.DB
	redis           *redis.Client
	captchaService  *utils.CaptchaService
	securityService *SecurityService
	globalConfig    *SpamProtectionConfig
	ctx             context.Context
	
	// Machine learning components
	mlModel         *NaiveBayesSpamClassifier
	behavioralModel *BehavioralAnalyzer
}

// NewSpamProtectionService creates a new spam protection service
func NewSpamProtectionService(db *sql.DB, redis *redis.Client, securityService *SecurityService) *SpamProtectionService {
	// Default configuration
	defaultConfig := &SpamProtectionConfig{
		Enabled:             true,
		DefaultLevel:        SpamProtectionMedium,
		BlockThreshold:      0.8,
		QuarantineThreshold: 0.6,
		CaptchaConfig: &utils.CaptchaConfig{
			RecaptchaV3MinScore: 0.5,
			FallbackEnabled:     true,
			Timeout:             10,
		},
		RequireCaptcha:      false,
		CaptchaProvider:     utils.ProviderRecaptchaV3,
		CaptchaFallback:     true,
		RateLimitEnabled:    true,
		MaxSubmissionsPerIP: 10,
		RateLimitWindowMin:  15,
		HoneypotEnabled:     true,
		HoneypotFields:      []string{"_honeypot", "_hp", "_bot_check", "_email_confirm"},
		ContentFilterEnabled: true,
		BlockedKeywords:     []string{"viagra", "casino", "lottery", "porn", "sex", "free money", "click here", "urgent"},
		BlockedDomains:      []string{"spam.com", "tempmail.org", "guerrillamail.com"},
		MaxUrlCount:         3,
		MaxLinkCount:        2,
		BehavioralEnabled:   true,
		MinTypingTime:       2.0,
		MaxTypingSpeed:      200.0,
		RequireInteraction:  true,
		IPReputationEnabled: true,
		BlockedCountries:    []string{}, // Empty by default
		VPNDetectionEnabled: false,
		MLEnabled:           true,
		MLThreshold:         0.7,
		EnableLearning:      true,
		NotifyOnBlock:       true,
		NotifyOnQuarantine:  true,
	}
	
	service := &SpamProtectionService{
		db:              db,
		redis:           redis,
		securityService: securityService,
		globalConfig:    defaultConfig,
		ctx:             context.Background(),
	}
	
	// Initialize CAPTCHA service
	service.captchaService = utils.NewCaptchaService(defaultConfig.CaptchaConfig)
	
	// Initialize ML components
	service.mlModel = NewNaiveBayesSpamClassifier(db, redis)
	service.behavioralModel = NewBehavioralAnalyzer(db, redis)
	
	// Load configuration from database
	service.loadConfiguration()
	
	return service
}

// AnalyzeSubmission performs comprehensive spam analysis on a form submission
func (sps *SpamProtectionService) AnalyzeSubmission(formID string, data map[string]interface{}, 
	metadata map[string]interface{}) (*SpamDetectionResult, error) {
	
	startTime := time.Now()
	
	result := &SpamDetectionResult{
		IsSpam:     false,
		SpamScore:  0.0,
		Confidence: 0.0,
		Action:     "allow",
		Triggers:   []SpamTrigger{},
		Metadata:   make(map[string]interface{}),
	}
	
	// Get form-specific configuration
	formConfig, err := sps.getFormConfig(formID)
	if err != nil {
		return nil, fmt.Errorf("failed to get form config: %w", err)
	}
	
	if !formConfig.Enabled {
		result.Metadata["protection_disabled"] = true
		result.ProcessingTime = time.Since(startTime)
		return result, nil
	}
	
	// Extract metadata
	clientIP := sps.extractStringMetadata(metadata, "client_ip")
	userAgent := sps.extractStringMetadata(metadata, "user_agent")
	referer := sps.extractStringMetadata(metadata, "referer")
	captchaToken := sps.extractStringMetadata(metadata, "captcha_token")
	captchaProvider := utils.CaptchaProvider(sps.extractStringMetadata(metadata, "captcha_provider"))
	
	// Parse behavioral data if available
	var behavioralData *BehavioralData
	if behaviorStr, ok := metadata["behavioral_data"].(string); ok {
		behavioralData = &BehavioralData{}
		json.Unmarshal([]byte(behaviorStr), behavioralData)
	}
	
	// 1. Rate limiting check
	if formConfig.CustomConfig != nil && formConfig.CustomConfig.RateLimitEnabled {
		if trigger, blocked := sps.checkRateLimit(clientIP, formConfig); blocked {
			result.Triggers = append(result.Triggers, *trigger)
			result.SpamScore += trigger.Score
		}
	}
	
	// 2. IP-based checks
	if ipTriggers := sps.analyzeIP(clientIP, formConfig); len(ipTriggers) > 0 {
		result.Triggers = append(result.Triggers, ipTriggers...)
		for _, trigger := range ipTriggers {
			result.SpamScore += trigger.Score
		}
	}
	
	// 3. Honeypot field check
	if formConfig.CustomConfig != nil && formConfig.CustomConfig.HoneypotEnabled {
		if trigger := sps.checkHoneypot(data, formConfig); trigger != nil {
			result.Triggers = append(result.Triggers, *trigger)
			result.SpamScore += trigger.Score
		}
	}
	
	// 4. Content analysis
	if formConfig.CustomConfig != nil && formConfig.CustomConfig.ContentFilterEnabled {
		if contentTriggers := sps.analyzeContent(data, formConfig); len(contentTriggers) > 0 {
			result.Triggers = append(result.Triggers, contentTriggers...)
			for _, trigger := range contentTriggers {
				result.SpamScore += trigger.Score
			}
		}
	}
	
	// 5. Behavioral analysis
	if formConfig.CustomConfig != nil && formConfig.CustomConfig.BehavioralEnabled && behavioralData != nil {
		if behavioralTriggers := sps.analyzeBehavior(behavioralData, formConfig); len(behavioralTriggers) > 0 {
			result.Triggers = append(result.Triggers, behavioralTriggers...)
			for _, trigger := range behavioralTriggers {
				result.SpamScore += trigger.Score
			}
		}
	}
	
	// 6. Custom rules
	if len(formConfig.CustomRules) > 0 {
		if customTriggers := sps.checkCustomRules(data, formConfig.CustomRules); len(customTriggers) > 0 {
			result.Triggers = append(result.Triggers, customTriggers...)
			for _, trigger := range customTriggers {
				result.SpamScore += trigger.Score
			}
		}
	}
	
	// 7. Machine learning analysis
	if formConfig.CustomConfig != nil && formConfig.CustomConfig.MLEnabled {
		mlScore, mlConfidence, err := sps.mlModel.PredictSpam(data, metadata)
		if err == nil {
			result.Metadata["ml_score"] = mlScore
			result.Metadata["ml_confidence"] = mlConfidence
			
			if mlScore > formConfig.CustomConfig.MLThreshold {
				trigger := SpamTrigger{
					Type:        "machine_learning",
					Rule:        "ml_classifier",
					Description: fmt.Sprintf("ML classifier flagged as spam (score: %.3f)", mlScore),
					Score:       mlScore * 0.5, // Weight ML score
					Severity:    sps.getScoreSeverity(mlScore),
				}
				result.Triggers = append(result.Triggers, trigger)
				result.SpamScore += trigger.Score
			}
		}
	}
	
	// 8. CAPTCHA verification
	captchaRequired := sps.shouldRequireCaptcha(result.SpamScore, formConfig)
	if captchaRequired {
		result.CaptchaRequired = true
		
		// If CAPTCHA token provided, verify it
		if captchaToken != "" {
			captchaResult, err := sps.captchaService.VerifyWithFallback(captchaProvider, captchaToken, clientIP)
			if err != nil || !captchaResult.Success {
				trigger := SpamTrigger{
					Type:        "captcha",
					Rule:        "captcha_failed",
					Description: "CAPTCHA verification failed",
					Score:       0.8,
					Severity:    "high",
				}
				result.Triggers = append(result.Triggers, trigger)
				result.SpamScore += trigger.Score
			} else {
				result.Metadata["captcha_verified"] = true
				result.Metadata["captcha_provider"] = captchaResult.Provider
				if captchaResult.Score > 0 {
					result.Metadata["captcha_score"] = captchaResult.Score
				}
			}
		} else {
			// Generate fallback CAPTCHA challenge
			if fallbackChallenge, err := sps.captchaService.GenerateFallbackCaptcha(clientIP); err == nil {
				result.CaptchaChallenge = fallbackChallenge
			}
		}
	}
	
	// Normalize spam score
	if result.SpamScore > 1.0 {
		result.SpamScore = 1.0
	}
	
	// Calculate confidence based on number and quality of triggers
	result.Confidence = sps.calculateConfidence(result.Triggers, result.SpamScore)
	
	// Determine action based on score and thresholds
	config := formConfig.CustomConfig
	if config == nil {
		config = sps.globalConfig
	}
	
	if result.SpamScore >= config.BlockThreshold || result.CaptchaRequired && captchaToken == "" {
		result.IsSpam = true
		result.Action = "block"
	} else if result.SpamScore >= config.QuarantineThreshold {
		result.IsSpam = true
		result.Action = "quarantine"
	}
	
	// Store analysis result for learning
	if config.EnableLearning {
		go sps.storeAnalysisResult(formID, data, metadata, result)
	}
	
	// Send webhook notification if configured
	if (result.Action == "block" && config.NotifyOnBlock) || 
	   (result.Action == "quarantine" && config.NotifyOnQuarantine) {
		go sps.sendWebhookNotification(formID, result, data, metadata)
	}
	
	result.ProcessingTime = time.Since(startTime)
	result.Metadata["form_id"] = formID
	result.Metadata["client_ip"] = clientIP
	result.Metadata["user_agent"] = userAgent
	result.Metadata["protection_level"] = formConfig.Level
	
	return result, nil
}

// checkRateLimit checks if the IP has exceeded rate limits
func (sps *SpamProtectionService) checkRateLimit(clientIP string, config *FormSpamConfig) (*SpamTrigger, bool) {
	if config.CustomConfig == nil || !config.CustomConfig.RateLimitEnabled {
		return nil, false
	}
	
	key := fmt.Sprintf("rate_limit:%s:%s", config.FormID, clientIP)
	window := time.Duration(config.CustomConfig.RateLimitWindowMin) * time.Minute
	maxRequests := config.CustomConfig.MaxSubmissionsPerIP
	
	// Sliding window rate limiting using Redis
	now := time.Now()
	windowStart := now.Add(-window)
	
	pipe := sps.redis.Pipeline()
	
	// Remove old entries
	pipe.ZRemRangeByScore(sps.ctx, key, "0", fmt.Sprintf("%d", windowStart.Unix()))
	
	// Count current requests in window
	countCmd := pipe.ZCount(sps.ctx, key, fmt.Sprintf("%d", windowStart.Unix()), fmt.Sprintf("%d", now.Unix()))
	
	// Add current request
	pipe.ZAdd(sps.ctx, key, redis.Z{Score: float64(now.Unix()), Member: now.UnixNano()})
	
	// Set expiry
	pipe.Expire(sps.ctx, key, window)
	
	_, err := pipe.Exec(sps.ctx)
	if err != nil {
		return nil, false
	}
	
	count, err := countCmd.Result()
	if err != nil {
		return nil, false
	}
	
	if count >= int64(maxRequests) {
		trigger := &SpamTrigger{
			Type:        "rate_limit",
			Rule:        "ip_rate_limit",
			Description: fmt.Sprintf("IP exceeded rate limit: %d requests in %v", count, window),
			Score:       0.9,
			Severity:    "high",
		}
		return trigger, true
	}
	
	return nil, false
}

// analyzeIP performs IP-based analysis
func (sps *SpamProtectionService) analyzeIP(clientIP string, config *FormSpamConfig) []SpamTrigger {
	triggers := []SpamTrigger{}
	
	if clientIP == "" {
		return triggers
	}
	
	// Check whitelist first
	for _, whiteIP := range config.Whitelist {
		if clientIP == whiteIP || sps.ipInCIDR(clientIP, whiteIP) {
			return triggers // Whitelisted, skip all IP checks
		}
	}
	
	// Check blacklist
	for _, blackIP := range config.Blacklist {
		if clientIP == blackIP || sps.ipInCIDR(clientIP, blackIP) {
			triggers = append(triggers, SpamTrigger{
				Type:        "ip_blacklist",
				Rule:        "blacklisted_ip",
				Description: fmt.Sprintf("IP %s is blacklisted", clientIP),
				Score:       1.0,
				Severity:    "critical",
			})
			break
		}
	}
	
	// IP reputation check
	if config.CustomConfig != nil && config.CustomConfig.IPReputationEnabled {
		if reputation := sps.checkIPReputation(clientIP); reputation != nil {
			triggers = append(triggers, *reputation)
		}
	}
	
	// VPN/Proxy detection
	if config.CustomConfig != nil && config.CustomConfig.VPNDetectionEnabled {
		if vpnTrigger := sps.detectVPN(clientIP); vpnTrigger != nil {
			triggers = append(triggers, *vpnTrigger)
		}
	}
	
	return triggers
}

// checkHoneypot checks for honeypot field violations
func (sps *SpamProtectionService) checkHoneypot(data map[string]interface{}, config *FormSpamConfig) *SpamTrigger {
	honeypotFields := config.HoneypotFieldOverride
	if len(honeypotFields) == 0 && config.CustomConfig != nil {
		honeypotFields = config.CustomConfig.HoneypotFields
	}
	if len(honeypotFields) == 0 {
		honeypotFields = sps.globalConfig.HoneypotFields
	}
	
	for _, field := range honeypotFields {
		if value, exists := data[field]; exists && value != "" {
			return &SpamTrigger{
				Type:        "honeypot",
				Rule:        "honeypot_filled",
				Description: fmt.Sprintf("Honeypot field '%s' was filled", field),
				Score:       0.95,
				Severity:    "critical",
				Field:       field,
				Value:       fmt.Sprintf("%v", value),
			}
		}
	}
	
	return nil
}

// analyzeContent performs content-based spam analysis
func (sps *SpamProtectionService) analyzeContent(data map[string]interface{}, config *FormSpamConfig) []SpamTrigger {
	triggers := []SpamTrigger{}
	
	if config.CustomConfig == nil {
		return triggers
	}
	
	for field, value := range data {
		valueStr, ok := value.(string)
		if !ok {
			continue
		}
		
		lowerValue := strings.ToLower(valueStr)
		
		// Check blocked keywords
		for _, keyword := range config.CustomConfig.BlockedKeywords {
			if strings.Contains(lowerValue, strings.ToLower(keyword)) {
				triggers = append(triggers, SpamTrigger{
					Type:        "content_filter",
					Rule:        "blocked_keyword",
					Description: fmt.Sprintf("Contains blocked keyword: %s", keyword),
					Score:       0.4,
					Severity:    "medium",
					Field:       field,
				})
			}
		}
		
		// Check URL/link count
		urlPattern := regexp.MustCompile(`https?://[^\s]+`)
		urls := urlPattern.FindAllString(valueStr, -1)
		if len(urls) > config.CustomConfig.MaxUrlCount {
			triggers = append(triggers, SpamTrigger{
				Type:        "content_filter",
				Rule:        "excessive_urls",
				Description: fmt.Sprintf("Contains %d URLs (max allowed: %d)", len(urls), config.CustomConfig.MaxUrlCount),
				Score:       0.6,
				Severity:    "medium",
				Field:       field,
			})
		}
		
		// Check for blocked domains
		for _, url := range urls {
			for _, domain := range config.CustomConfig.BlockedDomains {
				if strings.Contains(url, domain) {
					triggers = append(triggers, SpamTrigger{
						Type:        "content_filter",
						Rule:        "blocked_domain",
						Description: fmt.Sprintf("Contains link to blocked domain: %s", domain),
						Score:       0.8,
						Severity:    "high",
						Field:       field,
					})
				}
			}
		}
		
		// Check for excessive capitalization
		if len(valueStr) > 20 {
			upperCount := 0
			for _, char := range valueStr {
				if char >= 'A' && char <= 'Z' {
					upperCount++
				}
			}
			if float64(upperCount)/float64(len(valueStr)) > 0.7 {
				triggers = append(triggers, SpamTrigger{
					Type:        "content_filter",
					Rule:        "excessive_caps",
					Description: "Excessive use of capital letters",
					Score:       0.3,
					Severity:    "low",
					Field:       field,
				})
			}
		}
		
		// Check for suspicious patterns
		suspiciousPatterns := []struct {
			pattern string
			score   float64
			desc    string
		}{
			{`\b(urgent|act now|limited time|don't wait|hurry|call now)\b`, 0.4, "Contains urgent language"},
			{`\b(free money|get rich|make money fast|guaranteed income)\b`, 0.6, "Contains money-making claims"},
			{`\b(click here|visit now|buy now|order today)\b`, 0.3, "Contains call-to-action spam"},
			{`[!]{3,}`, 0.2, "Excessive exclamation marks"},
			{`\b\w*\d+\w*@\w+\.\w+`, 0.5, "Contains email with numbers (suspicious)"},
		}
		
		for _, pattern := range suspiciousPatterns {
			matched, _ := regexp.MatchString(pattern.pattern, lowerValue)
			if matched {
				triggers = append(triggers, SpamTrigger{
					Type:        "content_filter",
					Rule:        "suspicious_pattern",
					Description: pattern.desc,
					Score:       pattern.score,
					Severity:    sps.getScoreSeverity(pattern.score),
					Field:       field,
				})
			}
		}
	}
	
	return triggers
}

// analyzeBehavior performs behavioral analysis
func (sps *SpamProtectionService) analyzeBehavior(behavioral *BehavioralData, config *FormSpamConfig) []SpamTrigger {
	triggers := []SpamTrigger{}
	
	if config.CustomConfig == nil || !config.CustomConfig.BehavioralEnabled {
		return triggers
	}
	
	// Check typing time
	if behavioral.TypingTime < config.CustomConfig.MinTypingTime {
		triggers = append(triggers, SpamTrigger{
			Type:        "behavioral",
			Rule:        "too_fast_typing",
			Description: fmt.Sprintf("Form filled too quickly: %.2f seconds", behavioral.TypingTime),
			Score:       0.6,
			Severity:    "medium",
		})
	}
	
	// Check typing speed
	if behavioral.TypingSpeed > config.CustomConfig.MaxTypingSpeed {
		triggers = append(triggers, SpamTrigger{
			Type:        "behavioral",
			Rule:        "inhuman_typing_speed",
			Description: fmt.Sprintf("Typing speed too high: %.0f WPM", behavioral.TypingSpeed),
			Score:       0.8,
			Severity:    "high",
		})
	}
	
	// Check for lack of natural behavior
	if behavioral.MouseMovements == 0 && behavioral.Keystrokes > 50 {
		triggers = append(triggers, SpamTrigger{
			Type:        "behavioral",
			Rule:        "no_mouse_movement",
			Description: "No mouse movements detected with significant typing",
			Score:       0.5,
			Severity:    "medium",
		})
	}
	
	// Check copy-paste behavior
	if behavioral.CopyPastes > behavioral.Keystrokes/2 {
		triggers = append(triggers, SpamTrigger{
			Type:        "behavioral",
			Rule:        "excessive_copy_paste",
			Description: "Excessive copy-paste activity detected",
			Score:       0.4,
			Severity:    "low",
		})
	}
	
	// Check interaction time
	if config.CustomConfig.RequireInteraction && behavioral.InteractionTime < 1.0 {
		triggers = append(triggers, SpamTrigger{
			Type:        "behavioral",
			Rule:        "immediate_interaction",
			Description: "Form interaction started immediately (possible bot)",
			Score:       0.7,
			Severity:    "high",
		})
	}
	
	return triggers
}

// checkCustomRules evaluates custom spam rules
func (sps *SpamProtectionService) checkCustomRules(data map[string]interface{}, rules []CustomSpamRule) []SpamTrigger {
	triggers := []SpamTrigger{}
	
	for _, rule := range rules {
		if !rule.Enabled {
			continue
		}
		
		pattern, err := regexp.Compile(rule.Pattern)
		if err != nil {
			continue // Skip invalid patterns
		}
		
		// Check specific field or all fields
		fieldsToCheck := make(map[string]interface{})
		if rule.Field == "*" {
			fieldsToCheck = data
		} else if value, exists := data[rule.Field]; exists {
			fieldsToCheck[rule.Field] = value
		}
		
		for field, value := range fieldsToCheck {
			valueStr := fmt.Sprintf("%v", value)
			if pattern.MatchString(valueStr) {
				trigger := SpamTrigger{
					Type:        "custom_rule",
					Rule:        rule.Name,
					Description: rule.Description,
					Score:       rule.Score,
					Severity:    sps.getScoreSeverity(rule.Score),
					Field:       field,
				}
				
				triggers = append(triggers, trigger)
				
				// Stop at first match if action is block
				if rule.Action == "block" {
					break
				}
			}
		}
	}
	
	return triggers
}

// Helper methods

func (sps *SpamProtectionService) extractStringMetadata(metadata map[string]interface{}, key string) string {
	if value, ok := metadata[key].(string); ok {
		return value
	}
	return ""
}

func (sps *SpamProtectionService) ipInCIDR(ip, cidr string) bool {
	if !strings.Contains(cidr, "/") {
		return ip == cidr
	}
	
	_, network, err := net.ParseCIDR(cidr)
	if err != nil {
		return false
	}
	
	ipAddr := net.ParseIP(ip)
	return network.Contains(ipAddr)
}

func (sps *SpamProtectionService) checkIPReputation(ip string) *SpamTrigger {
	// This would integrate with IP reputation services in production
	// For now, return nil (placeholder)
	return nil
}

func (sps *SpamProtectionService) detectVPN(ip string) *SpamTrigger {
	// This would integrate with VPN detection services in production
	// For now, return nil (placeholder)
	return nil
}

func (sps *SpamProtectionService) shouldRequireCaptcha(spamScore float64, config *FormSpamConfig) bool {
	if config.CustomConfig == nil {
		return false
	}
	
	if !config.CustomConfig.RequireCaptcha {
		return false
	}
	
	// Require CAPTCHA if spam score is moderate but not high enough to block
	return spamScore >= 0.3 && spamScore < config.CustomConfig.BlockThreshold
}

func (sps *SpamProtectionService) calculateConfidence(triggers []SpamTrigger, spamScore float64) float64 {
	if len(triggers) == 0 {
		return 0.0
	}
	
	// Confidence increases with number of high-severity triggers
	highSeverityCount := 0
	for _, trigger := range triggers {
		if trigger.Severity == "high" || trigger.Severity == "critical" {
			highSeverityCount++
		}
	}
	
	// Base confidence on spam score and high-severity trigger count
	confidence := math.Min(spamScore + float64(highSeverityCount)*0.1, 0.95)
	return confidence
}

func (sps *SpamProtectionService) getScoreSeverity(score float64) string {
	if score >= 0.8 {
		return "critical"
	} else if score >= 0.6 {
		return "high"
	} else if score >= 0.3 {
		return "medium"
	}
	return "low"
}

func (sps *SpamProtectionService) storeAnalysisResult(formID string, data map[string]interface{}, 
	metadata map[string]interface{}, result *SpamDetectionResult) {
	
	// Store analysis result for machine learning improvement
	query := `
		INSERT INTO spam_analysis_logs (
			id, form_id, spam_score, confidence, action, triggers_count, 
			is_spam, created_at, metadata
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	
	triggersJSON, _ := json.Marshal(result.Triggers)
	metadataJSON, _ := json.Marshal(result.Metadata)
	
	sps.db.Exec(query, uuid.New(), formID, result.SpamScore, result.Confidence, 
		result.Action, len(result.Triggers), result.IsSpam, time.Now(), 
		string(metadataJSON))
}

func (sps *SpamProtectionService) sendWebhookNotification(formID string, result *SpamDetectionResult,
	data map[string]interface{}, metadata map[string]interface{}) {
	
	// Send webhook notification for blocked/quarantined submissions
	// Implementation would send HTTP POST to configured webhook URL
	log.Printf("Webhook notification: Form %s, Action: %s, Score: %.3f", 
		formID, result.Action, result.SpamScore)
}

func (sps *SpamProtectionService) getFormConfig(formID string) (*FormSpamConfig, error) {
	// Load form-specific configuration from database
	query := `
		SELECT config FROM form_spam_configs 
		WHERE form_id = ? AND enabled = 1
	`
	
	var configJSON string
	err := sps.db.QueryRow(query, formID).Scan(&configJSON)
	if err != nil {
		// Return default configuration if not found
		return &FormSpamConfig{
			FormID:       formID,
			Enabled:      true,
			Level:        sps.globalConfig.DefaultLevel,
			CustomConfig: sps.globalConfig,
		}, nil
	}
	
	var config FormSpamConfig
	if err := json.Unmarshal([]byte(configJSON), &config); err != nil {
		return nil, err
	}
	
	return &config, nil
}

func (sps *SpamProtectionService) loadConfiguration() {
	// Load global configuration from database or environment
	query := `SELECT config FROM spam_protection_config WHERE id = 'global' AND enabled = 1`
	
	var configJSON string
	err := sps.db.QueryRow(query).Scan(&configJSON)
	if err != nil {
		// Use default configuration
		return
	}
	
	var config SpamProtectionConfig
	if err := json.Unmarshal([]byte(configJSON), &config); err == nil {
		sps.globalConfig = &config
		// Reinitialize CAPTCHA service with new config
		sps.captchaService = utils.NewCaptchaService(config.CaptchaConfig)
	}
}

// Public API methods

// UpdateGlobalConfig updates the global spam protection configuration
func (sps *SpamProtectionService) UpdateGlobalConfig(config *SpamProtectionConfig) error {
	configJSON, err := json.Marshal(config)
	if err != nil {
		return err
	}
	
	query := `
		INSERT INTO spam_protection_config (id, config, enabled, updated_at)
		VALUES ('global', ?, 1, ?)
		ON DUPLICATE KEY UPDATE config = VALUES(config), updated_at = VALUES(updated_at)
	`
	
	_, err = sps.db.Exec(query, string(configJSON), time.Now())
	if err != nil {
		return err
	}
	
	sps.globalConfig = config
	sps.captchaService = utils.NewCaptchaService(config.CaptchaConfig)
	
	return nil
}

// UpdateFormConfig updates spam protection configuration for a specific form
func (sps *SpamProtectionService) UpdateFormConfig(config *FormSpamConfig) error {
	configJSON, err := json.Marshal(config)
	if err != nil {
		return err
	}
	
	query := `
		INSERT INTO form_spam_configs (form_id, config, enabled, updated_at)
		VALUES (?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE config = VALUES(config), enabled = VALUES(enabled), updated_at = VALUES(updated_at)
	`
	
	_, err = sps.db.Exec(query, config.FormID, string(configJSON), config.Enabled, time.Now())
	return err
}

// GetSpamStatistics returns spam detection statistics
func (sps *SpamProtectionService) GetSpamStatistics(formID string, days int) (map[string]interface{}, error) {
	query := `
		SELECT 
			COUNT(*) as total,
			SUM(CASE WHEN is_spam = 1 THEN 1 ELSE 0 END) as spam_count,
			AVG(spam_score) as avg_spam_score,
			action,
			DATE(created_at) as date
		FROM spam_analysis_logs 
		WHERE form_id = ? AND created_at >= DATE_SUB(NOW(), INTERVAL ? DAY)
		GROUP BY action, DATE(created_at)
		ORDER BY date DESC
	`
	
	rows, err := sps.db.Query(query, formID, days)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	stats := make(map[string]interface{})
	dailyStats := []map[string]interface{}{}
	
	for rows.Next() {
		var total, spamCount int
		var avgScore float64
		var action, date string
		
		err := rows.Scan(&total, &spamCount, &avgScore, &action, &date)
		if err != nil {
			continue
		}
		
		dailyStats = append(dailyStats, map[string]interface{}{
			"date":           date,
			"total":          total,
			"spam_count":     spamCount,
			"avg_spam_score": avgScore,
			"action":         action,
		})
	}
	
	stats["daily_stats"] = dailyStats
	stats["form_id"] = formID
	stats["period_days"] = days
	
	return stats, nil
}