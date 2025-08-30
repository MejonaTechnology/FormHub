package services

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"formhub/internal/models"
	"formhub/pkg/database"
	"html/template"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type ReportingService struct {
	db               *sqlx.DB
	redis            *database.RedisClient
	analyticsService *AnalyticsService
	emailService     EmailService // Assume this interface exists
	geoIPService     *GeoIPService
}

type EmailService interface {
	SendEmail(ctx context.Context, to []string, subject, htmlBody, textBody string, attachments []EmailAttachment) error
}

type EmailAttachment struct {
	Filename    string
	ContentType string
	Data        []byte
}

// ReportData represents the data structure for reports
type ReportData struct {
	User              *models.User                        `json:"user"`
	Forms             []models.Form                       `json:"forms"`
	Period            ReportPeriod                        `json:"period"`
	Summary           ReportSummary                       `json:"summary"`
	FormAnalytics     []FormAnalyticsReport              `json:"form_analytics"`
	GeographicData    []models.CountryStats               `json:"geographic_data"`
	DeviceData        []models.DeviceStats                `json:"device_data"`
	ConversionFunnels []models.FormConversionFunnel       `json:"conversion_funnels"`
	SpamAnalytics     SpamAnalyticsReport                 `json:"spam_analytics"`
	PerformanceMetrics PerformanceMetricsReport           `json:"performance_metrics"`
	GeneratedAt       time.Time                           `json:"generated_at"`
}

type ReportPeriod struct {
	StartDate time.Time `json:"start_date"`
	EndDate   time.Time `json:"end_date"`
	Type      string    `json:"type"` // daily, weekly, monthly, custom
}

type ReportSummary struct {
	TotalViews          int     `json:"total_views"`
	TotalSubmissions    int     `json:"total_submissions"`
	TotalSpamBlocked    int     `json:"total_spam_blocked"`
	ConversionRate      float64 `json:"conversion_rate"`
	TopPerformingForm   string  `json:"top_performing_form"`
	TopCountry          string  `json:"top_country"`
	TopDevice           string  `json:"top_device"`
	AverageResponseTime float64 `json:"average_response_time"`
}

type FormAnalyticsReport struct {
	FormID             uuid.UUID `json:"form_id"`
	FormName           string    `json:"form_name"`
	Views              int       `json:"views"`
	Submissions        int       `json:"submissions"`
	ConversionRate     float64   `json:"conversion_rate"`
	SpamRate           float64   `json:"spam_rate"`
	AverageCompletionTime int    `json:"average_completion_time"`
}

type SpamAnalyticsReport struct {
	TotalSpamBlocked  int                    `json:"total_spam_blocked"`
	SpamRate          float64                `json:"spam_rate"`
	TopSpamSources    []SpamSourceReport     `json:"top_spam_sources"`
	DetectionMethods  map[string]int         `json:"detection_methods"`
	FalsePositives    int                    `json:"false_positives"`
}

type SpamSourceReport struct {
	CountryCode string `json:"country_code"`
	CountryName string `json:"country_name"`
	Count       int    `json:"count"`
}

type PerformanceMetricsReport struct {
	AverageResponseTime   float64            `json:"average_response_time"`
	ErrorRate            float64            `json:"error_rate"`
	UptimePercentage     float64            `json:"uptime_percentage"`
	EndpointPerformance  []EndpointMetrics  `json:"endpoint_performance"`
}

type EndpointMetrics struct {
	Endpoint        string  `json:"endpoint"`
	AverageTime     float64 `json:"average_time"`
	RequestCount    int     `json:"request_count"`
	ErrorCount      int     `json:"error_count"`
	ErrorRate       float64 `json:"error_rate"`
}

func NewReportingService(
	db *sqlx.DB,
	redis *database.RedisClient,
	analyticsService *AnalyticsService,
	emailService EmailService,
	geoIPService *GeoIPService,
) *ReportingService {
	return &ReportingService{
		db:               db,
		redis:            redis,
		analyticsService: analyticsService,
		emailService:     emailService,
		geoIPService:     geoIPService,
	}
}

// CreateAutomatedReport creates a new automated report configuration
func (r *ReportingService) CreateAutomatedReport(ctx context.Context, userID uuid.UUID, req *models.CreateAutomatedReportRequest) (*models.AutomatedReport, error) {
	report := &models.AutomatedReport{
		ID:              uuid.New(),
		UserID:          userID,
		Name:            req.Name,
		Description:     req.Description,
		ReportType:      req.ReportType,
		FormsIncluded:   req.FormsIncluded,
		Frequency:       req.Frequency,
		EmailRecipients: req.EmailRecipients,
		ReportFormat:    req.ReportFormat,
		CustomConfig:    req.CustomConfig,
		Timezone:        req.Timezone,
		SendTime:        req.SendTime,
		IsActive:        true,
		CreatedAt:       time.Now().UTC(),
		UpdatedAt:       time.Now().UTC(),
	}

	// Calculate next send time
	report.NextSendAt = r.calculateNextSendTime(report.Frequency, report.SendTime, report.Timezone)

	query := `
		INSERT INTO automated_reports (
			id, user_id, name, description, report_type, forms_included, frequency,
			email_recipients, report_format, custom_config, timezone, send_time,
			is_active, next_send_at, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	formsJSON, _ := json.Marshal(report.FormsIncluded)
	recipientsJSON, _ := json.Marshal(report.EmailRecipients)
	configJSON, _ := json.Marshal(report.CustomConfig)

	_, err := r.db.ExecContext(ctx, query,
		report.ID, report.UserID, report.Name, report.Description, report.ReportType,
		string(formsJSON), report.Frequency, string(recipientsJSON), report.ReportFormat,
		string(configJSON), report.Timezone, report.SendTime, report.IsActive,
		report.NextSendAt, report.CreatedAt, report.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create automated report: %w", err)
	}

	return report, nil
}

// GenerateReport generates a report for the specified parameters
func (r *ReportingService) GenerateReport(ctx context.Context, userID uuid.UUID, reportType models.ReportType, startDate, endDate time.Time, formIDs []uuid.UUID, customConfig map[string]interface{}) (*ReportData, error) {
	data := &ReportData{
		Period: ReportPeriod{
			StartDate: startDate,
			EndDate:   endDate,
			Type:      r.getPeriodType(startDate, endDate),
		},
		GeneratedAt: time.Now().UTC(),
	}

	// Get user information
	user, err := r.getUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	data.User = user

	// Get forms information
	forms, err := r.getForms(ctx, userID, formIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to get forms: %w", err)
	}
	data.Forms = forms

	// Generate report based on type
	switch reportType {
	case models.ReportTypeDailySummary, models.ReportTypeWeeklySummary, models.ReportTypeMonthlySummary:
		err = r.generateSummaryReport(ctx, data)
	case models.ReportTypeConversionAnalysis:
		err = r.generateConversionAnalysisReport(ctx, data)
	case models.ReportTypeGeographicBreakdown:
		err = r.generateGeographicReport(ctx, data)
	case models.ReportTypeDeviceAnalysis:
		err = r.generateDeviceAnalysisReport(ctx, data)
	case models.ReportTypeFieldPerformance:
		err = r.generateFieldPerformanceReport(ctx, data)
	case models.ReportTypeSpamAnalysis:
		err = r.generateSpamAnalysisReport(ctx, data)
	case models.ReportTypeCustom:
		err = r.generateCustomReport(ctx, data, customConfig)
	default:
		err = r.generateSummaryReport(ctx, data) // Default to summary
	}

	if err != nil {
		return nil, fmt.Errorf("failed to generate %s report: %w", reportType, err)
	}

	return data, nil
}

// SendScheduledReports processes and sends scheduled reports
func (r *ReportingService) SendScheduledReports(ctx context.Context) error {
	// Get reports that are due to be sent
	query := `
		SELECT id, user_id, name, description, report_type, forms_included, frequency,
		       email_recipients, report_format, custom_config, timezone, send_time,
		       is_active, last_sent_at, next_send_at, created_at, updated_at
		FROM automated_reports 
		WHERE is_active = TRUE AND next_send_at <= NOW()
		ORDER BY next_send_at
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to get scheduled reports: %w", err)
	}
	defer rows.Close()

	var reports []models.AutomatedReport
	for rows.Next() {
		var report models.AutomatedReport
		var formsJSON, recipientsJSON, configJSON string
		
		err := rows.Scan(
			&report.ID, &report.UserID, &report.Name, &report.Description, &report.ReportType,
			&formsJSON, &report.Frequency, &recipientsJSON, &report.ReportFormat,
			&configJSON, &report.Timezone, &report.SendTime, &report.IsActive,
			&report.LastSentAt, &report.NextSendAt, &report.CreatedAt, &report.UpdatedAt,
		)
		if err != nil {
			log.Printf("Error scanning report: %v", err)
			continue
		}

		// Parse JSON fields
		json.Unmarshal([]byte(formsJSON), &report.FormsIncluded)
		json.Unmarshal([]byte(recipientsJSON), &report.EmailRecipients)
		json.Unmarshal([]byte(configJSON), &report.CustomConfig)

		reports = append(reports, report)
	}

	// Process each report
	for _, report := range reports {
		err := r.processScheduledReport(ctx, &report)
		if err != nil {
			log.Printf("Failed to process report %s: %v", report.ID, err)
			continue
		}

		// Update last sent time and calculate next send time
		now := time.Now().UTC()
		nextSend := r.calculateNextSendTime(report.Frequency, report.SendTime, report.Timezone)

		updateQuery := `
			UPDATE automated_reports 
			SET last_sent_at = ?, next_send_at = ?, updated_at = ?
			WHERE id = ?
		`
		r.db.ExecContext(ctx, updateQuery, now, nextSend, now, report.ID)

		log.Printf("Successfully sent report %s to %v", report.Name, report.EmailRecipients)
	}

	return nil
}

// ExportReportData exports report data in the specified format
func (r *ReportingService) ExportReportData(ctx context.Context, data *ReportData, format models.ReportFormat) ([]byte, string, error) {
	switch format {
	case models.ReportFormatJSON:
		return r.exportJSON(data)
	case models.ReportFormatCSV:
		return r.exportCSV(data)
	case models.ReportFormatHTML:
		return r.exportHTML(data)
	case models.ReportFormatPDF:
		return r.exportPDF(data)
	default:
		return r.exportJSON(data)
	}
}

// Private methods

func (r *ReportingService) processScheduledReport(ctx context.Context, report *models.AutomatedReport) error {
	// Calculate date range based on frequency
	endDate := time.Now().UTC()
	var startDate time.Time

	switch report.Frequency {
	case models.ReportFrequencyDaily:
		startDate = endDate.AddDate(0, 0, -1)
	case models.ReportFrequencyWeekly:
		startDate = endDate.AddDate(0, 0, -7)
	case models.ReportFrequencyMonthly:
		startDate = endDate.AddDate(0, -1, 0)
	case models.ReportFrequencyQuarterly:
		startDate = endDate.AddDate(0, -3, 0)
	}

	// Generate report data
	reportData, err := r.GenerateReport(ctx, report.UserID, report.ReportType, startDate, endDate, report.FormsIncluded, report.CustomConfig)
	if err != nil {
		return err
	}

	// Export in the specified format
	data, contentType, err := r.ExportReportData(ctx, reportData, report.ReportFormat)
	if err != nil {
		return err
	}

	// Create email subject and body
	subject := fmt.Sprintf("%s - %s", report.Name, reportData.Period.StartDate.Format("2006-01-02"))
	htmlBody, textBody := r.generateEmailContent(reportData, report)

	// Create attachment
	filename := fmt.Sprintf("%s_%s.%s", 
		strings.ReplaceAll(strings.ToLower(report.Name), " ", "_"),
		reportData.Period.StartDate.Format("20060102"),
		string(report.ReportFormat))

	attachment := EmailAttachment{
		Filename:    filename,
		ContentType: contentType,
		Data:        data,
	}

	// Send email
	return r.emailService.SendEmail(ctx, report.EmailRecipients, subject, htmlBody, textBody, []EmailAttachment{attachment})
}

func (r *ReportingService) generateSummaryReport(ctx context.Context, data *ReportData) error {
	// Get summary statistics
	summary := &ReportSummary{}

	// Calculate totals across all forms
	for _, form := range data.Forms {
		formStats, err := r.getFormSummaryStats(ctx, form.ID, data.Period.StartDate, data.Period.EndDate)
		if err != nil {
			continue
		}

		summary.TotalViews += formStats.Views
		summary.TotalSubmissions += formStats.Submissions
	}

	// Calculate conversion rate
	if summary.TotalViews > 0 {
		summary.ConversionRate = float64(summary.TotalSubmissions) / float64(summary.TotalViews) * 100
	}

	// Get spam statistics
	spamStats, _ := r.getSpamStats(ctx, data.User.ID, data.Period.StartDate, data.Period.EndDate)
	summary.TotalSpamBlocked = spamStats.TotalSpamBlocked

	// Get top performing form
	summary.TopPerformingForm = r.getTopPerformingForm(ctx, data.User.ID, data.Period.StartDate, data.Period.EndDate)

	// Get geographic and device data
	countries, _ := r.geoIPService.GetTopCountriesForUser(ctx, data.User.ID.String(), data.Period.StartDate, data.Period.EndDate, 1)
	if len(countries) > 0 {
		summary.TopCountry = countries[0].CountryName
	}

	devices, _ := r.geoIPService.GetDeviceBreakdownForUser(ctx, data.User.ID.String(), data.Period.StartDate, data.Period.EndDate)
	if len(devices) > 0 {
		summary.TopDevice = string(devices[0].DeviceType)
	}

	data.Summary = *summary

	// Get form analytics
	data.FormAnalytics = r.getFormAnalyticsReports(ctx, data.Forms, data.Period.StartDate, data.Period.EndDate)

	return nil
}

func (r *ReportingService) generateConversionAnalysisReport(ctx context.Context, data *ReportData) error {
	// Get conversion funnel data for all forms
	for _, form := range data.Forms {
		funnels, err := r.analyticsService.GetFormConversionFunnel(ctx, form.ID, form.UserID, data.Period.StartDate, data.Period.EndDate)
		if err != nil {
			continue
		}
		data.ConversionFunnels = append(data.ConversionFunnels, funnels...)
	}

	// Generate summary with conversion focus
	return r.generateSummaryReport(ctx, data)
}

func (r *ReportingService) generateGeographicReport(ctx context.Context, data *ReportData) error {
	countries, err := r.geoIPService.GetTopCountriesForUser(ctx, data.User.ID.String(), data.Period.StartDate, data.Period.EndDate, 20)
	if err != nil {
		return err
	}
	data.GeographicData = countries

	return r.generateSummaryReport(ctx, data)
}

func (r *ReportingService) generateDeviceAnalysisReport(ctx context.Context, data *ReportData) error {
	devices, err := r.geoIPService.GetDeviceBreakdownForUser(ctx, data.User.ID.String(), data.Period.StartDate, data.Period.EndDate)
	if err != nil {
		return err
	}
	data.DeviceData = devices

	return r.generateSummaryReport(ctx, data)
}

func (r *ReportingService) generateFieldPerformanceReport(ctx context.Context, data *ReportData) error {
	// This would generate field-level analytics
	// Implementation depends on specific field tracking requirements
	return r.generateSummaryReport(ctx, data)
}

func (r *ReportingService) generateSpamAnalysisReport(ctx context.Context, data *ReportData) error {
	spamStats, err := r.getDetailedSpamStats(ctx, data.User.ID, data.Period.StartDate, data.Period.EndDate)
	if err != nil {
		return err
	}
	data.SpamAnalytics = *spamStats

	return r.generateSummaryReport(ctx, data)
}

func (r *ReportingService) generateCustomReport(ctx context.Context, data *ReportData, config map[string]interface{}) error {
	// Implementation would depend on custom configuration parameters
	return r.generateSummaryReport(ctx, data)
}

// Helper methods for data retrieval

func (r *ReportingService) getUser(ctx context.Context, userID uuid.UUID) (*models.User, error) {
	var user models.User
	query := "SELECT id, email, first_name, last_name, company, plan_type, created_at FROM users WHERE id = ?"
	err := r.db.GetContext(ctx, &user, query, userID)
	return &user, err
}

func (r *ReportingService) getForms(ctx context.Context, userID uuid.UUID, formIDs []uuid.UUID) ([]models.Form, error) {
	var forms []models.Form
	var query string
	var args []interface{}

	if len(formIDs) > 0 {
		placeholders := make([]string, len(formIDs))
		args = make([]interface{}, len(formIDs)+1)
		args[0] = userID

		for i, formID := range formIDs {
			placeholders[i] = "?"
			args[i+1] = formID
		}

		query = fmt.Sprintf(`
			SELECT id, name, description, target_email, is_active, submission_count, created_at 
			FROM forms 
			WHERE user_id = ? AND id IN (%s)
		`, strings.Join(placeholders, ","))
	} else {
		query = "SELECT id, name, description, target_email, is_active, submission_count, created_at FROM forms WHERE user_id = ?"
		args = []interface{}{userID}
	}

	err := r.db.SelectContext(ctx, &forms, query, args...)
	return forms, err
}

func (r *ReportingService) getFormSummaryStats(ctx context.Context, formID uuid.UUID, startDate, endDate time.Time) (*FormAnalyticsReport, error) {
	var stats FormAnalyticsReport
	stats.FormID = formID

	query := `
		SELECT 
			COUNT(CASE WHEN event_type = 'form_view' THEN 1 END) as views,
			COUNT(CASE WHEN event_type = 'form_submit' THEN 1 END) as submissions,
			AVG(CASE WHEN event_type = 'form_complete' AND JSON_EXTRACT(event_data, '$.completion_time') IS NOT NULL 
				THEN JSON_EXTRACT(event_data, '$.completion_time') END) as avg_completion_time
		FROM form_analytics_events 
		WHERE form_id = ? AND created_at BETWEEN ? AND ?
	`

	var avgCompletionTime *float64
	err := r.db.QueryRowContext(ctx, query, formID, startDate, endDate).Scan(
		&stats.Views, &stats.Submissions, &avgCompletionTime)

	if err != nil {
		return &stats, err
	}

	if avgCompletionTime != nil {
		stats.AverageCompletionTime = int(*avgCompletionTime)
	}

	if stats.Views > 0 {
		stats.ConversionRate = float64(stats.Submissions) / float64(stats.Views) * 100
	}

	// Get spam rate
	var spamCount int
	r.db.GetContext(ctx, &spamCount, 
		"SELECT COUNT(*) FROM submissions WHERE form_id = ? AND is_spam = TRUE AND created_at BETWEEN ? AND ?", 
		formID, startDate, endDate)

	if stats.Submissions > 0 {
		stats.SpamRate = float64(spamCount) / float64(stats.Submissions) * 100
	}

	return &stats, nil
}

func (r *ReportingService) getSpamStats(ctx context.Context, userID uuid.UUID, startDate, endDate time.Time) (*SpamAnalyticsReport, error) {
	stats := &SpamAnalyticsReport{
		DetectionMethods: make(map[string]int),
	}

	// Get total spam blocked
	query := `
		SELECT COUNT(*) 
		FROM submissions s
		INNER JOIN forms f ON s.form_id = f.id
		WHERE f.user_id = ? AND s.is_spam = TRUE AND s.created_at BETWEEN ? AND ?
	`
	r.db.GetContext(ctx, &stats.TotalSpamBlocked, query, userID, startDate, endDate)

	// Get total submissions for rate calculation
	var totalSubmissions int
	query = `
		SELECT COUNT(*) 
		FROM submissions s
		INNER JOIN forms f ON s.form_id = f.id
		WHERE f.user_id = ? AND s.created_at BETWEEN ? AND ?
	`
	r.db.GetContext(ctx, &totalSubmissions, query, userID, startDate, endDate)

	if totalSubmissions > 0 {
		stats.SpamRate = float64(stats.TotalSpamBlocked) / float64(totalSubmissions) * 100
	}

	return stats, nil
}

func (r *ReportingService) getDetailedSpamStats(ctx context.Context, userID uuid.UUID, startDate, endDate time.Time) (*SpamAnalyticsReport, error) {
	stats, err := r.getSpamStats(ctx, userID, startDate, endDate)
	if err != nil {
		return stats, err
	}

	// Get top spam sources by country
	query := `
		SELECT ae.country_code, ae.country_name, COUNT(*) as count
		FROM submissions s
		INNER JOIN forms f ON s.form_id = f.id
		LEFT JOIN form_analytics_events ae ON s.form_id = ae.form_id AND ae.event_type = 'form_submit'
		WHERE f.user_id = ? AND s.is_spam = TRUE AND s.created_at BETWEEN ? AND ?
		      AND ae.country_code IS NOT NULL
		GROUP BY ae.country_code, ae.country_name
		ORDER BY count DESC
		LIMIT 10
	`

	rows, err := r.db.QueryContext(ctx, query, userID, startDate, endDate)
	if err != nil {
		return stats, err
	}
	defer rows.Close()

	for rows.Next() {
		var source SpamSourceReport
		err := rows.Scan(&source.CountryCode, &source.CountryName, &source.Count)
		if err != nil {
			continue
		}
		stats.TopSpamSources = append(stats.TopSpamSources, source)
	}

	return stats, nil
}

func (r *ReportingService) getTopPerformingForm(ctx context.Context, userID uuid.UUID, startDate, endDate time.Time) string {
	var formName string
	query := `
		SELECT f.name
		FROM forms f
		LEFT JOIN form_analytics_events ae ON f.id = ae.form_id
		WHERE f.user_id = ? AND ae.created_at BETWEEN ? AND ? AND ae.event_type = 'form_submit'
		GROUP BY f.id, f.name
		ORDER BY COUNT(*) DESC
		LIMIT 1
	`
	r.db.GetContext(ctx, &formName, query, userID, startDate, endDate)
	return formName
}

func (r *ReportingService) getFormAnalyticsReports(ctx context.Context, forms []models.Form, startDate, endDate time.Time) []FormAnalyticsReport {
	var reports []FormAnalyticsReport

	for _, form := range forms {
		report, err := r.getFormSummaryStats(ctx, form.ID, startDate, endDate)
		if err != nil {
			continue
		}
		report.FormName = form.Name
		reports = append(reports, *report)
	}

	return reports
}

// Export methods

func (r *ReportingService) exportJSON(data *ReportData) ([]byte, string, error) {
	jsonData, err := json.MarshalIndent(data, "", "  ")
	return jsonData, "application/json", err
}

func (r *ReportingService) exportCSV(data *ReportData) ([]byte, string, error) {
	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)

	// Write header
	header := []string{"Form Name", "Views", "Submissions", "Conversion Rate", "Spam Rate", "Avg Completion Time"}
	writer.Write(header)

	// Write data
	for _, analytics := range data.FormAnalytics {
		record := []string{
			analytics.FormName,
			strconv.Itoa(analytics.Views),
			strconv.Itoa(analytics.Submissions),
			fmt.Sprintf("%.2f", analytics.ConversionRate),
			fmt.Sprintf("%.2f", analytics.SpamRate),
			strconv.Itoa(analytics.AverageCompletionTime),
		}
		writer.Write(record)
	}

	writer.Flush()
	return buf.Bytes(), "text/csv", writer.Error()
}

func (r *ReportingService) exportHTML(data *ReportData) ([]byte, string, error) {
	tmpl := template.Must(template.New("report").Parse(htmlReportTemplate))
	
	var buf bytes.Buffer
	err := tmpl.Execute(&buf, data)
	return buf.Bytes(), "text/html", err
}

func (r *ReportingService) exportPDF(data *ReportData) ([]byte, string, error) {
	// This would require a PDF generation library like wkhtmltopdf, chromedp, or similar
	// For now, return HTML version
	return r.exportHTML(data)
}

// Utility methods

func (r *ReportingService) calculateNextSendTime(frequency models.ReportFrequency, sendTime, timezone string) time.Time {
	now := time.Now().UTC()
	
	// Parse send time (assuming format "15:04")
	timeOfDay, _ := time.Parse("15:04", sendTime)
	hour, minute := timeOfDay.Hour(), timeOfDay.Minute()

	var nextSend time.Time

	switch frequency {
	case models.ReportFrequencyDaily:
		nextSend = time.Date(now.Year(), now.Month(), now.Day()+1, hour, minute, 0, 0, time.UTC)
	case models.ReportFrequencyWeekly:
		daysUntilNextWeek := 7 - int(now.Weekday())
		nextSend = time.Date(now.Year(), now.Month(), now.Day()+daysUntilNextWeek, hour, minute, 0, 0, time.UTC)
	case models.ReportFrequencyMonthly:
		nextSend = time.Date(now.Year(), now.Month()+1, 1, hour, minute, 0, 0, time.UTC)
	case models.ReportFrequencyQuarterly:
		nextSend = time.Date(now.Year(), now.Month()+3, 1, hour, minute, 0, 0, time.UTC)
	default:
		nextSend = now.Add(24 * time.Hour)
	}

	return nextSend
}

func (r *ReportingService) getPeriodType(startDate, endDate time.Time) string {
	diff := endDate.Sub(startDate)
	days := int(diff.Hours() / 24)

	switch days {
	case 1:
		return "daily"
	case 7:
		return "weekly"
	case 30, 31:
		return "monthly"
	default:
		return "custom"
	}
}

func (r *ReportingService) generateEmailContent(data *ReportData, report *models.AutomatedReport) (string, string) {
	subject := fmt.Sprintf("Form Analytics Report - %s", data.Period.StartDate.Format("2006-01-02"))
	
	htmlBody := fmt.Sprintf(`
		<html>
		<body>
			<h2>%s</h2>
			<p>Hi %s,</p>
			<p>Here's your %s form analytics report for %s to %s.</p>
			
			<h3>Summary</h3>
			<ul>
				<li>Total Views: %d</li>
				<li>Total Submissions: %d</li>
				<li>Conversion Rate: %.2f%%</li>
				<li>Spam Blocked: %d</li>
			</ul>
			
			<p>Please find the detailed report attached.</p>
			
			<p>Best regards,<br>FormHub Team</p>
		</body>
		</html>
	`, report.Name, data.User.FirstName, string(report.ReportType), 
	   data.Period.StartDate.Format("2006-01-02"), data.Period.EndDate.Format("2006-01-02"),
	   data.Summary.TotalViews, data.Summary.TotalSubmissions, data.Summary.ConversionRate,
	   data.Summary.TotalSpamBlocked)

	textBody := fmt.Sprintf(`
%s

Hi %s,

Here's your %s form analytics report for %s to %s.

Summary:
- Total Views: %d
- Total Submissions: %d
- Conversion Rate: %.2f%%
- Spam Blocked: %d

Please find the detailed report attached.

Best regards,
FormHub Team
	`, report.Name, data.User.FirstName, string(report.ReportType),
	   data.Period.StartDate.Format("2006-01-02"), data.Period.EndDate.Format("2006-01-02"),
	   data.Summary.TotalViews, data.Summary.TotalSubmissions, data.Summary.ConversionRate,
	   data.Summary.TotalSpamBlocked)

	return htmlBody, textBody
}

// HTML template for reports
const htmlReportTemplate = `
<!DOCTYPE html>
<html>
<head>
    <title>Form Analytics Report</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        .header { background-color: #f5f5f5; padding: 20px; border-radius: 5px; }
        .summary { margin: 20px 0; }
        .metric { display: inline-block; margin: 10px; padding: 15px; background-color: #e8f4fd; border-radius: 5px; }
        table { width: 100%; border-collapse: collapse; margin: 20px 0; }
        th, td { border: 1px solid #ddd; padding: 8px; text-align: left; }
        th { background-color: #f2f2f2; }
    </style>
</head>
<body>
    <div class="header">
        <h1>Form Analytics Report</h1>
        <p>Period: {{.Period.StartDate.Format "2006-01-02"}} to {{.Period.EndDate.Format "2006-01-02"}}</p>
        <p>Generated: {{.GeneratedAt.Format "2006-01-02 15:04:05 UTC"}}</p>
    </div>

    <div class="summary">
        <h2>Summary</h2>
        <div class="metric">
            <strong>Total Views:</strong> {{.Summary.TotalViews}}
        </div>
        <div class="metric">
            <strong>Total Submissions:</strong> {{.Summary.TotalSubmissions}}
        </div>
        <div class="metric">
            <strong>Conversion Rate:</strong> {{printf "%.2f" .Summary.ConversionRate}}%
        </div>
        <div class="metric">
            <strong>Spam Blocked:</strong> {{.Summary.TotalSpamBlocked}}
        </div>
    </div>

    {{if .FormAnalytics}}
    <h2>Form Performance</h2>
    <table>
        <thead>
            <tr>
                <th>Form Name</th>
                <th>Views</th>
                <th>Submissions</th>
                <th>Conversion Rate</th>
                <th>Spam Rate</th>
            </tr>
        </thead>
        <tbody>
            {{range .FormAnalytics}}
            <tr>
                <td>{{.FormName}}</td>
                <td>{{.Views}}</td>
                <td>{{.Submissions}}</td>
                <td>{{printf "%.2f" .ConversionRate}}%</td>
                <td>{{printf "%.2f" .SpamRate}}%</td>
            </tr>
            {{end}}
        </tbody>
    </table>
    {{end}}

    {{if .GeographicData}}
    <h2>Top Countries</h2>
    <table>
        <thead>
            <tr>
                <th>Country</th>
                <th>Views</th>
                <th>Submissions</th>
                <th>Conversion Rate</th>
            </tr>
        </thead>
        <tbody>
            {{range .GeographicData}}
            <tr>
                <td>{{.CountryName}}</td>
                <td>{{.Views}}</td>
                <td>{{.Submissions}}</td>
                <td>{{printf "%.2f" .ConversionRate}}%</td>
            </tr>
            {{end}}
        </tbody>
    </table>
    {{end}}
</body>
</html>
`