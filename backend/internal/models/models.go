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

type SubmissionRequest struct {
	AccessKey    string                 `json:"access_key" form:"access_key" binding:"required"`
	Data         map[string]interface{} `json:"-" form:"-"`
	Email        string                 `json:"email" form:"email"`
	Subject      string                 `json:"subject" form:"subject"`
	Message      string                 `json:"message" form:"message"`
	RedirectURL  string                 `json:"redirect" form:"redirect"`
	RecaptchaResponse string            `json:"g-recaptcha-response" form:"g-recaptcha-response"`
}

type SubmissionResponse struct {
	Success    bool                   `json:"success"`
	StatusCode int                    `json:"statusCode"`
	Message    string                 `json:"message"`
	Data       map[string]interface{} `json:"data,omitempty"`
	RedirectURL string                `json:"redirect_url,omitempty"`
}