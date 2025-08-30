package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"formhub/internal/models"
	"formhub/pkg/database"
	"log"
	"math"
	"math/rand"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type ABTestingService struct {
	db               *sqlx.DB
	redis            *database.RedisClient
	analyticsService *AnalyticsService
}

// ABTestResult represents the results of an A/B test
type ABTestResult struct {
	VariantA         *ABTestVariantResult `json:"variant_a"`
	VariantB         *ABTestVariantResult `json:"variant_b"`
	Winner           *string              `json:"winner,omitempty"`
	ConfidenceLevel  float64              `json:"confidence_level"`
	StatisticallySignificant bool         `json:"statistically_significant"`
	RecommendedAction string              `json:"recommended_action"`
}

type ABTestVariantResult struct {
	Name             string  `json:"name"`
	Views            int     `json:"views"`
	Submissions      int     `json:"submissions"`
	ConversionRate   float64 `json:"conversion_rate"`
	ConfidenceInterval *ConfidenceInterval `json:"confidence_interval,omitempty"`
}

type ConfidenceInterval struct {
	Lower float64 `json:"lower"`
	Upper float64 `json:"upper"`
}

func NewABTestingService(db *sqlx.DB, redis *database.RedisClient, analyticsService *AnalyticsService) *ABTestingService {
	return &ABTestingService{
		db:               db,
		redis:            redis,
		analyticsService: analyticsService,
	}
}

// CreateABTest creates a new A/B test for a form
func (s *ABTestingService) CreateABTest(ctx context.Context, userID, formID uuid.UUID, testName string, variants []ABTestVariantConfig) (*models.FormABTestVariant, error) {
	if len(variants) != 2 {
		return nil, fmt.Errorf("A/B test requires exactly 2 variants")
	}

	// Validate traffic split totals to 100%
	totalTraffic := 0
	for _, variant := range variants {
		totalTraffic += variant.TrafficPercentage
	}
	if totalTraffic != 100 {
		return nil, fmt.Errorf("traffic split must total 100%%")
	}

	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()

	// Create variants
	var createdVariants []models.FormABTestVariant
	for i, variantConfig := range variants {
		variant := &models.FormABTestVariant{
			ID:                uuid.New(),
			FormID:            formID,
			UserID:            userID,
			TestName:          testName,
			VariantName:       variantConfig.Name,
			VariantConfig:     variantConfig.Config,
			TrafficPercentage: variantConfig.TrafficPercentage,
			IsActive:          true,
			Status:            models.ABTestStatusDraft,
			CreatedAt:         time.Now().UTC(),
			UpdatedAt:         time.Now().UTC(),
		}

		query := `
			INSERT INTO form_ab_test_variants (
				id, form_id, user_id, test_name, variant_name, variant_config,
				traffic_percentage, is_active, status, created_at, updated_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`

		configJSON, _ := json.Marshal(variant.VariantConfig)

		_, err = tx.ExecContext(ctx, query,
			variant.ID, variant.FormID, variant.UserID, variant.TestName,
			variant.VariantName, string(configJSON), variant.TrafficPercentage,
			variant.IsActive, variant.Status, variant.CreatedAt, variant.UpdatedAt,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to create variant %s: %w", variant.VariantName, err)
		}

		createdVariants = append(createdVariants, *variant)
	}

	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Return the first variant (both are created together)
	return &createdVariants[0], nil
}

// StartABTest starts an A/B test
func (s *ABTestingService) StartABTest(ctx context.Context, userID uuid.UUID, testName string, formID uuid.UUID) error {
	now := time.Now().UTC()

	query := `
		UPDATE form_ab_test_variants 
		SET status = ?, started_at = ?, updated_at = ?
		WHERE user_id = ? AND form_id = ? AND test_name = ? AND is_active = TRUE
	`

	result, err := s.db.ExecContext(ctx, query, models.ABTestStatusActive, now, now, userID, formID, testName)
	if err != nil {
		return fmt.Errorf("failed to start A/B test: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("no A/B test found to start")
	}

	log.Printf("Started A/B test '%s' for form %s", testName, formID)
	return nil
}

// StopABTest stops an A/B test
func (s *ABTestingService) StopABTest(ctx context.Context, userID uuid.UUID, testName string, formID uuid.UUID) error {
	now := time.Now().UTC()

	query := `
		UPDATE form_ab_test_variants 
		SET status = ?, ended_at = ?, updated_at = ?
		WHERE user_id = ? AND form_id = ? AND test_name = ? AND status = ?
	`

	result, err := s.db.ExecContext(ctx, query, models.ABTestStatusCompleted, now, now, userID, formID, testName, models.ABTestStatusActive)
	if err != nil {
		return fmt.Errorf("failed to stop A/B test: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("no active A/B test found to stop")
	}

	log.Printf("Stopped A/B test '%s' for form %s", testName, formID)
	return nil
}

// AssignVariant assigns a user session to a test variant
func (s *ABTestingService) AssignVariant(ctx context.Context, formID uuid.UUID, sessionID string) (*models.FormABTestVariant, error) {
	// Check if session already has a variant assigned
	cacheKey := fmt.Sprintf("ab_test_assignment:%s:%s", formID, sessionID)
	cachedVariantID, err := s.redis.Client.Get(ctx, cacheKey).Result()
	if err == nil {
		// Session already assigned, return the cached variant
		variantID, _ := uuid.Parse(cachedVariantID)
		return s.getVariantByID(ctx, variantID)
	}

	// Get active A/B tests for this form
	query := `
		SELECT id, form_id, user_id, test_name, variant_name, variant_config,
		       traffic_percentage, is_active, status, total_views, total_submissions,
		       conversion_rate, created_at, updated_at
		FROM form_ab_test_variants
		WHERE form_id = ? AND status = ? AND is_active = TRUE
		ORDER BY test_name, variant_name
	`

	rows, err := s.db.QueryContext(ctx, query, formID, models.ABTestStatusActive)
	if err != nil {
		return nil, fmt.Errorf("failed to get A/B test variants: %w", err)
	}
	defer rows.Close()

	var variants []models.FormABTestVariant
	for rows.Next() {
		var variant models.FormABTestVariant
		var configJSON string

		err := rows.Scan(
			&variant.ID, &variant.FormID, &variant.UserID, &variant.TestName,
			&variant.VariantName, &configJSON, &variant.TrafficPercentage,
			&variant.IsActive, &variant.Status, &variant.TotalViews,
			&variant.TotalSubmissions, &variant.ConversionRate,
			&variant.CreatedAt, &variant.UpdatedAt,
		)
		if err != nil {
			continue
		}

		json.Unmarshal([]byte(configJSON), &variant.VariantConfig)
		variants = append(variants, variant)
	}

	if len(variants) == 0 {
		return nil, nil // No active A/B tests
	}

	// Assign variant based on traffic split
	assignedVariant := s.selectVariantByTraffic(variants, sessionID)

	// Cache the assignment
	s.redis.Client.Set(ctx, cacheKey, assignedVariant.ID.String(), 24*time.Hour)

	// Record the assignment in analytics
	s.recordVariantAssignment(ctx, assignedVariant, sessionID)

	return assignedVariant, nil
}

// GetABTestResults returns the results of an A/B test
func (s *ABTestingService) GetABTestResults(ctx context.Context, userID uuid.UUID, testName string, formID uuid.UUID) (*ABTestResult, error) {
	// Get the two variants for this test
	query := `
		SELECT id, variant_name, traffic_percentage, total_views, total_submissions, conversion_rate
		FROM form_ab_test_variants
		WHERE user_id = ? AND form_id = ? AND test_name = ? AND is_active = TRUE
		ORDER BY variant_name
	`

	rows, err := s.db.QueryContext(ctx, query, userID, formID, testName)
	if err != nil {
		return nil, fmt.Errorf("failed to get A/B test variants: %w", err)
	}
	defer rows.Close()

	var variants []ABTestVariantResult
	for rows.Next() {
		var variant ABTestVariantResult
		var trafficPercentage int

		err := rows.Scan(
			nil, // Skip ID for now
			&variant.Name, &trafficPercentage, &variant.Views,
			&variant.Submissions, &variant.ConversionRate,
		)
		if err != nil {
			continue
		}

		// Recalculate conversion rate
		if variant.Views > 0 {
			variant.ConversionRate = float64(variant.Submissions) / float64(variant.Views) * 100
		}

		variants = append(variants, variant)
	}

	if len(variants) != 2 {
		return nil, fmt.Errorf("A/B test requires exactly 2 variants, found %d", len(variants))
	}

	result := &ABTestResult{
		VariantA: &variants[0],
		VariantB: &variants[1],
	}

	// Calculate statistical significance
	s.calculateStatisticalSignificance(result)

	return result, nil
}

// UpdateVariantStats updates the statistics for a variant
func (s *ABTestingService) UpdateVariantStats(ctx context.Context, variantID uuid.UUID, views, submissions int) error {
	var conversionRate float64
	if views > 0 {
		conversionRate = float64(submissions) / float64(views) * 100
	}

	query := `
		UPDATE form_ab_test_variants 
		SET total_views = ?, total_submissions = ?, conversion_rate = ?, updated_at = ?
		WHERE id = ?
	`

	_, err := s.db.ExecContext(ctx, query, views, submissions, conversionRate, time.Now().UTC(), variantID)
	if err != nil {
		return fmt.Errorf("failed to update variant stats: %w", err)
	}

	return nil
}

// GetActiveABTests returns all active A/B tests for a user
func (s *ABTestingService) GetActiveABTests(ctx context.Context, userID uuid.UUID) ([]models.FormABTestVariant, error) {
	query := `
		SELECT v.id, v.form_id, v.user_id, v.test_name, v.variant_name, v.variant_config,
		       v.traffic_percentage, v.is_active, v.status, v.started_at, v.ended_at,
		       v.total_views, v.total_submissions, v.conversion_rate, v.is_winner,
		       v.created_at, v.updated_at, f.name as form_name
		FROM form_ab_test_variants v
		INNER JOIN forms f ON v.form_id = f.id
		WHERE v.user_id = ? AND v.status = ? AND v.is_active = TRUE
		ORDER BY v.test_name, v.variant_name
	`

	rows, err := s.db.QueryContext(ctx, query, userID, models.ABTestStatusActive)
	if err != nil {
		return nil, fmt.Errorf("failed to get active A/B tests: %w", err)
	}
	defer rows.Close()

	var tests []models.FormABTestVariant
	for rows.Next() {
		var test models.FormABTestVariant
		var configJSON string
		var formName string

		err := rows.Scan(
			&test.ID, &test.FormID, &test.UserID, &test.TestName,
			&test.VariantName, &configJSON, &test.TrafficPercentage,
			&test.IsActive, &test.Status, &test.StartedAt, &test.EndedAt,
			&test.TotalViews, &test.TotalSubmissions, &test.ConversionRate,
			&test.IsWinner, &test.CreatedAt, &test.UpdatedAt, &formName,
		)
		if err != nil {
			continue
		}

		json.Unmarshal([]byte(configJSON), &test.VariantConfig)
		tests = append(tests, test)
	}

	return tests, nil
}

// DeclareWinner declares a winner for an A/B test
func (s *ABTestingService) DeclareWinner(ctx context.Context, userID uuid.UUID, testName string, formID uuid.UUID, winnerVariantID uuid.UUID) error {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()

	// Reset all variants in this test
	query := `
		UPDATE form_ab_test_variants 
		SET is_winner = FALSE, updated_at = ?
		WHERE user_id = ? AND form_id = ? AND test_name = ?
	`
	_, err = tx.ExecContext(ctx, query, time.Now().UTC(), userID, formID, testName)
	if err != nil {
		return fmt.Errorf("failed to reset winners: %w", err)
	}

	// Set the winner
	query = `
		UPDATE form_ab_test_variants 
		SET is_winner = TRUE, updated_at = ?
		WHERE id = ? AND user_id = ? AND form_id = ? AND test_name = ?
	`
	result, err := tx.ExecContext(ctx, query, time.Now().UTC(), winnerVariantID, userID, formID, testName)
	if err != nil {
		return fmt.Errorf("failed to set winner: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("variant not found or not owned by user")
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	log.Printf("Declared winner for A/B test '%s' on form %s: variant %s", testName, formID, winnerVariantID)
	return nil
}

// Private methods

func (s *ABTestingService) getVariantByID(ctx context.Context, variantID uuid.UUID) (*models.FormABTestVariant, error) {
	var variant models.FormABTestVariant
	var configJSON string

	query := `
		SELECT id, form_id, user_id, test_name, variant_name, variant_config,
		       traffic_percentage, is_active, status, total_views, total_submissions,
		       conversion_rate, is_winner, created_at, updated_at
		FROM form_ab_test_variants
		WHERE id = ?
	`

	err := s.db.QueryRowContext(ctx, query, variantID).Scan(
		&variant.ID, &variant.FormID, &variant.UserID, &variant.TestName,
		&variant.VariantName, &configJSON, &variant.TrafficPercentage,
		&variant.IsActive, &variant.Status, &variant.TotalViews,
		&variant.TotalSubmissions, &variant.ConversionRate, &variant.IsWinner,
		&variant.CreatedAt, &variant.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	json.Unmarshal([]byte(configJSON), &variant.VariantConfig)
	return &variant, nil
}

func (s *ABTestingService) selectVariantByTraffic(variants []models.FormABTestVariant, sessionID string) *models.FormABTestVariant {
	// Use session ID as seed for consistent assignment
	seed := int64(0)
	for _, b := range []byte(sessionID) {
		seed = seed*31 + int64(b)
	}

	rng := rand.New(rand.NewSource(seed))
	randomValue := rng.Intn(100) + 1 // 1-100

	cumulativePercentage := 0
	for i := range variants {
		cumulativePercentage += variants[i].TrafficPercentage
		if randomValue <= cumulativePercentage {
			return &variants[i]
		}
	}

	// Fallback to first variant
	return &variants[0]
}

func (s *ABTestingService) recordVariantAssignment(ctx context.Context, variant *models.FormABTestVariant, sessionID string) {
	// Record an analytics event for the variant assignment
	event := &models.FormAnalyticsEvent{
		FormID:    variant.FormID,
		UserID:    variant.UserID,
		SessionID: sessionID,
		EventType: models.AnalyticsEventType("ab_test_assignment"),
		IPAddress: "127.0.0.1", // System event
		EventData: map[string]interface{}{
			"test_name":    variant.TestName,
			"variant_name": variant.VariantName,
			"variant_id":   variant.ID,
		},
		CreatedAt: time.Now().UTC(),
	}

	// Record asynchronously
	go s.analyticsService.RecordEvent(context.Background(), event)
}

func (s *ABTestingService) calculateStatisticalSignificance(result *ABTestResult) {
	a := result.VariantA
	b := result.VariantB

	// Calculate conversion rates as proportions
	p1 := float64(a.Submissions) / float64(a.Views)
	p2 := float64(b.Submissions) / float64(b.Views)

	// Calculate pooled proportion
	pooledP := float64(a.Submissions+b.Submissions) / float64(a.Views+b.Views)

	// Calculate standard error
	se := math.Sqrt(pooledP*(1-pooledP)*(1/float64(a.Views)+1/float64(b.Views)))

	// Calculate z-score
	zScore := math.Abs(p1-p2) / se

	// Calculate confidence level (using normal distribution approximation)
	// For 95% confidence, critical value is 1.96
	// For 99% confidence, critical value is 2.576
	confidenceLevel := 0.0
	if zScore >= 1.96 {
		confidenceLevel = 95.0
	}
	if zScore >= 2.576 {
		confidenceLevel = 99.0
	}

	result.ConfidenceLevel = confidenceLevel
	result.StatisticallySignificant = confidenceLevel >= 95.0

	// Determine winner
	if result.StatisticallySignificant {
		if a.ConversionRate > b.ConversionRate {
			result.Winner = &a.Name
		} else if b.ConversionRate > a.ConversionRate {
			result.Winner = &b.Name
		}
	}

	// Generate recommendation
	if result.StatisticallySignificant && result.Winner != nil {
		result.RecommendedAction = fmt.Sprintf("Deploy %s - it has a significantly higher conversion rate", *result.Winner)
	} else if confidenceLevel > 0 && confidenceLevel < 95 {
		result.RecommendedAction = "Continue test - results are not statistically significant yet"
	} else {
		result.RecommendedAction = "Collect more data before making a decision"
	}

	// Calculate confidence intervals for each variant
	a.ConfidenceInterval = s.calculateConfidenceInterval(p1, a.Views)
	b.ConfidenceInterval = s.calculateConfidenceInterval(p2, b.Views)
}

func (s *ABTestingService) calculateConfidenceInterval(proportion float64, sampleSize int) *ConfidenceInterval {
	// 95% confidence interval
	z := 1.96
	standardError := math.Sqrt(proportion * (1 - proportion) / float64(sampleSize))
	margin := z * standardError

	return &ConfidenceInterval{
		Lower: math.Max(0, proportion-margin) * 100,
		Upper: math.Min(1, proportion+margin) * 100,
	}
}

// AutoOptimizeABTests automatically optimizes A/B tests based on performance
func (s *ABTestingService) AutoOptimizeABTests(ctx context.Context) error {
	// Get all active A/B tests that have been running for at least 7 days
	query := `
		SELECT DISTINCT user_id, form_id, test_name
		FROM form_ab_test_variants
		WHERE status = ? AND is_active = TRUE 
		      AND started_at <= DATE_SUB(NOW(), INTERVAL 7 DAY)
		      AND total_views >= 100  -- Minimum sample size
	`

	rows, err := s.db.QueryContext(ctx, query, models.ABTestStatusActive)
	if err != nil {
		return fmt.Errorf("failed to get A/B tests for optimization: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var userID, formID uuid.UUID
		var testName string

		err := rows.Scan(&userID, &formID, &testName)
		if err != nil {
			continue
		}

		// Get results for this test
		result, err := s.GetABTestResults(ctx, userID, testName, formID)
		if err != nil {
			log.Printf("Failed to get results for test %s: %v", testName, err)
			continue
		}

		// Auto-declare winner if statistically significant
		if result.StatisticallySignificant && result.Winner != nil {
			// Get the winning variant ID
			winnerVariantID, err := s.getVariantIDByName(ctx, userID, formID, testName, *result.Winner)
			if err != nil {
				log.Printf("Failed to get winner variant ID: %v", err)
				continue
			}

			err = s.DeclareWinner(ctx, userID, testName, formID, winnerVariantID)
			if err != nil {
				log.Printf("Failed to auto-declare winner for test %s: %v", testName, err)
				continue
			}

			// Stop the test
			err = s.StopABTest(ctx, userID, testName, formID)
			if err != nil {
				log.Printf("Failed to stop optimized test %s: %v", testName, err)
			} else {
				log.Printf("Auto-optimized A/B test '%s': Winner is %s", testName, *result.Winner)
			}
		}
	}

	return nil
}

func (s *ABTestingService) getVariantIDByName(ctx context.Context, userID, formID uuid.UUID, testName, variantName string) (uuid.UUID, error) {
	var variantID uuid.UUID
	query := `
		SELECT id FROM form_ab_test_variants
		WHERE user_id = ? AND form_id = ? AND test_name = ? AND variant_name = ?
	`
	err := s.db.GetContext(ctx, &variantID, query, userID, formID, testName, variantName)
	return variantID, err
}

// ABTestVariantConfig represents the configuration for creating an A/B test variant
type ABTestVariantConfig struct {
	Name              string                 `json:"name"`
	TrafficPercentage int                    `json:"traffic_percentage"`
	Config            map[string]interface{} `json:"config"`
}