package models

import (
	"time"

	"github.com/google/uuid"
)

// User represents a client who uses the form service
type User struct {
	ID          uuid.UUID `json:"id" db:"id"`
	Email       string    `json:"email" db:"email"`
	Password    string    `json:"-" db:"password_hash"`
	FirstName   string    `json:"first_name" db:"first_name"`
	LastName    string    `json:"last_name" db:"last_name"`
	Company     string    `json:"company" db:"company"`
	PlanType    string    `json:"plan_type" db:"plan_type"` // free, starter, professional, enterprise
	IsActive    bool      `json:"is_active" db:"is_active"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

// APIKey represents an API key for form submissions
type APIKey struct {
	ID          uuid.UUID `json:"id" db:"id"`
	UserID      uuid.UUID `json:"user_id" db:"user_id"`
	Name        string    `json:"name" db:"name"`
	KeyHash     string    `json:"-" db:"key_hash"`
	Key         string    `json:"key,omitempty"` // Only shown when created
	Permissions string    `json:"permissions" db:"permissions"` // JSON string of permissions
	RateLimit   int       `json:"rate_limit" db:"rate_limit"`   // requests per minute
	IsActive    bool      `json:"is_active" db:"is_active"`
	LastUsedAt  *time.Time `json:"last_used_at" db:"last_used_at"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

// Form represents a form configuration
type Form struct {
	ID              uuid.UUID `json:"id" db:"id"`
	UserID          uuid.UUID `json:"user_id" db:"user_id"`
	Name            string    `json:"name" db:"name"`
	Description     string    `json:"description" db:"description"`
	TargetEmail     string    `json:"target_email" db:"target_email"`
	CCEmails        string    `json:"cc_emails" db:"cc_emails"` // JSON array of emails
	Subject         string    `json:"subject" db:"subject"`
	SuccessMessage  string    `json:"success_message" db:"success_message"`
	RedirectURL     string    `json:"redirect_url" db:"redirect_url"`
	WebhookURL      string    `json:"webhook_url" db:"webhook_url"`
	SpamProtection  bool      `json:"spam_protection" db:"spam_protection"`
	RecaptchaSecret string    `json:"-" db:"recaptcha_secret"`
	FileUploads     bool      `json:"file_uploads" db:"file_uploads"`
	MaxFileSize     int64     `json:"max_file_size" db:"max_file_size"` // in bytes
	AllowedOrigins  string    `json:"allowed_origins" db:"allowed_origins"` // JSON array of domains
	IsActive        bool      `json:"is_active" db:"is_active"`
	SubmissionCount int64     `json:"submission_count" db:"submission_count"`
	CreatedAt       time.Time `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time `json:"updated_at" db:"updated_at"`
}

// Submission represents a form submission
type Submission struct {
	ID         uuid.UUID              `json:"id" db:"id"`
	FormID     uuid.UUID              `json:"form_id" db:"form_id"`
	Data       map[string]interface{} `json:"data" db:"data"` // JSON data
	Files      []FileUpload           `json:"files,omitempty"`
	IPAddress  string                 `json:"ip_address" db:"ip_address"`
	UserAgent  string                 `json:"user_agent" db:"user_agent"`
	Referrer   string                 `json:"referrer" db:"referrer"`
	IsSpam     bool                   `json:"is_spam" db:"is_spam"`
	SpamScore  float64                `json:"spam_score" db:"spam_score"`
	EmailSent  bool                   `json:"email_sent" db:"email_sent"`
	WebhookSent bool                  `json:"webhook_sent" db:"webhook_sent"`
	CreatedAt  time.Time              `json:"created_at" db:"created_at"`
}

// FileUpload represents an uploaded file
type FileUpload struct {
	ID           uuid.UUID `json:"id" db:"id"`
	SubmissionID uuid.UUID `json:"submission_id" db:"submission_id"`
	FileName     string    `json:"file_name" db:"file_name"`
	OriginalName string    `json:"original_name" db:"original_name"`
	ContentType  string    `json:"content_type" db:"content_type"`
	Size         int64     `json:"size" db:"size"`
	StoragePath  string    `json:"storage_path" db:"storage_path"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
}

// PlanLimits defines limits for each plan type
type PlanLimits struct {
	SubmissionsPerMonth int
	FormsLimit         int
	FileUploads        bool
	Webhooks           bool
	Analytics          bool
	EmailSupport       bool
	PrioritySupport    bool
	WhiteLabel         bool
}

var PlanLimitsMap = map[string]PlanLimits{
	"free": {
		SubmissionsPerMonth: 100,
		FormsLimit:         1,
		FileUploads:        false,
		Webhooks:           false,
		Analytics:          false,
		EmailSupport:       true,
		PrioritySupport:    false,
		WhiteLabel:         false,
	},
	"starter": {
		SubmissionsPerMonth: 1000,
		FormsLimit:         5,
		FileUploads:        true,
		Webhooks:           false,
		Analytics:          true,
		EmailSupport:       true,
		PrioritySupport:    false,
		WhiteLabel:         false,
	},
	"professional": {
		SubmissionsPerMonth: 10000,
		FormsLimit:         -1, // unlimited
		FileUploads:        true,
		Webhooks:           true,
		Analytics:          true,
		EmailSupport:       true,
		PrioritySupport:    true,
		WhiteLabel:         false,
	},
	"enterprise": {
		SubmissionsPerMonth: 100000,
		FormsLimit:         -1, // unlimited
		FileUploads:        true,
		Webhooks:           true,
		Analytics:          true,
		EmailSupport:       true,
		PrioritySupport:    true,
		WhiteLabel:         true,
	},
}

// Advanced Form Field Types
type FormFieldType string

const (
	FieldTypeText     FormFieldType = "text"
	FieldTypeEmail    FormFieldType = "email"
	FieldTypeNumber   FormFieldType = "number"
	FieldTypeDate     FormFieldType = "date"
	FieldTypeTime     FormFieldType = "time"
	FieldTypeDateTime FormFieldType = "datetime"
	FieldTypeURL      FormFieldType = "url"
	FieldTypeTel      FormFieldType = "tel"
	FieldTypeTextarea FormFieldType = "textarea"
	FieldTypeSelect   FormFieldType = "select"
	FieldTypeRadio    FormFieldType = "radio"
	FieldTypeCheckbox FormFieldType = "checkbox"
	FieldTypeFile     FormFieldType = "file"
	FieldTypeHidden   FormFieldType = "hidden"
	FieldTypePassword FormFieldType = "password"
)

// Form Field Configuration
type FormField struct {
	ID            uuid.UUID                 `json:"id" db:"id"`
	FormID        uuid.UUID                 `json:"form_id" db:"form_id"`
	Name          string                    `json:"name" db:"name"`
	Label         string                    `json:"label" db:"label"`
	Type          FormFieldType             `json:"type" db:"type"`
	Required      bool                      `json:"required" db:"required"`
	Placeholder   string                    `json:"placeholder" db:"placeholder"`
	DefaultValue  string                    `json:"default_value" db:"default_value"`
	Options       []FormFieldOption         `json:"options,omitempty"`
	Validation    FormFieldValidation       `json:"validation" db:"validation"`
	FileSettings  *FormFieldFileSettings    `json:"file_settings,omitempty" db:"file_settings"`
	Order         int                       `json:"order" db:"field_order"`
	IsActive      bool                      `json:"is_active" db:"is_active"`
	CreatedAt     time.Time                 `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time                 `json:"updated_at" db:"updated_at"`
}

// Form Field Options for select, radio, checkbox
type FormFieldOption struct {
	ID       uuid.UUID `json:"id" db:"id"`
	FieldID  uuid.UUID `json:"field_id" db:"field_id"`
	Label    string    `json:"label" db:"label"`
	Value    string    `json:"value" db:"value"`
	Selected bool      `json:"selected" db:"selected"`
	Order    int       `json:"order" db:"option_order"`
}

// Form Field Validation Rules
type FormFieldValidation struct {
	MinLength    *int     `json:"min_length,omitempty"`
	MaxLength    *int     `json:"max_length,omitempty"`
	MinValue     *float64 `json:"min_value,omitempty"`
	MaxValue     *float64 `json:"max_value,omitempty"`
	Pattern      string   `json:"pattern,omitempty"`
	CustomError  string   `json:"custom_error,omitempty"`
	AcceptedTypes []string `json:"accepted_types,omitempty"` // For file fields
}

// File Upload Settings for file fields
type FormFieldFileSettings struct {
	MaxFiles        int      `json:"max_files"`
	MaxFileSize     int64    `json:"max_file_size"`     // in bytes
	AllowedTypes    []string `json:"allowed_types"`     // mime types
	AllowedExts     []string `json:"allowed_extensions"` // file extensions
	RequirePreview  bool     `json:"require_preview"`
	AllowMultiple   bool     `json:"allow_multiple"`
}

// Enhanced Form with Field Configuration Support
type FormWithFields struct {
	Form   `json:",inline"`
	Fields []FormField `json:"fields,omitempty"`
}

// File Upload Result
type FileUploadResult struct {
	ID           uuid.UUID `json:"id"`
	FileName     string    `json:"file_name"`
	OriginalName string    `json:"original_name"`
	Size         int64     `json:"size"`
	ContentType  string    `json:"content_type"`
	URL          string    `json:"url,omitempty"`
	Error        string    `json:"error,omitempty"`
}

// Form Validation Result
type FormValidationResult struct {
	IsValid      bool                           `json:"is_valid"`
	Errors       map[string][]string           `json:"errors,omitempty"`
	FieldResults map[string]FieldValidationResult `json:"field_results,omitempty"`
}

// Field Validation Result
type FieldValidationResult struct {
	IsValid bool     `json:"is_valid"`
	Value   interface{} `json:"value"`
	Errors  []string `json:"errors,omitempty"`
}

// Request/Response DTOs
type RegisterRequest struct {
	Email     string `json:"email" binding:"required,email"`
	Password  string `json:"password" binding:"required,min=8"`
	FirstName string `json:"first_name" binding:"required"`
	LastName  string `json:"last_name" binding:"required"`
	Company   string `json:"company"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type AuthResponse struct {
	User         *User  `json:"user"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type CreateFormRequest struct {
	Name            string   `json:"name" binding:"required"`
	Description     string   `json:"description"`
	TargetEmail     string   `json:"target_email" binding:"required,email"`
	CCEmails        []string `json:"cc_emails"`
	Subject         string   `json:"subject"`
	SuccessMessage  string   `json:"success_message"`
	RedirectURL     string   `json:"redirect_url"`
	WebhookURL      string   `json:"webhook_url"`
	SpamProtection  bool     `json:"spam_protection"`
	RecaptchaSecret string   `json:"recaptcha_secret"`
	FileUploads     bool     `json:"file_uploads"`
	MaxFileSize     int64    `json:"max_file_size"`
	AllowedOrigins  []string `json:"allowed_origins"`
}

// Enhanced Form Creation with Field Configuration
type CreateFormWithFieldsRequest struct {
	CreateFormRequest `json:",inline"`
	Fields           []CreateFormFieldRequest `json:"fields,omitempty"`
}

// Create Form Field Request
type CreateFormFieldRequest struct {
	Name         string                    `json:"name" binding:"required"`
	Label        string                    `json:"label" binding:"required"`
	Type         FormFieldType             `json:"type" binding:"required"`
	Required     bool                      `json:"required"`
	Placeholder  string                    `json:"placeholder"`
	DefaultValue string                    `json:"default_value"`
	Options      []CreateFieldOptionRequest `json:"options,omitempty"`
	Validation   FormFieldValidation       `json:"validation"`
	FileSettings *FormFieldFileSettings    `json:"file_settings,omitempty"`
	Order        int                       `json:"order"`
}

// Create Field Option Request
type CreateFieldOptionRequest struct {
	Label    string `json:"label" binding:"required"`
	Value    string `json:"value" binding:"required"`
	Selected bool   `json:"selected"`
	Order    int    `json:"order"`
}

// Enhanced Submission Request with File Support
type SubmissionRequest struct {
	AccessKey         string                 `json:"access_key" form:"access_key" binding:"required"`
	Data              map[string]interface{} `json:"-" form:"-"`
	Email             string                 `json:"email" form:"email"`
	Subject           string                 `json:"subject" form:"subject"`
	Message           string                 `json:"message" form:"message"`
	RedirectURL       string                 `json:"redirect" form:"redirect"`
	RecaptchaResponse string                 `json:"g-recaptcha-response" form:"g-recaptcha-response"`
	Files             []FileUploadResult     `json:"files,omitempty"`
}

// File Upload Request
type FileUploadRequest struct {
	FormID    uuid.UUID `json:"form_id" binding:"required"`
	FieldName string    `json:"field_name" binding:"required"`
	AccessKey string    `json:"access_key" binding:"required"`
}

// Bulk File Upload Request
type BulkFileUploadRequest struct {
	FormID    uuid.UUID `json:"form_id" binding:"required"`
	FieldName string    `json:"field_name" binding:"required"`
	AccessKey string    `json:"access_key" binding:"required"`
	MaxFiles  int       `json:"max_files,omitempty"`
}

type SubmissionResponse struct {
	Success    bool                   `json:"success"`
	StatusCode int                    `json:"statusCode"`
	Message    string                 `json:"message"`
	Data       map[string]interface{} `json:"data,omitempty"`
	RedirectURL string                `json:"redirect_url,omitempty"`
}

// Email Template System Models

// EmailProvider represents different email service providers
type EmailProviderType string

const (
	ProviderSMTP     EmailProviderType = "smtp"
	ProviderSendGrid EmailProviderType = "sendgrid"
	ProviderMailgun  EmailProviderType = "mailgun"
	ProviderSES      EmailProviderType = "aws_ses"
	ProviderPostmark EmailProviderType = "postmark"
	ProviderMailjet  EmailProviderType = "mailjet"
)

// EmailProvider stores email service provider configurations
type EmailProvider struct {
	ID          uuid.UUID         `json:"id" db:"id"`
	UserID      uuid.UUID         `json:"user_id" db:"user_id"`
	Name        string            `json:"name" db:"name"`
	Type        EmailProviderType `json:"type" db:"type"`
	Config      EmailProviderConfig `json:"config" db:"config"` // JSON configuration
	IsActive    bool              `json:"is_active" db:"is_active"`
	IsDefault   bool              `json:"is_default" db:"is_default"`
	CreatedAt   time.Time         `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at" db:"updated_at"`
}

// EmailProviderConfig holds provider-specific configuration
type EmailProviderConfig struct {
	// SMTP Configuration
	Host     string `json:"host,omitempty"`
	Port     int    `json:"port,omitempty"`
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
	UseTLS   bool   `json:"use_tls,omitempty"`
	
	// API-based providers
	APIKey    string `json:"api_key,omitempty"`
	APISecret string `json:"api_secret,omitempty"`
	Domain    string `json:"domain,omitempty"`
	Region    string `json:"region,omitempty"`
	
	// Common settings
	FromName   string `json:"from_name,omitempty"`
	FromEmail  string `json:"from_email,omitempty"`
	ReplyTo    string `json:"reply_to,omitempty"`
	ReturnPath string `json:"return_path,omitempty"`
}

// EmailTemplateType represents different types of email templates
type EmailTemplateType string

const (
	TemplateTypeNotification   EmailTemplateType = "notification"    // Form submission notifications
	TemplateTypeAutoresponder  EmailTemplateType = "autoresponder"   // Auto-replies to users
	TemplateTypeWelcome        EmailTemplateType = "welcome"         // Welcome emails
	TemplateTypeConfirmation   EmailTemplateType = "confirmation"    // Email confirmations
	TemplateTypeFollowUp       EmailTemplateType = "follow_up"       // Follow-up emails
	TemplateTypeCustom         EmailTemplateType = "custom"          // Custom templates
)

// EmailTemplate represents an email template
type EmailTemplate struct {
	ID             uuid.UUID         `json:"id" db:"id"`
	UserID         uuid.UUID         `json:"user_id" db:"user_id"`
	FormID         *uuid.UUID        `json:"form_id,omitempty" db:"form_id"` // Optional: specific to a form
	Name           string            `json:"name" db:"name"`
	Description    string            `json:"description" db:"description"`
	Type           EmailTemplateType `json:"type" db:"type"`
	Language       string            `json:"language" db:"language"` // ISO language code (en, es, fr, etc.)
	Subject        string            `json:"subject" db:"subject"`
	HTMLContent    string            `json:"html_content" db:"html_content"`
	TextContent    string            `json:"text_content" db:"text_content"`
	Variables      []string          `json:"variables" db:"variables"`     // JSON array of available variables
	ParentID       *uuid.UUID        `json:"parent_id,omitempty" db:"parent_id"` // For template inheritance
	IsActive       bool              `json:"is_active" db:"is_active"`
	IsDefault      bool              `json:"is_default" db:"is_default"`
	Version        int               `json:"version" db:"version"`
	Tags           []string          `json:"tags" db:"tags"` // JSON array for categorization
	CreatedAt      time.Time         `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time         `json:"updated_at" db:"updated_at"`
}

// EmailAutoresponder represents an autoresponder configuration
type EmailAutoresponder struct {
	ID             uuid.UUID       `json:"id" db:"id"`
	UserID         uuid.UUID       `json:"user_id" db:"user_id"`
	FormID         uuid.UUID       `json:"form_id" db:"form_id"`
	Name           string          `json:"name" db:"name"`
	TemplateID     uuid.UUID       `json:"template_id" db:"template_id"`
	ProviderID     *uuid.UUID      `json:"provider_id,omitempty" db:"provider_id"`
	IsEnabled      bool            `json:"is_enabled" db:"is_enabled"`
	DelayMinutes   int             `json:"delay_minutes" db:"delay_minutes"` // 0 for immediate
	Conditions     AutoresponderConditions `json:"conditions" db:"conditions"` // JSON conditions
	SendToField    string          `json:"send_to_field" db:"send_to_field"` // Form field containing recipient email
	CCEmails       []string        `json:"cc_emails" db:"cc_emails"` // JSON array
	BCCEmails      []string        `json:"bcc_emails" db:"bcc_emails"` // JSON array
	ReplyTo        string          `json:"reply_to" db:"reply_to"`
	TrackOpens     bool            `json:"track_opens" db:"track_opens"`
	TrackClicks    bool            `json:"track_clicks" db:"track_clicks"`
	CreatedAt      time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at" db:"updated_at"`
}

// AutoresponderConditions defines when an autoresponder should trigger
type AutoresponderConditions struct {
	FieldConditions []FieldCondition `json:"field_conditions,omitempty"`
	TimeConditions  *TimeCondition   `json:"time_conditions,omitempty"`
	LogicalOperator string           `json:"logical_operator,omitempty"` // "AND" or "OR"
}

// FieldCondition represents a condition based on form field values
type FieldCondition struct {
	FieldName string   `json:"field_name"`
	Operator  string   `json:"operator"` // equals, contains, starts_with, ends_with, not_equals, etc.
	Value     string   `json:"value"`
	Values    []string `json:"values,omitempty"` // For "in" operator
}

// TimeCondition represents time-based conditions
type TimeCondition struct {
	StartTime string   `json:"start_time,omitempty"` // HH:MM format
	EndTime   string   `json:"end_time,omitempty"`   // HH:MM format
	Days      []string `json:"days,omitempty"`       // monday, tuesday, etc.
	TimeZone  string   `json:"timezone,omitempty"`
}

// EmailQueue represents queued emails for delayed/scheduled sending
type EmailQueue struct {
	ID             uuid.UUID              `json:"id" db:"id"`
	UserID         uuid.UUID              `json:"user_id" db:"user_id"`
	FormID         *uuid.UUID             `json:"form_id,omitempty" db:"form_id"`
	SubmissionID   *uuid.UUID             `json:"submission_id,omitempty" db:"submission_id"`
	TemplateID     uuid.UUID              `json:"template_id" db:"template_id"`
	ProviderID     *uuid.UUID             `json:"provider_id,omitempty" db:"provider_id"`
	ToEmails       []string               `json:"to_emails" db:"to_emails"` // JSON array
	CCEmails       []string               `json:"cc_emails" db:"cc_emails"` // JSON array
	BCCEmails      []string               `json:"bcc_emails" db:"bcc_emails"` // JSON array
	Subject        string                 `json:"subject" db:"subject"`
	HTMLContent    string                 `json:"html_content" db:"html_content"`
	TextContent    string                 `json:"text_content" db:"text_content"`
	Variables      map[string]interface{} `json:"variables" db:"variables"` // JSON template variables
	ScheduledAt    time.Time              `json:"scheduled_at" db:"scheduled_at"`
	SentAt         *time.Time             `json:"sent_at,omitempty" db:"sent_at"`
	Status         EmailStatus            `json:"status" db:"status"`
	Attempts       int                    `json:"attempts" db:"attempts"`
	LastError      string                 `json:"last_error" db:"last_error"`
	Priority       int                    `json:"priority" db:"priority"` // Higher number = higher priority
	CreatedAt      time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at" db:"updated_at"`
}

// EmailStatus represents the status of an email in the queue
type EmailStatus string

const (
	EmailStatusPending   EmailStatus = "pending"
	EmailStatusSending   EmailStatus = "sending"
	EmailStatusSent      EmailStatus = "sent"
	EmailStatusFailed    EmailStatus = "failed"
	EmailStatusCancelled EmailStatus = "cancelled"
	EmailStatusScheduled EmailStatus = "scheduled"
)

// EmailAnalytics tracks email delivery and engagement metrics
type EmailAnalytics struct {
	ID             uuid.UUID  `json:"id" db:"id"`
	QueueID        uuid.UUID  `json:"queue_id" db:"queue_id"`
	UserID         uuid.UUID  `json:"user_id" db:"user_id"`
	FormID         *uuid.UUID `json:"form_id,omitempty" db:"form_id"`
	TemplateID     uuid.UUID  `json:"template_id" db:"template_id"`
	EmailAddress   string     `json:"email_address" db:"email_address"`
	DeliveredAt    *time.Time `json:"delivered_at,omitempty" db:"delivered_at"`
	OpenedAt       *time.Time `json:"opened_at,omitempty" db:"opened_at"`
	FirstClickedAt *time.Time `json:"first_clicked_at,omitempty" db:"first_clicked_at"`
	OpenCount      int        `json:"open_count" db:"open_count"`
	ClickCount     int        `json:"click_count" db:"click_count"`
	Links          []LinkClick `json:"links" db:"links"` // JSON array of clicked links
	UserAgent      string     `json:"user_agent" db:"user_agent"`
	IPAddress      string     `json:"ip_address" db:"ip_address"`
	CreatedAt      time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at" db:"updated_at"`
}

// LinkClick tracks individual link clicks in emails
type LinkClick struct {
	URL       string    `json:"url"`
	ClickedAt time.Time `json:"clicked_at"`
	Count     int       `json:"count"`
}

// EmailABTest represents A/B testing for email templates
type EmailABTest struct {
	ID            uuid.UUID  `json:"id" db:"id"`
	UserID        uuid.UUID  `json:"user_id" db:"user_id"`
	FormID        *uuid.UUID `json:"form_id,omitempty" db:"form_id"`
	Name          string     `json:"name" db:"name"`
	Description   string     `json:"description" db:"description"`
	TemplateAID   uuid.UUID  `json:"template_a_id" db:"template_a_id"`
	TemplateBID   uuid.UUID  `json:"template_b_id" db:"template_b_id"`
	TrafficSplit  int        `json:"traffic_split" db:"traffic_split"` // Percentage for A (0-100)
	Status        ABTestStatus `json:"status" db:"status"`
	StartedAt     *time.Time `json:"started_at,omitempty" db:"started_at"`
	EndedAt       *time.Time `json:"ended_at,omitempty" db:"ended_at"`
	Winner        *uuid.UUID `json:"winner,omitempty" db:"winner"` // Template ID of the winner
	StatsSentA    int        `json:"stats_sent_a" db:"stats_sent_a"`
	StatsSentB    int        `json:"stats_sent_b" db:"stats_sent_b"`
	StatsOpenA    int        `json:"stats_open_a" db:"stats_open_a"`
	StatsOpenB    int        `json:"stats_open_b" db:"stats_open_b"`
	StatsClickA   int        `json:"stats_click_a" db:"stats_click_a"`
	StatsClickB   int        `json:"stats_click_b" db:"stats_click_b"`
	CreatedAt     time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at" db:"updated_at"`
}

// Removed duplicate ABTestStatus - now defined in analytics_models.go

// EmailTemplateCategory for organizing templates
type EmailTemplateCategory struct {
	ID          uuid.UUID `json:"id" db:"id"`
	UserID      uuid.UUID `json:"user_id" db:"user_id"`
	Name        string    `json:"name" db:"name"`
	Description string    `json:"description" db:"description"`
	Color       string    `json:"color" db:"color"` // Hex color for UI
	Icon        string    `json:"icon" db:"icon"`   // Icon identifier
	Order       int       `json:"order" db:"category_order"`
	IsSystem    bool      `json:"is_system" db:"is_system"` // System-defined categories
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

// EmailTemplateLibrary for pre-built templates
type EmailTemplateLibrary struct {
	ID          uuid.UUID         `json:"id" db:"id"`
	Name        string            `json:"name" db:"name"`
	Description string            `json:"description" db:"description"`
	Type        EmailTemplateType `json:"type" db:"type"`
	CategoryID  uuid.UUID         `json:"category_id" db:"category_id"`
	Subject     string            `json:"subject" db:"subject"`
	HTMLContent string            `json:"html_content" db:"html_content"`
	TextContent string            `json:"text_content" db:"text_content"`
	Variables   []string          `json:"variables" db:"variables"`
	Tags        []string          `json:"tags" db:"tags"`
	Preview     string            `json:"preview" db:"preview"` // Base64 image or URL
	IsPublic    bool              `json:"is_public" db:"is_public"`
	UsageCount  int               `json:"usage_count" db:"usage_count"`
	Rating      float64           `json:"rating" db:"rating"`
	CreatedAt   time.Time         `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at" db:"updated_at"`
}

// Request/Response DTOs for Email Templates

// CreateEmailProviderRequest
type CreateEmailProviderRequest struct {
	Name      string              `json:"name" binding:"required"`
	Type      EmailProviderType   `json:"type" binding:"required"`
	Config    EmailProviderConfig `json:"config" binding:"required"`
	IsDefault bool                `json:"is_default"`
}

// CreateEmailTemplateRequest
type CreateEmailTemplateRequest struct {
	FormID      *uuid.UUID        `json:"form_id,omitempty"`
	Name        string            `json:"name" binding:"required"`
	Description string            `json:"description"`
	Type        EmailTemplateType `json:"type" binding:"required"`
	Language    string            `json:"language"`
	Subject     string            `json:"subject" binding:"required"`
	HTMLContent string            `json:"html_content" binding:"required"`
	TextContent string            `json:"text_content"`
	Variables   []string          `json:"variables"`
	ParentID    *uuid.UUID        `json:"parent_id,omitempty"`
	Tags        []string          `json:"tags"`
}

// CreateAutoresponderRequest
type CreateAutoresponderRequest struct {
	FormID         uuid.UUID               `json:"form_id" binding:"required"`
	Name           string                  `json:"name" binding:"required"`
	TemplateID     uuid.UUID               `json:"template_id" binding:"required"`
	ProviderID     *uuid.UUID              `json:"provider_id,omitempty"`
	DelayMinutes   int                     `json:"delay_minutes"`
	Conditions     AutoresponderConditions `json:"conditions"`
	SendToField    string                  `json:"send_to_field" binding:"required"`
	CCEmails       []string                `json:"cc_emails"`
	BCCEmails      []string                `json:"bcc_emails"`
	ReplyTo        string                  `json:"reply_to"`
	TrackOpens     bool                    `json:"track_opens"`
	TrackClicks    bool                    `json:"track_clicks"`
}

// EmailTemplatePreviewRequest
type EmailTemplatePreviewRequest struct {
	TemplateID uuid.UUID              `json:"template_id" binding:"required"`
	Variables  map[string]interface{} `json:"variables"`
}

// EmailAnalyticsResponse
type EmailAnalyticsResponse struct {
	TemplateID     uuid.UUID `json:"template_id"`
	TotalSent      int       `json:"total_sent"`
	TotalDelivered int       `json:"total_delivered"`
	TotalOpened    int       `json:"total_opened"`
	TotalClicked   int       `json:"total_clicked"`
	OpenRate       float64   `json:"open_rate"`
	ClickRate      float64   `json:"click_rate"`
	DeliveryRate   float64   `json:"delivery_rate"`
}