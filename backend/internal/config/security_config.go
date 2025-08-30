package config

import (
	"time"
)

// SecurityConfig contains all security-related configuration
type SecurityConfig struct {
	// File Upload Security
	FileUpload FileUploadSecurity `json:"file_upload"`
	
	// Rate Limiting
	RateLimit RateLimitConfig `json:"rate_limit"`
	
	// Authentication & Authorization
	Auth AuthSecurity `json:"auth"`
	
	// Content Security
	Content ContentSecurity `json:"content"`
	
	// Network Security
	Network NetworkSecurity `json:"network"`
	
	// Logging & Monitoring
	Logging LoggingSecurity `json:"logging"`
}

type FileUploadSecurity struct {
	MaxFileSize         int64    `json:"max_file_size"`          // Maximum file size in bytes (50MB default)
	MaxFilesPerRequest  int      `json:"max_files_per_request"`  // Maximum number of files per request
	MaxTotalSize        int64    `json:"max_total_size"`         // Maximum total size for bulk uploads
	AllowedExtensions   []string `json:"allowed_extensions"`     // Allowed file extensions
	AllowedMimeTypes    []string `json:"allowed_mime_types"`     // Allowed MIME types
	BlockedExtensions   []string `json:"blocked_extensions"`     // Explicitly blocked extensions
	ScanForMalware      bool     `json:"scan_for_malware"`       // Enable malware scanning
	QuarantineDir       string   `json:"quarantine_dir"`         // Directory for quarantined files
	VirusTotalAPIKey    string   `json:"virustotal_api_key"`     // VirusTotal API key for scanning
	EnableContentHash   bool     `json:"enable_content_hash"`    // Calculate file hashes for deduplication
}

type RateLimitConfig struct {
	EnableRateLimit     bool          `json:"enable_rate_limit"`
	RequestsPerMinute   int           `json:"requests_per_minute"`    // General API requests per minute
	SubmissionsPerHour  int           `json:"submissions_per_hour"`   // Form submissions per hour per IP
	UploadsPerHour      int           `json:"uploads_per_hour"`       // File uploads per hour per IP
	BurstSize           int           `json:"burst_size"`             // Burst allowance
	CleanupInterval     time.Duration `json:"cleanup_interval"`       // How often to clean old rate limiters
	BlockDuration       time.Duration `json:"block_duration"`         // How long to block after rate limit hit
}

type AuthSecurity struct {
	EnableCSRFProtection    bool          `json:"enable_csrf_protection"`
	CSRFTokenTimeout        time.Duration `json:"csrf_token_timeout"`
	SessionTimeout          time.Duration `json:"session_timeout"`
	MaxLoginAttempts        int           `json:"max_login_attempts"`
	LoginAttemptWindow      time.Duration `json:"login_attempt_window"`
	LockoutDuration         time.Duration `json:"lockout_duration"`
	RequireStrongPasswords  bool          `json:"require_strong_passwords"`
	PasswordMinLength       int           `json:"password_min_length"`
	PasswordRequireSpecial  bool          `json:"password_require_special"`
	EnableTwoFactor         bool          `json:"enable_two_factor"`
}

type ContentSecurity struct {
	EnableXSSProtection     bool     `json:"enable_xss_protection"`
	EnableSQLInjectionCheck bool     `json:"enable_sql_injection_check"`
	EnableSpamFilter        bool     `json:"enable_spam_filter"`
	SpamThreshold          float64   `json:"spam_threshold"`           // 0.0 to 1.0
	MaxFieldLength         int       `json:"max_field_length"`         // Maximum length for form fields
	AllowedHTMLTags        []string  `json:"allowed_html_tags"`        // Allowed HTML tags in content
	BlockedKeywords        []string  `json:"blocked_keywords"`         // Keywords that trigger blocking
	EnableHoneypot         bool      `json:"enable_honeypot"`          // Enable honeypot fields
}

type NetworkSecurity struct {
	EnableIPWhitelist      bool     `json:"enable_ip_whitelist"`
	WhitelistedIPs         []string `json:"whitelisted_ips"`
	EnableIPBlacklist      bool     `json:"enable_ip_blacklist"`
	BlacklistedIPs         []string `json:"blacklisted_ips"`
	EnableGeoblocking      bool     `json:"enable_geoblocking"`
	BlockedCountries       []string `json:"blocked_countries"`
	EnableBotDetection     bool     `json:"enable_bot_detection"`
	TrustedProxies         []string `json:"trusted_proxies"`
	EnableCORS             bool     `json:"enable_cors"`
	AllowedOrigins         []string `json:"allowed_origins"`
	EnableHTTPS            bool     `json:"enable_https"`
	HSTSMaxAge             int      `json:"hsts_max_age"`
}

type LoggingSecurity struct {
	EnableSecurityLogging  bool   `json:"enable_security_logging"`
	LogLevel              string  `json:"log_level"`                // debug, info, warn, error
	LogToFile             bool    `json:"log_to_file"`
	LogFile               string  `json:"log_file"`
	LogToDatabase         bool    `json:"log_to_database"`
	LogToSyslog           bool    `json:"log_to_syslog"`
	EnableAuditLog        bool    `json:"enable_audit_log"`
	EnableAlerts          bool    `json:"enable_alerts"`
	AlertThreshold        int     `json:"alert_threshold"`          // Number of threats before alert
	WebhookURL            string  `json:"webhook_url"`              // Webhook for security alerts
	EnableMetrics         bool    `json:"enable_metrics"`
}

// GetDefaultSecurityConfig returns a secure default configuration
func GetDefaultSecurityConfig() SecurityConfig {
	return SecurityConfig{
		FileUpload: FileUploadSecurity{
			MaxFileSize:        50 * 1024 * 1024, // 50MB
			MaxFilesPerRequest: 10,
			MaxTotalSize:       100 * 1024 * 1024, // 100MB
			AllowedExtensions: []string{
				".jpg", ".jpeg", ".png", ".gif", ".webp", ".svg",
				".pdf", ".doc", ".docx", ".xls", ".xlsx", ".ppt", ".pptx",
				".txt", ".csv", ".json", ".xml",
				".zip", ".rar", ".7z",
				".mp3", ".wav", ".mp4", ".webm", ".mov",
			},
			AllowedMimeTypes: []string{
				"image/jpeg", "image/png", "image/gif", "image/webp", "image/svg+xml",
				"application/pdf",
				"application/msword",
				"application/vnd.openxmlformats-officedocument.wordprocessingml.document",
				"application/vnd.ms-excel",
				"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
				"text/plain", "text/csv", "application/json", "application/xml",
				"application/zip", "application/x-zip-compressed",
				"audio/mpeg", "audio/wav", "video/mp4", "video/webm",
			},
			BlockedExtensions: []string{
				".exe", ".bat", ".cmd", ".scr", ".pif", ".com", ".vbs", ".js",
				".jar", ".msi", ".msp", ".hta", ".cpl", ".msc", ".ps1", ".sh",
			},
			ScanForMalware:    true,
			QuarantineDir:     "./quarantine",
			EnableContentHash: true,
		},
		RateLimit: RateLimitConfig{
			EnableRateLimit:    true,
			RequestsPerMinute:  60,
			SubmissionsPerHour: 100,
			UploadsPerHour:     50,
			BurstSize:          10,
			CleanupInterval:    time.Hour,
			BlockDuration:      15 * time.Minute,
		},
		Auth: AuthSecurity{
			EnableCSRFProtection:   true,
			CSRFTokenTimeout:       time.Hour,
			SessionTimeout:         24 * time.Hour,
			MaxLoginAttempts:       5,
			LoginAttemptWindow:     15 * time.Minute,
			LockoutDuration:        30 * time.Minute,
			RequireStrongPasswords: true,
			PasswordMinLength:      8,
			PasswordRequireSpecial: true,
			EnableTwoFactor:        false, // Can be enabled per user
		},
		Content: ContentSecurity{
			EnableXSSProtection:     true,
			EnableSQLInjectionCheck: true,
			EnableSpamFilter:        true,
			SpamThreshold:          0.7,
			MaxFieldLength:         10000, // 10KB per field
			AllowedHTMLTags: []string{
				"<b>", "<i>", "<u>", "<strong>", "<em>", "<br>", "<p>", "<a>",
			},
			BlockedKeywords: []string{
				"viagra", "casino", "lottery", "bitcoin", "crypto", "loan",
				"make money fast", "work from home", "free money",
			},
			EnableHoneypot: true,
		},
		Network: NetworkSecurity{
			EnableIPWhitelist:  false,
			WhitelistedIPs:     []string{},
			EnableIPBlacklist:  true,
			BlacklistedIPs:     []string{},
			EnableGeoblocking:  false,
			BlockedCountries:   []string{},
			EnableBotDetection: true,
			TrustedProxies:     []string{"127.0.0.1", "::1"},
			EnableCORS:         true,
			AllowedOrigins:     []string{"*"}, // Configure based on your needs
			EnableHTTPS:        true,
			HSTSMaxAge:         31536000, // 1 year
		},
		Logging: LoggingSecurity{
			EnableSecurityLogging: true,
			LogLevel:             "info",
			LogToFile:            true,
			LogFile:              "./logs/security.log",
			LogToDatabase:        true,
			LogToSyslog:          false,
			EnableAuditLog:       true,
			EnableAlerts:         true,
			AlertThreshold:       5,
			WebhookURL:           "",
			EnableMetrics:        true,
		},
	}
}

// GetProductionSecurityConfig returns a production-ready configuration
func GetProductionSecurityConfig() SecurityConfig {
	config := GetDefaultSecurityConfig()
	
	// More restrictive settings for production
	config.RateLimit.RequestsPerMinute = 30
	config.RateLimit.SubmissionsPerHour = 50
	config.RateLimit.UploadsPerHour = 20
	
	config.FileUpload.MaxFileSize = 10 * 1024 * 1024 // 10MB
	config.FileUpload.MaxFilesPerRequest = 5
	config.FileUpload.MaxTotalSize = 50 * 1024 * 1024 // 50MB
	
	config.Content.SpamThreshold = 0.5 // More aggressive spam detection
	config.Content.MaxFieldLength = 5000 // 5KB per field
	
	config.Network.AllowedOrigins = []string{} // Must be explicitly configured
	config.Network.EnableGeoblocking = true
	config.Network.BlockedCountries = []string{
		// Add high-risk countries based on your threat model
	}
	
	config.Logging.LogLevel = "warn"
	config.Logging.AlertThreshold = 3
	
	return config
}

// GetDevelopmentSecurityConfig returns a development-friendly configuration
func GetDevelopmentSecurityConfig() SecurityConfig {
	config := GetDefaultSecurityConfig()
	
	// More lenient settings for development
	config.RateLimit.EnableRateLimit = false
	config.Auth.EnableCSRFProtection = false
	config.Network.EnableBotDetection = false
	config.Content.EnableSpamFilter = false
	
	config.Logging.LogLevel = "debug"
	config.Logging.EnableAlerts = false
	
	return config
}

// ValidateSecurityConfig validates the security configuration
func ValidateSecurityConfig(config SecurityConfig) []string {
	var errors []string
	
	// Validate file upload settings
	if config.FileUpload.MaxFileSize <= 0 {
		errors = append(errors, "file_upload.max_file_size must be greater than 0")
	}
	
	if config.FileUpload.MaxFileSize > 500*1024*1024 { // 500MB
		errors = append(errors, "file_upload.max_file_size should not exceed 500MB")
	}
	
	if config.FileUpload.MaxFilesPerRequest <= 0 {
		errors = append(errors, "file_upload.max_files_per_request must be greater than 0")
	}
	
	if config.FileUpload.MaxFilesPerRequest > 100 {
		errors = append(errors, "file_upload.max_files_per_request should not exceed 100")
	}
	
	// Validate rate limiting
	if config.RateLimit.EnableRateLimit {
		if config.RateLimit.RequestsPerMinute <= 0 {
			errors = append(errors, "rate_limit.requests_per_minute must be greater than 0")
		}
		
		if config.RateLimit.BurstSize < config.RateLimit.RequestsPerMinute {
			errors = append(errors, "rate_limit.burst_size should be >= requests_per_minute")
		}
	}
	
	// Validate authentication settings
	if config.Auth.PasswordMinLength < 6 {
		errors = append(errors, "auth.password_min_length should be at least 6")
	}
	
	if config.Auth.MaxLoginAttempts <= 0 {
		errors = append(errors, "auth.max_login_attempts must be greater than 0")
	}
	
	// Validate content security
	if config.Content.SpamThreshold < 0.0 || config.Content.SpamThreshold > 1.0 {
		errors = append(errors, "content.spam_threshold must be between 0.0 and 1.0")
	}
	
	if config.Content.MaxFieldLength <= 0 {
		errors = append(errors, "content.max_field_length must be greater than 0")
	}
	
	// Validate logging settings
	validLogLevels := []string{"debug", "info", "warn", "error"}
	validLevel := false
	for _, level := range validLogLevels {
		if config.Logging.LogLevel == level {
			validLevel = true
			break
		}
	}
	if !validLevel {
		errors = append(errors, "logging.log_level must be one of: debug, info, warn, error")
	}
	
	return errors
}