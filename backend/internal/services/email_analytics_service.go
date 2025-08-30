package services

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"formhub/internal/models"
	"time"

	"github.com/google/uuid"
)

type EmailAnalyticsService struct {
	db *sql.DB
}

type AnalyticsReport struct {
	TemplateID     uuid.UUID                `json:"template_id"`
	TemplateName   string                   `json:"template_name"`
	TotalSent      int                      `json:"total_sent"`
	TotalDelivered int                      `json:"total_delivered"`
	TotalOpened    int                      `json:"total_opened"`
	TotalClicked   int                      `json:"total_clicked"`
	UniqueOpened   int                      `json:"unique_opened"`
	UniqueClicked  int                      `json:"unique_clicked"`
	OpenRate       float64                  `json:"open_rate"`
	ClickRate      float64                  `json:"click_rate"`
	DeliveryRate   float64                  `json:"delivery_rate"`
	BounceRate     float64                  `json:"bounce_rate"`
	TimeSeriesData []AnalyticsTimePoint     `json:"time_series_data"`
	TopLinks       []LinkAnalytics          `json:"top_links"`
	GeographicData []GeographicAnalytics    `json:"geographic_data"`
	DeviceData     []DeviceAnalytics        `json:"device_data"`
	DateRange      DateRange                `json:"date_range"`
}

type AnalyticsTimePoint struct {
	Date     time.Time `json:"date"`
	Sent     int       `json:"sent"`
	Opened   int       `json:"opened"`
	Clicked  int       `json:"clicked"`
	Bounced  int       `json:"bounced"`
}

type LinkAnalytics struct {
	URL        string  `json:"url"`
	Clicks     int     `json:"clicks"`
	UniqueClicks int   `json:"unique_clicks"`
	ClickRate  float64 `json:"click_rate"`
}

type GeographicAnalytics struct {
	Country    string  `json:"country"`
	Region     string  `json:"region"`
	City       string  `json:"city"`
	Opens      int     `json:"opens"`
	Clicks     int     `json:"clicks"`
	Recipients int     `json:"recipients"`
}

type DeviceAnalytics struct {
	DeviceType string  `json:"device_type"`
	Browser    string  `json:"browser"`
	OS         string  `json:"os"`
	Opens      int     `json:"opens"`
	Clicks     int     `json:"clicks"`
	Percentage float64 `json:"percentage"`
}

type DateRange struct {
	StartDate time.Time `json:"start_date"`
	EndDate   time.Time `json:"end_date"`
}

type ComparisonReport struct {
	TemplateA      AnalyticsReport `json:"template_a"`
	TemplateB      AnalyticsReport `json:"template_b"`
	Significance   float64         `json:"significance"`
	Winner         string          `json:"winner"`
	Improvement    float64         `json:"improvement"`
	Recommendation string          `json:"recommendation"`
}

func NewEmailAnalyticsService(db *sql.DB) *EmailAnalyticsService {
	return &EmailAnalyticsService{
		db: db,
	}
}

// CreateAnalytics creates a new analytics entry
func (s *EmailAnalyticsService) CreateAnalytics(analytics *models.EmailAnalytics) error {
	if analytics.ID == uuid.Nil {
		analytics.ID = uuid.New()
	}
	if analytics.CreatedAt.IsZero() {
		analytics.CreatedAt = time.Now()
	}
	if analytics.UpdatedAt.IsZero() {
		analytics.UpdatedAt = time.Now()
	}

	// Insert into database
	linksJSON, _ := json.Marshal(analytics.Links)

	query := `
		INSERT INTO email_analytics (
			id, queue_id, user_id, form_id, template_id, email_address,
			delivered_at, opened_at, first_clicked_at, open_count, click_count,
			links, user_agent, ip_address, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := s.db.Exec(query,
		analytics.ID, analytics.QueueID, analytics.UserID, analytics.FormID,
		analytics.TemplateID, analytics.EmailAddress, analytics.DeliveredAt,
		analytics.OpenedAt, analytics.FirstClickedAt, analytics.OpenCount,
		analytics.ClickCount, linksJSON, analytics.UserAgent,
		analytics.IPAddress, analytics.CreatedAt, analytics.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create analytics: %w", err)
	}

	return nil
}

// RecordEmailOpen records an email open event
func (s *EmailAnalyticsService) RecordEmailOpen(queueID uuid.UUID, userAgent, ipAddress string) error {
	now := time.Now()

	// Get analytics record
	query := `SELECT id, open_count, opened_at FROM email_analytics WHERE queue_id = ?`
	
	var analyticsID uuid.UUID
	var openCount int
	var openedAt sql.NullTime

	err := s.db.QueryRow(query, queueID).Scan(&analyticsID, &openCount, &openedAt)
	if err != nil {
		return fmt.Errorf("failed to get analytics record: %w", err)
	}

	// Update open tracking
	updateQuery := `
		UPDATE email_analytics SET 
			open_count = open_count + 1,
			opened_at = COALESCE(opened_at, ?),
			user_agent = COALESCE(user_agent, ?),
			ip_address = COALESCE(ip_address, ?),
			updated_at = ?
		WHERE id = ?`

	_, err = s.db.Exec(updateQuery, now, userAgent, ipAddress, now, analyticsID)
	if err != nil {
		return fmt.Errorf("failed to record email open: %w", err)
	}

	return nil
}

// RecordEmailClick records an email click event
func (s *EmailAnalyticsService) RecordEmailClick(queueID uuid.UUID, url, userAgent, ipAddress string) error {
	now := time.Now()

	// Get analytics record
	query := `SELECT id, click_count, first_clicked_at, links FROM email_analytics WHERE queue_id = ?`
	
	var analyticsID uuid.UUID
	var clickCount int
	var firstClickedAt sql.NullTime
	var linksJSON []byte

	err := s.db.QueryRow(query, queueID).Scan(&analyticsID, &clickCount, &firstClickedAt, &linksJSON)
	if err != nil {
		return fmt.Errorf("failed to get analytics record: %w", err)
	}

	// Parse existing links
	var links []models.LinkClick
	if len(linksJSON) > 0 {
		json.Unmarshal(linksJSON, &links)
	}

	// Update or add link click
	found := false
	for i := range links {
		if links[i].URL == url {
			links[i].Count++
			links[i].ClickedAt = now
			found = true
			break
		}
	}

	if !found {
		links = append(links, models.LinkClick{
			URL:       url,
			ClickedAt: now,
			Count:     1,
		})
	}

	// Update analytics record
	updatedLinksJSON, _ := json.Marshal(links)
	
	updateQuery := `
		UPDATE email_analytics SET 
			click_count = click_count + 1,
			first_clicked_at = COALESCE(first_clicked_at, ?),
			links = ?,
			user_agent = COALESCE(user_agent, ?),
			ip_address = COALESCE(ip_address, ?),
			updated_at = ?
		WHERE id = ?`

	_, err = s.db.Exec(updateQuery, now, updatedLinksJSON, userAgent, ipAddress, now, analyticsID)
	if err != nil {
		return fmt.Errorf("failed to record email click: %w", err)
	}

	return nil
}

// GetTemplateAnalytics gets analytics for a specific template
func (s *EmailAnalyticsService) GetTemplateAnalytics(userID, templateID uuid.UUID, startDate, endDate time.Time) (*AnalyticsReport, error) {
	report := &AnalyticsReport{
		TemplateID: templateID,
		DateRange: DateRange{
			StartDate: startDate,
			EndDate:   endDate,
		},
	}

	// Get template name
	templateQuery := `SELECT name FROM email_templates WHERE id = ? AND user_id = ?`
	s.db.QueryRow(templateQuery, templateID, userID).Scan(&report.TemplateName)

	// Get basic metrics
	metricsQuery := `
		SELECT 
			COUNT(DISTINCT ea.queue_id) as total_sent,
			COUNT(DISTINCT CASE WHEN ea.delivered_at IS NOT NULL THEN ea.queue_id END) as total_delivered,
			COUNT(DISTINCT CASE WHEN ea.opened_at IS NOT NULL THEN ea.queue_id END) as unique_opened,
			COUNT(DISTINCT CASE WHEN ea.first_clicked_at IS NOT NULL THEN ea.queue_id END) as unique_clicked,
			SUM(ea.open_count) as total_opened,
			SUM(ea.click_count) as total_clicked
		FROM email_analytics ea
		WHERE ea.template_id = ? AND ea.user_id = ?
		AND ea.created_at BETWEEN ? AND ?`

	err := s.db.QueryRow(metricsQuery, templateID, userID, startDate, endDate).Scan(
		&report.TotalSent, &report.TotalDelivered, &report.UniqueOpened,
		&report.UniqueClicked, &report.TotalOpened, &report.TotalClicked,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get metrics: %w", err)
	}

	// Calculate rates
	if report.TotalSent > 0 {
		report.DeliveryRate = float64(report.TotalDelivered) / float64(report.TotalSent) * 100
	}
	if report.TotalDelivered > 0 {
		report.OpenRate = float64(report.UniqueOpened) / float64(report.TotalDelivered) * 100
		report.ClickRate = float64(report.UniqueClicked) / float64(report.TotalDelivered) * 100
	}
	report.BounceRate = 100 - report.DeliveryRate

	// Get time series data
	timeSeriesQuery := `
		SELECT 
			DATE(ea.created_at) as date,
			COUNT(DISTINCT ea.queue_id) as sent,
			COUNT(DISTINCT CASE WHEN ea.opened_at IS NOT NULL THEN ea.queue_id END) as opened,
			COUNT(DISTINCT CASE WHEN ea.first_clicked_at IS NOT NULL THEN ea.queue_id END) as clicked
		FROM email_analytics ea
		WHERE ea.template_id = ? AND ea.user_id = ?
		AND ea.created_at BETWEEN ? AND ?
		GROUP BY DATE(ea.created_at)
		ORDER BY date`

	timeRows, err := s.db.Query(timeSeriesQuery, templateID, userID, startDate, endDate)
	if err == nil {
		defer timeRows.Close()
		for timeRows.Next() {
			var point AnalyticsTimePoint
			timeRows.Scan(&point.Date, &point.Sent, &point.Opened, &point.Clicked)
			report.TimeSeriesData = append(report.TimeSeriesData, point)
		}
	}

	// Get top links
	linkQuery := `
		SELECT 
			JSON_UNQUOTE(JSON_EXTRACT(links, '$[*].url')) as url,
			SUM(JSON_EXTRACT(links, '$[*].count')) as total_clicks,
			COUNT(DISTINCT ea.id) as unique_clicks
		FROM email_analytics ea
		WHERE ea.template_id = ? AND ea.user_id = ?
		AND ea.created_at BETWEEN ? AND ?
		AND JSON_LENGTH(ea.links) > 0
		GROUP BY url
		ORDER BY total_clicks DESC
		LIMIT 10`

	linkRows, err := s.db.Query(linkQuery, templateID, userID, startDate, endDate)
	if err == nil {
		defer linkRows.Close()
		for linkRows.Next() {
			var link LinkAnalytics
			linkRows.Scan(&link.URL, &link.Clicks, &link.UniqueClicks)
			if report.TotalDelivered > 0 {
				link.ClickRate = float64(link.UniqueClicks) / float64(report.TotalDelivered) * 100
			}
			report.TopLinks = append(report.TopLinks, link)
		}
	}

	return report, nil
}

// GetUserAnalytics gets comprehensive analytics for a user
func (s *EmailAnalyticsService) GetUserAnalytics(userID uuid.UUID, startDate, endDate time.Time) (*AnalyticsReport, error) {
	report := &AnalyticsReport{
		DateRange: DateRange{
			StartDate: startDate,
			EndDate:   endDate,
		},
	}

	// Get overall metrics for user
	metricsQuery := `
		SELECT 
			COUNT(DISTINCT ea.queue_id) as total_sent,
			COUNT(DISTINCT CASE WHEN ea.delivered_at IS NOT NULL THEN ea.queue_id END) as total_delivered,
			COUNT(DISTINCT CASE WHEN ea.opened_at IS NOT NULL THEN ea.queue_id END) as unique_opened,
			COUNT(DISTINCT CASE WHEN ea.first_clicked_at IS NOT NULL THEN ea.queue_id END) as unique_clicked,
			SUM(ea.open_count) as total_opened,
			SUM(ea.click_count) as total_clicked
		FROM email_analytics ea
		WHERE ea.user_id = ?
		AND ea.created_at BETWEEN ? AND ?`

	err := s.db.QueryRow(metricsQuery, userID, startDate, endDate).Scan(
		&report.TotalSent, &report.TotalDelivered, &report.UniqueOpened,
		&report.UniqueClicked, &report.TotalOpened, &report.TotalClicked,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get user metrics: %w", err)
	}

	// Calculate rates
	if report.TotalSent > 0 {
		report.DeliveryRate = float64(report.TotalDelivered) / float64(report.TotalSent) * 100
	}
	if report.TotalDelivered > 0 {
		report.OpenRate = float64(report.UniqueOpened) / float64(report.TotalDelivered) * 100
		report.ClickRate = float64(report.UniqueClicked) / float64(report.TotalDelivered) * 100
	}
	report.BounceRate = 100 - report.DeliveryRate

	// Get time series data
	timeSeriesQuery := `
		SELECT 
			DATE(ea.created_at) as date,
			COUNT(DISTINCT ea.queue_id) as sent,
			COUNT(DISTINCT CASE WHEN ea.opened_at IS NOT NULL THEN ea.queue_id END) as opened,
			COUNT(DISTINCT CASE WHEN ea.first_clicked_at IS NOT NULL THEN ea.queue_id END) as clicked
		FROM email_analytics ea
		WHERE ea.user_id = ?
		AND ea.created_at BETWEEN ? AND ?
		GROUP BY DATE(ea.created_at)
		ORDER BY date`

	timeRows, err := s.db.Query(timeSeriesQuery, userID, startDate, endDate)
	if err == nil {
		defer timeRows.Close()
		for timeRows.Next() {
			var point AnalyticsTimePoint
			timeRows.Scan(&point.Date, &point.Sent, &point.Opened, &point.Clicked)
			report.TimeSeriesData = append(report.TimeSeriesData, point)
		}
	}

	return report, nil
}

// GetFormAnalytics gets analytics for a specific form
func (s *EmailAnalyticsService) GetFormAnalytics(userID, formID uuid.UUID, startDate, endDate time.Time) (*AnalyticsReport, error) {
	report := &AnalyticsReport{
		DateRange: DateRange{
			StartDate: startDate,
			EndDate:   endDate,
		},
	}

	// Get form metrics
	metricsQuery := `
		SELECT 
			COUNT(DISTINCT ea.queue_id) as total_sent,
			COUNT(DISTINCT CASE WHEN ea.delivered_at IS NOT NULL THEN ea.queue_id END) as total_delivered,
			COUNT(DISTINCT CASE WHEN ea.opened_at IS NOT NULL THEN ea.queue_id END) as unique_opened,
			COUNT(DISTINCT CASE WHEN ea.first_clicked_at IS NOT NULL THEN ea.queue_id END) as unique_clicked,
			SUM(ea.open_count) as total_opened,
			SUM(ea.click_count) as total_clicked
		FROM email_analytics ea
		WHERE ea.user_id = ? AND ea.form_id = ?
		AND ea.created_at BETWEEN ? AND ?`

	err := s.db.QueryRow(metricsQuery, userID, formID, startDate, endDate).Scan(
		&report.TotalSent, &report.TotalDelivered, &report.UniqueOpened,
		&report.UniqueClicked, &report.TotalOpened, &report.TotalClicked,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get form metrics: %w", err)
	}

	// Calculate rates
	if report.TotalSent > 0 {
		report.DeliveryRate = float64(report.TotalDelivered) / float64(report.TotalSent) * 100
	}
	if report.TotalDelivered > 0 {
		report.OpenRate = float64(report.UniqueOpened) / float64(report.TotalDelivered) * 100
		report.ClickRate = float64(report.UniqueClicked) / float64(report.TotalDelivered) * 100
	}
	report.BounceRate = 100 - report.DeliveryRate

	return report, nil
}

// GetTemplateComparison compares two templates
func (s *EmailAnalyticsService) GetTemplateComparison(userID, templateAID, templateBID uuid.UUID, startDate, endDate time.Time) (*ComparisonReport, error) {
	// Get analytics for both templates
	templateA, err := s.GetTemplateAnalytics(userID, templateAID, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to get template A analytics: %w", err)
	}

	templateB, err := s.GetTemplateAnalytics(userID, templateBID, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to get template B analytics: %w", err)
	}

	comparison := &ComparisonReport{
		TemplateA: *templateA,
		TemplateB: *templateB,
	}

	// Determine winner based on primary metric (open rate)
	if templateA.OpenRate > templateB.OpenRate {
		comparison.Winner = "A"
		comparison.Improvement = ((templateA.OpenRate - templateB.OpenRate) / templateB.OpenRate) * 100
	} else if templateB.OpenRate > templateA.OpenRate {
		comparison.Winner = "B"
		comparison.Improvement = ((templateB.OpenRate - templateA.OpenRate) / templateA.OpenRate) * 100
	} else {
		comparison.Winner = "Tie"
		comparison.Improvement = 0
	}

	// Calculate statistical significance (simplified)
	comparison.Significance = s.calculateSignificance(
		templateA.UniqueOpened, templateA.TotalDelivered,
		templateB.UniqueOpened, templateB.TotalDelivered,
	)

	// Generate recommendation
	comparison.Recommendation = s.generateRecommendation(comparison)

	return comparison, nil
}

// GenerateTrackingPixelURL generates a tracking pixel URL for email opens
func (s *EmailAnalyticsService) GenerateTrackingPixelURL(queueID uuid.UUID) string {
	// In a real implementation, this would generate a unique tracking URL
	// that your email service can serve and track opens
	return fmt.Sprintf("https://your-domain.com/track/open/%s.gif", queueID.String())
}

// GenerateTrackingClickURL generates a tracking URL for email clicks
func (s *EmailAnalyticsService) GenerateTrackingClickURL(queueID uuid.UUID, originalURL string) string {
	// In a real implementation, this would generate a unique tracking URL
	// that redirects to the original URL after recording the click
	return fmt.Sprintf("https://your-domain.com/track/click/%s?url=%s", queueID.String(), originalURL)
}

// GetTopPerformingTemplates gets the best performing templates
func (s *EmailAnalyticsService) GetTopPerformingTemplates(userID uuid.UUID, limit int, startDate, endDate time.Time) ([]AnalyticsReport, error) {
	query := `
		SELECT 
			ea.template_id,
			et.name,
			COUNT(DISTINCT ea.queue_id) as total_sent,
			COUNT(DISTINCT CASE WHEN ea.delivered_at IS NOT NULL THEN ea.queue_id END) as total_delivered,
			COUNT(DISTINCT CASE WHEN ea.opened_at IS NOT NULL THEN ea.queue_id END) as unique_opened,
			COUNT(DISTINCT CASE WHEN ea.first_clicked_at IS NOT NULL THEN ea.queue_id END) as unique_clicked,
			(COUNT(DISTINCT CASE WHEN ea.opened_at IS NOT NULL THEN ea.queue_id END) * 100.0 / 
			 NULLIF(COUNT(DISTINCT CASE WHEN ea.delivered_at IS NOT NULL THEN ea.queue_id END), 0)) as open_rate
		FROM email_analytics ea
		JOIN email_templates et ON ea.template_id = et.id
		WHERE ea.user_id = ? AND ea.created_at BETWEEN ? AND ?
		GROUP BY ea.template_id, et.name
		HAVING total_sent >= 10  -- Minimum threshold for meaningful data
		ORDER BY open_rate DESC
		LIMIT ?`

	rows, err := s.db.Query(query, userID, startDate, endDate, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get top performing templates: %w", err)
	}
	defer rows.Close()

	var reports []AnalyticsReport
	for rows.Next() {
		var report AnalyticsReport
		rows.Scan(
			&report.TemplateID, &report.TemplateName, &report.TotalSent,
			&report.TotalDelivered, &report.UniqueOpened, &report.UniqueClicked,
			&report.OpenRate,
		)
		
		// Calculate other rates
		if report.TotalSent > 0 {
			report.DeliveryRate = float64(report.TotalDelivered) / float64(report.TotalSent) * 100
		}
		if report.TotalDelivered > 0 {
			report.ClickRate = float64(report.UniqueClicked) / float64(report.TotalDelivered) * 100
		}

		reports = append(reports, report)
	}

	return reports, nil
}

// Private helper methods

func (s *EmailAnalyticsService) calculateSignificance(opensA, sentA, opensB, sentB int) float64 {
	// Simplified statistical significance calculation
	// In a real implementation, you'd want a proper statistical test
	if sentA < 30 || sentB < 30 {
		return 0.0 // Not enough data for significance
	}

	// Very basic calculation - in practice you'd use proper statistical tests
	rateA := float64(opensA) / float64(sentA)
	rateB := float64(opensB) / float64(sentB)
	
	if rateA == rateB {
		return 0.0
	}

	// Simplified confidence calculation
	diff := abs(rateA - rateB)
	avgRate := (rateA + rateB) / 2
	
	if avgRate == 0 {
		return 0.0
	}

	significance := (diff / avgRate) * 100
	if significance > 95 {
		significance = 95
	}
	
	return significance
}

func (s *EmailAnalyticsService) generateRecommendation(comparison *ComparisonReport) string {
	if comparison.Significance < 80 {
		return "The test results are not statistically significant. Consider running the test longer or with a larger sample size."
	}

	if comparison.Winner == "Tie" {
		return "Both templates perform similarly. Consider testing different elements like subject lines, call-to-action buttons, or content structure."
	}

	winnerTemplate := "Template A"
	if comparison.Winner == "B" {
		winnerTemplate = "Template B"
	}

	return fmt.Sprintf("%s is the clear winner with %.1f%% better performance. Consider using this template for your campaigns and applying its successful elements to other templates.", 
		winnerTemplate, comparison.Improvement)
}

// Helper function for absolute value
func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}