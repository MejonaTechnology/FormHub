package services

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// BehavioralAnalyzer analyzes user behavior patterns to detect bots
type BehavioralAnalyzer struct {
	db    *sql.DB
	redis *redis.Client
	
	// Statistical models for normal human behavior
	typingSpeedModel    *StatisticalModel
	interactionModel    *StatisticalModel
	mouseMovementModel  *StatisticalModel
	keystrokeModel      *StatisticalModel
	
	// Thresholds and parameters
	botScoreThreshold   float64
	anomalyThreshold    float64
	minDataPoints       int
	updateInterval      time.Duration
	lastModelUpdate     time.Time
}

// StatisticalModel represents a statistical model for behavioral metrics
type StatisticalModel struct {
	Mean              float64   `json:"mean"`
	StandardDeviation float64   `json:"standard_deviation"`
	Percentiles       []float64 `json:"percentiles"` // 5th, 25th, 50th, 75th, 95th
	SampleCount       int       `json:"sample_count"`
	LastUpdate        time.Time `json:"last_update"`
}

// BehavioralProfile represents a user's behavioral profile
type BehavioralProfile struct {
	SessionID         string            `json:"session_id"`
	UserAgent         string            `json:"user_agent"`
	IPAddress         string            `json:"ip_address"`
	TypingSpeed       float64           `json:"typing_speed"`        // WPM
	AvgKeystrokeDelay float64           `json:"avg_keystroke_delay"` // milliseconds
	MouseMovements    int               `json:"mouse_movements"`
	MouseDistance     float64           `json:"mouse_distance"`      // pixels
	ScrollEvents      int               `json:"scroll_events"`
	ClickEvents       int               `json:"click_events"`
	FocusEvents       int               `json:"focus_events"`
	TabSwitches       int               `json:"tab_switches"`
	CopyPasteEvents   int               `json:"copy_paste_events"`
	BackspaceRatio    float64           `json:"backspace_ratio"`     // backspaces/total keystrokes
	TypingRhythm      []float64         `json:"typing_rhythm"`       // keystroke intervals
	TimeOnPage        float64           `json:"time_on_page"`        // seconds
	InteractionDelay  float64           `json:"interaction_delay"`   // seconds before first interaction
	FormFillPattern   map[string]float64 `json:"form_fill_pattern"`  // field_name -> time_spent
	DeviceInfo        DeviceInfo        `json:"device_info"`
	Timestamps        []int64           `json:"timestamps"`          // interaction timestamps
}

// DeviceInfo contains device and browser information
type DeviceInfo struct {
	ScreenWidth       int     `json:"screen_width"`
	ScreenHeight      int     `json:"screen_height"`
	ViewportWidth     int     `json:"viewport_width"`
	ViewportHeight    int     `json:"viewport_height"`
	ColorDepth        int     `json:"color_depth"`
	PixelRatio        float64 `json:"pixel_ratio"`
	TouchSupport      bool    `json:"touch_support"`
	Platform          string  `json:"platform"`
	Language          string  `json:"language"`
	TimeZone          string  `json:"timezone"`
	CookiesEnabled    bool    `json:"cookies_enabled"`
	JavaScriptEnabled bool    `json:"javascript_enabled"`
	DoNotTrack        bool    `json:"do_not_track"`
}

// BehavioralAnalysisResult contains the result of behavioral analysis
type BehavioralAnalysisResult struct {
	BotScore         float64                `json:"bot_score"`         // 0.0-1.0, higher = more bot-like
	HumanScore       float64                `json:"human_score"`       // 0.0-1.0, higher = more human-like
	Confidence       float64                `json:"confidence"`        // 0.0-1.0
	AnomalyFlags     []AnomalyFlag          `json:"anomaly_flags"`
	BehavioralFeats  BehavioralFeatures     `json:"behavioral_features"`
	Recommendation   string                 `json:"recommendation"`    // "allow", "challenge", "block"
	Analysis         map[string]interface{} `json:"analysis"`
}

// AnomalyFlag represents a behavioral anomaly
type AnomalyFlag struct {
	Type        string  `json:"type"`
	Description string  `json:"description"`
	Severity    string  `json:"severity"`      // "low", "medium", "high", "critical"
	Score       float64 `json:"score"`         // Contribution to bot score
	Threshold   float64 `json:"threshold"`     // The threshold that was exceeded
	Value       float64 `json:"value"`         // The actual value
	ZScore      float64 `json:"z_score"`       // Standard deviations from mean
}

// BehavioralFeatures represents extracted behavioral features
type BehavioralFeatures struct {
	TypingConsistency    float64 `json:"typing_consistency"`     // 0.0-1.0
	MouseNaturalness     float64 `json:"mouse_naturalness"`      // 0.0-1.0
	InteractionPatterns  float64 `json:"interaction_patterns"`   // 0.0-1.0
	TimingAnomalies      float64 `json:"timing_anomalies"`       // 0.0-1.0
	DeviceConsistency    float64 `json:"device_consistency"`     // 0.0-1.0
	NavigationBehavior   float64 `json:"navigation_behavior"`    // 0.0-1.0
	FormFillBehavior     float64 `json:"form_fill_behavior"`     // 0.0-1.0
}

// NewBehavioralAnalyzer creates a new behavioral analyzer
func NewBehavioralAnalyzer(db *sql.DB, redis *redis.Client) *BehavioralAnalyzer {
	ba := &BehavioralAnalyzer{
		db:                 db,
		redis:              redis,
		botScoreThreshold:  0.7,
		anomalyThreshold:   2.0, // Standard deviations
		minDataPoints:      100,
		updateInterval:     24 * time.Hour,
		typingSpeedModel:   &StatisticalModel{},
		interactionModel:   &StatisticalModel{},
		mouseMovementModel: &StatisticalModel{},
		keystrokeModel:     &StatisticalModel{},
	}
	
	// Load existing models
	ba.loadModels()
	
	// Update models if needed
	if time.Since(ba.lastModelUpdate) > ba.updateInterval {
		go ba.updateModels()
	}
	
	return ba
}

// AnalyzeBehavior analyzes behavioral data and returns bot likelihood
func (ba *BehavioralAnalyzer) AnalyzeBehavior(profile *BehavioralProfile) (*BehavioralAnalysisResult, error) {
	result := &BehavioralAnalysisResult{
		AnomalyFlags: []AnomalyFlag{},
		Analysis:     make(map[string]interface{}),
	}
	
	// Extract behavioral features
	result.BehavioralFeats = ba.extractBehavioralFeatures(profile)
	
	// Analyze typing behavior
	typingAnomalies := ba.analyzeTypingBehavior(profile)
	result.AnomalyFlags = append(result.AnomalyFlags, typingAnomalies...)
	
	// Analyze mouse behavior
	mouseAnomalies := ba.analyzeMouseBehavior(profile)
	result.AnomalyFlags = append(result.AnomalyFlags, mouseAnomalies...)
	
	// Analyze interaction patterns
	interactionAnomalies := ba.analyzeInteractionPatterns(profile)
	result.AnomalyFlags = append(result.AnomalyFlags, interactionAnomalies...)
	
	// Analyze timing patterns
	timingAnomalies := ba.analyzeTimingPatterns(profile)
	result.AnomalyFlags = append(result.AnomalyFlags, timingAnomalies...)
	
	// Analyze device consistency
	deviceAnomalies := ba.analyzeDeviceConsistency(profile)
	result.AnomalyFlags = append(result.AnomalyFlags, deviceAnomalies...)
	
	// Calculate overall bot score
	result.BotScore = ba.calculateBotScore(result.AnomalyFlags, result.BehavioralFeats)
	result.HumanScore = 1.0 - result.BotScore
	
	// Calculate confidence based on data quality and quantity
	result.Confidence = ba.calculateConfidence(profile, result.AnomalyFlags)
	
	// Determine recommendation
	result.Recommendation = ba.getRecommendation(result.BotScore, result.Confidence)
	
	// Add detailed analysis
	result.Analysis["total_anomalies"] = len(result.AnomalyFlags)
	result.Analysis["high_severity_anomalies"] = ba.countAnomaliesBySeverity(result.AnomalyFlags, "high")
	result.Analysis["critical_anomalies"] = ba.countAnomaliesBySeverity(result.AnomalyFlags, "critical")
	result.Analysis["typing_speed"] = profile.TypingSpeed
	result.Analysis["mouse_movements"] = profile.MouseMovements
	result.Analysis["time_on_page"] = profile.TimeOnPage
	result.Analysis["interaction_delay"] = profile.InteractionDelay
	
	return result, nil
}

// extractBehavioralFeatures extracts high-level behavioral features
func (ba *BehavioralAnalyzer) extractBehavioralFeatures(profile *BehavioralProfile) BehavioralFeatures {
	features := BehavioralFeatures{}
	
	// Typing consistency - analyze rhythm and speed variations
	features.TypingConsistency = ba.calculateTypingConsistency(profile)
	
	// Mouse naturalness - analyze movement patterns
	features.MouseNaturalness = ba.calculateMouseNaturalness(profile)
	
	// Interaction patterns - analyze sequence and timing
	features.InteractionPatterns = ba.calculateInteractionNaturalness(profile)
	
	// Timing anomalies - detect unnatural timing patterns
	features.TimingAnomalies = ba.calculateTimingNaturalness(profile)
	
	// Device consistency - check for inconsistent device info
	features.DeviceConsistency = ba.calculateDeviceConsistency(profile)
	
	// Navigation behavior - analyze page interaction patterns
	features.NavigationBehavior = ba.calculateNavigationNaturalness(profile)
	
	// Form fill behavior - analyze form completion patterns
	features.FormFillBehavior = ba.calculateFormFillNaturalness(profile)
	
	return features
}

// analyzeTypingBehavior detects typing-related anomalies
func (ba *BehavioralAnalyzer) analyzeTypingBehavior(profile *BehavioralProfile) []AnomalyFlag {
	var anomalies []AnomalyFlag
	
	// Check typing speed
	if ba.typingSpeedModel.SampleCount > ba.minDataPoints {
		zScore := (profile.TypingSpeed - ba.typingSpeedModel.Mean) / ba.typingSpeedModel.StandardDeviation
		if math.Abs(zScore) > ba.anomalyThreshold {
			severity := "medium"
			score := 0.3
			if math.Abs(zScore) > 3.0 {
				severity = "high"
				score = 0.5
			}
			
			anomalies = append(anomalies, AnomalyFlag{
				Type:        "typing_speed",
				Description: fmt.Sprintf("Unusual typing speed: %.1f WPM", profile.TypingSpeed),
				Severity:    severity,
				Score:       score,
				Threshold:   ba.typingSpeedModel.Mean + ba.anomalyThreshold*ba.typingSpeedModel.StandardDeviation,
				Value:       profile.TypingSpeed,
				ZScore:      zScore,
			})
		}
	}
	
	// Check keystroke consistency
	if len(profile.TypingRhythm) > 10 {
		consistency := ba.calculateVariationCoefficient(profile.TypingRhythm)
		if consistency > 2.0 { // High variation indicates bot-like behavior
			anomalies = append(anomalies, AnomalyFlag{
				Type:        "keystroke_rhythm",
				Description: "Inconsistent keystroke rhythm",
				Severity:    "medium",
				Score:       0.4,
				Threshold:   2.0,
				Value:       consistency,
			})
		} else if consistency < 0.1 { // Too consistent also indicates bot
			anomalies = append(anomalies, AnomalyFlag{
				Type:        "keystroke_rhythm",
				Description: "Overly consistent keystroke rhythm",
				Severity:    "high",
				Score:       0.6,
				Threshold:   0.1,
				Value:       consistency,
			})
		}
	}
	
	// Check backspace ratio
	if profile.BackspaceRatio > 0.5 {
		anomalies = append(anomalies, AnomalyFlag{
			Type:        "backspace_ratio",
			Description: "Excessive backspace usage",
			Severity:    "low",
			Score:       0.2,
			Threshold:   0.5,
			Value:       profile.BackspaceRatio,
		})
	} else if profile.BackspaceRatio < 0.01 && profile.TypingSpeed > 30 {
		anomalies = append(anomalies, AnomalyFlag{
			Type:        "backspace_ratio",
			Description: "Unusually low backspace usage for typing speed",
			Severity:    "medium",
			Score:       0.4,
			Threshold:   0.01,
			Value:       profile.BackspaceRatio,
		})
	}
	
	return anomalies
}

// analyzeMouseBehavior detects mouse-related anomalies
func (ba *BehavioralAnalyzer) analyzeMouseBehavior(profile *BehavioralProfile) []AnomalyFlag {
	var anomalies []AnomalyFlag
	
	// Check for absence of mouse movements with significant typing
	if profile.MouseMovements == 0 && profile.TypingSpeed > 0 {
		anomalies = append(anomalies, AnomalyFlag{
			Type:        "no_mouse_movement",
			Description: "No mouse movements detected during typing",
			Severity:    "high",
			Score:       0.7,
			Threshold:   1,
			Value:       float64(profile.MouseMovements),
		})
	}
	
	// Check mouse movement distance vs. events
	if profile.MouseMovements > 0 && profile.MouseDistance/float64(profile.MouseMovements) < 10 {
		anomalies = append(anomalies, AnomalyFlag{
			Type:        "minimal_mouse_distance",
			Description: "Mouse movements cover minimal distance",
			Severity:    "medium",
			Score:       0.3,
			Threshold:   10,
			Value:       profile.MouseDistance / float64(profile.MouseMovements),
		})
	}
	
	// Check for excessive click events without mouse movements
	if profile.ClickEvents > profile.MouseMovements*2 {
		anomalies = append(anomalies, AnomalyFlag{
			Type:        "click_without_movement",
			Description: "High click rate without corresponding mouse movements",
			Severity:    "medium",
			Score:       0.4,
			Threshold:   2,
			Value:       float64(profile.ClickEvents) / float64(profile.MouseMovements),
		})
	}
	
	return anomalies
}

// analyzeInteractionPatterns detects interaction-related anomalies
func (ba *BehavioralAnalyzer) analyzeInteractionPatterns(profile *BehavioralProfile) []AnomalyFlag {
	var anomalies []AnomalyFlag
	
	// Check interaction delay (immediate interaction is suspicious)
	if profile.InteractionDelay < 0.5 {
		anomalies = append(anomalies, AnomalyFlag{
			Type:        "immediate_interaction",
			Description: "Form interaction started immediately",
			Severity:    "high",
			Score:       0.6,
			Threshold:   0.5,
			Value:       profile.InteractionDelay,
		})
	}
	
	// Check for excessive copy-paste
	totalInteractions := profile.MouseMovements + len(profile.TypingRhythm) + profile.ScrollEvents
	if totalInteractions > 0 && float64(profile.CopyPasteEvents)/float64(totalInteractions) > 0.5 {
		anomalies = append(anomalies, AnomalyFlag{
			Type:        "excessive_copy_paste",
			Description: "High ratio of copy-paste events",
			Severity:    "medium",
			Score:       0.4,
			Threshold:   0.5,
			Value:       float64(profile.CopyPasteEvents) / float64(totalInteractions),
		})
	}
	
	// Check for lack of focus events
	if profile.FocusEvents == 0 && totalInteractions > 10 {
		anomalies = append(anomalies, AnomalyFlag{
			Type:        "no_focus_events",
			Description: "No focus events detected during interaction",
			Severity:    "low",
			Score:       0.2,
			Threshold:   1,
			Value:       float64(profile.FocusEvents),
		})
	}
	
	return anomalies
}

// analyzeTimingPatterns detects timing-related anomalies
func (ba *BehavioralAnalyzer) analyzeTimingPatterns(profile *BehavioralProfile) []AnomalyFlag {
	var anomalies []AnomalyFlag
	
	// Analyze timestamp patterns for regularity
	if len(profile.Timestamps) > 5 {
		intervals := make([]float64, len(profile.Timestamps)-1)
		for i := 1; i < len(profile.Timestamps); i++ {
			intervals[i-1] = float64(profile.Timestamps[i] - profile.Timestamps[i-1])
		}
		
		// Check for overly regular intervals (bot-like)
		consistency := ba.calculateVariationCoefficient(intervals)
		if consistency < 0.1 {
			anomalies = append(anomalies, AnomalyFlag{
				Type:        "regular_timing",
				Description: "Overly regular interaction timing",
				Severity:    "high",
				Score:       0.8,
				Threshold:   0.1,
				Value:       consistency,
			})
		}
		
		// Check for burst patterns (multiple rapid interactions)
		burstCount := 0
		for _, interval := range intervals {
			if interval < 100 { // Less than 100ms between interactions
				burstCount++
			}
		}
		
		if float64(burstCount)/float64(len(intervals)) > 0.3 {
			anomalies = append(anomalies, AnomalyFlag{
				Type:        "burst_interactions",
				Description: "High frequency of rapid interactions",
				Severity:    "medium",
				Score:       0.5,
				Threshold:   0.3,
				Value:       float64(burstCount) / float64(len(intervals)),
			})
		}
	}
	
	// Check time on page vs. form complexity
	expectedMinTime := float64(len(profile.FormFillPattern)) * 5.0 // 5 seconds per field minimum
	if profile.TimeOnPage < expectedMinTime {
		anomalies = append(anomalies, AnomalyFlag{
			Type:        "insufficient_time",
			Description: "Form completed too quickly for complexity",
			Severity:    "high",
			Score:       0.7,
			Threshold:   expectedMinTime,
			Value:       profile.TimeOnPage,
		})
	}
	
	return anomalies
}

// analyzeDeviceConsistency checks for device information inconsistencies
func (ba *BehavioralAnalyzer) analyzeDeviceConsistency(profile *BehavioralProfile) []AnomalyFlag {
	var anomalies []AnomalyFlag
	
	// Check viewport vs screen size consistency
	if profile.DeviceInfo.ViewportWidth > profile.DeviceInfo.ScreenWidth ||
	   profile.DeviceInfo.ViewportHeight > profile.DeviceInfo.ScreenHeight {
		anomalies = append(anomalies, AnomalyFlag{
			Type:        "viewport_inconsistency",
			Description: "Viewport larger than screen size",
			Severity:    "medium",
			Score:       0.4,
			Threshold:   1,
			Value:       1,
		})
	}
	
	// Check for headless browser indicators
	if !profile.DeviceInfo.JavaScriptEnabled || profile.DeviceInfo.ColorDepth == 0 {
		anomalies = append(anomalies, AnomalyFlag{
			Type:        "headless_browser",
			Description: "Indicators of headless browser usage",
			Severity:    "critical",
			Score:       0.9,
			Threshold:   1,
			Value:       1,
		})
	}
	
	// Check for unusual screen dimensions
	if profile.DeviceInfo.ScreenWidth < 800 || profile.DeviceInfo.ScreenHeight < 600 ||
	   profile.DeviceInfo.ScreenWidth > 8000 || profile.DeviceInfo.ScreenHeight > 8000 {
		anomalies = append(anomalies, AnomalyFlag{
			Type:        "unusual_screen_size",
			Description: "Unusual screen dimensions detected",
			Severity:    "low",
			Score:       0.2,
			Threshold:   800,
			Value:       float64(profile.DeviceInfo.ScreenWidth),
		})
	}
	
	return anomalies
}

// Helper methods for feature calculation

func (ba *BehavioralAnalyzer) calculateTypingConsistency(profile *BehavioralProfile) float64 {
	if len(profile.TypingRhythm) < 5 {
		return 0.5 // Neutral score for insufficient data
	}
	
	consistency := ba.calculateVariationCoefficient(profile.TypingRhythm)
	
	// Convert to 0-1 score where 1 = natural human typing
	if consistency < 0.1 {
		return 0.1 // Too consistent (bot-like)
	} else if consistency > 2.0 {
		return 0.2 // Too inconsistent
	} else {
		// Optimal range is around 0.3-0.8
		return math.Max(0.1, math.Min(1.0, 1.0-math.Abs(consistency-0.5)))
	}
}

func (ba *BehavioralAnalyzer) calculateMouseNaturalness(profile *BehavioralProfile) float64 {
	if profile.MouseMovements == 0 {
		return 0.1 // Very unnatural
	}
	
	score := 1.0
	
	// Penalize minimal movement distance
	avgDistance := profile.MouseDistance / float64(profile.MouseMovements)
	if avgDistance < 20 {
		score -= 0.4
	}
	
	// Reward appropriate click-to-movement ratio
	clickRatio := float64(profile.ClickEvents) / float64(profile.MouseMovements)
	if clickRatio > 0.1 && clickRatio < 0.5 {
		score += 0.1
	} else {
		score -= 0.2
	}
	
	return math.Max(0.0, math.Min(1.0, score))
}

func (ba *BehavioralAnalyzer) calculateInteractionNaturalness(profile *BehavioralProfile) float64 {
	score := 1.0
	
	// Penalize immediate interaction
	if profile.InteractionDelay < 1.0 {
		score -= 0.5
	}
	
	// Reward natural focus patterns
	if profile.FocusEvents > 0 {
		score += 0.1
	}
	
	// Penalize excessive copy-paste
	totalEvents := profile.MouseMovements + len(profile.TypingRhythm) + profile.ScrollEvents
	if totalEvents > 0 {
		pasteRatio := float64(profile.CopyPasteEvents) / float64(totalEvents)
		if pasteRatio > 0.3 {
			score -= pasteRatio
		}
	}
	
	return math.Max(0.0, math.Min(1.0, score))
}

func (ba *BehavioralAnalyzer) calculateTimingNaturalness(profile *BehavioralProfile) float64 {
	if len(profile.Timestamps) < 3 {
		return 0.5
	}
	
	intervals := make([]float64, len(profile.Timestamps)-1)
	for i := 1; i < len(profile.Timestamps); i++ {
		intervals[i-1] = float64(profile.Timestamps[i] - profile.Timestamps[i-1])
	}
	
	consistency := ba.calculateVariationCoefficient(intervals)
	
	// Natural timing should have some variation but not be too chaotic
	if consistency < 0.2 {
		return 0.2 // Too regular
	} else if consistency > 3.0 {
		return 0.3 // Too chaotic
	} else {
		return math.Min(1.0, consistency/2.0)
	}
}

func (ba *BehavioralAnalyzer) calculateDeviceConsistency(profile *BehavioralProfile) float64 {
	score := 1.0
	
	// Check for basic consistency issues
	if profile.DeviceInfo.ViewportWidth > profile.DeviceInfo.ScreenWidth {
		score -= 0.5
	}
	
	if !profile.DeviceInfo.JavaScriptEnabled || profile.DeviceInfo.ColorDepth == 0 {
		score -= 0.8
	}
	
	if profile.DeviceInfo.ScreenWidth < 800 || profile.DeviceInfo.ScreenHeight < 600 {
		score -= 0.2
	}
	
	return math.Max(0.0, score)
}

func (ba *BehavioralAnalyzer) calculateNavigationNaturalness(profile *BehavioralProfile) float64 {
	score := 1.0
	
	// Reward scroll events (natural human behavior)
	if profile.ScrollEvents > 0 {
		score += 0.1
	}
	
	// Penalize excessive tab switches
	if profile.TabSwitches > 5 {
		score -= 0.3
	}
	
	// Reward appropriate time on page
	if profile.TimeOnPage > 10 && profile.TimeOnPage < 600 { // 10 seconds to 10 minutes
		score += 0.1
	}
	
	return math.Max(0.0, math.Min(1.0, score))
}

func (ba *BehavioralAnalyzer) calculateFormFillNaturalness(profile *BehavioralProfile) float64 {
	if len(profile.FormFillPattern) == 0 {
		return 0.5
	}
	
	score := 1.0
	
	// Check for natural field completion order and timing
	times := make([]float64, 0, len(profile.FormFillPattern))
	for _, time := range profile.FormFillPattern {
		times = append(times, time)
	}
	
	if len(times) > 1 {
		// Check for natural progression (generally increasing times)
		decreasing := 0
		for i := 1; i < len(times); i++ {
			if times[i] < times[i-1] {
				decreasing++
			}
		}
		
		decreasingRatio := float64(decreasing) / float64(len(times)-1)
		if decreasingRatio > 0.5 {
			score -= 0.4 // Too much jumping around
		}
	}
	
	return math.Max(0.0, math.Min(1.0, score))
}

func (ba *BehavioralAnalyzer) calculateVariationCoefficient(values []float64) float64 {
	if len(values) < 2 {
		return 0
	}
	
	// Calculate mean
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	mean := sum / float64(len(values))
	
	if mean == 0 {
		return 0
	}
	
	// Calculate standard deviation
	sumSquaredDiff := 0.0
	for _, v := range values {
		diff := v - mean
		sumSquaredDiff += diff * diff
	}
	stdDev := math.Sqrt(sumSquaredDiff / float64(len(values)-1))
	
	// Return coefficient of variation
	return stdDev / mean
}

func (ba *BehavioralAnalyzer) calculateBotScore(anomalies []AnomalyFlag, features BehavioralFeatures) float64 {
	// Start with feature-based score
	featureScore := (2.0 - features.TypingConsistency - features.MouseNaturalness - 
					features.InteractionPatterns - features.TimingAnomalies - 
					features.DeviceConsistency - features.NavigationBehavior - 
					features.FormFillBehavior) / 7.0
	
	// Add anomaly-based score
	anomalyScore := 0.0
	for _, anomaly := range anomalies {
		anomalyScore += anomaly.Score
	}
	
	// Combine scores (weighted average)
	finalScore := 0.6*featureScore + 0.4*math.Min(1.0, anomalyScore)
	
	return math.Max(0.0, math.Min(1.0, finalScore))
}

func (ba *BehavioralAnalyzer) calculateConfidence(profile *BehavioralProfile, anomalies []AnomalyFlag) float64 {
	// Base confidence on data quantity and quality
	dataQuality := 0.0
	
	// More data points = higher confidence
	if len(profile.TypingRhythm) > 10 {
		dataQuality += 0.2
	}
	if profile.MouseMovements > 5 {
		dataQuality += 0.2
	}
	if profile.TimeOnPage > 10 {
		dataQuality += 0.2
	}
	if len(profile.FormFillPattern) > 3 {
		dataQuality += 0.2
	}
	if len(profile.Timestamps) > 10 {
		dataQuality += 0.2
	}
	
	// High-severity anomalies increase confidence
	highSeverityCount := ba.countAnomaliesBySeverity(anomalies, "high") +
						ba.countAnomaliesBySeverity(anomalies, "critical")
	
	severityBonus := math.Min(0.3, float64(highSeverityCount)*0.1)
	
	return math.Min(1.0, dataQuality + severityBonus)
}

func (ba *BehavioralAnalyzer) getRecommendation(botScore, confidence float64) string {
	if confidence < 0.3 {
		return "challenge" // Low confidence, challenge with CAPTCHA
	}
	
	if botScore >= 0.8 && confidence >= 0.6 {
		return "block"
	} else if botScore >= 0.5 {
		return "challenge"
	} else {
		return "allow"
	}
}

func (ba *BehavioralAnalyzer) countAnomaliesBySeverity(anomalies []AnomalyFlag, severity string) int {
	count := 0
	for _, anomaly := range anomalies {
		if anomaly.Severity == severity {
			count++
		}
	}
	return count
}

// Model training and updating methods

func (ba *BehavioralAnalyzer) updateModels() error {
	// Update statistical models with recent data
	query := `
		SELECT behavioral_data FROM form_submissions 
		WHERE created_at >= DATE_SUB(NOW(), INTERVAL 7 DAY)
		AND action = 'allow'
		AND behavioral_data IS NOT NULL
		LIMIT 10000
	`
	
	rows, err := ba.db.Query(query)
	if err != nil {
		return err
	}
	defer rows.Close()
	
	var profiles []BehavioralProfile
	
	for rows.Next() {
		var dataJSON string
		if err := rows.Scan(&dataJSON); err != nil {
			continue
		}
		
		var profile BehavioralProfile
		if json.Unmarshal([]byte(dataJSON), &profile) == nil {
			profiles = append(profiles, profile)
		}
	}
	
	if len(profiles) >= ba.minDataPoints {
		ba.updateTypingSpeedModel(profiles)
		ba.updateInteractionModel(profiles)
		ba.updateMouseMovementModel(profiles)
		ba.updateKeystrokeModel(profiles)
		
		ba.lastModelUpdate = time.Now()
		ba.saveModels()
	}
	
	return nil
}

func (ba *BehavioralAnalyzer) updateTypingSpeedModel(profiles []BehavioralProfile) {
	speeds := make([]float64, 0, len(profiles))
	for _, profile := range profiles {
		if profile.TypingSpeed > 0 && profile.TypingSpeed < 200 { // Reasonable range
			speeds = append(speeds, profile.TypingSpeed)
		}
	}
	
	if len(speeds) > ba.minDataPoints {
		ba.typingSpeedModel = ba.calculateStatisticalModel(speeds)
	}
}

func (ba *BehavioralAnalyzer) updateInteractionModel(profiles []BehavioralProfile) {
	delays := make([]float64, 0, len(profiles))
	for _, profile := range profiles {
		if profile.InteractionDelay >= 0 && profile.InteractionDelay < 300 { // Reasonable range
			delays = append(delays, profile.InteractionDelay)
		}
	}
	
	if len(delays) > ba.minDataPoints {
		ba.interactionModel = ba.calculateStatisticalModel(delays)
	}
}

func (ba *BehavioralAnalyzer) updateMouseMovementModel(profiles []BehavioralProfile) {
	movements := make([]float64, 0, len(profiles))
	for _, profile := range profiles {
		if profile.MouseMovements >= 0 {
			movements = append(movements, float64(profile.MouseMovements))
		}
	}
	
	if len(movements) > ba.minDataPoints {
		ba.mouseMovementModel = ba.calculateStatisticalModel(movements)
	}
}

func (ba *BehavioralAnalyzer) updateKeystrokeModel(profiles []BehavioralProfile) {
	delays := make([]float64, 0)
	for _, profile := range profiles {
		if profile.AvgKeystrokeDelay > 0 && profile.AvgKeystrokeDelay < 1000 {
			delays = append(delays, profile.AvgKeystrokeDelay)
		}
	}
	
	if len(delays) > ba.minDataPoints {
		ba.keystrokeModel = ba.calculateStatisticalModel(delays)
	}
}

func (ba *BehavioralAnalyzer) calculateStatisticalModel(values []float64) *StatisticalModel {
	if len(values) == 0 {
		return &StatisticalModel{}
	}
	
	// Sort values for percentile calculation
	sortedValues := make([]float64, len(values))
	copy(sortedValues, values)
	sort.Float64s(sortedValues)
	
	// Calculate mean
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	mean := sum / float64(len(values))
	
	// Calculate standard deviation
	sumSquaredDiff := 0.0
	for _, v := range values {
		diff := v - mean
		sumSquaredDiff += diff * diff
	}
	stdDev := math.Sqrt(sumSquaredDiff / float64(len(values)-1))
	
	// Calculate percentiles
	percentiles := make([]float64, 5)
	percentiles[0] = sortedValues[int(float64(len(sortedValues))*0.05)] // 5th
	percentiles[1] = sortedValues[int(float64(len(sortedValues))*0.25)] // 25th
	percentiles[2] = sortedValues[int(float64(len(sortedValues))*0.50)] // 50th (median)
	percentiles[3] = sortedValues[int(float64(len(sortedValues))*0.75)] // 75th
	percentiles[4] = sortedValues[int(float64(len(sortedValues))*0.95)] // 95th
	
	return &StatisticalModel{
		Mean:              mean,
		StandardDeviation: stdDev,
		Percentiles:       percentiles,
		SampleCount:       len(values),
		LastUpdate:        time.Now(),
	}
}

func (ba *BehavioralAnalyzer) saveModels() error {
	models := map[string]*StatisticalModel{
		"typing_speed":    ba.typingSpeedModel,
		"interaction":     ba.interactionModel,
		"mouse_movement":  ba.mouseMovementModel,
		"keystroke":       ba.keystrokeModel,
	}
	
	for modelName, model := range models {
		modelJSON, err := json.Marshal(model)
		if err != nil {
			continue
		}
		
		query := `
			INSERT INTO behavioral_models (model_name, model_data, updated_at)
			VALUES (?, ?, ?)
			ON DUPLICATE KEY UPDATE model_data = VALUES(model_data), updated_at = VALUES(updated_at)
		`
		
		ba.db.Exec(query, modelName, string(modelJSON), time.Now())
	}
	
	return nil
}

func (ba *BehavioralAnalyzer) loadModels() error {
	query := `SELECT model_name, model_data FROM behavioral_models WHERE updated_at >= DATE_SUB(NOW(), INTERVAL 30 DAY)`
	
	rows, err := ba.db.Query(query)
	if err != nil {
		return err
	}
	defer rows.Close()
	
	for rows.Next() {
		var modelName, modelJSON string
		if err := rows.Scan(&modelName, &modelJSON); err != nil {
			continue
		}
		
		var model StatisticalModel
		if json.Unmarshal([]byte(modelJSON), &model) != nil {
			continue
		}
		
		switch modelName {
		case "typing_speed":
			ba.typingSpeedModel = &model
		case "interaction":
			ba.interactionModel = &model
		case "mouse_movement":
			ba.mouseMovementModel = &model
		case "keystroke":
			ba.keystrokeModel = &model
		}
	}
	
	return nil
}