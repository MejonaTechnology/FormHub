package services

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/robfig/cron/v3"
	"gopkg.in/yaml.v3"
)

// EnhancedWebhookService provides comprehensive webhook and integration capabilities
type EnhancedWebhookService struct {
	db                *sql.DB
	redis             *redis.Client
	httpClient        *http.Client
	secureClient      *http.Client
	ctx               context.Context
	
	// Configuration
	maxRetries        int
	baseRetryDelay    time.Duration
	maxRetryDelay     time.Duration
	timeout           time.Duration
	maxPayloadSize    int64
	rateLimitWindow   time.Duration
	maxWebhooksPerMin int
	
	// Security
	signatureHeader   string
	timestampHeader   string
	userAgent         string
	allowedDomains    map[string]bool
	blockedDomains    map[string]bool
	
	// Load balancing and failover
	loadBalancer      *WebhookLoadBalancer
	circuitBreaker    *CircuitBreaker
	
	// Background processing
	workerPool        *WorkerPool
	scheduler         *cron.Cron
	
	// Analytics and monitoring
	analytics         *WebhookAnalytics
	monitor           *WebhookMonitor
	
	// Third-party integrations
	integrations      *IntegrationManager
	
	// Mutex for thread safety
	mu                sync.RWMutex
}

// WebhookEndpoint represents a single webhook endpoint configuration
type WebhookEndpoint struct {
	ID                string            `json:"id"`
	Name              string            `json:"name"`
	URL               string            `json:"url"`
	Secret            string            `json:"secret,omitempty"`
	Events            []string          `json:"events"`
	Headers           map[string]string `json:"headers,omitempty"`
	ContentType       string            `json:"content_type"`
	Method            string            `json:"method"`
	Timeout           int               `json:"timeout"`
	MaxRetries        int               `json:"max_retries"`
	RetryDelay        int               `json:"retry_delay"`
	Enabled           bool              `json:"enabled"`
	RateLimitEnabled  bool              `json:"rate_limit_enabled"`
	VerifySSL         bool              `json:"verify_ssl"`
	CustomPayload     string            `json:"custom_payload,omitempty"`
	TransformConfig   *TransformConfig  `json:"transform_config,omitempty"`
	ConditionalRules  []ConditionalRule `json:"conditional_rules,omitempty"`
	Priority          int               `json:"priority"` // 1 = highest
	Tags              []string          `json:"tags"`
	CreatedAt         time.Time         `json:"created_at"`
	UpdatedAt         time.Time         `json:"updated_at"`
}

// FormWebhookConfig holds all webhook configurations for a form
type FormWebhookConfig struct {
	FormID           string            `json:"form_id"`
	Endpoints        []WebhookEndpoint `json:"endpoints"`
	GlobalConfig     *GlobalConfig     `json:"global_config,omitempty"`
	IntegrationRules *IntegrationRules `json:"integration_rules,omitempty"`
	CreatedAt        time.Time         `json:"created_at"`
	UpdatedAt        time.Time         `json:"updated_at"`
}

// TransformConfig defines field mapping and data transformation
type TransformConfig struct {
	FieldMappings  map[string]string   `json:"field_mappings"`
	DataFilters    []DataFilter        `json:"data_filters"`
	CustomTemplate string              `json:"custom_template,omitempty"`
	OutputFormat   string              `json:"output_format"` // json, xml, form-data
}

// ConditionalRule defines when a webhook should trigger
type ConditionalRule struct {
	Field     string      `json:"field"`
	Operator  string      `json:"operator"` // equals, contains, greater_than, etc.
	Value     interface{} `json:"value"`
	LogicOp   string      `json:"logic_op"` // and, or
}

// DataFilter applies filtering/transformation to data
type DataFilter struct {
	Type   string      `json:"type"`   // include, exclude, transform, validate
	Field  string      `json:"field"`
	Action string      `json:"action"`
	Value  interface{} `json:"value,omitempty"`
}

// GlobalConfig provides form-level webhook settings
type GlobalConfig struct {
	EnableSignatures     bool              `json:"enable_signatures"`
	RequireHTTPS        bool              `json:"require_https"`
	MaxConcurrentSends  int               `json:"max_concurrent_sends"`
	DeliveryTimeout     time.Duration     `json:"delivery_timeout"`
	DefaultRetries      int               `json:"default_retries"`
	RateLimitPerEndpoint int              `json:"rate_limit_per_endpoint"`
	CustomHeaders       map[string]string `json:"custom_headers"`
	NotificationSettings *NotificationSettings `json:"notification_settings"`
}

// IntegrationRules defines third-party integration settings
type IntegrationRules struct {
	GoogleSheets  *GoogleSheetsConfig  `json:"google_sheets,omitempty"`
	Airtable      *AirtableConfig      `json:"airtable,omitempty"`
	Notion        *NotionConfig        `json:"notion,omitempty"`
	Slack         *SlackConfig         `json:"slack,omitempty"`
	Telegram      *TelegramConfig      `json:"telegram,omitempty"`
	Discord       *DiscordConfig       `json:"discord,omitempty"`
	Zapier        *ZapierConfig        `json:"zapier,omitempty"`
	CustomIntegrations []CustomIntegration `json:"custom_integrations,omitempty"`
}

// NotificationSettings for webhook status notifications
type NotificationSettings struct {
	EmailOnFailure   bool     `json:"email_on_failure"`
	SlackOnFailure   bool     `json:"slack_on_failure"`
	NotifyAfterRetries int    `json:"notify_after_retries"`
	Recipients       []string `json:"recipients"`
}

// WebhookLoadBalancer handles multiple endpoint load balancing
type WebhookLoadBalancer struct {
	strategy    string // round_robin, weighted, priority, random
	endpoints   []WeightedEndpoint
	currentIdx  int
	mu          sync.Mutex
}

// WeightedEndpoint for load balancing
type WeightedEndpoint struct {
	Endpoint   *WebhookEndpoint
	Weight     int
	HealthScore float64
	LastCheck  time.Time
}

// CircuitBreaker prevents cascading failures
type CircuitBreaker struct {
	maxFailures    int
	resetTimeout   time.Duration
	endpoints      map[string]*EndpointState
	mu             sync.RWMutex
}

// EndpointState tracks circuit breaker state per endpoint
type EndpointState struct {
	State        string    // closed, open, half_open
	Failures     int
	LastFailure  time.Time
	NextReset    time.Time
}

// WorkerPool for concurrent webhook processing
type WorkerPool struct {
	workerCount int
	jobChan     chan *WebhookJob
	workers     []*WebhookWorker
	ctx         context.Context
	cancel      context.CancelFunc
	wg          sync.WaitGroup
}

// WebhookJob represents a webhook processing job
type WebhookJob struct {
	ID           string
	FormID       string
	Event        *EnhancedWebhookEvent
	Endpoints    []WebhookEndpoint
	Priority     int
	CreatedAt    time.Time
	ProcessedAt  *time.Time
	CompletedAt  *time.Time
}

// WebhookWorker processes webhook jobs
type WebhookWorker struct {
	id      int
	service *EnhancedWebhookService
	jobChan chan *WebhookJob
	quit    chan bool
}

// EnhancedWebhookEvent extends the basic webhook event
type EnhancedWebhookEvent struct {
	ID              string                 `json:"id"`
	Type            string                 `json:"type"`
	Timestamp       time.Time              `json:"timestamp"`
	FormID          string                 `json:"form_id"`
	SubmissionID    string                 `json:"submission_id,omitempty"`
	UserID          string                 `json:"user_id,omitempty"`
	Data            map[string]interface{} `json:"data"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
	Source          string                 `json:"source"`
	Version         string                 `json:"version"`
	EventSequence   int64                  `json:"event_sequence"`
	CorrelationID   string                 `json:"correlation_id,omitempty"`
	Environment     string                 `json:"environment"`
	IPAddress       string                 `json:"ip_address,omitempty"`
	UserAgent       string                 `json:"user_agent,omitempty"`
	Geolocation     *GeolocationData       `json:"geolocation,omitempty"`
	DeviceInfo      *DeviceInfo            `json:"device_info,omitempty"`
}

// GeolocationData contains location information
type GeolocationData struct {
	Country     string  `json:"country"`
	Region      string  `json:"region"`
	City        string  `json:"city"`
	Latitude    float64 `json:"latitude"`
	Longitude   float64 `json:"longitude"`
	Timezone    string  `json:"timezone"`
}

// DeviceInfo contains device and browser information
type DeviceInfo struct {
	Browser         string `json:"browser"`
	BrowserVersion  string `json:"browser_version"`
	OS              string `json:"os"`
	OSVersion       string `json:"os_version"`
	Device          string `json:"device"`
	IsMobile        bool   `json:"is_mobile"`
	IsTablet        bool   `json:"is_tablet"`
	IsDesktop       bool   `json:"is_desktop"`
}

// WebhookAnalytics provides detailed analytics and metrics
type WebhookAnalytics struct {
	redis     *redis.Client
	db        *sql.DB
	mu        sync.RWMutex
}

// WebhookMonitor provides real-time monitoring
type WebhookMonitor struct {
	redis          *redis.Client
	db             *sql.DB
	alertThreshold float64
	checkInterval  time.Duration
	subscribers    map[string]chan *MonitorEvent
	mu             sync.RWMutex
}

// MonitorEvent represents a monitoring event
type MonitorEvent struct {
	Type        string                 `json:"type"`
	Timestamp   time.Time              `json:"timestamp"`
	EndpointID  string                 `json:"endpoint_id"`
	FormID      string                 `json:"form_id"`
	Severity    string                 `json:"severity"`
	Message     string                 `json:"message"`
	Data        map[string]interface{} `json:"data"`
}

// IntegrationManager handles third-party integrations
type IntegrationManager struct {
	integrations map[string]Integration
	oauth        *OAuthManager
	templates    *TemplateManager
	marketplace  *IntegrationMarketplace
}

// Integration interface for third-party services
type Integration interface {
	Name() string
	Authenticate(config map[string]interface{}) error
	Send(event *EnhancedWebhookEvent, config map[string]interface{}) error
	ValidateConfig(config map[string]interface{}) error
	GetSchema() *IntegrationSchema
}

// IntegrationSchema defines configuration schema
type IntegrationSchema struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Version     string                 `json:"version"`
	Fields      []SchemaField          `json:"fields"`
	Examples    []map[string]interface{} `json:"examples"`
	Documentation string               `json:"documentation"`
}

// SchemaField defines a configuration field
type SchemaField struct {
	Name        string      `json:"name"`
	Type        string      `json:"type"`
	Required    bool        `json:"required"`
	Description string      `json:"description"`
	Default     interface{} `json:"default,omitempty"`
	Options     []string    `json:"options,omitempty"`
	Validation  *FieldValidation `json:"validation,omitempty"`
}

// FieldValidation defines field validation rules
type FieldValidation struct {
	MinLength int         `json:"min_length,omitempty"`
	MaxLength int         `json:"max_length,omitempty"`
	Pattern   string      `json:"pattern,omitempty"`
	Range     *ValueRange `json:"range,omitempty"`
}

// ValueRange defines numeric value ranges
type ValueRange struct {
	Min *float64 `json:"min,omitempty"`
	Max *float64 `json:"max,omitempty"`
}

// OAuthManager handles OAuth authentication for integrations
type OAuthManager struct {
	configs map[string]*OAuthConfig
	tokens  map[string]*OAuthToken
	mu      sync.RWMutex
}

// OAuthConfig stores OAuth configuration
type OAuthConfig struct {
	ClientID     string   `json:"client_id"`
	ClientSecret string   `json:"client_secret"`
	AuthURL      string   `json:"auth_url"`
	TokenURL     string   `json:"token_url"`
	RedirectURL  string   `json:"redirect_url"`
	Scopes       []string `json:"scopes"`
}

// OAuthToken stores OAuth tokens
type OAuthToken struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	TokenType    string    `json:"token_type"`
	ExpiresAt    time.Time `json:"expires_at"`
}

// TemplateManager handles payload templates and transformations
type TemplateManager struct {
	templates map[string]*PayloadTemplate
	mu        sync.RWMutex
}

// PayloadTemplate defines a payload transformation template
type PayloadTemplate struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Template    string            `json:"template"`
	OutputType  string            `json:"output_type"` // json, xml, form-data, yaml
	Variables   map[string]string `json:"variables"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

// IntegrationMarketplace provides pre-built integrations
type IntegrationMarketplace struct {
	integrations map[string]*MarketplaceIntegration
	categories   map[string][]string
	mu           sync.RWMutex
}

// MarketplaceIntegration represents a marketplace integration
type MarketplaceIntegration struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Category    string                 `json:"category"`
	Version     string                 `json:"version"`
	Author      string                 `json:"author"`
	Icon        string                 `json:"icon"`
	Tags        []string               `json:"tags"`
	Config      map[string]interface{} `json:"config"`
	Template    string                 `json:"template"`
	Schema      *IntegrationSchema     `json:"schema"`
	Popular     bool                   `json:"popular"`
	Featured    bool                   `json:"featured"`
	Downloads   int64                  `json:"downloads"`
	Rating      float64                `json:"rating"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

// NewEnhancedWebhookService creates a new enhanced webhook service
func NewEnhancedWebhookService(db *sql.DB, redis *redis.Client) *EnhancedWebhookService {
	ctx := context.Background()
	
	// Create HTTP clients with different security settings
	httpClient := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:       100,
			IdleConnTimeout:    90 * time.Second,
			DisableCompression: false,
			TLSClientConfig:    &tls.Config{InsecureSkipVerify: true},
		},
	}
	
	secureClient := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:       100,
			IdleConnTimeout:    90 * time.Second,
			DisableCompression: false,
			TLSClientConfig:    &tls.Config{InsecureSkipVerify: false},
		},
	}
	
	// Initialize components
	loadBalancer := &WebhookLoadBalancer{
		strategy: "round_robin",
		endpoints: make([]WeightedEndpoint, 0),
	}
	
	circuitBreaker := &CircuitBreaker{
		maxFailures:  5,
		resetTimeout: 60 * time.Second,
		endpoints:    make(map[string]*EndpointState),
	}
	
	analytics := &WebhookAnalytics{
		redis: redis,
		db:    db,
	}
	
	monitor := &WebhookMonitor{
		redis:          redis,
		db:             db,
		alertThreshold: 0.8, // 80% failure rate triggers alert
		checkInterval:  5 * time.Minute,
		subscribers:    make(map[string]chan *MonitorEvent),
	}
	
	integrationManager := NewIntegrationManager(db, redis)
	
	service := &EnhancedWebhookService{
		db:                db,
		redis:             redis,
		httpClient:        httpClient,
		secureClient:      secureClient,
		ctx:               ctx,
		maxRetries:        5,
		baseRetryDelay:    2 * time.Second,
		maxRetryDelay:     300 * time.Second, // 5 minutes
		timeout:           30 * time.Second,
		maxPayloadSize:    5 * 1024 * 1024, // 5MB
		rateLimitWindow:   time.Minute,
		maxWebhooksPerMin: 100,
		signatureHeader:   "X-FormHub-Signature-256",
		timestampHeader:   "X-FormHub-Timestamp",
		userAgent:         "FormHub-Webhooks/2.0",
		allowedDomains:    make(map[string]bool),
		blockedDomains:    make(map[string]bool),
		loadBalancer:      loadBalancer,
		circuitBreaker:    circuitBreaker,
		analytics:         analytics,
		monitor:           monitor,
		integrations:      integrationManager,
		scheduler:         cron.New(),
	}
	
	// Initialize worker pool
	service.workerPool = NewWorkerPool(service, 10) // 10 concurrent workers
	
	// Start background services
	service.startBackgroundServices()
	
	return service
}

// SendWebhook sends webhooks to all configured endpoints for a form
func (ews *EnhancedWebhookService) SendWebhook(formID string, event *EnhancedWebhookEvent) error {
	// Get webhook configuration
	config, err := ews.getFormWebhookConfig(formID)
	if err != nil {
		return fmt.Errorf("failed to get webhook config: %w", err)
	}
	
	if config == nil || len(config.Endpoints) == 0 {
		return nil // No webhooks configured
	}
	
	// Filter endpoints based on event type and conditions
	eligibleEndpoints := ews.filterEligibleEndpoints(config.Endpoints, event)
	if len(eligibleEndpoints) == 0 {
		return nil // No eligible endpoints
	}
	
	// Create webhook job
	job := &WebhookJob{
		ID:        uuid.New().String(),
		FormID:    formID,
		Event:     event,
		Endpoints: eligibleEndpoints,
		Priority:  ews.calculateJobPriority(eligibleEndpoints),
		CreatedAt: time.Now(),
	}
	
	// Add to analytics
	ews.analytics.RecordWebhookJob(job)
	
	// Process through worker pool
	ews.workerPool.AddJob(job)
	
	return nil
}

// CreateWebhookEndpoint creates a new webhook endpoint for a form
func (ews *EnhancedWebhookService) CreateWebhookEndpoint(formID string, endpoint *WebhookEndpoint) error {
	// Validate endpoint
	if err := ews.validateEndpoint(endpoint); err != nil {
		return fmt.Errorf("invalid endpoint configuration: %w", err)
	}
	
	// Generate ID if not provided
	if endpoint.ID == "" {
		endpoint.ID = uuid.New().String()
	}
	
	// Set timestamps
	endpoint.CreatedAt = time.Now()
	endpoint.UpdatedAt = time.Now()
	
	// Get existing config or create new
	config, err := ews.getFormWebhookConfig(formID)
	if err != nil {
		return fmt.Errorf("failed to get existing config: %w", err)
	}
	
	if config == nil {
		config = &FormWebhookConfig{
			FormID:    formID,
			Endpoints: []WebhookEndpoint{*endpoint},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
	} else {
		config.Endpoints = append(config.Endpoints, *endpoint)
		config.UpdatedAt = time.Now()
	}
	
	// Save to database
	return ews.saveFormWebhookConfig(config)
}

// UpdateWebhookEndpoint updates an existing webhook endpoint
func (ews *EnhancedWebhookService) UpdateWebhookEndpoint(formID, endpointID string, updates *WebhookEndpoint) error {
	config, err := ews.getFormWebhookConfig(formID)
	if err != nil {
		return fmt.Errorf("failed to get webhook config: %w", err)
	}
	
	if config == nil {
		return fmt.Errorf("webhook config not found for form %s", formID)
	}
	
	// Find and update endpoint
	found := false
	for i, endpoint := range config.Endpoints {
		if endpoint.ID == endpointID {
			// Validate updates
			if err := ews.validateEndpoint(updates); err != nil {
				return fmt.Errorf("invalid endpoint updates: %w", err)
			}
			
			// Update fields
			updates.ID = endpointID
			updates.CreatedAt = endpoint.CreatedAt
			updates.UpdatedAt = time.Now()
			config.Endpoints[i] = *updates
			found = true
			break
		}
	}
	
	if !found {
		return fmt.Errorf("endpoint %s not found in form %s", endpointID, formID)
	}
	
	config.UpdatedAt = time.Now()
	return ews.saveFormWebhookConfig(config)
}

// DeleteWebhookEndpoint removes a webhook endpoint
func (ews *EnhancedWebhookService) DeleteWebhookEndpoint(formID, endpointID string) error {
	config, err := ews.getFormWebhookConfig(formID)
	if err != nil {
		return fmt.Errorf("failed to get webhook config: %w", err)
	}
	
	if config == nil {
		return fmt.Errorf("webhook config not found for form %s", formID)
	}
	
	// Find and remove endpoint
	newEndpoints := make([]WebhookEndpoint, 0, len(config.Endpoints))
	found := false
	for _, endpoint := range config.Endpoints {
		if endpoint.ID != endpointID {
			newEndpoints = append(newEndpoints, endpoint)
		} else {
			found = true
		}
	}
	
	if !found {
		return fmt.Errorf("endpoint %s not found in form %s", endpointID, formID)
	}
	
	config.Endpoints = newEndpoints
	config.UpdatedAt = time.Now()
	
	// Archive endpoint data
	ews.archiveEndpointData(formID, endpointID)
	
	return ews.saveFormWebhookConfig(config)
}

// TestWebhookEndpoint tests a webhook endpoint
func (ews *EnhancedWebhookService) TestWebhookEndpoint(formID, endpointID string) (*WebhookTestResult, error) {
	config, err := ews.getFormWebhookConfig(formID)
	if err != nil {
		return nil, fmt.Errorf("failed to get webhook config: %w", err)
	}
	
	if config == nil {
		return nil, fmt.Errorf("webhook config not found")
	}
	
	// Find endpoint
	var endpoint *WebhookEndpoint
	for _, ep := range config.Endpoints {
		if ep.ID == endpointID {
			endpoint = &ep
			break
		}
	}
	
	if endpoint == nil {
		return nil, fmt.Errorf("endpoint not found")
	}
	
	// Create test event
	testEvent := &EnhancedWebhookEvent{
		ID:            uuid.New().String(),
		Type:          "test",
		Timestamp:     time.Now().UTC(),
		FormID:        formID,
		Source:        "test",
		Version:       "2.0",
		EventSequence: 1,
		Environment:   "test",
		Data: map[string]interface{}{
			"message": "This is a test webhook from FormHub",
			"test":    true,
		},
	}
	
	// Send test webhook
	result := ews.sendSingleWebhook(endpoint, testEvent)
	
	return &WebhookTestResult{
		EndpointID:   endpointID,
		Success:      result.Success,
		StatusCode:   result.StatusCode,
		ResponseTime: result.ResponseTime,
		Response:     result.ResponseBody,
		Error:        result.Error,
		TestedAt:     time.Now(),
	}, nil
}

// GetWebhookAnalytics returns comprehensive webhook analytics
func (ews *EnhancedWebhookService) GetWebhookAnalytics(formID string, timeRange TimeRange) (*WebhookAnalyticsData, error) {
	return ews.analytics.GetAnalytics(formID, timeRange)
}

// GetWebhookMonitoringData returns real-time monitoring data
func (ews *EnhancedWebhookService) GetWebhookMonitoringData(formID string) (*WebhookMonitoringData, error) {
	return ews.monitor.GetMonitoringData(formID)
}

// Helper methods

func (ews *EnhancedWebhookService) filterEligibleEndpoints(endpoints []WebhookEndpoint, event *EnhancedWebhookEvent) []WebhookEndpoint {
	eligible := make([]WebhookEndpoint, 0)
	
	for _, endpoint := range endpoints {
		// Check if endpoint is enabled
		if !endpoint.Enabled {
			continue
		}
		
		// Check if event type is supported
		if !ews.isEventTypeSupported(endpoint.Events, event.Type) {
			continue
		}
		
		// Check conditional rules
		if len(endpoint.ConditionalRules) > 0 && !ews.evaluateConditionalRules(endpoint.ConditionalRules, event) {
			continue
		}
		
		// Check circuit breaker
		if ews.circuitBreaker.IsOpen(endpoint.ID) {
			continue
		}
		
		eligible = append(eligible, endpoint)
	}
	
	return eligible
}

func (ews *EnhancedWebhookService) isEventTypeSupported(supportedEvents []string, eventType string) bool {
	if len(supportedEvents) == 0 {
		return true // Support all events if none specified
	}
	
	for _, supported := range supportedEvents {
		if supported == "*" || supported == eventType {
			return true
		}
		
		// Support wildcard matching (e.g., "form.*" matches "form.submitted")
		if strings.HasSuffix(supported, "*") {
			prefix := strings.TrimSuffix(supported, "*")
			if strings.HasPrefix(eventType, prefix) {
				return true
			}
		}
	}
	
	return false
}

func (ews *EnhancedWebhookService) evaluateConditionalRules(rules []ConditionalRule, event *EnhancedWebhookEvent) bool {
	if len(rules) == 0 {
		return true
	}
	
	results := make([]bool, len(rules))
	
	for i, rule := range rules {
		results[i] = ews.evaluateRule(rule, event)
	}
	
	// Apply logic operators (simple AND/OR evaluation)
	// For more complex logic, implement proper expression parsing
	result := results[0]
	for i := 1; i < len(results); i++ {
		if i < len(rules) && rules[i].LogicOp == "or" {
			result = result || results[i]
		} else {
			result = result && results[i]
		}
	}
	
	return result
}

func (ews *EnhancedWebhookService) evaluateRule(rule ConditionalRule, event *EnhancedWebhookEvent) bool {
	// Get field value from event data
	var fieldValue interface{}
	
	// Check in main data
	if val, exists := event.Data[rule.Field]; exists {
		fieldValue = val
	} else if val, exists := event.Metadata[rule.Field]; exists {
		fieldValue = val
	} else {
		// Check top-level fields
		switch rule.Field {
		case "type":
			fieldValue = event.Type
		case "form_id":
			fieldValue = event.FormID
		case "submission_id":
			fieldValue = event.SubmissionID
		case "user_id":
			fieldValue = event.UserID
		case "source":
			fieldValue = event.Source
		case "environment":
			fieldValue = event.Environment
		default:
			return false // Field not found
		}
	}
	
	// Evaluate based on operator
	switch rule.Operator {
	case "equals", "eq":
		return fmt.Sprintf("%v", fieldValue) == fmt.Sprintf("%v", rule.Value)
	case "not_equals", "neq":
		return fmt.Sprintf("%v", fieldValue) != fmt.Sprintf("%v", rule.Value)
	case "contains":
		return strings.Contains(strings.ToLower(fmt.Sprintf("%v", fieldValue)), strings.ToLower(fmt.Sprintf("%v", rule.Value)))
	case "not_contains":
		return !strings.Contains(strings.ToLower(fmt.Sprintf("%v", fieldValue)), strings.ToLower(fmt.Sprintf("%v", rule.Value)))
	case "starts_with":
		return strings.HasPrefix(strings.ToLower(fmt.Sprintf("%v", fieldValue)), strings.ToLower(fmt.Sprintf("%v", rule.Value)))
	case "ends_with":
		return strings.HasSuffix(strings.ToLower(fmt.Sprintf("%v", fieldValue)), strings.ToLower(fmt.Sprintf("%v", rule.Value)))
	case "greater_than", "gt":
		return ews.compareNumeric(fieldValue, rule.Value, ">")
	case "greater_than_equal", "gte":
		return ews.compareNumeric(fieldValue, rule.Value, ">=")
	case "less_than", "lt":
		return ews.compareNumeric(fieldValue, rule.Value, "<")
	case "less_than_equal", "lte":
		return ews.compareNumeric(fieldValue, rule.Value, "<=")
	case "in":
		if valueList, ok := rule.Value.([]interface{}); ok {
			fieldStr := fmt.Sprintf("%v", fieldValue)
			for _, v := range valueList {
				if fieldStr == fmt.Sprintf("%v", v) {
					return true
				}
			}
		}
		return false
	case "not_in":
		if valueList, ok := rule.Value.([]interface{}); ok {
			fieldStr := fmt.Sprintf("%v", fieldValue)
			for _, v := range valueList {
				if fieldStr == fmt.Sprintf("%v", v) {
					return false
				}
			}
			return true
		}
		return true
	case "exists":
		return fieldValue != nil
	case "not_exists":
		return fieldValue == nil
	default:
		return false
	}
}

func (ews *EnhancedWebhookService) compareNumeric(a, b interface{}, operator string) bool {
	aFloat, aOk := ews.toFloat64(a)
	bFloat, bOk := ews.toFloat64(b)
	
	if !aOk || !bOk {
		return false
	}
	
	switch operator {
	case ">":
		return aFloat > bFloat
	case ">=":
		return aFloat >= bFloat
	case "<":
		return aFloat < bFloat
	case "<=":
		return aFloat <= bFloat
	default:
		return false
	}
}

func (ews *EnhancedWebhookService) toFloat64(val interface{}) (float64, bool) {
	switch v := val.(type) {
	case float64:
		return v, true
	case float32:
		return float64(v), true
	case int:
		return float64(v), true
	case int32:
		return float64(v), true
	case int64:
		return float64(v), true
	case string:
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f, true
		}
	}
	return 0, false
}

func (ews *EnhancedWebhookService) calculateJobPriority(endpoints []WebhookEndpoint) int {
	if len(endpoints) == 0 {
		return 5 // Default priority
	}
	
	minPriority := endpoints[0].Priority
	for _, endpoint := range endpoints[1:] {
		if endpoint.Priority < minPriority && endpoint.Priority > 0 {
			minPriority = endpoint.Priority
		}
	}
	
	return minPriority
}

func (ews *EnhancedWebhookService) validateEndpoint(endpoint *WebhookEndpoint) error {
	// URL validation
	if endpoint.URL == "" {
		return fmt.Errorf("URL is required")
	}
	
	u, err := url.Parse(endpoint.URL)
	if err != nil {
		return fmt.Errorf("invalid URL format: %w", err)
	}
	
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("unsupported URL scheme: %s", u.Scheme)
	}
	
	// Check for localhost/private IPs in production
	if ews.isProductionEnvironment() && ews.isPrivateAddress(u.Host) {
		return fmt.Errorf("private addresses not allowed in production")
	}
	
	// Check blocked domains
	host := strings.ToLower(u.Host)
	if ews.blockedDomains[host] {
		return fmt.Errorf("domain is blocked: %s", host)
	}
	
	// Validate method
	if endpoint.Method != "" {
		validMethods := map[string]bool{"GET": true, "POST": true, "PUT": true, "PATCH": true, "DELETE": true}
		if !validMethods[strings.ToUpper(endpoint.Method)] {
			return fmt.Errorf("unsupported HTTP method: %s", endpoint.Method)
		}
	}
	
	// Validate timeout
	if endpoint.Timeout < 0 || endpoint.Timeout > 300 {
		return fmt.Errorf("timeout must be between 0 and 300 seconds")
	}
	
	// Validate max retries
	if endpoint.MaxRetries < 0 || endpoint.MaxRetries > 10 {
		return fmt.Errorf("max retries must be between 0 and 10")
	}
	
	// Validate retry delay
	if endpoint.RetryDelay < 0 || endpoint.RetryDelay > 3600 {
		return fmt.Errorf("retry delay must be between 0 and 3600 seconds")
	}
	
	// Validate priority
	if endpoint.Priority < 1 || endpoint.Priority > 10 {
		return fmt.Errorf("priority must be between 1 and 10")
	}
	
	return nil
}

func (ews *EnhancedWebhookService) isProductionEnvironment() bool {
	// Check environment variable or config
	// For now, return false for development
	return false
}

func (ews *EnhancedWebhookService) isPrivateAddress(host string) bool {
	// Check for localhost
	if strings.Contains(host, "localhost") || strings.Contains(host, "127.0.0.1") {
		return true
	}
	
	// Parse IP address
	ip := net.ParseIP(host)
	if ip == nil {
		// If it's not an IP, check for common private domain patterns
		return false
	}
	
	// Check for private IP ranges
	privateRanges := []string{
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
		"127.0.0.0/8",
	}
	
	for _, cidr := range privateRanges {
		_, network, err := net.ParseCIDR(cidr)
		if err != nil {
			continue
		}
		if network.Contains(ip) {
			return true
		}
	}
	
	return false
}

// Additional types needed for the service

// WebhookTestResult represents the result of a webhook test
type WebhookTestResult struct {
	EndpointID   string        `json:"endpoint_id"`
	Success      bool          `json:"success"`
	StatusCode   int           `json:"status_code"`
	ResponseTime time.Duration `json:"response_time"`
	Response     string        `json:"response"`
	Error        string        `json:"error,omitempty"`
	TestedAt     time.Time     `json:"tested_at"`
}

// TimeRange represents a time range for analytics
type TimeRange struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

// WebhookAnalyticsData contains comprehensive analytics data
type WebhookAnalyticsData struct {
	FormID           string                    `json:"form_id"`
	TimeRange        TimeRange                 `json:"time_range"`
	TotalWebhooks    int64                     `json:"total_webhooks"`
	SuccessfulSent   int64                     `json:"successful_sent"`
	Failed           int64                     `json:"failed"`
	SuccessRate      float64                   `json:"success_rate"`
	AvgResponseTime  time.Duration             `json:"avg_response_time"`
	EndpointStats    []EndpointAnalytics       `json:"endpoint_stats"`
	HourlyStats      []HourlyWebhookStats      `json:"hourly_stats"`
	DailyStats       []DailyWebhookStats       `json:"daily_stats"`
	ErrorBreakdown   map[string]int64          `json:"error_breakdown"`
	ResponseCodes    map[int]int64             `json:"response_codes"`
	TopErrors        []ErrorStat               `json:"top_errors"`
	PerformanceMetrics *PerformanceMetrics     `json:"performance_metrics"`
}

// EndpointAnalytics contains analytics for a specific endpoint
type EndpointAnalytics struct {
	EndpointID      string        `json:"endpoint_id"`
	Name            string        `json:"name"`
	URL             string        `json:"url"`
	TotalRequests   int64         `json:"total_requests"`
	SuccessfulRequests int64      `json:"successful_requests"`
	FailedRequests  int64         `json:"failed_requests"`
	SuccessRate     float64       `json:"success_rate"`
	AvgResponseTime time.Duration `json:"avg_response_time"`
	LastSuccess     *time.Time    `json:"last_success"`
	LastFailure     *time.Time    `json:"last_failure"`
}

// HourlyWebhookStats contains hourly statistics
type HourlyWebhookStats struct {
	Hour            time.Time `json:"hour"`
	TotalRequests   int64     `json:"total_requests"`
	SuccessfulRequests int64  `json:"successful_requests"`
	FailedRequests  int64     `json:"failed_requests"`
	AvgResponseTime time.Duration `json:"avg_response_time"`
}

// DailyWebhookStats contains daily statistics
type DailyWebhookStats struct {
	Date            time.Time `json:"date"`
	TotalRequests   int64     `json:"total_requests"`
	SuccessfulRequests int64  `json:"successful_requests"`
	FailedRequests  int64     `json:"failed_requests"`
	AvgResponseTime time.Duration `json:"avg_response_time"`
}

// ErrorStat contains error statistics
type ErrorStat struct {
	Error      string `json:"error"`
	Count      int64  `json:"count"`
	Percentage float64 `json:"percentage"`
}

// PerformanceMetrics contains detailed performance metrics
type PerformanceMetrics struct {
	P50ResponseTime time.Duration `json:"p50_response_time"`
	P90ResponseTime time.Duration `json:"p90_response_time"`
	P95ResponseTime time.Duration `json:"p95_response_time"`
	P99ResponseTime time.Duration `json:"p99_response_time"`
	MinResponseTime time.Duration `json:"min_response_time"`
	MaxResponseTime time.Duration `json:"max_response_time"`
}

// WebhookMonitoringData contains real-time monitoring information
type WebhookMonitoringData struct {
	FormID          string                  `json:"form_id"`
	Status          string                  `json:"status"` // healthy, warning, critical
	ActiveEndpoints int                     `json:"active_endpoints"`
	FailingEndpoints int                    `json:"failing_endpoints"`
	CurrentLoad     int                     `json:"current_load"`
	QueueSize       int                     `json:"queue_size"`
	RecentEvents    []MonitorEvent          `json:"recent_events"`
	HealthChecks    []EndpointHealthCheck   `json:"health_checks"`
	Alerts          []MonitoringAlert       `json:"alerts"`
}

// EndpointHealthCheck contains health check information
type EndpointHealthCheck struct {
	EndpointID      string    `json:"endpoint_id"`
	Status          string    `json:"status"` // healthy, unhealthy, unknown
	LastCheck       time.Time `json:"last_check"`
	ResponseTime    time.Duration `json:"response_time"`
	SuccessRate     float64   `json:"success_rate"`
	ErrorMessage    string    `json:"error_message,omitempty"`
}

// MonitoringAlert represents a monitoring alert
type MonitoringAlert struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`
	Severity    string                 `json:"severity"`
	Message     string                 `json:"message"`
	EndpointID  string                 `json:"endpoint_id,omitempty"`
	Timestamp   time.Time              `json:"timestamp"`
	Acknowledged bool                  `json:"acknowledged"`
	Data        map[string]interface{} `json:"data,omitempty"`
}

// Database operations

func (ews *EnhancedWebhookService) getFormWebhookConfig(formID string) (*FormWebhookConfig, error) {
	query := `SELECT webhook_config FROM forms WHERE id = ? AND webhook_config IS NOT NULL`
	
	var configJSON string
	err := ews.db.QueryRow(query, formID).Scan(&configJSON)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // No config found
		}
		return nil, err
	}
	
	var config FormWebhookConfig
	if err := json.Unmarshal([]byte(configJSON), &config); err != nil {
		return nil, err
	}
	
	return &config, nil
}

func (ews *EnhancedWebhookService) saveFormWebhookConfig(config *FormWebhookConfig) error {
	configJSON, err := json.Marshal(config)
	if err != nil {
		return err
	}
	
	query := `UPDATE forms SET webhook_config = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`
	_, err = ews.db.Exec(query, string(configJSON), config.FormID)
	return err
}

func (ews *EnhancedWebhookService) archiveEndpointData(formID, endpointID string) error {
	// Archive webhook logs and analytics data
	query := `
		UPDATE webhook_notifications 
		SET archived = TRUE, archived_at = CURRENT_TIMESTAMP 
		WHERE form_id = ? AND webhook_endpoint_id = ?
	`
	_, err := ews.db.Exec(query, formID, endpointID)
	return err
}

// Background services initialization

func (ews *EnhancedWebhookService) startBackgroundServices() {
	// Start worker pool
	ews.workerPool.Start()
	
	// Start monitoring
	go ews.monitor.Start()
	
	// Start scheduler for cleanup tasks
	ews.scheduler.AddFunc("@hourly", func() {
		ews.cleanupOldWebhookData()
	})
	
	ews.scheduler.AddFunc("*/5 * * * *", func() { // Every 5 minutes
		ews.updateEndpointHealthChecks()
	})
	
	ews.scheduler.Start()
	
	log.Println("Enhanced webhook service background services started")
}

func (ews *EnhancedWebhookService) cleanupOldWebhookData() {
	// Clean up webhook logs older than 90 days
	query := `
		DELETE FROM webhook_notifications 
		WHERE created_at < DATE_SUB(NOW(), INTERVAL 90 DAY) 
		AND archived = TRUE
		LIMIT 1000
	`
	
	result, err := ews.db.Exec(query)
	if err != nil {
		log.Printf("Failed to cleanup old webhook data: %v", err)
		return
	}
	
	affected, _ := result.RowsAffected()
	if affected > 0 {
		log.Printf("Cleaned up %d old webhook records", affected)
	}
}

func (ews *EnhancedWebhookService) updateEndpointHealthChecks() {
	// Update endpoint health checks
	// This is a placeholder for actual health check implementation
	log.Println("Updating endpoint health checks...")
}

// Shutdown gracefully shuts down the webhook service
func (ews *EnhancedWebhookService) Shutdown(ctx context.Context) error {
	log.Println("Shutting down enhanced webhook service...")
	
	// Stop scheduler
	ews.scheduler.Stop()
	
	// Stop worker pool
	ews.workerPool.Stop()
	
	// Stop monitor
	ews.monitor.Stop()
	
	log.Println("Enhanced webhook service shutdown complete")
	return nil
}