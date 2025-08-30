package models

import (
	"time"

	"github.com/google/uuid"
)

// Form Analytics Event Types
type AnalyticsEventType string

const (
	EventTypeFormView           AnalyticsEventType = "form_view"
	EventTypeFormStart          AnalyticsEventType = "form_start"
	EventTypeFieldFocus         AnalyticsEventType = "field_focus"
	EventTypeFieldBlur          AnalyticsEventType = "field_blur"
	EventTypeFieldChange        AnalyticsEventType = "field_change"
	EventTypeFormSubmit         AnalyticsEventType = "form_submit"
	EventTypeFormComplete       AnalyticsEventType = "form_complete"
	EventTypeFormAbandon        AnalyticsEventType = "form_abandon"
	EventTypeValidationError    AnalyticsEventType = "validation_error"
	EventTypeFileUploadStart    AnalyticsEventType = "file_upload_start"
	EventTypeFileUploadComplete AnalyticsEventType = "file_upload_complete"
	EventTypeRecaptchaSolve     AnalyticsEventType = "recaptcha_solve"
)

// Device Types
type DeviceType string

const (
	DeviceTypeDesktop DeviceType = "desktop"
	DeviceTypeTablet  DeviceType = "tablet"
	DeviceTypeMobile  DeviceType = "mobile"
	DeviceTypeUnknown DeviceType = "unknown"
)

// FormAnalyticsEvent represents a single analytics event
type FormAnalyticsEvent struct {
	ID                   uuid.UUID          `json:"id" db:"id"`
	FormID               uuid.UUID          `json:"form_id" db:"form_id"`
	UserID               uuid.UUID          `json:"user_id" db:"user_id"`
	SessionID            string             `json:"session_id" db:"session_id"`
	EventType            AnalyticsEventType `json:"event_type" db:"event_type"`
	FieldName            *string            `json:"field_name,omitempty" db:"field_name"`
	FieldValueLength     *int               `json:"field_value_length,omitempty" db:"field_value_length"`
	FieldValidationError *string            `json:"field_validation_error,omitempty" db:"field_validation_error"`
	PageURL              string             `json:"page_url" db:"page_url"`
	Referrer             *string            `json:"referrer,omitempty" db:"referrer"`
	UTMSource            *string            `json:"utm_source,omitempty" db:"utm_source"`
	UTMMedium            *string            `json:"utm_medium,omitempty" db:"utm_medium"`
	UTMCampaign          *string            `json:"utm_campaign,omitempty" db:"utm_campaign"`
	UTMTerm              *string            `json:"utm_term,omitempty" db:"utm_term"`
	UTMContent           *string            `json:"utm_content,omitempty" db:"utm_content"`
	DeviceType           DeviceType         `json:"device_type" db:"device_type"`
	BrowserName          *string            `json:"browser_name,omitempty" db:"browser_name"`
	BrowserVersion       *string            `json:"browser_version,omitempty" db:"browser_version"`
	OSName               *string            `json:"os_name,omitempty" db:"os_name"`
	OSVersion            *string            `json:"os_version,omitempty" db:"os_version"`
	ScreenResolution     *string            `json:"screen_resolution,omitempty" db:"screen_resolution"`
	ViewportSize         *string            `json:"viewport_size,omitempty" db:"viewport_size"`
	IPAddress            string             `json:"ip_address" db:"ip_address"`
	CountryCode          *string            `json:"country_code,omitempty" db:"country_code"`
	CountryName          *string            `json:"country_name,omitempty" db:"country_name"`
	Region               *string            `json:"region,omitempty" db:"region"`
	City                 *string            `json:"city,omitempty" db:"city"`
	Latitude             *float64           `json:"latitude,omitempty" db:"latitude"`
	Longitude            *float64           `json:"longitude,omitempty" db:"longitude"`
	Timezone             *string            `json:"timezone,omitempty" db:"timezone"`
	UserAgent            *string            `json:"user_agent,omitempty" db:"user_agent"`
	EventData            map[string]interface{} `json:"event_data,omitempty" db:"event_data"`
	CreatedAt            time.Time          `json:"created_at" db:"created_at"`
}

// FormConversionFunnel represents daily conversion funnel data
type FormConversionFunnel struct {
	ID                   uuid.UUID         `json:"id" db:"id"`
	FormID               uuid.UUID         `json:"form_id" db:"form_id"`
	UserID               uuid.UUID         `json:"user_id" db:"user_id"`
	Date                 time.Time         `json:"date" db:"date"`
	TotalViews           int               `json:"total_views" db:"total_views"`
	TotalStarts          int               `json:"total_starts" db:"total_starts"`
	TotalSubmits         int               `json:"total_submits" db:"total_submits"`
	TotalCompletes       int               `json:"total_completes" db:"total_completes"`
	TotalAbandons        int               `json:"total_abandons" db:"total_abandons"`
	AbandonmentPoints    map[string]int    `json:"abandonment_points" db:"abandonment_points"`
	ConversionRate       float64           `json:"conversion_rate" db:"conversion_rate"`
	CompletionRate       float64           `json:"completion_rate" db:"completion_rate"`
	AvgTimeToSubmit      *int              `json:"average_time_to_submit,omitempty" db:"average_time_to_submit"`
	AvgTimeToAbandon     *int              `json:"average_time_to_abandon,omitempty" db:"average_time_to_abandon"`
	CreatedAt            time.Time         `json:"created_at" db:"created_at"`
	UpdatedAt            time.Time         `json:"updated_at" db:"updated_at"`
}

// Submission Lifecycle Tracking
type SubmissionStatus string

const (
	SubmissionStatusReceived    SubmissionStatus = "received"
	SubmissionStatusProcessing  SubmissionStatus = "processing"
	SubmissionStatusValidated   SubmissionStatus = "validated"
	SubmissionStatusSpamFlagged SubmissionStatus = "spam_flagged"
	SubmissionStatusEmailSent   SubmissionStatus = "email_sent"
	SubmissionStatusWebhookSent SubmissionStatus = "webhook_sent"
	SubmissionStatusCompleted   SubmissionStatus = "completed"
	SubmissionStatusFailed      SubmissionStatus = "failed"
	SubmissionStatusResponded   SubmissionStatus = "responded"
	SubmissionStatusArchived    SubmissionStatus = "archived"
)

type EmailDeliveryStatus string

const (
	EmailDeliveryStatusPending   EmailDeliveryStatus = "pending"
	EmailDeliveryStatusSent      EmailDeliveryStatus = "sent"
	EmailDeliveryStatusDelivered EmailDeliveryStatus = "delivered"
	EmailDeliveryStatusBounced   EmailDeliveryStatus = "bounced"
	EmailDeliveryStatusFailed    EmailDeliveryStatus = "failed"
)

type WebhookDeliveryStatus string

const (
	WebhookDeliveryStatusPending WebhookDeliveryStatus = "pending"
	WebhookDeliveryStatusSent    WebhookDeliveryStatus = "sent"
	WebhookDeliveryStatusSuccess WebhookDeliveryStatus = "success"
	WebhookDeliveryStatusFailed  WebhookDeliveryStatus = "failed"
	WebhookDeliveryStatusRetry   WebhookDeliveryStatus = "retry"
)

type ResponseMethod string

const (
	ResponseMethodEmail    ResponseMethod = "email"
	ResponseMethodPhone    ResponseMethod = "phone"
	ResponseMethodInPerson ResponseMethod = "in_person"
	ResponseMethodOther    ResponseMethod = "other"
)

// SubmissionLifecycle tracks the complete lifecycle of a form submission
type SubmissionLifecycle struct {
	ID                      uuid.UUID              `json:"id" db:"id"`
	SubmissionID            uuid.UUID              `json:"submission_id" db:"submission_id"`
	FormID                  uuid.UUID              `json:"form_id" db:"form_id"`
	UserID                  uuid.UUID              `json:"user_id" db:"user_id"`
	TrackingID              string                 `json:"tracking_id" db:"tracking_id"`
	Status                  SubmissionStatus       `json:"status" db:"status"`
	ProcessingTimeMs        *int                   `json:"processing_time_ms,omitempty" db:"processing_time_ms"`
	ValidationErrors        []string               `json:"validation_errors,omitempty" db:"validation_errors"`
	SpamDetectionScore      *float64               `json:"spam_detection_score,omitempty" db:"spam_detection_score"`
	SpamDetectionReasons    []string               `json:"spam_detection_reasons,omitempty" db:"spam_detection_reasons"`
	EmailDeliveryStatus     *EmailDeliveryStatus   `json:"email_delivery_status,omitempty" db:"email_delivery_status"`
	EmailDeliveryTimeMs     *int                   `json:"email_delivery_time_ms,omitempty" db:"email_delivery_time_ms"`
	WebhookDeliveryStatus   *WebhookDeliveryStatus `json:"webhook_delivery_status,omitempty" db:"webhook_delivery_status"`
	WebhookDeliveryTimeMs   *int                   `json:"webhook_delivery_time_ms,omitempty" db:"webhook_delivery_time_ms"`
	WebhookResponseCode     *int                   `json:"webhook_response_code,omitempty" db:"webhook_response_code"`
	ResponseTime            *time.Time             `json:"response_time,omitempty" db:"response_time"`
	ResponseMethod          *ResponseMethod        `json:"response_method,omitempty" db:"response_method"`
	Notes                   *string                `json:"notes,omitempty" db:"notes"`
	CreatedAt               time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt               time.Time              `json:"updated_at" db:"updated_at"`
}

// UserSession represents a user session for journey tracking
type UserSession struct {
	ID                   uuid.UUID    `json:"id" db:"id"`
	SessionID            string       `json:"session_id" db:"session_id"`
	UserID               *uuid.UUID   `json:"user_id,omitempty" db:"user_id"`
	IPAddress            string       `json:"ip_address" db:"ip_address"`
	UserAgent            *string      `json:"user_agent,omitempty" db:"user_agent"`
	DeviceType           DeviceType   `json:"device_type" db:"device_type"`
	BrowserName          *string      `json:"browser_name,omitempty" db:"browser_name"`
	BrowserVersion       *string      `json:"browser_version,omitempty" db:"browser_version"`
	OSName               *string      `json:"os_name,omitempty" db:"os_name"`
	OSVersion            *string      `json:"os_version,omitempty" db:"os_version"`
	CountryCode          *string      `json:"country_code,omitempty" db:"country_code"`
	CountryName          *string      `json:"country_name,omitempty" db:"country_name"`
	Region               *string      `json:"region,omitempty" db:"region"`
	City                 *string      `json:"city,omitempty" db:"city"`
	Timezone             *string      `json:"timezone,omitempty" db:"timezone"`
	Referrer             *string      `json:"referrer,omitempty" db:"referrer"`
	LandingPage          *string      `json:"landing_page,omitempty" db:"landing_page"`
	UTMSource            *string      `json:"utm_source,omitempty" db:"utm_source"`
	UTMMedium            *string      `json:"utm_medium,omitempty" db:"utm_medium"`
	UTMCampaign          *string      `json:"utm_campaign,omitempty" db:"utm_campaign"`
	UTMTerm              *string      `json:"utm_term,omitempty" db:"utm_term"`
	UTMContent           *string      `json:"utm_content,omitempty" db:"utm_content"`
	TotalFormsViewed     int          `json:"total_forms_viewed" db:"total_forms_viewed"`
	TotalFormsStarted    int          `json:"total_forms_started" db:"total_forms_started"`
	TotalFormsSubmitted  int          `json:"total_forms_submitted" db:"total_forms_submitted"`
	TotalSessionTime     int          `json:"total_session_time" db:"total_session_time"`
	IsBot                bool         `json:"is_bot" db:"is_bot"`
	BotDetectionScore    float64      `json:"bot_detection_score" db:"bot_detection_score"`
	StartedAt            time.Time    `json:"started_at" db:"started_at"`
	LastActivityAt       time.Time    `json:"last_activity_at" db:"last_activity_at"`
	EndedAt              *time.Time   `json:"ended_at,omitempty" db:"ended_at"`
}

// Form A/B Test Variants
type ABTestStatus string

const (
	ABTestStatusDraft     ABTestStatus = "draft"
	ABTestStatusActive    ABTestStatus = "active"
	ABTestStatusPaused    ABTestStatus = "paused"
	ABTestStatusCompleted ABTestStatus = "completed"
)

type FormABTestVariant struct {
	ID                 uuid.UUID                  `json:"id" db:"id"`
	FormID             uuid.UUID                  `json:"form_id" db:"form_id"`
	UserID             uuid.UUID                  `json:"user_id" db:"user_id"`
	TestName           string                     `json:"test_name" db:"test_name"`
	VariantName        string                     `json:"variant_name" db:"variant_name"`
	VariantConfig      map[string]interface{}     `json:"variant_config" db:"variant_config"`
	TrafficPercentage  int                        `json:"traffic_percentage" db:"traffic_percentage"`
	IsActive           bool                       `json:"is_active" db:"is_active"`
	Status             ABTestStatus               `json:"status" db:"status"`
	StartedAt          *time.Time                 `json:"started_at,omitempty" db:"started_at"`
	EndedAt            *time.Time                 `json:"ended_at,omitempty" db:"ended_at"`
	TotalViews         int                        `json:"total_views" db:"total_views"`
	TotalSubmissions   int                        `json:"total_submissions" db:"total_submissions"`
	ConversionRate     float64                    `json:"conversion_rate" db:"conversion_rate"`
	ConfidenceLevel    *float64                   `json:"confidence_level,omitempty" db:"confidence_level"`
	IsWinner           bool                       `json:"is_winner" db:"is_winner"`
	CreatedAt          time.Time                  `json:"created_at" db:"created_at"`
	UpdatedAt          time.Time                  `json:"updated_at" db:"updated_at"`
}

// Geographic Analytics
type FormGeographicAnalytics struct {
	ID                   uuid.UUID `json:"id" db:"id"`
	FormID               uuid.UUID `json:"form_id" db:"form_id"`
	UserID               uuid.UUID `json:"user_id" db:"user_id"`
	Date                 time.Time `json:"date" db:"date"`
	CountryCode          string    `json:"country_code" db:"country_code"`
	CountryName          string    `json:"country_name" db:"country_name"`
	Region               *string   `json:"region,omitempty" db:"region"`
	City                 *string   `json:"city,omitempty" db:"city"`
	TotalViews           int       `json:"total_views" db:"total_views"`
	TotalSubmissions     int       `json:"total_submissions" db:"total_submissions"`
	ConversionRate       float64   `json:"conversion_rate" db:"conversion_rate"`
	BounceRate           float64   `json:"bounce_rate" db:"bounce_rate"`
	AverageSessionTime   int       `json:"average_session_time" db:"average_session_time"`
	CreatedAt            time.Time `json:"created_at" db:"created_at"`
	UpdatedAt            time.Time `json:"updated_at" db:"updated_at"`
}

// Device Analytics
type FormDeviceAnalytics struct {
	ID                     uuid.UUID  `json:"id" db:"id"`
	FormID                 uuid.UUID  `json:"form_id" db:"form_id"`
	UserID                 uuid.UUID  `json:"user_id" db:"user_id"`
	Date                   time.Time  `json:"date" db:"date"`
	DeviceType             DeviceType `json:"device_type" db:"device_type"`
	BrowserName            string     `json:"browser_name" db:"browser_name"`
	BrowserVersion         *string    `json:"browser_version,omitempty" db:"browser_version"`
	OSName                 string     `json:"os_name" db:"os_name"`
	OSVersion              *string    `json:"os_version,omitempty" db:"os_version"`
	TotalViews             int        `json:"total_views" db:"total_views"`
	TotalSubmissions       int        `json:"total_submissions" db:"total_submissions"`
	ConversionRate         float64    `json:"conversion_rate" db:"conversion_rate"`
	BounceRate             float64    `json:"bounce_rate" db:"bounce_rate"`
	AverageCompletionTime  int        `json:"average_completion_time" db:"average_completion_time"`
	CreatedAt              time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt              time.Time  `json:"updated_at" db:"updated_at"`
}

// Field Analytics
type FormFieldAnalytics struct {
	ID                   uuid.UUID              `json:"id" db:"id"`
	FormID               uuid.UUID              `json:"form_id" db:"form_id"`
	UserID               uuid.UUID              `json:"user_id" db:"user_id"`
	FieldName            string                 `json:"field_name" db:"field_name"`
	Date                 time.Time              `json:"date" db:"date"`
	TotalInteractions    int                    `json:"total_interactions" db:"total_interactions"`
	TotalFocusEvents     int                    `json:"total_focus_events" db:"total_focus_events"`
	TotalBlurEvents      int                    `json:"total_blur_events" db:"total_blur_events"`
	TotalChanges         int                    `json:"total_changes" db:"total_changes"`
	TotalValidationErrors int                   `json:"total_validation_errors" db:"total_validation_errors"`
	AverageTimeToFill    int                    `json:"average_time_to_fill" db:"average_time_to_fill"`
	AbandonmentRate      float64                `json:"abandonment_rate" db:"abandonment_rate"`
	ErrorRate            float64                `json:"error_rate" db:"error_rate"`
	CommonErrors         map[string]int         `json:"common_errors" db:"common_errors"`
	CreatedAt            time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt            time.Time              `json:"updated_at" db:"updated_at"`
}

// Automated Reports
type ReportType string

const (
	ReportTypeDailySummary      ReportType = "daily_summary"
	ReportTypeWeeklySummary     ReportType = "weekly_summary"
	ReportTypeMonthlySummary    ReportType = "monthly_summary"
	ReportTypeConversionAnalysis ReportType = "conversion_analysis"
	ReportTypeGeographicBreakdown ReportType = "geographic_breakdown"
	ReportTypeDeviceAnalysis    ReportType = "device_analysis"
	ReportTypeFieldPerformance  ReportType = "field_performance"
	ReportTypeSpamAnalysis      ReportType = "spam_analysis"
	ReportTypeCustom            ReportType = "custom"
)

type ReportFrequency string

const (
	ReportFrequencyDaily     ReportFrequency = "daily"
	ReportFrequencyWeekly    ReportFrequency = "weekly"
	ReportFrequencyMonthly   ReportFrequency = "monthly"
	ReportFrequencyQuarterly ReportFrequency = "quarterly"
)

type ReportFormat string

const (
	ReportFormatPDF  ReportFormat = "pdf"
	ReportFormatHTML ReportFormat = "html"
	ReportFormatCSV  ReportFormat = "csv"
	ReportFormatJSON ReportFormat = "json"
)

type AutomatedReport struct {
	ID               uuid.UUID                  `json:"id" db:"id"`
	UserID           uuid.UUID                  `json:"user_id" db:"user_id"`
	Name             string                     `json:"name" db:"name"`
	Description      *string                    `json:"description,omitempty" db:"description"`
	ReportType       ReportType                 `json:"report_type" db:"report_type"`
	FormsIncluded    []uuid.UUID                `json:"forms_included,omitempty" db:"forms_included"`
	Frequency        ReportFrequency            `json:"frequency" db:"frequency"`
	EmailRecipients  []string                   `json:"email_recipients" db:"email_recipients"`
	ReportFormat     ReportFormat               `json:"report_format" db:"report_format"`
	CustomConfig     map[string]interface{}     `json:"custom_config,omitempty" db:"custom_config"`
	Timezone         string                     `json:"timezone" db:"timezone"`
	SendTime         string                     `json:"send_time" db:"send_time"`
	IsActive         bool                       `json:"is_active" db:"is_active"`
	LastSentAt       *time.Time                 `json:"last_sent_at,omitempty" db:"last_sent_at"`
	NextSendAt       time.Time                  `json:"next_send_at" db:"next_send_at"`
	CreatedAt        time.Time                  `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time                  `json:"updated_at" db:"updated_at"`
}

// Monitoring Alerts
type AlertType string

const (
	AlertTypeHighSpamRate        AlertType = "high_spam_rate"
	AlertTypeLowConversionRate   AlertType = "low_conversion_rate"
	AlertTypeHighAbandonmentRate AlertType = "high_abandonment_rate"
	AlertTypeUnusualTraffic      AlertType = "unusual_traffic"
	AlertTypeFormErrors          AlertType = "form_errors"
	AlertTypeWebhookFailures     AlertType = "webhook_failures"
	AlertTypeEmailDeliveryIssues AlertType = "email_delivery_issues"
	AlertTypeSuspiciousActivity  AlertType = "suspicious_activity"
	AlertTypeFormDowntime        AlertType = "form_downtime"
	AlertTypeCustom              AlertType = "custom"
)

type AlertSeverity string

const (
	AlertSeverityLow      AlertSeverity = "low"
	AlertSeverityMedium   AlertSeverity = "medium"
	AlertSeverityHigh     AlertSeverity = "high"
	AlertSeverityCritical AlertSeverity = "critical"
)

type MonitoringAlert struct {
	ID                    uuid.UUID                  `json:"id" db:"id"`
	UserID                uuid.UUID                  `json:"user_id" db:"user_id"`
	AlertName             string                     `json:"alert_name" db:"alert_name"`
	AlertType             AlertType                  `json:"alert_type" db:"alert_type"`
	FormIDs               []uuid.UUID                `json:"form_ids,omitempty" db:"form_ids"`
	Conditions            map[string]interface{}     `json:"conditions" db:"conditions"`
	NotificationMethods   []string                   `json:"notification_methods" db:"notification_methods"`
	NotificationConfig    map[string]interface{}     `json:"notification_config" db:"notification_config"`
	CooldownMinutes       int                        `json:"cooldown_minutes" db:"cooldown_minutes"`
	IsActive              bool                       `json:"is_active" db:"is_active"`
	LastTriggeredAt       *time.Time                 `json:"last_triggered_at,omitempty" db:"last_triggered_at"`
	TriggerCount          int                        `json:"trigger_count" db:"trigger_count"`
	CreatedAt             time.Time                  `json:"created_at" db:"created_at"`
	UpdatedAt             time.Time                  `json:"updated_at" db:"updated_at"`
}

type AlertTriggerHistory struct {
	ID             uuid.UUID     `json:"id" db:"id"`
	AlertID        uuid.UUID     `json:"alert_id" db:"alert_id"`
	FormID         *uuid.UUID    `json:"form_id,omitempty" db:"form_id"`
	TriggerReason  string        `json:"trigger_reason" db:"trigger_reason"`
	TriggerData    map[string]interface{} `json:"trigger_data" db:"trigger_data"`
	Severity       AlertSeverity `json:"severity" db:"severity"`
	IsResolved     bool          `json:"is_resolved" db:"is_resolved"`
	ResolvedAt     *time.Time    `json:"resolved_at,omitempty" db:"resolved_at"`
	ResolvedBy     *uuid.UUID    `json:"resolved_by,omitempty" db:"resolved_by"`
	ResolutionNotes *string      `json:"resolution_notes,omitempty" db:"resolution_notes"`
	CreatedAt      time.Time     `json:"created_at" db:"created_at"`
}

// API Performance Metrics
type APIPerformanceMetrics struct {
	ID                  uuid.UUID  `json:"id" db:"id"`
	EndpointPath        string     `json:"endpoint_path" db:"endpoint_path"`
	HTTPMethod          string     `json:"http_method" db:"http_method"`
	ResponseTimeMs      int        `json:"response_time_ms" db:"response_time_ms"`
	StatusCode          int        `json:"status_code" db:"status_code"`
	UserID              *uuid.UUID `json:"user_id,omitempty" db:"user_id"`
	FormID              *uuid.UUID `json:"form_id,omitempty" db:"form_id"`
	IPAddress           string     `json:"ip_address" db:"ip_address"`
	UserAgent           *string    `json:"user_agent,omitempty" db:"user_agent"`
	RequestSizeBytes    *int       `json:"request_size_bytes,omitempty" db:"request_size_bytes"`
	ResponseSizeBytes   *int       `json:"response_size_bytes,omitempty" db:"response_size_bytes"`
	ErrorMessage        *string    `json:"error_message,omitempty" db:"error_message"`
	CreatedAt           time.Time  `json:"created_at" db:"created_at"`
}

// Dashboard Analytics DTOs
type FormAnalyticsDashboard struct {
	FormID             uuid.UUID              `json:"form_id"`
	FormName           string                 `json:"form_name"`
	TotalViews         int                    `json:"total_views"`
	TotalSubmissions   int                    `json:"total_submissions"`
	ConversionRate     float64                `json:"conversion_rate"`
	SpamRate           float64                `json:"spam_rate"`
	AverageCompletionTime int                 `json:"average_completion_time"`
	TopCountries       []CountryStats         `json:"top_countries"`
	DeviceBreakdown    []DeviceStats          `json:"device_breakdown"`
	HourlyStats        []HourlyStats          `json:"hourly_stats"`
	FieldAnalytics     []FieldAnalyticsStats  `json:"field_analytics"`
	RecentSubmissions  []RecentSubmissionStat `json:"recent_submissions"`
}

type CountryStats struct {
	CountryCode      string  `json:"country_code"`
	CountryName      string  `json:"country_name"`
	Views            int     `json:"views"`
	Submissions      int     `json:"submissions"`
	ConversionRate   float64 `json:"conversion_rate"`
}

type DeviceStats struct {
	DeviceType       DeviceType `json:"device_type"`
	Views            int        `json:"views"`
	Submissions      int        `json:"submissions"`
	ConversionRate   float64    `json:"conversion_rate"`
}

type HourlyStats struct {
	Hour         int     `json:"hour"`
	Views        int     `json:"views"`
	Submissions  int     `json:"submissions"`
	ConversionRate float64 `json:"conversion_rate"`
}

type FieldAnalyticsStats struct {
	FieldName        string  `json:"field_name"`
	Interactions     int     `json:"interactions"`
	AbandonmentRate  float64 `json:"abandonment_rate"`
	ErrorRate        float64 `json:"error_rate"`
	AvgTimeToFill    int     `json:"avg_time_to_fill"`
}

type RecentSubmissionStat struct {
	SubmissionID  uuid.UUID `json:"submission_id"`
	TrackingID    string    `json:"tracking_id"`
	Status        SubmissionStatus `json:"status"`
	CountryName   *string   `json:"country_name"`
	DeviceType    DeviceType `json:"device_type"`
	CreatedAt     time.Time `json:"created_at"`
}

// Real-time Analytics
type RealTimeStats struct {
	ActiveSessions       int                    `json:"active_sessions"`
	SubmissionsLastHour  int                    `json:"submissions_last_hour"`
	SubmissionsLast24h   int                    `json:"submissions_last_24h"`
	SpamBlockedLastHour  int                    `json:"spam_blocked_last_hour"`
	TopFormsLastHour     []FormActivityStat     `json:"top_forms_last_hour"`
	LiveSubmissions      []LiveSubmission       `json:"live_submissions"`
	SystemHealth         SystemHealthStats      `json:"system_health"`
}

type FormActivityStat struct {
	FormID      uuid.UUID `json:"form_id"`
	FormName    string    `json:"form_name"`
	Submissions int       `json:"submissions"`
	Views       int       `json:"views"`
}

type LiveSubmission struct {
	SubmissionID uuid.UUID `json:"submission_id"`
	FormName     string    `json:"form_name"`
	CountryName  *string   `json:"country_name"`
	CreatedAt    time.Time `json:"created_at"`
	IsSpam       bool      `json:"is_spam"`
}

type SystemHealthStats struct {
	APIResponseTime      float64 `json:"api_response_time"`
	EmailDeliveryRate    float64 `json:"email_delivery_rate"`
	WebhookSuccessRate   float64 `json:"webhook_success_rate"`
	SpamDetectionRate    float64 `json:"spam_detection_rate"`
	DatabaseConnections  int     `json:"database_connections"`
	RedisConnections     int     `json:"redis_connections"`
}

// Request DTOs for Analytics API
type AnalyticsEventRequest struct {
	FormID               uuid.UUID              `json:"form_id" binding:"required"`
	SessionID            string                 `json:"session_id" binding:"required"`
	EventType            AnalyticsEventType     `json:"event_type" binding:"required"`
	FieldName            *string                `json:"field_name,omitempty"`
	FieldValueLength     *int                   `json:"field_value_length,omitempty"`
	FieldValidationError *string                `json:"field_validation_error,omitempty"`
	PageURL              string                 `json:"page_url" binding:"required"`
	Referrer             *string                `json:"referrer,omitempty"`
	UTMData              *UTMData               `json:"utm_data,omitempty"`
	EventData            map[string]interface{} `json:"event_data,omitempty"`
}

type UTMData struct {
	Source   *string `json:"source,omitempty"`
	Medium   *string `json:"medium,omitempty"`
	Campaign *string `json:"campaign,omitempty"`
	Term     *string `json:"term,omitempty"`
	Content  *string `json:"content,omitempty"`
}

type AnalyticsQueryParams struct {
	FormIDs   []uuid.UUID `json:"form_ids,omitempty"`
	StartDate *time.Time  `json:"start_date,omitempty"`
	EndDate   *time.Time  `json:"end_date,omitempty"`
	Timezone  *string     `json:"timezone,omitempty"`
	Granularity string    `json:"granularity,omitempty"` // hour, day, week, month
	Filters   map[string]interface{} `json:"filters,omitempty"`
}

type CreateAutomatedReportRequest struct {
	Name             string                     `json:"name" binding:"required"`
	Description      *string                    `json:"description,omitempty"`
	ReportType       ReportType                 `json:"report_type" binding:"required"`
	FormsIncluded    []uuid.UUID                `json:"forms_included,omitempty"`
	Frequency        ReportFrequency            `json:"frequency" binding:"required"`
	EmailRecipients  []string                   `json:"email_recipients" binding:"required,min=1"`
	ReportFormat     ReportFormat               `json:"report_format" binding:"required"`
	CustomConfig     map[string]interface{}     `json:"custom_config,omitempty"`
	Timezone         string                     `json:"timezone"`
	SendTime         string                     `json:"send_time"`
}

type CreateMonitoringAlertRequest struct {
	AlertName           string                     `json:"alert_name" binding:"required"`
	AlertType           AlertType                  `json:"alert_type" binding:"required"`
	FormIDs             []uuid.UUID                `json:"form_ids,omitempty"`
	Conditions          map[string]interface{}     `json:"conditions" binding:"required"`
	NotificationMethods []string                   `json:"notification_methods" binding:"required,min=1"`
	NotificationConfig  map[string]interface{}     `json:"notification_config" binding:"required"`
	CooldownMinutes     int                        `json:"cooldown_minutes"`
}

type UpdateSubmissionLifecycleRequest struct {
	Status         SubmissionStatus `json:"status" binding:"required"`
	ResponseMethod *ResponseMethod  `json:"response_method,omitempty"`
	Notes          *string          `json:"notes,omitempty"`
}