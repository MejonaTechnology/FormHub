package services

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// Third-party integration configurations
type GoogleSheetsConfig struct {
	SpreadsheetID    string            `json:"spreadsheet_id"`
	WorksheetName    string            `json:"worksheet_name,omitempty"`
	CredentialsJSON  string            `json:"credentials_json"`
	FieldMappings    map[string]string `json:"field_mappings"`
	AppendOnly       bool              `json:"append_only"`
	CreateHeaders    bool              `json:"create_headers"`
}

type AirtableConfig struct {
	APIKey      string            `json:"api_key"`
	BaseID      string            `json:"base_id"`
	TableName   string            `json:"table_name"`
	ViewID      string            `json:"view_id,omitempty"`
	FieldMappings map[string]string `json:"field_mappings"`
}

type NotionConfig struct {
	APIKey       string            `json:"api_key"`
	DatabaseID   string            `json:"database_id"`
	FieldMappings map[string]string `json:"field_mappings"`
}

type SlackConfig struct {
	WebhookURL   string            `json:"webhook_url,omitempty"`
	BotToken     string            `json:"bot_token,omitempty"`
	Channel      string            `json:"channel"`
	Username     string            `json:"username,omitempty"`
	IconEmoji    string            `json:"icon_emoji,omitempty"`
	MessageTemplate string         `json:"message_template,omitempty"`
}

type TelegramConfig struct {
	BotToken    string `json:"bot_token"`
	ChatID      string `json:"chat_id"`
	MessageTemplate string `json:"message_template,omitempty"`
	ParseMode   string `json:"parse_mode,omitempty"` // Markdown, HTML
}

type DiscordConfig struct {
	WebhookURL      string `json:"webhook_url"`
	Username        string `json:"username,omitempty"`
	AvatarURL       string `json:"avatar_url,omitempty"`
	MessageTemplate string `json:"message_template,omitempty"`
}

type ZapierConfig struct {
	WebhookURL    string            `json:"webhook_url"`
	CustomFields  map[string]string `json:"custom_fields,omitempty"`
}

type CustomIntegration struct {
	ID            string            `json:"id"`
	Name          string            `json:"name"`
	Type          string            `json:"type"` // webhook, api, email
	Configuration map[string]interface{} `json:"configuration"`
	Enabled       bool              `json:"enabled"`
}

// NewIntegrationManager creates a new integration manager
func NewIntegrationManager(db *sql.DB, redis *redis.Client) *IntegrationManager {
	manager := &IntegrationManager{
		integrations: make(map[string]Integration),
		oauth:        NewOAuthManager(),
		templates:    NewTemplateManager(),
		marketplace:  NewIntegrationMarketplace(),
	}
	
	// Register built-in integrations
	manager.registerIntegration(NewGoogleSheetsIntegration(db, redis))
	manager.registerIntegration(NewAirtableIntegration(db, redis))
	manager.registerIntegration(NewNotionIntegration(db, redis))
	manager.registerIntegration(NewSlackIntegration(db, redis))
	manager.registerIntegration(NewTelegramIntegration(db, redis))
	manager.registerIntegration(NewDiscordIntegration(db, redis))
	manager.registerIntegration(NewZapierIntegration(db, redis))
	
	return manager
}

// registerIntegration registers a new integration
func (im *IntegrationManager) registerIntegration(integration Integration) {
	im.integrations[integration.Name()] = integration
	log.Printf("Registered integration: %s", integration.Name())
}

// GetIntegration returns an integration by name
func (im *IntegrationManager) GetIntegration(name string) (Integration, bool) {
	integration, exists := im.integrations[name]
	return integration, exists
}

// ListIntegrations returns all available integrations
func (im *IntegrationManager) ListIntegrations() []Integration {
	integrations := make([]Integration, 0, len(im.integrations))
	for _, integration := range im.integrations {
		integrations = append(integrations, integration)
	}
	return integrations
}

// SendToIntegration sends data to a specific integration
func (im *IntegrationManager) SendToIntegration(name string, event *EnhancedWebhookEvent, config map[string]interface{}) error {
	integration, exists := im.GetIntegration(name)
	if !exists {
		return fmt.Errorf("integration '%s' not found", name)
	}
	
	// Validate configuration
	if err := integration.ValidateConfig(config); err != nil {
		return fmt.Errorf("configuration validation failed: %w", err)
	}
	
	// Send to integration
	return integration.Send(event, config)
}

// OAuth Manager Implementation

func NewOAuthManager() *OAuthManager {
	return &OAuthManager{
		configs: make(map[string]*OAuthConfig),
		tokens:  make(map[string]*OAuthToken),
	}
}

func (om *OAuthManager) SetOAuthConfig(service string, config *OAuthConfig) {
	om.mu.Lock()
	defer om.mu.Unlock()
	om.configs[service] = config
}

func (om *OAuthManager) GetOAuthConfig(service string) (*OAuthConfig, bool) {
	om.mu.RLock()
	defer om.mu.RUnlock()
	config, exists := om.configs[service]
	return config, exists
}

func (om *OAuthManager) StoreToken(service string, token *OAuthToken) {
	om.mu.Lock()
	defer om.mu.Unlock()
	om.tokens[service] = token
}

func (om *OAuthManager) GetToken(service string) (*OAuthToken, bool) {
	om.mu.RLock()
	defer om.mu.RUnlock()
	token, exists := om.tokens[service]
	
	// Check if token is expired and needs refresh
	if exists && token.ExpiresAt.Before(time.Now()) {
		// Token expired, try to refresh
		if refreshedToken, err := om.refreshToken(service, token); err == nil {
			om.mu.RUnlock()
			om.mu.Lock()
			om.tokens[service] = refreshedToken
			om.mu.Unlock()
			om.mu.RLock()
			return refreshedToken, true
		}
	}
	
	return token, exists
}

func (om *OAuthManager) refreshToken(service string, token *OAuthToken) (*OAuthToken, error) {
	config, exists := om.configs[service]
	if !exists {
		return nil, fmt.Errorf("OAuth config not found for service: %s", service)
	}
	
	oauth2Config := &oauth2.Config{
		ClientID:     config.ClientID,
		ClientSecret: config.ClientSecret,
		Endpoint: oauth2.Endpoint{
			AuthURL:  config.AuthURL,
			TokenURL: config.TokenURL,
		},
		RedirectURL: config.RedirectURL,
		Scopes:      config.Scopes,
	}
	
	oauth2Token := &oauth2.Token{
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		TokenType:    token.TokenType,
		Expiry:       token.ExpiresAt,
	}
	
	newToken, err := oauth2Config.TokenSource(nil, oauth2Token).Token()
	if err != nil {
		return nil, fmt.Errorf("failed to refresh token: %w", err)
	}
	
	return &OAuthToken{
		AccessToken:  newToken.AccessToken,
		RefreshToken: newToken.RefreshToken,
		TokenType:    newToken.TokenType,
		ExpiresAt:    newToken.Expiry,
	}, nil
}

// Template Manager Implementation

func NewTemplateManager() *TemplateManager {
	manager := &TemplateManager{
		templates: make(map[string]*PayloadTemplate),
	}
	
	// Load default templates
	manager.loadDefaultTemplates()
	
	return manager
}

func (tm *TemplateManager) loadDefaultTemplates() {
	// Slack message template
	tm.templates["slack_default"] = &PayloadTemplate{
		ID:          "slack_default",
		Name:        "Default Slack Message",
		Description: "Standard Slack notification template",
		Template:    `{"text": "New form submission received from {{.FormID}}", "blocks": [{"type": "section", "text": {"type": "mrkdwn", "text": "*Form:* {{.FormID}}\n*Time:* {{.Timestamp.Format \"2006-01-02 15:04:05\"}}\n*Data:* {{range $key, $value := .Data}}{{$key}}: {{$value}}\n{{end}}"}}]}`,
		OutputType:  "json",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	
	// Discord webhook template
	tm.templates["discord_default"] = &PayloadTemplate{
		ID:          "discord_default",
		Name:        "Default Discord Webhook",
		Description: "Standard Discord webhook template",
		Template:    `{"content": "New form submission", "embeds": [{"title": "Form Submission", "description": "Form ID: {{.FormID}}", "fields": [{{range $i, $key := .Data}}{{if $i}},{{end}}{"name": "{{$key}}", "value": "{{index $.Data $key}}", "inline": true}{{end}}], "timestamp": "{{.Timestamp.Format \"2006-01-02T15:04:05Z\"}}"}]}`,
		OutputType:  "json",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	
	// Email template
	tm.templates["email_default"] = &PayloadTemplate{
		ID:          "email_default",
		Name:        "Default Email Notification",
		Description: "Standard email notification template",
		Template:    `{"subject": "New Form Submission - {{.FormID}}", "body": "A new form submission has been received.\n\nForm ID: {{.FormID}}\nSubmission ID: {{.SubmissionID}}\nTimestamp: {{.Timestamp.Format \"2006-01-02 15:04:05\"}}\n\nData:\n{{range $key, $value := .Data}}{{$key}}: {{$value}}\n{{end}}"}`,
		OutputType:  "json",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}

func (tm *TemplateManager) GetTemplate(id string) (*PayloadTemplate, bool) {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	template, exists := tm.templates[id]
	return template, exists
}

func (tm *TemplateManager) CreateTemplate(template *PayloadTemplate) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	
	template.CreatedAt = time.Now()
	template.UpdatedAt = time.Now()
	tm.templates[template.ID] = template
	
	return nil
}

func (tm *TemplateManager) ListTemplates() []*PayloadTemplate {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	
	templates := make([]*PayloadTemplate, 0, len(tm.templates))
	for _, template := range tm.templates {
		templates = append(templates, template)
	}
	
	return templates
}

// Integration Marketplace Implementation

func NewIntegrationMarketplace() *IntegrationMarketplace {
	marketplace := &IntegrationMarketplace{
		integrations: make(map[string]*MarketplaceIntegration),
		categories:   make(map[string][]string),
	}
	
	// Load marketplace integrations
	marketplace.loadMarketplaceIntegrations()
	
	return marketplace
}

func (imp *IntegrationMarketplace) loadMarketplaceIntegrations() {
	// Popular integrations
	integrations := []*MarketplaceIntegration{
		{
			ID:          "google_sheets",
			Name:        "Google Sheets",
			Description: "Send form submissions directly to Google Sheets",
			Category:    "productivity",
			Version:     "1.0.0",
			Author:      "FormHub",
			Icon:        "https://cdn.formhub.io/icons/google-sheets.svg",
			Tags:        []string{"sheets", "google", "productivity", "data"},
			Popular:     true,
			Featured:    true,
			Downloads:   15420,
			Rating:      4.8,
			CreatedAt:   time.Now().AddDate(0, -6, 0),
			UpdatedAt:   time.Now().AddDate(0, -1, 0),
		},
		{
			ID:          "slack",
			Name:        "Slack",
			Description: "Get notified in Slack channels when forms are submitted",
			Category:    "communication",
			Version:     "2.1.0",
			Author:      "FormHub",
			Icon:        "https://cdn.formhub.io/icons/slack.svg",
			Tags:        []string{"slack", "notification", "communication", "team"},
			Popular:     true,
			Featured:    true,
			Downloads:   12847,
			Rating:      4.7,
			CreatedAt:   time.Now().AddDate(0, -8, 0),
			UpdatedAt:   time.Now().AddDate(0, 0, -15),
		},
		{
			ID:          "airtable",
			Name:        "Airtable",
			Description: "Organize form submissions in Airtable bases",
			Category:    "database",
			Version:     "1.5.0",
			Author:      "FormHub",
			Icon:        "https://cdn.formhub.io/icons/airtable.svg",
			Tags:        []string{"airtable", "database", "organization", "crm"},
			Popular:     true,
			Featured:    false,
			Downloads:   8932,
			Rating:      4.6,
			CreatedAt:   time.Now().AddDate(0, -4, 0),
			UpdatedAt:   time.Now().AddDate(0, 0, -7),
		},
		{
			ID:          "notion",
			Name:        "Notion",
			Description: "Add form submissions to Notion databases",
			Category:    "productivity",
			Version:     "1.2.0",
			Author:      "FormHub",
			Icon:        "https://cdn.formhub.io/icons/notion.svg",
			Tags:        []string{"notion", "database", "productivity", "organization"},
			Popular:     false,
			Featured:    true,
			Downloads:   5621,
			Rating:      4.5,
			CreatedAt:   time.Now().AddDate(0, -3, 0),
			UpdatedAt:   time.Now().AddDate(0, 0, -3),
		},
		{
			ID:          "discord",
			Name:        "Discord",
			Description: "Send form notifications to Discord channels",
			Category:    "communication",
			Version:     "1.0.0",
			Author:      "FormHub",
			Icon:        "https://cdn.formhub.io/icons/discord.svg",
			Tags:        []string{"discord", "notification", "gaming", "community"},
			Popular:     false,
			Featured:    false,
			Downloads:   3247,
			Rating:      4.4,
			CreatedAt:   time.Now().AddDate(0, -2, 0),
			UpdatedAt:   time.Now().AddDate(0, 0, -10),
		},
		{
			ID:          "zapier",
			Name:        "Zapier",
			Description: "Connect to thousands of apps through Zapier",
			Category:    "automation",
			Version:     "2.0.0",
			Author:      "FormHub",
			Icon:        "https://cdn.formhub.io/icons/zapier.svg",
			Tags:        []string{"zapier", "automation", "integration", "workflow"},
			Popular:     true,
			Featured:    true,
			Downloads:   18932,
			Rating:      4.9,
			CreatedAt:   time.Now().AddDate(0, -10, 0),
			UpdatedAt:   time.Now().AddDate(0, 0, -2),
		},
		{
			ID:          "telegram",
			Name:        "Telegram",
			Description: "Get form notifications in Telegram chats",
			Category:    "communication",
			Version:     "1.1.0",
			Author:      "FormHub",
			Icon:        "https://cdn.formhub.io/icons/telegram.svg",
			Tags:        []string{"telegram", "notification", "messaging", "mobile"},
			Popular:     false,
			Featured:    false,
			Downloads:   2103,
			Rating:      4.3,
			CreatedAt:   time.Now().AddDate(0, -1, 0),
			UpdatedAt:   time.Now().AddDate(0, 0, -5),
		},
	}
	
	// Store integrations
	for _, integration := range integrations {
		imp.integrations[integration.ID] = integration
		
		// Update categories
		if _, exists := imp.categories[integration.Category]; !exists {
			imp.categories[integration.Category] = make([]string, 0)
		}
		imp.categories[integration.Category] = append(imp.categories[integration.Category], integration.ID)
	}
}

func (imp *IntegrationMarketplace) GetIntegration(id string) (*MarketplaceIntegration, bool) {
	imp.mu.RLock()
	defer imp.mu.RUnlock()
	integration, exists := imp.integrations[id]
	return integration, exists
}

func (imp *IntegrationMarketplace) ListIntegrations(filter *MarketplaceFilter) []*MarketplaceIntegration {
	imp.mu.RLock()
	defer imp.mu.RUnlock()
	
	integrations := make([]*MarketplaceIntegration, 0)
	
	for _, integration := range imp.integrations {
		// Apply filters
		if filter != nil {
			if filter.Category != "" && integration.Category != filter.Category {
				continue
			}
			if filter.Popular && !integration.Popular {
				continue
			}
			if filter.Featured && !integration.Featured {
				continue
			}
			if filter.MinRating > 0 && integration.Rating < filter.MinRating {
				continue
			}
		}
		
		integrations = append(integrations, integration)
	}
	
	// Sort integrations
	if filter != nil && filter.SortBy != "" {
		imp.sortIntegrations(integrations, filter.SortBy, filter.SortOrder)
	}
	
	// Apply limit
	if filter != nil && filter.Limit > 0 && len(integrations) > filter.Limit {
		integrations = integrations[:filter.Limit]
	}
	
	return integrations
}

func (imp *IntegrationMarketplace) GetCategories() map[string][]string {
	imp.mu.RLock()
	defer imp.mu.RUnlock()
	
	// Return a copy to prevent external modification
	categories := make(map[string][]string)
	for category, integrations := range imp.categories {
		categories[category] = make([]string, len(integrations))
		copy(categories[category], integrations)
	}
	
	return categories
}

func (imp *IntegrationMarketplace) IncrementDownloads(id string) error {
	imp.mu.Lock()
	defer imp.mu.Unlock()
	
	if integration, exists := imp.integrations[id]; exists {
		integration.Downloads++
		integration.UpdatedAt = time.Now()
		return nil
	}
	
	return fmt.Errorf("integration not found: %s", id)
}

func (imp *IntegrationMarketplace) sortIntegrations(integrations []*MarketplaceIntegration, sortBy, sortOrder string) {
	// Implementation of sorting logic
	// For brevity, this is a placeholder
	log.Printf("Sorting %d integrations by %s (%s)", len(integrations), sortBy, sortOrder)
}

// MarketplaceFilter defines filtering options for marketplace integrations
type MarketplaceFilter struct {
	Category  string  `json:"category,omitempty"`
	Popular   bool    `json:"popular,omitempty"`
	Featured  bool    `json:"featured,omitempty"`
	MinRating float64 `json:"min_rating,omitempty"`
	SortBy    string  `json:"sort_by,omitempty"`    // name, rating, downloads, created_at
	SortOrder string  `json:"sort_order,omitempty"` // asc, desc
	Limit     int     `json:"limit,omitempty"`
}

// Helper function to create Google Sheets OAuth2 config
func CreateGoogleSheetsOAuthConfig(clientID, clientSecret, redirectURL string) *oauth2.Config {
	return &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		Scopes: []string{
			"https://www.googleapis.com/auth/spreadsheets",
			"https://www.googleapis.com/auth/drive.file",
		},
		Endpoint: google.Endpoint,
	}
}

// Integration validation helpers

func ValidateGoogleSheetsConfig(config map[string]interface{}) error {
	required := []string{"spreadsheet_id", "credentials_json"}
	for _, field := range required {
		if _, exists := config[field]; !exists {
			return fmt.Errorf("missing required field: %s", field)
		}
	}
	
	// Validate spreadsheet ID format
	if spreadsheetID, ok := config["spreadsheet_id"].(string); ok {
		if len(spreadsheetID) == 0 {
			return fmt.Errorf("spreadsheet_id cannot be empty")
		}
	}
	
	return nil
}

func ValidateSlackConfig(config map[string]interface{}) error {
	// Must have either webhook_url or bot_token
	hasWebhookURL := false
	hasBotToken := false
	
	if webhookURL, exists := config["webhook_url"]; exists {
		if str, ok := webhookURL.(string); ok && len(str) > 0 {
			hasWebhookURL = true
		}
	}
	
	if botToken, exists := config["bot_token"]; exists {
		if str, ok := botToken.(string); ok && len(str) > 0 {
			hasBotToken = true
		}
	}
	
	if !hasWebhookURL && !hasBotToken {
		return fmt.Errorf("either webhook_url or bot_token is required")
	}
	
	// Channel is required
	if _, exists := config["channel"]; !exists {
		return fmt.Errorf("channel is required")
	}
	
	return nil
}

func ValidateAirtableConfig(config map[string]interface{}) error {
	required := []string{"api_key", "base_id", "table_name"}
	for _, field := range required {
		if _, exists := config[field]; !exists {
			return fmt.Errorf("missing required field: %s", field)
		}
	}
	return nil
}

func ValidateNotionConfig(config map[string]interface{}) error {
	required := []string{"api_key", "database_id"}
	for _, field := range required {
		if _, exists := config[field]; !exists {
			return fmt.Errorf("missing required field: %s", field)
		}
	}
	return nil
}

func ValidateTelegramConfig(config map[string]interface{}) error {
	required := []string{"bot_token", "chat_id"}
	for _, field := range required {
		if _, exists := config[field]; !exists {
			return fmt.Errorf("missing required field: %s", field)
		}
	}
	return nil
}

func ValidateDiscordConfig(config map[string]interface{}) error {
	if _, exists := config["webhook_url"]; !exists {
		return fmt.Errorf("webhook_url is required")
	}
	return nil
}

func ValidateZapierConfig(config map[string]interface{}) error {
	if _, exists := config["webhook_url"]; !exists {
		return fmt.Errorf("webhook_url is required")
	}
	return nil
}