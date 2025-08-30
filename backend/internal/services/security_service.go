package services

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"formhub/internal/models"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type SecurityService struct {
	db               *sql.DB
	redis            *redis.Client
	maxRequestsPerIP int
	rateLimitWindow  time.Duration
	quarantineDir    string
}

func NewSecurityService(db *sql.DB, redis *redis.Client, quarantineDir string) *SecurityService {
	// Ensure quarantine directory exists
	os.MkdirAll(quarantineDir, 0700) // Restricted permissions
	
	return &SecurityService{
		db:               db,
		redis:            redis,
		maxRequestsPerIP: 100,                // Max 100 requests per IP per window
		rateLimitWindow:  15 * time.Minute,   // 15-minute window
		quarantineDir:    quarantineDir,
	}
}

// SecurityCheckResult contains the result of security validation
type SecurityCheckResult struct {
	IsSecure     bool                   `json:"is_secure"`
	Threats      []SecurityThreat       `json:"threats,omitempty"`
	RiskScore    float64               `json:"risk_score"`
	Actions      []SecurityAction       `json:"actions,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

type SecurityThreat struct {
	Type        string  `json:"type"`
	Description string  `json:"description"`
	Severity    string  `json:"severity"` // low, medium, high, critical
	Score       float64 `json:"score"`
}

type SecurityAction struct {
	Action      string `json:"action"`      // block, quarantine, log, alert
	Description string `json:"description"`
}

// ValidateFileUpload performs comprehensive security validation on uploaded files
func (s *SecurityService) ValidateFileUpload(filePath, originalName, contentType string, size int64) (*SecurityCheckResult, error) {
	result := &SecurityCheckResult{
		IsSecure:  true,
		Threats:   []SecurityThreat{},
		RiskScore: 0.0,
		Actions:   []SecurityAction{},
		Metadata:  make(map[string]interface{}),
	}

	// 1. File size validation
	if size > 50*1024*1024 { // 50MB limit
		s.addThreat(result, "oversized_file", "File size exceeds maximum allowed limit", "medium", 0.3)
	}

	// 2. File extension validation
	ext := strings.ToLower(filepath.Ext(originalName))
	if !s.isAllowedExtension(ext) {
		s.addThreat(result, "dangerous_extension", fmt.Sprintf("File extension %s is not allowed", ext), "high", 0.7)
	}

	// 3. MIME type validation
	if !s.isAllowedMimeType(contentType) {
		s.addThreat(result, "dangerous_mimetype", fmt.Sprintf("MIME type %s is not allowed", contentType), "high", 0.7)
	}

	// 4. File content analysis
	contentThreats, err := s.analyzeFileContent(filePath, contentType)
	if err != nil {
		s.addThreat(result, "content_analysis_failed", "Failed to analyze file content", "low", 0.2)
	} else {
		for _, threat := range contentThreats {
			result.Threats = append(result.Threats, threat)
			result.RiskScore += threat.Score
		}
	}

	// 5. Filename validation
	if s.hasSuspiciousFilename(originalName) {
		s.addThreat(result, "suspicious_filename", "Filename contains suspicious patterns", "medium", 0.4)
	}

	// 6. Check for known malicious hashes
	hash, err := s.calculateFileHash(filePath)
	if err == nil {
		if s.isKnownMaliciousHash(hash) {
			s.addThreat(result, "known_malware", "File matches known malware signature", "critical", 1.0)
		}
		result.Metadata["file_hash"] = hash
	}

	// 7. Double extension check
	if s.hasDoubleExtension(originalName) {
		s.addThreat(result, "double_extension", "File has double extension (potential obfuscation)", "medium", 0.5)
	}

	// Determine final security status
	if result.RiskScore >= 0.7 {
		result.IsSecure = false
		result.Actions = append(result.Actions, SecurityAction{
			Action:      "quarantine",
			Description: "File moved to quarantine due to high risk score",
		})
	} else if result.RiskScore >= 0.4 {
		result.Actions = append(result.Actions, SecurityAction{
			Action:      "log",
			Description: "File flagged for monitoring due to medium risk score",
		})
	}

	return result, nil
}

// ValidateFormSubmission performs security validation on form data
func (s *SecurityService) ValidateFormSubmission(data map[string]interface{}, ipAddress, userAgent string) (*SecurityCheckResult, error) {
	result := &SecurityCheckResult{
		IsSecure:  true,
		Threats:   []SecurityThreat{},
		RiskScore: 0.0,
		Actions:   []SecurityAction{},
		Metadata:  make(map[string]interface{}),
	}

	// 1. Rate limiting check
	if s.isRateLimited(ipAddress) {
		s.addThreat(result, "rate_limit_exceeded", "Too many requests from this IP address", "high", 0.8)
		result.Actions = append(result.Actions, SecurityAction{
			Action:      "block",
			Description: "Request blocked due to rate limiting",
		})
	}

	// 2. SQL injection detection
	for field, value := range data {
		if valueStr, ok := value.(string); ok {
			if s.detectSQLInjection(valueStr) {
				s.addThreat(result, "sql_injection", fmt.Sprintf("Potential SQL injection in field %s", field), "critical", 1.0)
			}
		}
	}

	// 3. XSS detection
	for field, value := range data {
		if valueStr, ok := value.(string); ok {
			if s.detectXSS(valueStr) {
				s.addThreat(result, "xss_attempt", fmt.Sprintf("Potential XSS attack in field %s", field), "high", 0.9)
			}
		}
	}

	// 4. Spam detection
	spamScore := s.calculateSpamScore(data)
	if spamScore > 0.7 {
		s.addThreat(result, "spam_content", "Content appears to be spam", "medium", spamScore)
	}

	// 5. Suspicious user agent detection
	if s.isSuspiciousUserAgent(userAgent) {
		s.addThreat(result, "suspicious_user_agent", "User agent appears to be automated/suspicious", "low", 0.3)
	}

	// 6. Honeypot field check
	if honeypotValue, exists := data["_honeypot"]; exists && honeypotValue != "" {
		s.addThreat(result, "honeypot_triggered", "Honeypot field was filled (likely bot)", "high", 0.9)
	}

	// 7. Geolocation risk assessment
	geoRisk := s.assessGeolocationRisk(ipAddress)
	if geoRisk > 0.5 {
		s.addThreat(result, "high_risk_location", "Request from high-risk geographic location", "medium", geoRisk)
	}

	result.Metadata["spam_score"] = spamScore
	result.Metadata["geo_risk"] = geoRisk
	result.Metadata["user_agent"] = userAgent

	// Determine actions based on risk score
	if result.RiskScore >= 0.9 {
		result.IsSecure = false
		result.Actions = append(result.Actions, SecurityAction{
			Action:      "block",
			Description: "Request blocked due to critical security threats",
		})
	} else if result.RiskScore >= 0.7 {
		result.IsSecure = false
		result.Actions = append(result.Actions, SecurityAction{
			Action:      "quarantine",
			Description: "Submission quarantined for manual review",
		})
	} else if result.RiskScore >= 0.4 {
		result.Actions = append(result.Actions, SecurityAction{
			Action:      "log",
			Description: "Submission logged for monitoring",
		})
	}

	return result, nil
}

// QuarantineFile moves a potentially dangerous file to quarantine
func (s *SecurityService) QuarantineFile(filePath, reason string) error {
	filename := filepath.Base(filePath)
	quarantinePath := filepath.Join(s.quarantineDir, fmt.Sprintf("%s_%s_%s", 
		time.Now().Format("20060102_150405"), 
		uuid.New().String()[:8], 
		filename))

	// Move file to quarantine
	err := os.Rename(filePath, quarantinePath)
	if err != nil {
		return fmt.Errorf("failed to quarantine file: %w", err)
	}

	// Log quarantine event
	s.logSecurityEvent("file_quarantined", map[string]interface{}{
		"original_path":   filePath,
		"quarantine_path": quarantinePath,
		"reason":          reason,
		"timestamp":       time.Now(),
	})

	return nil
}

// Helper methods

func (s *SecurityService) addThreat(result *SecurityCheckResult, threatType, description, severity string, score float64) {
	threat := SecurityThreat{
		Type:        threatType,
		Description: description,
		Severity:    severity,
		Score:       score,
	}
	result.Threats = append(result.Threats, threat)
	result.RiskScore += score
}

func (s *SecurityService) isAllowedExtension(ext string) bool {
	allowedExtensions := map[string]bool{
		".jpg":  true, ".jpeg": true, ".png": true, ".gif": true, ".webp": true, ".svg": true,
		".pdf":  true, ".doc": true, ".docx": true, ".xls": true, ".xlsx": true, ".ppt": true, ".pptx": true,
		".txt":  true, ".csv": true, ".json": true, ".xml": true,
		".zip":  true, ".rar": true, ".7z": true,
		".mp3":  true, ".wav": true, ".mp4": true, ".webm": true, ".mov": true,
	}
	return allowedExtensions[ext]
}

func (s *SecurityService) isAllowedMimeType(mimeType string) bool {
	allowedTypes := map[string]bool{
		"image/jpeg":    true, "image/jpg": true, "image/png": true, "image/gif": true, "image/webp": true,
		"application/pdf": true,
		"application/msword": true,
		"application/vnd.openxmlformats-officedocument.wordprocessingml.document": true,
		"application/vnd.ms-excel": true,
		"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet": true,
		"text/plain": true, "text/csv": true, "application/json": true, "application/xml": true,
		"application/zip": true, "application/x-zip-compressed": true,
		"audio/mpeg": true, "audio/wav": true, "video/mp4": true, "video/webm": true,
	}
	return allowedTypes[mimeType]
}

func (s *SecurityService) analyzeFileContent(filePath, contentType string) ([]SecurityThreat, error) {
	threats := []SecurityThreat{}

	file, err := os.Open(filePath)
	if err != nil {
		return threats, err
	}
	defer file.Close()

	// Read first 512 bytes for magic number detection
	buffer := make([]byte, 512)
	n, err := file.Read(buffer)
	if err != nil && err != io.EOF {
		return threats, err
	}

	// Check if actual content matches declared MIME type
	detectedType := http.DetectContentType(buffer[:n])
	if !s.mimeTypesMatch(contentType, detectedType) {
		threats = append(threats, SecurityThreat{
			Type:        "mime_type_mismatch",
			Description: fmt.Sprintf("Declared type %s doesn't match detected type %s", contentType, detectedType),
			Severity:    "medium",
			Score:       0.5,
		})
	}

	// Check for embedded executables or scripts
	if s.containsExecutableContent(buffer[:n]) {
		threats = append(threats, SecurityThreat{
			Type:        "embedded_executable",
			Description: "File contains embedded executable content",
			Severity:    "high",
			Score:       0.8,
		})
	}

	// Check for suspicious content patterns
	if s.containsSuspiciousPatterns(buffer[:n]) {
		threats = append(threats, SecurityThreat{
			Type:        "suspicious_patterns",
			Description: "File contains suspicious content patterns",
			Severity:    "medium",
			Score:       0.4,
		})
	}

	return threats, nil
}

func (s *SecurityService) hasSuspiciousFilename(filename string) bool {
	suspicious := []string{
		"cmd", "bat", "exe", "scr", "vbs", "js", "jar", "com", "pif", "application", "gadget",
		"msi", "msp", "hta", "cpl", "msc", "jar", "ws", "wsf", "wsc", "wsh", "ps1", "ps1xml",
		"ps2", "ps2xml", "psc1", "psc2", "msh", "msh1", "msh2", "mshxml", "msh1xml", "msh2xml",
	}
	
	lowerName := strings.ToLower(filename)
	for _, pattern := range suspicious {
		if strings.Contains(lowerName, pattern) {
			return true
		}
	}
	return false
}

func (s *SecurityService) calculateFileHash(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

func (s *SecurityService) isKnownMaliciousHash(hash string) bool {
	// In production, this would check against threat intelligence databases
	// For now, maintain a local blacklist
	maliciousHashes := map[string]bool{
		// Add known bad hashes here
	}
	return maliciousHashes[hash]
}

func (s *SecurityService) hasDoubleExtension(filename string) bool {
	parts := strings.Split(filename, ".")
	return len(parts) > 2
}

func (s *SecurityService) isRateLimited(ipAddress string) bool {
	// Check current request count for IP
	key := fmt.Sprintf("rate_limit:%s", ipAddress)
	
	// This is a simplified implementation
	// In production, use proper sliding window rate limiting
	return false // Placeholder
}

func (s *SecurityService) detectSQLInjection(input string) bool {
	patterns := []string{
		"'", "\"", ";", "--", "/*", "*/", "xp_", "sp_", "union", "select", "insert", "update", "delete",
		"drop", "create", "alter", "exec", "execute", "script", "javascript:", "vbscript:", "onload",
		"onerror", "onclick", "onmouseover",
	}
	
	lowerInput := strings.ToLower(input)
	for _, pattern := range patterns {
		if strings.Contains(lowerInput, pattern) {
			return true
		}
	}
	return false
}

func (s *SecurityService) detectXSS(input string) bool {
	patterns := []string{
		"<script", "</script>", "javascript:", "vbscript:", "onload=", "onerror=", "onclick=",
		"onmouseover=", "<iframe", "<object", "<embed", "eval(", "alert(", "confirm(",
		"prompt(", "document.cookie", "document.write", "innerHTML", "outerHTML",
	}
	
	lowerInput := strings.ToLower(input)
	for _, pattern := range patterns {
		if strings.Contains(lowerInput, pattern) {
			return true
		}
	}
	return false
}

func (s *SecurityService) calculateSpamScore(data map[string]interface{}) float64 {
	score := 0.0
	
	for _, value := range data {
		if valueStr, ok := value.(string); ok {
			// URL count check
			urlCount := strings.Count(strings.ToLower(valueStr), "http://") + 
					   strings.Count(strings.ToLower(valueStr), "https://")
			if urlCount > 3 {
				score += 0.3
			}
			
			// Excessive capitalization
			if len(valueStr) > 20 {
				upperCount := 0
				for _, char := range valueStr {
					if char >= 'A' && char <= 'Z' {
						upperCount++
					}
				}
				if float64(upperCount)/float64(len(valueStr)) > 0.7 {
					score += 0.2
				}
			}
			
			// Spam keywords
			spamKeywords := []string{"viagra", "casino", "lottery", "winner", "congratulations", "urgent"}
			lowerValue := strings.ToLower(valueStr)
			for _, keyword := range spamKeywords {
				if strings.Contains(lowerValue, keyword) {
					score += 0.2
				}
			}
		}
	}
	
	if score > 1.0 {
		score = 1.0
	}
	
	return score
}

func (s *SecurityService) isSuspiciousUserAgent(userAgent string) bool {
	suspicious := []string{"bot", "crawler", "spider", "scraper", "automated", "curl", "wget", "python"}
	lowerUA := strings.ToLower(userAgent)
	
	for _, pattern := range suspicious {
		if strings.Contains(lowerUA, pattern) {
			return true
		}
	}
	
	return len(userAgent) < 10 || userAgent == ""
}

func (s *SecurityService) assessGeolocationRisk(ipAddress string) float64 {
	// This would integrate with geolocation services in production
	// Return risk score based on IP location
	return 0.0 // Placeholder
}

func (s *SecurityService) mimeTypesMatch(declared, detected string) bool {
	// Handle cases where detected type is more specific
	if strings.HasPrefix(detected, declared) || strings.HasPrefix(declared, detected) {
		return true
	}
	
	// Handle common aliases
	aliases := map[string][]string{
		"image/jpeg": {"image/jpg"},
		"image/jpg":  {"image/jpeg"},
	}
	
	if aliasTypes, exists := aliases[declared]; exists {
		for _, alias := range aliasTypes {
			if detected == alias {
				return true
			}
		}
	}
	
	return declared == detected
}

func (s *SecurityService) containsExecutableContent(data []byte) bool {
	// Check for PE/ELF/Mach-O headers
	patterns := [][]byte{
		{0x4D, 0x5A},             // PE header (MZ)
		{0x7F, 0x45, 0x4C, 0x46}, // ELF header
		{0xFE, 0xED, 0xFA, 0xCE}, // Mach-O 32-bit
		{0xFE, 0xED, 0xFA, 0xCF}, // Mach-O 64-bit
	}
	
	for _, pattern := range patterns {
		if len(data) >= len(pattern) {
			match := true
			for i, b := range pattern {
				if data[i] != b {
					match = false
					break
				}
			}
			if match {
				return true
			}
		}
	}
	
	return false
}

func (s *SecurityService) containsSuspiciousPatterns(data []byte) bool {
	// Check for suspicious strings that might indicate malware
	suspicious := []string{
		"eval(", "shell_exec", "system(", "exec(", "passthru", "base64_decode",
		"powershell", "cmd.exe", "/bin/sh", "wget", "curl",
	}
	
	dataStr := strings.ToLower(string(data))
	for _, pattern := range suspicious {
		if strings.Contains(dataStr, pattern) {
			return true
		}
	}
	
	return false
}

func (s *SecurityService) logSecurityEvent(eventType string, details map[string]interface{}) {
	// Log security event to database and/or external systems
	query := `
		INSERT INTO security_logs (id, event_type, details, created_at)
		VALUES (?, ?, ?, ?)
	`
	
	detailsJSON := ""
	// In production, properly marshal details to JSON
	
	s.db.Exec(query, uuid.New(), eventType, detailsJSON, time.Now())
}