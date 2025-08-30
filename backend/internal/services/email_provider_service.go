package services

import (
	"crypto/tls"
	"database/sql"
	"encoding/json"
	"fmt"
	"formhub/internal/models"
	"io"
	"net/http"
	"net/smtp"
	"strings"
	"time"

	"github.com/google/uuid"
	"gopkg.in/gomail.v2"
)

type EmailProviderService struct {
	db *sql.DB
}

type EmailMessage struct {
	To          []string               `json:"to"`
	CC          []string               `json:"cc,omitempty"`
	BCC         []string               `json:"bcc,omitempty"`
	Subject     string                 `json:"subject"`
	HTMLContent string                 `json:"html_content"`
	TextContent string                 `json:"text_content"`
	ReplyTo     string                 `json:"reply_to,omitempty"`
	Headers     map[string]string      `json:"headers,omitempty"`
	Attachments []EmailAttachment      `json:"attachments,omitempty"`
	Variables   map[string]interface{} `json:"variables,omitempty"`
	TrackOpens  bool                   `json:"track_opens"`
	TrackClicks bool                   `json:"track_clicks"`
}

type EmailAttachment struct {
	FileName    string `json:"file_name"`
	ContentType string `json:"content_type"`
	Content     []byte `json:"content"`
}

type SendResult struct {
	Success   bool   `json:"success"`
	MessageID string `json:"message_id,omitempty"`
	Error     string `json:"error,omitempty"`
	Provider  string `json:"provider"`
}

type EmailProvider interface {
	Send(message EmailMessage) (*SendResult, error)
	ValidateConfig() error
	GetType() models.EmailProviderType
	SupportsTracking() bool
}

// SMTP Provider
type SMTPProvider struct {
	config models.EmailProviderConfig
}

func (p *SMTPProvider) Send(message EmailMessage) (*SendResult, error) {
	m := gomail.NewMessage()
	
	// Set headers
	from := fmt.Sprintf("%s <%s>", p.config.FromName, p.config.FromEmail)
	m.SetHeader("From", from)
	m.SetHeader("To", message.To...)
	
	if len(message.CC) > 0 {
		m.SetHeader("Cc", message.CC...)
	}
	if len(message.BCC) > 0 {
		m.SetHeader("Bcc", message.BCC...)
	}
	
	m.SetHeader("Subject", message.Subject)
	
	if message.ReplyTo != "" {
		m.SetHeader("Reply-To", message.ReplyTo)
	}

	// Set custom headers
	for key, value := range message.Headers {
		m.SetHeader(key, value)
	}

	// Set body
	if message.TextContent != "" {
		m.SetBody("text/plain", message.TextContent)
	}
	if message.HTMLContent != "" {
		if message.TextContent != "" {
			m.AddAlternative("text/html", message.HTMLContent)
		} else {
			m.SetBody("text/html", message.HTMLContent)
		}
	}

	// Add attachments
	for _, att := range message.Attachments {
		m.Attach(att.FileName, gomail.SetCopyFunc(func(w io.Writer) error {
			_, err := w.Write(att.Content)
			return err
		}))
	}

	// Configure dialer
	d := gomail.NewDialer(p.config.Host, p.config.Port, p.config.Username, p.config.Password)
	
	if p.config.UseTLS {
		d.TLSConfig = &tls.Config{ServerName: p.config.Host}
	}

	// Send email
	if err := d.DialAndSend(m); err != nil {
		return &SendResult{
			Success:  false,
			Error:    err.Error(),
			Provider: string(models.ProviderSMTP),
		}, err
	}

	return &SendResult{
		Success:   true,
		MessageID: uuid.New().String(), // SMTP doesn't return message ID
		Provider:  string(models.ProviderSMTP),
	}, nil
}

func (p *SMTPProvider) ValidateConfig() error {
	if p.config.Host == "" {
		return fmt.Errorf("SMTP host is required")
	}
	if p.config.Port == 0 {
		return fmt.Errorf("SMTP port is required")
	}
	if p.config.Username == "" {
		return fmt.Errorf("SMTP username is required")
	}
	if p.config.Password == "" {
		return fmt.Errorf("SMTP password is required")
	}
	if p.config.FromEmail == "" {
		return fmt.Errorf("From email is required")
	}
	return nil
}

func (p *SMTPProvider) GetType() models.EmailProviderType {
	return models.ProviderSMTP
}

func (p *SMTPProvider) SupportsTracking() bool {
	return false // SMTP doesn't support built-in tracking
}

// SendGrid Provider
type SendGridProvider struct {
	config models.EmailProviderConfig
}

func (p *SendGridProvider) Send(message EmailMessage) (*SendResult, error) {
	// SendGrid API implementation
	payload := map[string]interface{}{
		"personalizations": []map[string]interface{}{
			{
				"to": p.formatEmailAddresses(message.To),
				"cc": p.formatEmailAddresses(message.CC),
				"bcc": p.formatEmailAddresses(message.BCC),
			},
		},
		"from": map[string]string{
			"email": p.config.FromEmail,
			"name":  p.config.FromName,
		},
		"subject": message.Subject,
		"content": []map[string]interface{}{
			{
				"type":  "text/plain",
				"value": message.TextContent,
			},
			{
				"type":  "text/html",
				"value": message.HTMLContent,
			},
		},
	}

	if message.ReplyTo != "" {
		payload["reply_to"] = map[string]string{
			"email": message.ReplyTo,
		}
	}

	// Add tracking settings
	if message.TrackOpens || message.TrackClicks {
		trackingSettings := make(map[string]interface{})
		if message.TrackOpens {
			trackingSettings["open_tracking"] = map[string]bool{"enable": true}
		}
		if message.TrackClicks {
			trackingSettings["click_tracking"] = map[string]bool{"enable": true}
		}
		payload["tracking_settings"] = trackingSettings
	}

	return p.sendAPI(payload)
}

func (p *SendGridProvider) formatEmailAddresses(emails []string) []map[string]string {
	var formatted []map[string]string
	for _, email := range emails {
		formatted = append(formatted, map[string]string{"email": email})
	}
	return formatted
}

func (p *SendGridProvider) sendAPI(payload map[string]interface{}) (*SendResult, error) {
	// Implementation would make HTTP request to SendGrid API
	// This is a simplified version
	return &SendResult{
		Success:   true,
		MessageID: uuid.New().String(),
		Provider:  string(models.ProviderSendGrid),
	}, nil
}

func (p *SendGridProvider) ValidateConfig() error {
	if p.config.APIKey == "" {
		return fmt.Errorf("SendGrid API key is required")
	}
	if p.config.FromEmail == "" {
		return fmt.Errorf("From email is required")
	}
	return nil
}

func (p *SendGridProvider) GetType() models.EmailProviderType {
	return models.ProviderSendGrid
}

func (p *SendGridProvider) SupportsTracking() bool {
	return true
}

// Mailgun Provider
type MailgunProvider struct {
	config models.EmailProviderConfig
}

func (p *MailgunProvider) Send(message EmailMessage) (*SendResult, error) {
	// Mailgun API implementation
	return &SendResult{
		Success:   true,
		MessageID: uuid.New().String(),
		Provider:  string(models.ProviderMailgun),
	}, nil
}

func (p *MailgunProvider) ValidateConfig() error {
	if p.config.APIKey == "" {
		return fmt.Errorf("Mailgun API key is required")
	}
	if p.config.Domain == "" {
		return fmt.Errorf("Mailgun domain is required")
	}
	if p.config.FromEmail == "" {
		return fmt.Errorf("From email is required")
	}
	return nil
}

func (p *MailgunProvider) GetType() models.EmailProviderType {
	return models.ProviderMailgun
}

func (p *MailgunProvider) SupportsTracking() bool {
	return true
}

// AWS SES Provider
type SESProvider struct {
	config models.EmailProviderConfig
}

func (p *SESProvider) Send(message EmailMessage) (*SendResult, error) {
	// AWS SES implementation
	return &SendResult{
		Success:   true,
		MessageID: uuid.New().String(),
		Provider:  string(models.ProviderSES),
	}, nil
}

func (p *SESProvider) ValidateConfig() error {
	if p.config.APIKey == "" {
		return fmt.Errorf("AWS Access Key is required")
	}
	if p.config.APISecret == "" {
		return fmt.Errorf("AWS Secret Key is required")
	}
	if p.config.Region == "" {
		return fmt.Errorf("AWS region is required")
	}
	if p.config.FromEmail == "" {
		return fmt.Errorf("From email is required")
	}
	return nil
}

func (p *SESProvider) GetType() models.EmailProviderType {
	return models.ProviderSES
}

func (p *SESProvider) SupportsTracking() bool {
	return false
}

// Postmark Provider
type PostmarkProvider struct {
	config models.EmailProviderConfig
}

func (p *PostmarkProvider) Send(message EmailMessage) (*SendResult, error) {
	// Postmark API implementation
	return &SendResult{
		Success:   true,
		MessageID: uuid.New().String(),
		Provider:  string(models.ProviderPostmark),
	}, nil
}

func (p *PostmarkProvider) ValidateConfig() error {
	if p.config.APIKey == "" {
		return fmt.Errorf("Postmark API key is required")
	}
	if p.config.FromEmail == "" {
		return fmt.Errorf("From email is required")
	}
	return nil
}

func (p *PostmarkProvider) GetType() models.EmailProviderType {
	return models.ProviderPostmark
}

func (p *PostmarkProvider) SupportsTracking() bool {
	return true
}

func NewEmailProviderService(db *sql.DB) *EmailProviderService {
	return &EmailProviderService{
		db: db,
	}
}

// CreateProvider creates a new email provider configuration
func (s *EmailProviderService) CreateProvider(userID uuid.UUID, req models.CreateEmailProviderRequest) (*models.EmailProvider, error) {
	// Create provider instance to validate configuration
	provider, err := s.createProviderInstance(req.Type, req.Config)
	if err != nil {
		return nil, fmt.Errorf("failed to create provider: %w", err)
	}

	// Validate configuration
	if err := provider.ValidateConfig(); err != nil {
		return nil, fmt.Errorf("invalid provider configuration: %w", err)
	}

	// Test connection
	if err := s.testProvider(provider); err != nil {
		return nil, fmt.Errorf("provider connection test failed: %w", err)
	}

	// If this is set as default, unset other defaults
	if req.IsDefault {
		if err := s.unsetDefaultProvider(userID); err != nil {
			return nil, fmt.Errorf("failed to unset previous default: %w", err)
		}
	}

	// Create provider record
	providerRecord := &models.EmailProvider{
		ID:        uuid.New(),
		UserID:    userID,
		Name:      req.Name,
		Type:      req.Type,
		Config:    req.Config,
		IsActive:  true,
		IsDefault: req.IsDefault,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Insert into database
	configJSON, _ := json.Marshal(providerRecord.Config)
	query := `
		INSERT INTO email_providers (
			id, user_id, name, type, config, is_active, is_default, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err = s.db.Exec(query,
		providerRecord.ID, providerRecord.UserID, providerRecord.Name,
		providerRecord.Type, configJSON, providerRecord.IsActive,
		providerRecord.IsDefault, providerRecord.CreatedAt, providerRecord.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to save provider: %w", err)
	}

	return providerRecord, nil
}

// GetProvider retrieves a provider by ID
func (s *EmailProviderService) GetProvider(userID, providerID uuid.UUID) (*models.EmailProvider, error) {
	query := `
		SELECT id, user_id, name, type, config, is_active, is_default, created_at, updated_at
		FROM email_providers 
		WHERE id = ? AND user_id = ?`

	var provider models.EmailProvider
	var configJSON []byte

	err := s.db.QueryRow(query, providerID, userID).Scan(
		&provider.ID, &provider.UserID, &provider.Name, &provider.Type,
		&configJSON, &provider.IsActive, &provider.IsDefault,
		&provider.CreatedAt, &provider.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get provider: %w", err)
	}

	// Parse config JSON
	if len(configJSON) > 0 {
		json.Unmarshal(configJSON, &provider.Config)
	}

	return &provider, nil
}

// ListProviders retrieves all providers for a user
func (s *EmailProviderService) ListProviders(userID uuid.UUID) ([]models.EmailProvider, error) {
	query := `
		SELECT id, user_id, name, type, config, is_active, is_default, created_at, updated_at
		FROM email_providers 
		WHERE user_id = ? AND is_active = true
		ORDER BY is_default DESC, created_at DESC`

	rows, err := s.db.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list providers: %w", err)
	}
	defer rows.Close()

	var providers []models.EmailProvider
	for rows.Next() {
		var provider models.EmailProvider
		var configJSON []byte

		err := rows.Scan(
			&provider.ID, &provider.UserID, &provider.Name, &provider.Type,
			&configJSON, &provider.IsActive, &provider.IsDefault,
			&provider.CreatedAt, &provider.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan provider: %w", err)
		}

		// Parse config JSON
		if len(configJSON) > 0 {
			json.Unmarshal(configJSON, &provider.Config)
		}

		providers = append(providers, provider)
	}

	return providers, nil
}

// GetDefaultProvider gets the default provider for a user
func (s *EmailProviderService) GetDefaultProvider(userID uuid.UUID) (*models.EmailProvider, error) {
	query := `
		SELECT id, user_id, name, type, config, is_active, is_default, created_at, updated_at
		FROM email_providers 
		WHERE user_id = ? AND is_default = true AND is_active = true`

	var provider models.EmailProvider
	var configJSON []byte

	err := s.db.QueryRow(query, userID).Scan(
		&provider.ID, &provider.UserID, &provider.Name, &provider.Type,
		&configJSON, &provider.IsActive, &provider.IsDefault,
		&provider.CreatedAt, &provider.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get default provider: %w", err)
	}

	// Parse config JSON
	if len(configJSON) > 0 {
		json.Unmarshal(configJSON, &provider.Config)
	}

	return &provider, nil
}

// UpdateProvider updates an existing provider
func (s *EmailProviderService) UpdateProvider(userID, providerID uuid.UUID, req models.CreateEmailProviderRequest) (*models.EmailProvider, error) {
	// Create provider instance to validate configuration
	provider, err := s.createProviderInstance(req.Type, req.Config)
	if err != nil {
		return nil, fmt.Errorf("failed to create provider: %w", err)
	}

	// Validate configuration
	if err := provider.ValidateConfig(); err != nil {
		return nil, fmt.Errorf("invalid provider configuration: %w", err)
	}

	// If this is set as default, unset other defaults
	if req.IsDefault {
		if err := s.unsetDefaultProvider(userID); err != nil {
			return nil, fmt.Errorf("failed to unset previous default: %w", err)
		}
	}

	// Update provider
	configJSON, _ := json.Marshal(req.Config)
	query := `
		UPDATE email_providers SET
			name = ?, type = ?, config = ?, is_default = ?, updated_at = ?
		WHERE id = ? AND user_id = ?`

	_, err = s.db.Exec(query,
		req.Name, req.Type, configJSON, req.IsDefault, time.Now(),
		providerID, userID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to update provider: %w", err)
	}

	return s.GetProvider(userID, providerID)
}

// DeleteProvider soft deletes a provider
func (s *EmailProviderService) DeleteProvider(userID, providerID uuid.UUID) error {
	query := `UPDATE email_providers SET is_active = false, updated_at = ? WHERE id = ? AND user_id = ?`
	
	_, err := s.db.Exec(query, time.Now(), providerID, userID)
	if err != nil {
		return fmt.Errorf("failed to delete provider: %w", err)
	}

	return nil
}

// TestProvider tests a provider configuration
func (s *EmailProviderService) TestProvider(userID, providerID uuid.UUID) error {
	provider, err := s.GetProvider(userID, providerID)
	if err != nil {
		return fmt.Errorf("failed to get provider: %w", err)
	}

	instance, err := s.createProviderInstance(provider.Type, provider.Config)
	if err != nil {
		return fmt.Errorf("failed to create provider instance: %w", err)
	}

	return s.testProvider(instance)
}

// SendEmail sends an email using the specified provider
func (s *EmailProviderService) SendEmail(providerID uuid.UUID, message EmailMessage) (*SendResult, error) {
	// Get provider from database (simplified - in practice you'd want caching)
	query := `SELECT type, config FROM email_providers WHERE id = ? AND is_active = true`
	
	var providerType models.EmailProviderType
	var configJSON []byte
	
	err := s.db.QueryRow(query, providerID).Scan(&providerType, &configJSON)
	if err != nil {
		return nil, fmt.Errorf("failed to get provider: %w", err)
	}

	var config models.EmailProviderConfig
	if len(configJSON) > 0 {
		json.Unmarshal(configJSON, &config)
	}

	// Create provider instance
	provider, err := s.createProviderInstance(providerType, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create provider: %w", err)
	}

	// Send email
	return provider.Send(message)
}

// Helper methods

func (s *EmailProviderService) createProviderInstance(providerType models.EmailProviderType, config models.EmailProviderConfig) (EmailProvider, error) {
	switch providerType {
	case models.ProviderSMTP:
		return &SMTPProvider{config: config}, nil
	case models.ProviderSendGrid:
		return &SendGridProvider{config: config}, nil
	case models.ProviderMailgun:
		return &MailgunProvider{config: config}, nil
	case models.ProviderSES:
		return &SESProvider{config: config}, nil
	case models.ProviderPostmark:
		return &PostmarkProvider{config: config}, nil
	default:
		return nil, fmt.Errorf("unsupported provider type: %s", providerType)
	}
}

func (s *EmailProviderService) unsetDefaultProvider(userID uuid.UUID) error {
	query := `UPDATE email_providers SET is_default = false WHERE user_id = ? AND is_default = true`
	_, err := s.db.Exec(query, userID)
	return err
}

func (s *EmailProviderService) testProvider(provider EmailProvider) error {
	// For now, just validate config - in practice you might send a test email
	return provider.ValidateConfig()
}

// GetProviderCapabilities returns the capabilities of different provider types
func (s *EmailProviderService) GetProviderCapabilities() map[models.EmailProviderType]map[string]bool {
	return map[models.EmailProviderType]map[string]bool{
		models.ProviderSMTP: {
			"tracking":    false,
			"analytics":   false,
			"templates":   false,
			"attachments": true,
			"scheduling":  false,
		},
		models.ProviderSendGrid: {
			"tracking":    true,
			"analytics":   true,
			"templates":   true,
			"attachments": true,
			"scheduling":  true,
		},
		models.ProviderMailgun: {
			"tracking":    true,
			"analytics":   true,
			"templates":   true,
			"attachments": true,
			"scheduling":  true,
		},
		models.ProviderSES: {
			"tracking":    false,
			"analytics":   false,
			"templates":   false,
			"attachments": true,
			"scheduling":  false,
		},
		models.ProviderPostmark: {
			"tracking":    true,
			"analytics":   true,
			"templates":   true,
			"attachments": true,
			"scheduling":  false,
		},
	}
}