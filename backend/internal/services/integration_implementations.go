package services

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/slack-go/slack"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

// Google Sheets Integration
type GoogleSheetsIntegration struct {
	db    *sql.DB
	redis *redis.Client
	name  string
}

func NewGoogleSheetsIntegration(db *sql.DB, redis *redis.Client) *GoogleSheetsIntegration {
	return &GoogleSheetsIntegration{
		db:    db,
		redis: redis,
		name:  "google_sheets",
	}
}

func (gsi *GoogleSheetsIntegration) Name() string {
	return gsi.name
}

func (gsi *GoogleSheetsIntegration) Authenticate(config map[string]interface{}) error {
	// Validate credentials JSON
	credentialsJSON, exists := config["credentials_json"]
	if !exists {
		return fmt.Errorf("credentials_json is required")
	}
	
	credentialsStr, ok := credentialsJSON.(string)
	if !ok {
		return fmt.Errorf("credentials_json must be a string")
	}
	
	// Test connection by creating a sheets service
	ctx := context.Background()
	service, err := sheets.NewService(ctx, option.WithCredentialsJSON([]byte(credentialsStr)))
	if err != nil {
		return fmt.Errorf("failed to create sheets service: %w", err)
	}
	
	// Test access to spreadsheet
	spreadsheetID := config["spreadsheet_id"].(string)
	_, err = service.Spreadsheets.Get(spreadsheetID).Do()
	if err != nil {
		return fmt.Errorf("failed to access spreadsheet: %w", err)
	}
	
	return nil
}

func (gsi *GoogleSheetsIntegration) Send(event *EnhancedWebhookEvent, config map[string]interface{}) error {
	ctx := context.Background()
	
	// Create sheets service
	credentialsJSON := config["credentials_json"].(string)
	service, err := sheets.NewService(ctx, option.WithCredentialsJSON([]byte(credentialsJSON)))
	if err != nil {
		return fmt.Errorf("failed to create sheets service: %w", err)
	}
	
	spreadsheetID := config["spreadsheet_id"].(string)
	worksheetName := "Sheet1" // Default
	if name, exists := config["worksheet_name"]; exists {
		worksheetName = name.(string)
	}
	
	// Prepare data row
	var values []interface{}
	fieldMappings := make(map[string]string)
	if mappings, exists := config["field_mappings"]; exists {
		if mappingsMap, ok := mappings.(map[string]interface{}); ok {
			for k, v := range mappingsMap {
				fieldMappings[k] = v.(string)
			}
		}
	}
	
	// Create header row if needed
	createHeaders, _ := config["create_headers"].(bool)
	if createHeaders {
		if err := gsi.createHeaders(service, spreadsheetID, worksheetName, event, fieldMappings); err != nil {
			return fmt.Errorf("failed to create headers: %w", err)
		}
	}
	
	// Build values array
	if len(fieldMappings) > 0 {
		// Use field mappings
		for originalField, _ := range fieldMappings {
			if value, exists := event.Data[originalField]; exists {
				values = append(values, value)
			} else {
				values = append(values, "")
			}
		}
	} else {
		// Use all data fields
		values = append(values, event.Timestamp.Format("2006-01-02 15:04:05"))
		values = append(values, event.FormID)
		values = append(values, event.SubmissionID)
		for _, value := range event.Data {
			values = append(values, value)
		}
	}
	
	// Append to sheet
	valueRange := &sheets.ValueRange{
		Values: [][]interface{}{values},
	}
	
	appendRange := worksheetName
	_, err = service.Spreadsheets.Values.Append(spreadsheetID, appendRange, valueRange).
		ValueInputOption("RAW").
		InsertDataOption("INSERT_ROWS").
		Do()
	
	if err != nil {
		return fmt.Errorf("failed to append to sheet: %w", err)
	}
	
	return nil
}

func (gsi *GoogleSheetsIntegration) ValidateConfig(config map[string]interface{}) error {
	return ValidateGoogleSheetsConfig(config)
}

func (gsi *GoogleSheetsIntegration) GetSchema() *IntegrationSchema {
	return &IntegrationSchema{
		Name:        "Google Sheets",
		Description: "Send form submissions to Google Sheets spreadsheets",
		Version:     "1.0.0",
		Fields: []SchemaField{
			{
				Name:        "spreadsheet_id",
				Type:        "string",
				Required:    true,
				Description: "The ID of the Google Sheets spreadsheet",
			},
			{
				Name:        "worksheet_name",
				Type:        "string",
				Required:    false,
				Description: "Name of the worksheet (default: Sheet1)",
				Default:     "Sheet1",
			},
			{
				Name:        "credentials_json",
				Type:        "string",
				Required:    true,
				Description: "Google Service Account credentials JSON",
			},
			{
				Name:        "field_mappings",
				Type:        "object",
				Required:    false,
				Description: "Map form fields to sheet columns",
			},
			{
				Name:        "create_headers",
				Type:        "boolean",
				Required:    false,
				Description: "Create header row automatically",
				Default:     false,
			},
		},
		Examples: []map[string]interface{}{
			{
				"spreadsheet_id":  "1BxiMVs0XRA5nFMdKvBdBZjgmUUqptlbs74OgvE2upms",
				"worksheet_name":  "Form Submissions",
				"credentials_json": "{}",
				"field_mappings": map[string]string{
					"email": "Email Address",
					"name":  "Full Name",
				},
			},
		},
		Documentation: "https://docs.formhub.io/integrations/google-sheets",
	}
}

func (gsi *GoogleSheetsIntegration) createHeaders(service *sheets.Service, spreadsheetID, worksheetName string, 
	event *EnhancedWebhookEvent, fieldMappings map[string]string) error {
	
	// Check if headers already exist
	resp, err := service.Spreadsheets.Values.Get(spreadsheetID, fmt.Sprintf("%s!A1:Z1", worksheetName)).Do()
	if err != nil {
		return err
	}
	
	if len(resp.Values) > 0 && len(resp.Values[0]) > 0 {
		// Headers already exist
		return nil
	}
	
	// Create header row
	var headers []interface{}
	if len(fieldMappings) > 0 {
		for _, header := range fieldMappings {
			headers = append(headers, header)
		}
	} else {
		headers = append(headers, "Timestamp", "Form ID", "Submission ID")
		for key := range event.Data {
			headers = append(headers, key)
		}
	}
	
	valueRange := &sheets.ValueRange{
		Values: [][]interface{}{headers},
	}
	
	_, err = service.Spreadsheets.Values.Update(spreadsheetID, fmt.Sprintf("%s!A1", worksheetName), valueRange).
		ValueInputOption("RAW").
		Do()
	
	return err
}

// Slack Integration
type SlackIntegration struct {
	db    *sql.DB
	redis *redis.Client
	name  string
}

func NewSlackIntegration(db *sql.DB, redis *redis.Client) *SlackIntegration {
	return &SlackIntegration{
		db:    db,
		redis: redis,
		name:  "slack",
	}
}

func (si *SlackIntegration) Name() string {
	return si.name
}

func (si *SlackIntegration) Authenticate(config map[string]interface{}) error {
	// Test webhook URL if provided
	if webhookURL, exists := config["webhook_url"]; exists {
		if url, ok := webhookURL.(string); ok && url != "" {
			return si.testWebhookURL(url)
		}
	}
	
	// Test bot token if provided
	if botToken, exists := config["bot_token"]; exists {
		if token, ok := botToken.(string); ok && token != "" {
			return si.testBotToken(token)
		}
	}
	
	return fmt.Errorf("either webhook_url or bot_token is required")
}

func (si *SlackIntegration) Send(event *EnhancedWebhookEvent, config map[string]interface{}) error {
	// Use webhook URL if available
	if webhookURL, exists := config["webhook_url"]; exists {
		if url, ok := webhookURL.(string); ok && url != "" {
			return si.sendWebhook(url, event, config)
		}
	}
	
	// Use bot token
	if botToken, exists := config["bot_token"]; exists {
		if token, ok := botToken.(string); ok && token != "" {
			return si.sendBotMessage(token, event, config)
		}
	}
	
	return fmt.Errorf("no valid authentication method found")
}

func (si *SlackIntegration) ValidateConfig(config map[string]interface{}) error {
	return ValidateSlackConfig(config)
}

func (si *SlackIntegration) GetSchema() *IntegrationSchema {
	return &IntegrationSchema{
		Name:        "Slack",
		Description: "Send form submission notifications to Slack channels",
		Version:     "2.1.0",
		Fields: []SchemaField{
			{
				Name:        "webhook_url",
				Type:        "string",
				Required:    false,
				Description: "Slack incoming webhook URL",
			},
			{
				Name:        "bot_token",
				Type:        "string",
				Required:    false,
				Description: "Slack bot token (alternative to webhook)",
			},
			{
				Name:        "channel",
				Type:        "string",
				Required:    true,
				Description: "Slack channel name or ID",
			},
			{
				Name:        "username",
				Type:        "string",
				Required:    false,
				Description: "Custom username for the bot",
				Default:     "FormHub",
			},
			{
				Name:        "icon_emoji",
				Type:        "string",
				Required:    false,
				Description: "Emoji icon for the bot",
				Default:     ":memo:",
			},
			{
				Name:        "message_template",
				Type:        "string",
				Required:    false,
				Description: "Custom message template",
			},
		},
		Examples: []map[string]interface{}{
			{
				"webhook_url": "https://hooks.slack.com/services/T00000000/B00000000/XXXXXXXXXXXXXXXXXXXXXXXX",
				"channel":     "#general",
				"username":    "FormHub Bot",
				"icon_emoji":  ":memo:",
			},
		},
		Documentation: "https://docs.formhub.io/integrations/slack",
	}
}

func (si *SlackIntegration) testWebhookURL(webhookURL string) error {
	testPayload := map[string]interface{}{
		"text": "FormHub integration test",
	}
	
	payloadBytes, _ := json.Marshal(testPayload)
	resp, err := http.Post(webhookURL, "application/json", bytes.NewBuffer(payloadBytes))
	if err != nil {
		return fmt.Errorf("webhook test failed: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("webhook test failed with status: %d", resp.StatusCode)
	}
	
	return nil
}

func (si *SlackIntegration) testBotToken(botToken string) error {
	api := slack.New(botToken)
	_, err := api.AuthTest()
	return err
}

func (si *SlackIntegration) sendWebhook(webhookURL string, event *EnhancedWebhookEvent, config map[string]interface{}) error {
	payload := si.buildSlackPayload(event, config)
	payloadBytes, _ := json.Marshal(payload)
	
	resp, err := http.Post(webhookURL, "application/json", bytes.NewBuffer(payloadBytes))
	if err != nil {
		return fmt.Errorf("slack webhook failed: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("slack webhook failed: %d - %s", resp.StatusCode, string(body))
	}
	
	return nil
}

func (si *SlackIntegration) sendBotMessage(botToken string, event *EnhancedWebhookEvent, config map[string]interface{}) error {
	api := slack.New(botToken)
	
	channel := config["channel"].(string)
	payload := si.buildSlackPayload(event, config)
	
	// Extract blocks if they exist
	var blocks []slack.Block
	if blocksData, exists := payload["blocks"]; exists {
		// Convert blocks to Slack block format
		// This is a simplified implementation
		blocks = []slack.Block{
			slack.NewSectionBlock(
				&slack.TextBlockObject{
					Type: slack.MarkdownType,
					Text: fmt.Sprintf("*New form submission*\nForm: %s\nTime: %s", 
						event.FormID, event.Timestamp.Format("2006-01-02 15:04:05")),
				},
				nil, nil,
			),
		}
	}
	
	var options []slack.MsgOption
	if text, exists := payload["text"]; exists {
		options = append(options, slack.MsgOptionText(text.(string), false))
	}
	if len(blocks) > 0 {
		options = append(options, slack.MsgOptionBlocks(blocks...))
	}
	if username, exists := config["username"]; exists {
		options = append(options, slack.MsgOptionUsername(username.(string)))
	}
	if iconEmoji, exists := config["icon_emoji"]; exists {
		options = append(options, slack.MsgOptionIconEmoji(iconEmoji.(string)))
	}
	
	_, _, err := api.PostMessage(channel, options...)
	return err
}

func (si *SlackIntegration) buildSlackPayload(event *EnhancedWebhookEvent, config map[string]interface{}) map[string]interface{} {
	// Check for custom template
	if template, exists := config["message_template"]; exists {
		if templateStr, ok := template.(string); ok {
			return si.processSlackTemplate(templateStr, event)
		}
	}
	
	// Default payload
	var fields []map[string]interface{}
	for key, value := range event.Data {
		fields = append(fields, map[string]interface{}{
			"title": key,
			"value": fmt.Sprintf("%v", value),
			"short": true,
		})
	}
	
	return map[string]interface{}{
		"text": "New form submission received",
		"attachments": []map[string]interface{}{
			{
				"color":      "good",
				"title":      "Form Submission Details",
				"title_link": fmt.Sprintf("https://formhub.io/forms/%s/submissions/%s", event.FormID, event.SubmissionID),
				"fields": fields,
				"footer": "FormHub",
				"ts":     event.Timestamp.Unix(),
			},
		},
	}
}

func (si *SlackIntegration) processSlackTemplate(template string, event *EnhancedWebhookEvent) map[string]interface{} {
	// Simple template processing - in production use a proper template engine
	processed := template
	processed = strings.ReplaceAll(processed, "{{.FormID}}", event.FormID)
	processed = strings.ReplaceAll(processed, "{{.SubmissionID}}", event.SubmissionID)
	processed = strings.ReplaceAll(processed, "{{.Timestamp}}", event.Timestamp.Format("2006-01-02 15:04:05"))
	
	var result map[string]interface{}
	json.Unmarshal([]byte(processed), &result)
	return result
}

// Airtable Integration
type AirtableIntegration struct {
	db    *sql.DB
	redis *redis.Client
	name  string
}

func NewAirtableIntegration(db *sql.DB, redis *redis.Client) *AirtableIntegration {
	return &AirtableIntegration{
		db:    db,
		redis: redis,
		name:  "airtable",
	}
}

func (ai *AirtableIntegration) Name() string {
	return ai.name
}

func (ai *AirtableIntegration) Authenticate(config map[string]interface{}) error {
	apiKey := config["api_key"].(string)
	baseID := config["base_id"].(string)
	tableName := config["table_name"].(string)
	
	// Test by attempting to read the table schema
	url := fmt.Sprintf("https://api.airtable.com/v0/%s/%s", baseID, url.QueryEscape(tableName))
	
	req, err := http.NewRequest("GET", url+"?maxRecords=1", nil)
	if err != nil {
		return err
	}
	
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")
	
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("airtable authentication failed: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("airtable authentication failed with status: %d", resp.StatusCode)
	}
	
	return nil
}

func (ai *AirtableIntegration) Send(event *EnhancedWebhookEvent, config map[string]interface{}) error {
	apiKey := config["api_key"].(string)
	baseID := config["base_id"].(string)
	tableName := config["table_name"].(string)
	
	// Prepare fields
	fields := make(map[string]interface{})
	
	fieldMappings := make(map[string]string)
	if mappings, exists := config["field_mappings"]; exists {
		if mappingsMap, ok := mappings.(map[string]interface{}); ok {
			for k, v := range mappingsMap {
				fieldMappings[k] = v.(string)
			}
		}
	}
	
	// Map fields
	if len(fieldMappings) > 0 {
		for originalField, airtableField := range fieldMappings {
			if value, exists := event.Data[originalField]; exists {
				fields[airtableField] = value
			}
		}
	} else {
		// Use original field names
		for key, value := range event.Data {
			fields[key] = value
		}
	}
	
	// Add metadata
	fields["Form ID"] = event.FormID
	fields["Submission ID"] = event.SubmissionID
	fields["Submitted At"] = event.Timestamp.Format("2006-01-02T15:04:05Z")
	
	// Create record
	payload := map[string]interface{}{
		"records": []map[string]interface{}{
			{
				"fields": fields,
			},
		},
	}
	
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}
	
	url := fmt.Sprintf("https://api.airtable.com/v0/%s/%s", baseID, url.QueryEscape(tableName))
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return err
	}
	
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")
	
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("airtable request failed: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("airtable request failed: %d - %s", resp.StatusCode, string(body))
	}
	
	return nil
}

func (ai *AirtableIntegration) ValidateConfig(config map[string]interface{}) error {
	return ValidateAirtableConfig(config)
}

func (ai *AirtableIntegration) GetSchema() *IntegrationSchema {
	return &IntegrationSchema{
		Name:        "Airtable",
		Description: "Send form submissions to Airtable bases",
		Version:     "1.5.0",
		Fields: []SchemaField{
			{
				Name:        "api_key",
				Type:        "string",
				Required:    true,
				Description: "Airtable Personal Access Token",
			},
			{
				Name:        "base_id",
				Type:        "string",
				Required:    true,
				Description: "Airtable Base ID",
			},
			{
				Name:        "table_name",
				Type:        "string",
				Required:    true,
				Description: "Name of the table in the base",
			},
			{
				Name:        "field_mappings",
				Type:        "object",
				Required:    false,
				Description: "Map form fields to Airtable columns",
			},
		},
		Examples: []map[string]interface{}{
			{
				"api_key":    "patAbcDefGhiJklMnoPqrStUvWxYz",
				"base_id":    "appABCDEFGHIJKLMN",
				"table_name": "Form Submissions",
				"field_mappings": map[string]string{
					"email": "Email",
					"name":  "Name",
				},
			},
		},
		Documentation: "https://docs.formhub.io/integrations/airtable",
	}
}

// Additional integrations would follow similar patterns...
// For brevity, I'll implement stubs for the remaining integrations

// Notion Integration
type NotionIntegration struct {
	db    *sql.DB
	redis *redis.Client
	name  string
}

func NewNotionIntegration(db *sql.DB, redis *redis.Client) *NotionIntegration {
	return &NotionIntegration{db: db, redis: redis, name: "notion"}
}

func (ni *NotionIntegration) Name() string { return ni.name }
func (ni *NotionIntegration) Authenticate(config map[string]interface{}) error { return nil }
func (ni *NotionIntegration) Send(event *EnhancedWebhookEvent, config map[string]interface{}) error { return nil }
func (ni *NotionIntegration) ValidateConfig(config map[string]interface{}) error { return ValidateNotionConfig(config) }
func (ni *NotionIntegration) GetSchema() *IntegrationSchema { return &IntegrationSchema{Name: "Notion", Version: "1.0.0"} }

// Telegram Integration
type TelegramIntegration struct {
	db    *sql.DB
	redis *redis.Client
	name  string
}

func NewTelegramIntegration(db *sql.DB, redis *redis.Client) *TelegramIntegration {
	return &TelegramIntegration{db: db, redis: redis, name: "telegram"}
}

func (ti *TelegramIntegration) Name() string { return ti.name }
func (ti *TelegramIntegration) Authenticate(config map[string]interface{}) error { return nil }
func (ti *TelegramIntegration) Send(event *EnhancedWebhookEvent, config map[string]interface{}) error { return nil }
func (ti *TelegramIntegration) ValidateConfig(config map[string]interface{}) error { return ValidateTelegramConfig(config) }
func (ti *TelegramIntegration) GetSchema() *IntegrationSchema { return &IntegrationSchema{Name: "Telegram", Version: "1.0.0"} }

// Discord Integration
type DiscordIntegration struct {
	db    *sql.DB
	redis *redis.Client
	name  string
}

func NewDiscordIntegration(db *sql.DB, redis *redis.Client) *DiscordIntegration {
	return &DiscordIntegration{db: db, redis: redis, name: "discord"}
}

func (di *DiscordIntegration) Name() string { return di.name }
func (di *DiscordIntegration) Authenticate(config map[string]interface{}) error { return nil }
func (di *DiscordIntegration) Send(event *EnhancedWebhookEvent, config map[string]interface{}) error { return nil }
func (di *DiscordIntegration) ValidateConfig(config map[string]interface{}) error { return ValidateDiscordConfig(config) }
func (di *DiscordIntegration) GetSchema() *IntegrationSchema { return &IntegrationSchema{Name: "Discord", Version: "1.0.0"} }

// Zapier Integration
type ZapierIntegration struct {
	db    *sql.DB
	redis *redis.Client
	name  string
}

func NewZapierIntegration(db *sql.DB, redis *redis.Client) *ZapierIntegration {
	return &ZapierIntegration{db: db, redis: redis, name: "zapier"}
}

func (zi *ZapierIntegration) Name() string { return zi.name }
func (zi *ZapierIntegration) Authenticate(config map[string]interface{}) error { return nil }
func (zi *ZapierIntegration) Send(event *EnhancedWebhookEvent, config map[string]interface{}) error { return nil }
func (zi *ZapierIntegration) ValidateConfig(config map[string]interface{}) error { return ValidateZapierConfig(config) }
func (zi *ZapierIntegration) GetSchema() *IntegrationSchema { return &IntegrationSchema{Name: "Zapier", Version: "1.0.0"} }