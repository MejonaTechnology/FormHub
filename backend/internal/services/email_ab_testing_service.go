package services

import (
	"database/sql"
	"fmt"
	"formhub/internal/models"
	"math"
	"math/rand"
	"time"

	"github.com/google/uuid"
)

type EmailABTestingService struct {
	db                   *sql.DB
	templateService      *EmailTemplateService
	analyticsService     *EmailAnalyticsService
	queueService         *EmailQueueService
}

type ABTestResult struct {
	Test         models.EmailABTest `json:"test"`
	IsSignificant bool              `json:"is_significant"`
	Confidence   float64            `json:"confidence"`
	Winner       string             `json:"winner"` // "A", "B", or "inconclusive"
	Improvement  float64            `json:"improvement_percentage"`
	Recommendation string           `json:"recommendation"`
	MinSampleSize  int              `json:"min_sample_size"`
	CurrentPower   float64          `json:"current_power"`
}

type ABTestMetrics struct {
	VariantA ABTestVariantMetrics `json:"variant_a"`
	VariantB ABTestVariantMetrics `json:"variant_b"`
	Duration time.Duration        `json:"duration"`
}

type ABTestVariantMetrics struct {
	Sent         int     `json:"sent"`
	Opened       int     `json:"opened"`
	Clicked      int     `json:"clicked"`
	OpenRate     float64 `json:"open_rate"`
	ClickRate    float64 `json:"click_rate"`
	ConversionRate float64 `json:"conversion_rate"`
}

type CreateABTestRequest struct {
	FormID         *uuid.UUID `json:"form_id,omitempty"`
	Name           string     `json:"name" binding:"required"`
	Description    string     `json:"description"`
	TemplateAID    uuid.UUID  `json:"template_a_id" binding:"required"`
	TemplateBID    uuid.UUID  `json:"template_b_id" binding:"required"`
	TrafficSplit   int        `json:"traffic_split"` // Percentage for A (0-100)
	TestMetric     string     `json:"test_metric"`   // "open_rate", "click_rate"
	MinSampleSize  int        `json:"min_sample_size"`
	MaxDuration    int        `json:"max_duration_days"`
}

func NewEmailABTestingService(db *sql.DB, templateService *EmailTemplateService, analyticsService *EmailAnalyticsService, queueService *EmailQueueService) *EmailABTestingService {
	return &EmailABTestingService{
		db:              db,
		templateService: templateService,
		analyticsService: analyticsService,
		queueService:    queueService,
	}
}

// CreateABTest creates a new A/B test
func (s *EmailABTestingService) CreateABTest(userID uuid.UUID, req CreateABTestRequest) (*models.EmailABTest, error) {
	// Validate templates exist and belong to user
	templateA, err := s.templateService.GetTemplate(userID, req.TemplateAID)
	if err != nil {
		return nil, fmt.Errorf("template A not found or access denied: %w", err)
	}

	templateB, err := s.templateService.GetTemplate(userID, req.TemplateBID)
	if err != nil {
		return nil, fmt.Errorf("template B not found or access denied: %w", err)
	}

	// Validate traffic split
	if req.TrafficSplit < 10 || req.TrafficSplit > 90 {
		return nil, fmt.Errorf("traffic split must be between 10 and 90 percent")
	}

	// Ensure templates are different
	if req.TemplateAID == req.TemplateBID {
		return nil, fmt.Errorf("template A and B must be different")
	}

	test := &models.EmailABTest{
		ID:            uuid.New(),
		UserID:        userID,
		FormID:        req.FormID,
		Name:          req.Name,
		Description:   req.Description,
		TemplateAID:   req.TemplateAID,
		TemplateBID:   req.TemplateBID,
		TrafficSplit:  req.TrafficSplit,
		Status:        models.ABTestStatusDraft,
		StatsSentA:    0,
		StatsSentB:    0,
		StatsOpenA:    0,
		StatsOpenB:    0,
		StatsClickA:   0,
		StatsClickB:   0,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	// Insert into database
	query := `
		INSERT INTO email_ab_tests (
			id, user_id, form_id, name, description, template_a_id, template_b_id,
			traffic_split, status, stats_sent_a, stats_sent_b, stats_open_a,
			stats_open_b, stats_click_a, stats_click_b, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err = s.db.Exec(query,
		test.ID, test.UserID, test.FormID, test.Name, test.Description,
		test.TemplateAID, test.TemplateBID, test.TrafficSplit, test.Status,
		test.StatsSentA, test.StatsSentB, test.StatsOpenA, test.StatsOpenB,
		test.StatsClickA, test.StatsClickB, test.CreatedAt, test.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create A/B test: %w", err)
	}

	return test, nil
}

// GetABTest retrieves an A/B test by ID
func (s *EmailABTestingService) GetABTest(userID, testID uuid.UUID) (*models.EmailABTest, error) {
	query := `
		SELECT id, user_id, form_id, name, description, template_a_id, template_b_id,
		       traffic_split, status, started_at, ended_at, winner, stats_sent_a,
		       stats_sent_b, stats_open_a, stats_open_b, stats_click_a, stats_click_b,
		       created_at, updated_at
		FROM email_ab_tests 
		WHERE id = ? AND user_id = ?`

	var test models.EmailABTest
	var formID, winner sql.NullString
	var startedAt, endedAt sql.NullTime

	err := s.db.QueryRow(query, testID, userID).Scan(
		&test.ID, &test.UserID, &formID, &test.Name, &test.Description,
		&test.TemplateAID, &test.TemplateBID, &test.TrafficSplit, &test.Status,
		&startedAt, &endedAt, &winner, &test.StatsSentA, &test.StatsSentB,
		&test.StatsOpenA, &test.StatsOpenB, &test.StatsClickA, &test.StatsClickB,
		&test.CreatedAt, &test.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get A/B test: %w", err)
	}

	// Parse optional fields
	if formID.Valid {
		if fid, err := uuid.Parse(formID.String); err == nil {
			test.FormID = &fid
		}
	}
	if winner.Valid {
		if wid, err := uuid.Parse(winner.String); err == nil {
			test.Winner = &wid
		}
	}
	if startedAt.Valid {
		test.StartedAt = &startedAt.Time
	}
	if endedAt.Valid {
		test.EndedAt = &endedAt.Time
	}

	return &test, nil
}

// ListABTests retrieves A/B tests for a user
func (s *EmailABTestingService) ListABTests(userID uuid.UUID, status *models.ABTestStatus, formID *uuid.UUID) ([]models.EmailABTest, error) {
	query := `
		SELECT id, user_id, form_id, name, description, template_a_id, template_b_id,
		       traffic_split, status, started_at, ended_at, winner, stats_sent_a,
		       stats_sent_b, stats_open_a, stats_open_b, stats_click_a, stats_click_b,
		       created_at, updated_at
		FROM email_ab_tests 
		WHERE user_id = ?`
	
	args := []interface{}{userID}

	if status != nil {
		query += " AND status = ?"
		args = append(args, *status)
	}

	if formID != nil {
		query += " AND form_id = ?"
		args = append(args, *formID)
	}

	query += " ORDER BY created_at DESC"

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list A/B tests: %w", err)
	}
	defer rows.Close()

	var tests []models.EmailABTest
	for rows.Next() {
		var test models.EmailABTest
		var formID, winner sql.NullString
		var startedAt, endedAt sql.NullTime

		err := rows.Scan(
			&test.ID, &test.UserID, &formID, &test.Name, &test.Description,
			&test.TemplateAID, &test.TemplateBID, &test.TrafficSplit, &test.Status,
			&startedAt, &endedAt, &winner, &test.StatsSentA, &test.StatsSentB,
			&test.StatsOpenA, &test.StatsOpenB, &test.StatsClickA, &test.StatsClickB,
			&test.CreatedAt, &test.UpdatedAt,
		)
		if err != nil {
			continue
		}

		// Parse optional fields
		if formID.Valid {
			if fid, err := uuid.Parse(formID.String); err == nil {
				test.FormID = &fid
			}
		}
		if winner.Valid {
			if wid, err := uuid.Parse(winner.String); err == nil {
				test.Winner = &wid
			}
		}
		if startedAt.Valid {
			test.StartedAt = &startedAt.Time
		}
		if endedAt.Valid {
			test.EndedAt = &endedAt.Time
		}

		tests = append(tests, test)
	}

	return tests, nil
}

// StartABTest starts an A/B test
func (s *EmailABTestingService) StartABTest(userID, testID uuid.UUID) (*models.EmailABTest, error) {
	// Get test and validate status
	test, err := s.GetABTest(userID, testID)
	if err != nil {
		return nil, fmt.Errorf("failed to get test: %w", err)
	}

	if test.Status != models.ABTestStatusDraft {
		return nil, fmt.Errorf("test is not in draft status")
	}

	// Update test status
	now := time.Now()
	query := `UPDATE email_ab_tests SET status = ?, started_at = ?, updated_at = ? WHERE id = ? AND user_id = ?`
	
	_, err = s.db.Exec(query, models.ABTestStatusActive, now, now, testID, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to start test: %w", err)
	}

	return s.GetABTest(userID, testID)
}

// PauseABTest pauses an active A/B test
func (s *EmailABTestingService) PauseABTest(userID, testID uuid.UUID) error {
	query := `UPDATE email_ab_tests SET status = ?, updated_at = ? WHERE id = ? AND user_id = ? AND status = ?`
	
	_, err := s.db.Exec(query, models.ABTestStatusPaused, time.Now(), testID, userID, models.ABTestStatusActive)
	if err != nil {
		return fmt.Errorf("failed to pause test: %w", err)
	}

	return nil
}

// ResumeABTest resumes a paused A/B test
func (s *EmailABTestingService) ResumeABTest(userID, testID uuid.UUID) error {
	query := `UPDATE email_ab_tests SET status = ?, updated_at = ? WHERE id = ? AND user_id = ? AND status = ?`
	
	_, err := s.db.Exec(query, models.ABTestStatusActive, time.Now(), testID, userID, models.ABTestStatusPaused)
	if err != nil {
		return fmt.Errorf("failed to resume test: %w", err)
	}

	return nil
}

// EndABTest ends an A/B test and determines the winner
func (s *EmailABTestingService) EndABTest(userID, testID uuid.UUID) (*ABTestResult, error) {
	test, err := s.GetABTest(userID, testID)
	if err != nil {
		return nil, fmt.Errorf("failed to get test: %w", err)
	}

	if test.Status != models.ABTestStatusActive && test.Status != models.ABTestStatusPaused {
		return nil, fmt.Errorf("test is not active or paused")
	}

	// Calculate result
	result := s.calculateABTestResult(test)

	// Update test in database
	now := time.Now()
	query := `UPDATE email_ab_tests SET status = ?, ended_at = ?, winner = ?, updated_at = ? WHERE id = ? AND user_id = ?`
	
	var winnerID *uuid.UUID
	if result.Winner == "A" {
		winnerID = &test.TemplateAID
	} else if result.Winner == "B" {
		winnerID = &test.TemplateBID
	}

	_, err = s.db.Exec(query, models.ABTestStatusEnded, now, winnerID, now, testID, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to end test: %w", err)
	}

	// Update result with latest test data
	updatedTest, _ := s.GetABTest(userID, testID)
	result.Test = *updatedTest

	return result, nil
}

// SelectVariant selects which variant to use for a specific email send
func (s *EmailABTestingService) SelectVariant(testID uuid.UUID) (uuid.UUID, string, error) {
	// Get test
	query := `SELECT template_a_id, template_b_id, traffic_split, status FROM email_ab_tests WHERE id = ?`
	
	var templateAID, templateBID uuid.UUID
	var trafficSplit int
	var status models.ABTestStatus

	err := s.db.QueryRow(query, testID).Scan(&templateAID, &templateBID, &trafficSplit, &status)
	if err != nil {
		return uuid.Nil, "", fmt.Errorf("failed to get test for variant selection: %w", err)
	}

	if status != models.ABTestStatusActive {
		return uuid.Nil, "", fmt.Errorf("test is not active")
	}

	// Random selection based on traffic split
	if rand.Intn(100) < trafficSplit {
		return templateAID, "A", nil
	}

	return templateBID, "B", nil
}

// UpdateABTestStats updates the statistics for an A/B test
func (s *EmailABTestingService) UpdateABTestStats(testID uuid.UUID, variant string, sent, opened, clicked int) error {
	var query string
	
	switch variant {
	case "A":
		query = `UPDATE email_ab_tests SET stats_sent_a = stats_sent_a + ?, stats_open_a = stats_open_a + ?, stats_click_a = stats_click_a + ?, updated_at = ? WHERE id = ?`
	case "B":
		query = `UPDATE email_ab_tests SET stats_sent_b = stats_sent_b + ?, stats_open_b = stats_open_b + ?, stats_click_b = stats_click_b + ?, updated_at = ? WHERE id = ?`
	default:
		return fmt.Errorf("invalid variant: %s", variant)
	}

	_, err := s.db.Exec(query, sent, opened, clicked, time.Now(), testID)
	if err != nil {
		return fmt.Errorf("failed to update test stats: %w", err)
	}

	return nil
}

// GetABTestResults analyzes and returns A/B test results
func (s *EmailABTestingService) GetABTestResults(userID, testID uuid.UUID) (*ABTestResult, error) {
	test, err := s.GetABTest(userID, testID)
	if err != nil {
		return nil, fmt.Errorf("failed to get test: %w", err)
	}

	return s.calculateABTestResult(test), nil
}

// GetABTestMetrics returns detailed metrics for an A/B test
func (s *EmailABTestingService) GetABTestMetrics(userID, testID uuid.UUID) (*ABTestMetrics, error) {
	test, err := s.GetABTest(userID, testID)
	if err != nil {
		return nil, fmt.Errorf("failed to get test: %w", err)
	}

	metrics := &ABTestMetrics{}

	// Calculate metrics for variant A
	metrics.VariantA.Sent = test.StatsSentA
	metrics.VariantA.Opened = test.StatsOpenA
	metrics.VariantA.Clicked = test.StatsClickA
	
	if test.StatsSentA > 0 {
		metrics.VariantA.OpenRate = float64(test.StatsOpenA) / float64(test.StatsSentA) * 100
		metrics.VariantA.ClickRate = float64(test.StatsClickA) / float64(test.StatsSentA) * 100
	}

	// Calculate metrics for variant B
	metrics.VariantB.Sent = test.StatsSentB
	metrics.VariantB.Opened = test.StatsOpenB
	metrics.VariantB.Clicked = test.StatsClickB
	
	if test.StatsSentB > 0 {
		metrics.VariantB.OpenRate = float64(test.StatsOpenB) / float64(test.StatsSentB) * 100
		metrics.VariantB.ClickRate = float64(test.StatsClickB) / float64(test.StatsSentB) * 100
	}

	// Calculate duration
	if test.StartedAt != nil {
		endTime := time.Now()
		if test.EndedAt != nil {
			endTime = *test.EndedAt
		}
		metrics.Duration = endTime.Sub(*test.StartedAt)
	}

	return metrics, nil
}

// DeleteABTest deletes an A/B test (only if not active)
func (s *EmailABTestingService) DeleteABTest(userID, testID uuid.UUID) error {
	// Check if test can be deleted
	test, err := s.GetABTest(userID, testID)
	if err != nil {
		return fmt.Errorf("failed to get test: %w", err)
	}

	if test.Status == models.ABTestStatusActive {
		return fmt.Errorf("cannot delete active test - pause or end it first")
	}

	query := `DELETE FROM email_ab_tests WHERE id = ? AND user_id = ?`
	
	_, err = s.db.Exec(query, testID, userID)
	if err != nil {
		return fmt.Errorf("failed to delete test: %w", err)
	}

	return nil
}

// Helper methods

func (s *EmailABTestingService) calculateABTestResult(test *models.EmailABTest) *ABTestResult {
	result := &ABTestResult{
		Test: *test,
	}

	// Calculate rates
	openRateA := float64(0)
	openRateB := float64(0)
	clickRateA := float64(0)
	clickRateB := float64(0)

	if test.StatsSentA > 0 {
		openRateA = float64(test.StatsOpenA) / float64(test.StatsSentA)
		clickRateA = float64(test.StatsClickA) / float64(test.StatsSentA)
	}

	if test.StatsSentB > 0 {
		openRateB = float64(test.StatsOpenB) / float64(test.StatsSentB)
		clickRateB = float64(test.StatsClickB) / float64(test.StatsSentB)
	}

	// Determine winner based on primary metric (open rate)
	primaryMetricA := openRateA
	primaryMetricB := openRateB

	// Calculate statistical significance using Z-test for proportions
	result.IsSignificant, result.Confidence = s.calculateSignificance(
		test.StatsOpenA, test.StatsSentA,
		test.StatsOpenB, test.StatsSentB,
	)

	// Determine winner
	if primaryMetricA > primaryMetricB && result.IsSignificant {
		result.Winner = "A"
		result.Improvement = ((primaryMetricA - primaryMetricB) / primaryMetricB) * 100
	} else if primaryMetricB > primaryMetricA && result.IsSignificant {
		result.Winner = "B"
		result.Improvement = ((primaryMetricB - primaryMetricA) / primaryMetricA) * 100
	} else {
		result.Winner = "inconclusive"
		result.Improvement = 0
	}

	// Calculate minimum sample size needed
	result.MinSampleSize = s.calculateMinSampleSize(primaryMetricA, primaryMetricB, 0.05, 0.80)

	// Generate recommendation
	result.Recommendation = s.generateRecommendation(result, test)

	return result
}

func (s *EmailABTestingService) calculateSignificance(conversionsA, visitorsA, conversionsB, visitorsB int) (bool, float64) {
	// Minimum sample size for meaningful results
	if visitorsA < 30 || visitorsB < 30 {
		return false, 0.0
	}

	// Calculate conversion rates
	rateA := float64(conversionsA) / float64(visitorsA)
	rateB := float64(conversionsB) / float64(visitorsB)

	// If rates are identical, no significance
	if rateA == rateB {
		return false, 0.0
	}

	// Calculate pooled probability
	pooledRate := float64(conversionsA+conversionsB) / float64(visitorsA+visitorsB)

	// Calculate standard error
	standardError := math.Sqrt(pooledRate * (1 - pooledRate) * (1.0/float64(visitorsA) + 1.0/float64(visitorsB)))

	// Calculate Z-score
	zScore := math.Abs(rateA - rateB) / standardError

	// Convert Z-score to confidence level (two-tailed test)
	confidence := (1.0 - 2.0*normalCDF(-math.Abs(zScore))) * 100

	// Significant if confidence > 95%
	isSignificant := confidence > 95.0

	return isSignificant, confidence
}

func (s *EmailABTestingService) calculateMinSampleSize(rateA, rateB, alpha, power float64) int {
	if rateA == 0 || rateB == 0 {
		return 1000 // Default minimum
	}

	// Simplified sample size calculation
	// In practice, you'd use a more sophisticated formula
	avgRate := (rateA + rateB) / 2
	effect := math.Abs(rateA - rateB)
	
	if effect == 0 {
		return 10000 // Very large sample needed for no effect
	}

	// Simplified calculation (should be replaced with proper statistical formula)
	sampleSize := int((avgRate * (1 - avgRate)) / (effect * effect) * 16) // Rough approximation
	
	if sampleSize < 100 {
		sampleSize = 100
	}
	if sampleSize > 10000 {
		sampleSize = 10000
	}
	
	return sampleSize
}

func (s *EmailABTestingService) generateRecommendation(result *ABTestResult, test *models.EmailABTest) string {
	totalSample := test.StatsSentA + test.StatsSentB
	
	if totalSample < result.MinSampleSize {
		return fmt.Sprintf("Test needs more data. Current sample: %d, recommended: %d. Continue running the test to reach statistical significance.", 
			totalSample, result.MinSampleSize)
	}

	if !result.IsSignificant {
		return "The test results are not statistically significant. Consider running the test longer, increasing the sample size, or testing more dramatic variations."
	}

	if result.Winner == "inconclusive" {
		return "Both variants perform similarly. Consider testing different elements like subject lines, content, or call-to-action buttons to find more significant differences."
	}

	winnerTemplate := "Template A"
	if result.Winner == "B" {
		winnerTemplate = "Template B"
	}

	return fmt.Sprintf("%s is the clear winner with %.1f%% improvement and %.1f%% confidence. Implement this template for your campaigns and analyze its successful elements for future optimization.", 
		winnerTemplate, result.Improvement, result.Confidence)
}

// Approximate normal cumulative distribution function
func normalCDF(x float64) float64 {
	// Abramowitz & Stegun approximation
	const (
		a1 =  0.254829592
		a2 = -0.284496736
		a3 =  1.421413741
		a4 = -1.453152027
		a5 =  1.061405429
		p  =  0.3275911
	)

	sign := 1.0
	if x < 0 {
		sign = -1.0
	}
	x = math.Abs(x) / math.Sqrt(2.0)

	t := 1.0 / (1.0 + p*x)
	y := 1.0 - (((((a5*t+a4)*t)+a3)*t+a2)*t+a1)*t*math.Exp(-x*x)

	return 0.5 * (1.0 + sign*y)
}