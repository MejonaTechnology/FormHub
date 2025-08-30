package services

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// NaiveBayesSpamClassifier implements a Naive Bayes classifier for spam detection
type NaiveBayesSpamClassifier struct {
	db    *sql.DB
	redis *redis.Client
	
	// Model parameters
	spamProbability    float64
	hamProbability     float64
	vocabulary         map[string]bool
	spamWordCounts     map[string]int
	hamWordCounts      map[string]int
	totalSpamWords     int
	totalHamWords      int
	spamDocumentCount  int
	hamDocumentCount   int
	
	// Feature extractors
	wordExtractor     *WordFeatureExtractor
	patternExtractor  *PatternFeatureExtractor
	metaExtractor     *MetaFeatureExtractor
	
	// Configuration
	minWordFrequency  int
	smoothingFactor   float64
	lastTrainTime     time.Time
}

// WordFeature represents a word and its frequency
type WordFeature struct {
	Word      string  `json:"word"`
	Frequency int     `json:"frequency"`
	Weight    float64 `json:"weight"`
}

// PatternFeature represents a pattern match
type PatternFeature struct {
	Pattern     string  `json:"pattern"`
	Description string  `json:"description"`
	Count       int     `json:"count"`
	Weight      float64 `json:"weight"`
}

// MetaFeature represents metadata-based features
type MetaFeature struct {
	Name   string      `json:"name"`
	Value  interface{} `json:"value"`
	Weight float64     `json:"weight"`
}

// FeatureVector represents extracted features from a submission
type FeatureVector struct {
	Words    []WordFeature    `json:"words"`
	Patterns []PatternFeature `json:"patterns"`
	Meta     []MetaFeature    `json:"meta"`
}

// ClassificationResult contains the results of spam classification
type ClassificationResult struct {
	SpamProbability float64       `json:"spam_probability"`
	HamProbability  float64       `json:"ham_probability"`
	Confidence      float64       `json:"confidence"`
	Features        FeatureVector `json:"features"`
	ModelVersion    string        `json:"model_version"`
}

// NewNaiveBayesSpamClassifier creates a new Naive Bayes spam classifier
func NewNaiveBayesSpamClassifier(db *sql.DB, redis *redis.Client) *NaiveBayesSpamClassifier {
	classifier := &NaiveBayesSpamClassifier{
		db:                db,
		redis:             redis,
		vocabulary:        make(map[string]bool),
		spamWordCounts:    make(map[string]int),
		hamWordCounts:     make(map[string]int),
		minWordFrequency:  2,
		smoothingFactor:   1.0, // Laplace smoothing
		wordExtractor:     NewWordFeatureExtractor(),
		patternExtractor:  NewPatternFeatureExtractor(),
		metaExtractor:     NewMetaFeatureExtractor(),
	}
	
	// Load or initialize the model
	classifier.loadModel()
	
	return classifier
}

// PredictSpam predicts if a submission is spam
func (nb *NaiveBayesSpamClassifier) PredictSpam(data map[string]interface{}, 
	metadata map[string]interface{}) (float64, float64, error) {
	
	// Extract features from the submission
	features := nb.extractFeatures(data, metadata)
	
	// Calculate probabilities using Naive Bayes
	result := nb.classify(features)
	
	return result.SpamProbability, result.Confidence, nil
}

// TrainModel trains the Naive Bayes model with labeled data
func (nb *NaiveBayesSpamClassifier) TrainModel(trainingData []TrainingExample) error {
	// Reset counters
	nb.spamWordCounts = make(map[string]int)
	nb.hamWordCounts = make(map[string]int)
	nb.vocabulary = make(map[string]bool)
	nb.totalSpamWords = 0
	nb.totalHamWords = 0
	nb.spamDocumentCount = 0
	nb.hamDocumentCount = 0
	
	// Process training examples
	for _, example := range trainingData {
		features := nb.extractFeatures(example.Data, example.Metadata)
		
		if example.IsSpam {
			nb.spamDocumentCount++
			nb.trainOnFeatures(features, true)
		} else {
			nb.hamDocumentCount++
			nb.trainOnFeatures(features, false)
		}
	}
	
	// Calculate prior probabilities
	totalDocs := nb.spamDocumentCount + nb.hamDocumentCount
	if totalDocs > 0 {
		nb.spamProbability = float64(nb.spamDocumentCount) / float64(totalDocs)
		nb.hamProbability = float64(nb.hamDocumentCount) / float64(totalDocs)
	}
	
	// Filter vocabulary by minimum frequency
	nb.filterVocabulary()
	
	// Save the trained model
	nb.lastTrainTime = time.Now()
	return nb.saveModel()
}

// IncrementalTrain updates the model with new examples
func (nb *NaiveBayesSpamClassifier) IncrementalTrain(examples []TrainingExample) error {
	for _, example := range examples {
		features := nb.extractFeatures(example.Data, example.Metadata)
		
		if example.IsSpam {
			nb.spamDocumentCount++
			nb.trainOnFeatures(features, true)
		} else {
			nb.hamDocumentCount++
			nb.trainOnFeatures(features, false)
		}
	}
	
	// Recalculate priors
	totalDocs := nb.spamDocumentCount + nb.hamDocumentCount
	if totalDocs > 0 {
		nb.spamProbability = float64(nb.spamDocumentCount) / float64(totalDocs)
		nb.hamProbability = float64(nb.hamDocumentCount) / float64(totalDocs)
	}
	
	nb.lastTrainTime = time.Now()
	return nb.saveModel()
}

// extractFeatures extracts features from submission data
func (nb *NaiveBayesSpamClassifier) extractFeatures(data map[string]interface{}, 
	metadata map[string]interface{}) FeatureVector {
	
	var features FeatureVector
	
	// Combine all text content
	var textContent strings.Builder
	for _, value := range data {
		if str, ok := value.(string); ok {
			textContent.WriteString(str)
			textContent.WriteString(" ")
		}
	}
	
	text := textContent.String()
	
	// Extract word features
	features.Words = nb.wordExtractor.ExtractWords(text)
	
	// Extract pattern features
	features.Patterns = nb.patternExtractor.ExtractPatterns(text)
	
	// Extract metadata features
	features.Meta = nb.metaExtractor.ExtractMeta(data, metadata)
	
	return features
}

// trainOnFeatures trains the model on extracted features
func (nb *NaiveBayesSpamClassifier) trainOnFeatures(features FeatureVector, isSpam bool) {
	// Train on word features
	for _, wordFeature := range features.Words {
		word := wordFeature.Word
		nb.vocabulary[word] = true
		
		if isSpam {
			nb.spamWordCounts[word] += wordFeature.Frequency
			nb.totalSpamWords += wordFeature.Frequency
		} else {
			nb.hamWordCounts[word] += wordFeature.Frequency
			nb.totalHamWords += wordFeature.Frequency
		}
	}
	
	// Train on pattern features (treated as words for simplicity)
	for _, pattern := range features.Patterns {
		patternKey := "PATTERN:" + pattern.Pattern
		nb.vocabulary[patternKey] = true
		
		if isSpam {
			nb.spamWordCounts[patternKey] += pattern.Count
			nb.totalSpamWords += pattern.Count
		} else {
			nb.hamWordCounts[patternKey] += pattern.Count
			nb.totalHamWords += pattern.Count
		}
	}
}

// classify performs Naive Bayes classification
func (nb *NaiveBayesSpamClassifier) classify(features FeatureVector) ClassificationResult {
	// Start with prior probabilities (in log space to avoid underflow)
	logSpamProb := math.Log(nb.spamProbability)
	logHamProb := math.Log(nb.hamProbability)
	
	// Calculate likelihood for each feature
	for _, wordFeature := range features.Words {
		word := wordFeature.Word
		
		// Get word counts with smoothing
		spamCount := nb.spamWordCounts[word]
		hamCount := nb.hamWordCounts[word]
		
		// Apply Laplace smoothing
		spamProb := float64(spamCount+1) / float64(nb.totalSpamWords+len(nb.vocabulary))
		hamProb := float64(hamCount+1) / float64(nb.totalHamWords+len(nb.vocabulary))
		
		// Add to log probabilities (multiply frequencies for repeated words)
		logSpamProb += float64(wordFeature.Frequency) * math.Log(spamProb)
		logHamProb += float64(wordFeature.Frequency) * math.Log(hamProb)
	}
	
	// Handle pattern features
	for _, pattern := range features.Patterns {
		patternKey := "PATTERN:" + pattern.Pattern
		
		spamCount := nb.spamWordCounts[patternKey]
		hamCount := nb.hamWordCounts[patternKey]
		
		spamProb := float64(spamCount+1) / float64(nb.totalSpamWords+len(nb.vocabulary))
		hamProb := float64(hamCount+1) / float64(nb.totalHamWords+len(nb.vocabulary))
		
		logSpamProb += float64(pattern.Count) * math.Log(spamProb)
		logHamProb += float64(pattern.Count) * math.Log(hamProb)
	}
	
	// Convert back to probabilities and normalize
	maxLogProb := math.Max(logSpamProb, logHamProb)
	spamProb := math.Exp(logSpamProb - maxLogProb)
	hamProb := math.Exp(logHamProb - maxLogProb)
	
	total := spamProb + hamProb
	if total > 0 {
		spamProb /= total
		hamProb /= total
	}
	
	// Calculate confidence based on the margin between probabilities
	confidence := math.Abs(spamProb - hamProb)
	
	return ClassificationResult{
		SpamProbability: spamProb,
		HamProbability:  hamProb,
		Confidence:      confidence,
		Features:        features,
		ModelVersion:    nb.getModelVersion(),
	}
}

// filterVocabulary removes low-frequency words from vocabulary
func (nb *NaiveBayesSpamClassifier) filterVocabulary() {
	filteredVocab := make(map[string]bool)
	filteredSpamCounts := make(map[string]int)
	filteredHamCounts := make(map[string]int)
	
	for word := range nb.vocabulary {
		spamCount := nb.spamWordCounts[word]
		hamCount := nb.hamWordCounts[word]
		totalCount := spamCount + hamCount
		
		if totalCount >= nb.minWordFrequency {
			filteredVocab[word] = true
			if spamCount > 0 {
				filteredSpamCounts[word] = spamCount
			}
			if hamCount > 0 {
				filteredHamCounts[word] = hamCount
			}
		}
	}
	
	nb.vocabulary = filteredVocab
	nb.spamWordCounts = filteredSpamCounts
	nb.hamWordCounts = filteredHamCounts
}

// saveModel saves the trained model to database
func (nb *NaiveBayesSpamClassifier) saveModel() error {
	modelData := map[string]interface{}{
		"spam_probability":    nb.spamProbability,
		"ham_probability":     nb.hamProbability,
		"vocabulary":          nb.vocabulary,
		"spam_word_counts":    nb.spamWordCounts,
		"ham_word_counts":     nb.hamWordCounts,
		"total_spam_words":    nb.totalSpamWords,
		"total_ham_words":     nb.totalHamWords,
		"spam_document_count": nb.spamDocumentCount,
		"ham_document_count":  nb.hamDocumentCount,
		"min_word_frequency":  nb.minWordFrequency,
		"smoothing_factor":    nb.smoothingFactor,
		"last_train_time":     nb.lastTrainTime,
		"model_version":       nb.getModelVersion(),
	}
	
	modelJSON, err := json.Marshal(modelData)
	if err != nil {
		return err
	}
	
	query := `
		INSERT INTO ml_models (id, model_type, model_data, version, created_at, updated_at)
		VALUES (?, 'naive_bayes_spam', ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE 
			model_data = VALUES(model_data),
			version = VALUES(version),
			updated_at = VALUES(updated_at)
	`
	
	version := nb.getModelVersion()
	now := time.Now()
	
	_, err = nb.db.Exec(query, "nb_spam_classifier", string(modelJSON), version, now, now)
	return err
}

// loadModel loads the trained model from database
func (nb *NaiveBayesSpamClassifier) loadModel() error {
	query := `
		SELECT model_data FROM ml_models 
		WHERE id = 'nb_spam_classifier' AND model_type = 'naive_bayes_spam'
		ORDER BY updated_at DESC LIMIT 1
	`
	
	var modelJSON string
	err := nb.db.QueryRow(query).Scan(&modelJSON)
	if err != nil {
		// No existing model, initialize with defaults
		nb.spamProbability = 0.5
		nb.hamProbability = 0.5
		return nil
	}
	
	var modelData map[string]interface{}
	if err := json.Unmarshal([]byte(modelJSON), &modelData); err != nil {
		return err
	}
	
	// Load model parameters
	if val, ok := modelData["spam_probability"].(float64); ok {
		nb.spamProbability = val
	}
	if val, ok := modelData["ham_probability"].(float64); ok {
		nb.hamProbability = val
	}
	if val, ok := modelData["total_spam_words"].(float64); ok {
		nb.totalSpamWords = int(val)
	}
	if val, ok := modelData["total_ham_words"].(float64); ok {
		nb.totalHamWords = int(val)
	}
	if val, ok := modelData["spam_document_count"].(float64); ok {
		nb.spamDocumentCount = int(val)
	}
	if val, ok := modelData["ham_document_count"].(float64); ok {
		nb.hamDocumentCount = int(val)
	}
	
	// Load vocabulary and word counts
	if vocab, ok := modelData["vocabulary"].(map[string]interface{}); ok {
		nb.vocabulary = make(map[string]bool)
		for word := range vocab {
			nb.vocabulary[word] = true
		}
	}
	
	if spamCounts, ok := modelData["spam_word_counts"].(map[string]interface{}); ok {
		nb.spamWordCounts = make(map[string]int)
		for word, count := range spamCounts {
			if countFloat, ok := count.(float64); ok {
				nb.spamWordCounts[word] = int(countFloat)
			}
		}
	}
	
	if hamCounts, ok := modelData["ham_word_counts"].(map[string]interface{}); ok {
		nb.hamWordCounts = make(map[string]int)
		for word, count := range hamCounts {
			if countFloat, ok := count.(float64); ok {
				nb.hamWordCounts[word] = int(countFloat)
			}
		}
	}
	
	if trainTime, ok := modelData["last_train_time"].(string); ok {
		if parsed, err := time.Parse(time.RFC3339, trainTime); err == nil {
			nb.lastTrainTime = parsed
		}
	}
	
	return nil
}

func (nb *NaiveBayesSpamClassifier) getModelVersion() string {
	return fmt.Sprintf("v1.0_%d", nb.lastTrainTime.Unix())
}

// GetModelStats returns statistics about the trained model
func (nb *NaiveBayesSpamClassifier) GetModelStats() map[string]interface{} {
	return map[string]interface{}{
		"vocabulary_size":      len(nb.vocabulary),
		"spam_documents":       nb.spamDocumentCount,
		"ham_documents":        nb.hamDocumentCount,
		"total_spam_words":     nb.totalSpamWords,
		"total_ham_words":      nb.totalHamWords,
		"spam_probability":     nb.spamProbability,
		"ham_probability":      nb.hamProbability,
		"last_train_time":      nb.lastTrainTime,
		"model_version":        nb.getModelVersion(),
		"min_word_frequency":   nb.minWordFrequency,
		"smoothing_factor":     nb.smoothingFactor,
	}
}

// Feature extractors

// WordFeatureExtractor extracts word-based features
type WordFeatureExtractor struct {
	stopWords map[string]bool
}

func NewWordFeatureExtractor() *WordFeatureExtractor {
	stopWords := map[string]bool{
		"the": true, "a": true, "an": true, "and": true, "or": true, "but": true,
		"in": true, "on": true, "at": true, "to": true, "for": true, "of": true,
		"with": true, "by": true, "from": true, "up": true, "about": true,
		"into": true, "through": true, "during": true, "before": true, "after": true,
		"above": true, "below": true, "between": true, "among": true, "this": true,
		"that": true, "these": true, "those": true, "i": true, "you": true, "he": true,
		"she": true, "it": true, "we": true, "they": true, "me": true, "him": true,
		"her": true, "us": true, "them": true, "my": true, "your": true, "his": true,
		"its": true, "our": true, "their": true, "is": true, "am": true, "are": true,
		"was": true, "were": true, "be": true, "been": true, "being": true, "have": true,
		"has": true, "had": true, "do": true, "does": true, "did": true, "will": true,
		"would": true, "should": true, "could": true, "can": true, "may": true, "might": true,
	}
	
	return &WordFeatureExtractor{stopWords: stopWords}
}

func (wfe *WordFeatureExtractor) ExtractWords(text string) []WordFeature {
	// Normalize text
	text = strings.ToLower(text)
	
	// Extract words using regex
	wordRegex := regexp.MustCompile(`\b[a-zA-Z]{2,}\b`)
	words := wordRegex.FindAllString(text, -1)
	
	// Count word frequencies
	wordCounts := make(map[string]int)
	for _, word := range words {
		if !wfe.stopWords[word] && len(word) >= 2 {
			wordCounts[word]++
		}
	}
	
	// Convert to features
	var features []WordFeature
	for word, count := range wordCounts {
		features = append(features, WordFeature{
			Word:      word,
			Frequency: count,
			Weight:    1.0,
		})
	}
	
	// Sort by frequency (most frequent first)
	sort.Slice(features, func(i, j int) bool {
		return features[i].Frequency > features[j].Frequency
	})
	
	// Limit to top 100 words to avoid overfitting
	if len(features) > 100 {
		features = features[:100]
	}
	
	return features
}

// PatternFeatureExtractor extracts pattern-based features
type PatternFeatureExtractor struct {
	patterns []struct {
		regex       *regexp.Regexp
		name        string
		description string
		weight      float64
	}
}

func NewPatternFeatureExtractor() *PatternFeatureExtractor {
	pfe := &PatternFeatureExtractor{}
	
	patterns := []struct {
		pattern     string
		name        string
		description string
		weight      float64
	}{
		{`\b(urgent|hurry|act now|limited time|don't wait|call now)\b`, "urgency", "Urgent language", 1.5},
		{`\b(free|no cost|gratis|complimentary)\b`, "free_offers", "Free offers", 1.2},
		{`\b(money|cash|dollars?|€|£|\$\d+)\b`, "money_mentions", "Money mentions", 1.3},
		{`\b(click here|visit now|buy now|order today)\b`, "call_to_action", "Call to action", 1.4},
		{`[!]{2,}`, "excessive_exclamation", "Excessive exclamation", 1.1},
		{`[A-Z]{3,}`, "all_caps_words", "All caps words", 1.2},
		{`https?://[^\s]+`, "urls", "URLs", 1.0},
		{`\b\w+@\w+\.\w+\b`, "emails", "Email addresses", 1.1},
		{`\b\d{3}[-.\s]?\d{3}[-.\s]?\d{4}\b`, "phone_numbers", "Phone numbers", 1.0},
		{`\b(sex|porn|adult|xxx|viagra|casino|lottery|gambling)\b`, "adult_content", "Adult/gambling content", 2.0},
		{`\b(winner|congratulations|selected|chosen|lucky)\b`, "lottery_language", "Lottery language", 1.6},
		{`\b(guarantee|guaranteed|promise|assured)\b`, "guarantees", "Guarantee language", 1.3},
		{`[0-9]{1,3}%`, "percentages", "Percentage offers", 1.1},
		{`\b(save|discount|offer|deal|special|promotion)\b`, "offers", "Special offers", 1.2},
	}
	
	for _, p := range patterns {
		compiled := regexp.MustCompile(`(?i)` + p.pattern)
		pfe.patterns = append(pfe.patterns, struct {
			regex       *regexp.Regexp
			name        string
			description string
			weight      float64
		}{compiled, p.name, p.description, p.weight})
	}
	
	return pfe
}

func (pfe *PatternFeatureExtractor) ExtractPatterns(text string) []PatternFeature {
	var features []PatternFeature
	
	for _, pattern := range pfe.patterns {
		matches := pattern.regex.FindAllString(text, -1)
		if len(matches) > 0 {
			features = append(features, PatternFeature{
				Pattern:     pattern.name,
				Description: pattern.description,
				Count:       len(matches),
				Weight:      pattern.weight,
			})
		}
	}
	
	return features
}

// MetaFeatureExtractor extracts metadata-based features
type MetaFeatureExtractor struct{}

func NewMetaFeatureExtractor() *MetaFeatureExtractor {
	return &MetaFeatureExtractor{}
}

func (mfe *MetaFeatureExtractor) ExtractMeta(data map[string]interface{}, 
	metadata map[string]interface{}) []MetaFeature {
	
	var features []MetaFeature
	
	// Extract user agent features
	if userAgent, ok := metadata["user_agent"].(string); ok {
		features = append(features, MetaFeature{
			Name:   "user_agent_length",
			Value:  len(userAgent),
			Weight: 0.5,
		})
		
		// Check for bot indicators in user agent
		botIndicators := []string{"bot", "crawler", "spider", "scraper"}
		for _, indicator := range botIndicators {
			if strings.Contains(strings.ToLower(userAgent), indicator) {
				features = append(features, MetaFeature{
					Name:   "bot_user_agent",
					Value:  1,
					Weight: 2.0,
				})
				break
			}
		}
	}
	
	// Extract form field count and types
	fieldCount := len(data)
	features = append(features, MetaFeature{
		Name:   "field_count",
		Value:  fieldCount,
		Weight: 0.3,
	})
	
	// Analyze field values
	totalTextLength := 0
	urlCount := 0
	emailCount := 0
	
	urlRegex := regexp.MustCompile(`https?://[^\s]+`)
	emailRegex := regexp.MustCompile(`\b\w+@\w+\.\w+\b`)
	
	for _, value := range data {
		if str, ok := value.(string); ok {
			totalTextLength += len(str)
			urlCount += len(urlRegex.FindAllString(str, -1))
			emailCount += len(emailRegex.FindAllString(str, -1))
		}
	}
	
	features = append(features, MetaFeature{
		Name:   "total_text_length",
		Value:  totalTextLength,
		Weight: 0.4,
	})
	
	if urlCount > 0 {
		features = append(features, MetaFeature{
			Name:   "url_count",
			Value:  urlCount,
			Weight: 1.2,
		})
	}
	
	if emailCount > 0 {
		features = append(features, MetaFeature{
			Name:   "email_count",
			Value:  emailCount,
			Weight: 0.8,
		})
	}
	
	// Extract time-based features if available
	if behavioralStr, ok := metadata["behavioral_data"].(string); ok {
		var behavioral map[string]interface{}
		if json.Unmarshal([]byte(behavioralStr), &behavioral) == nil {
			if typingTime, ok := behavioral["typing_time"].(float64); ok {
				features = append(features, MetaFeature{
					Name:   "typing_time",
					Value:  typingTime,
					Weight: 1.0,
				})
			}
			
			if typingSpeed, ok := behavioral["typing_speed"].(float64); ok {
				features = append(features, MetaFeature{
					Name:   "typing_speed",
					Value:  typingSpeed,
					Weight: 1.2,
				})
			}
		}
	}
	
	return features
}

// TrainingExample represents a training example for the classifier
type TrainingExample struct {
	Data     map[string]interface{} `json:"data"`
	Metadata map[string]interface{} `json:"metadata"`
	IsSpam   bool                   `json:"is_spam"`
	Source   string                 `json:"source"` // "manual", "feedback", "honeypot"
}

// AutomaticTrainingService automatically collects training data
type AutomaticTrainingService struct {
	db         *sql.DB
	redis      *redis.Client
	classifier *NaiveBayesSpamClassifier
}

func NewAutomaticTrainingService(db *sql.DB, redis *redis.Client, 
	classifier *NaiveBayesSpamClassifier) *AutomaticTrainingService {
	
	return &AutomaticTrainingService{
		db:         db,
		redis:      redis,
		classifier: classifier,
	}
}

// CollectTrainingData collects training examples from various sources
func (ats *AutomaticTrainingService) CollectTrainingData() ([]TrainingExample, error) {
	var examples []TrainingExample
	
	// Collect from honeypot submissions (definitely spam)
	honeypotExamples, err := ats.getHoneypotExamples()
	if err == nil {
		examples = append(examples, honeypotExamples...)
	}
	
	// Collect from manually verified submissions
	manualExamples, err := ats.getManualExamples()
	if err == nil {
		examples = append(examples, manualExamples...)
	}
	
	// Collect from user feedback
	feedbackExamples, err := ats.getFeedbackExamples()
	if err == nil {
		examples = append(examples, feedbackExamples...)
	}
	
	return examples, nil
}

func (ats *AutomaticTrainingService) getHoneypotExamples() ([]TrainingExample, error) {
	query := `
		SELECT submission_data, metadata FROM form_submissions 
		WHERE spam_score >= 0.9 AND action = 'block' 
		AND JSON_EXTRACT(triggers, '$[*].type') LIKE '%honeypot%'
		AND created_at >= DATE_SUB(NOW(), INTERVAL 30 DAY)
		LIMIT 1000
	`
	
	rows, err := ats.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var examples []TrainingExample
	
	for rows.Next() {
		var dataJSON, metadataJSON string
		if err := rows.Scan(&dataJSON, &metadataJSON); err != nil {
			continue
		}
		
		var data, metadata map[string]interface{}
		if json.Unmarshal([]byte(dataJSON), &data) != nil ||
		   json.Unmarshal([]byte(metadataJSON), &metadata) != nil {
			continue
		}
		
		examples = append(examples, TrainingExample{
			Data:     data,
			Metadata: metadata,
			IsSpam:   true,
			Source:   "honeypot",
		})
	}
	
	return examples, nil
}

func (ats *AutomaticTrainingService) getManualExamples() ([]TrainingExample, error) {
	query := `
		SELECT submission_data, metadata, is_spam FROM manual_spam_labels 
		WHERE created_at >= DATE_SUB(NOW(), INTERVAL 90 DAY)
		LIMIT 5000
	`
	
	rows, err := ats.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var examples []TrainingExample
	
	for rows.Next() {
		var dataJSON, metadataJSON string
		var isSpam bool
		if err := rows.Scan(&dataJSON, &metadataJSON, &isSpam); err != nil {
			continue
		}
		
		var data, metadata map[string]interface{}
		if json.Unmarshal([]byte(dataJSON), &data) != nil ||
		   json.Unmarshal([]byte(metadataJSON), &metadata) != nil {
			continue
		}
		
		examples = append(examples, TrainingExample{
			Data:     data,
			Metadata: metadata,
			IsSpam:   isSpam,
			Source:   "manual",
		})
	}
	
	return examples, nil
}

func (ats *AutomaticTrainingService) getFeedbackExamples() ([]TrainingExample, error) {
	query := `
		SELECT submission_data, metadata, feedback_spam FROM user_feedback 
		WHERE created_at >= DATE_SUB(NOW(), INTERVAL 60 DAY)
		AND feedback_spam IS NOT NULL
		LIMIT 2000
	`
	
	rows, err := ats.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var examples []TrainingExample
	
	for rows.Next() {
		var dataJSON, metadataJSON string
		var isSpam bool
		if err := rows.Scan(&dataJSON, &metadataJSON, &isSpam); err != nil {
			continue
		}
		
		var data, metadata map[string]interface{}
		if json.Unmarshal([]byte(dataJSON), &data) != nil ||
		   json.Unmarshal([]byte(metadataJSON), &metadata) != nil {
			continue
		}
		
		examples = append(examples, TrainingExample{
			Data:     data,
			Metadata: metadata,
			IsSpam:   isSpam,
			Source:   "feedback",
		})
	}
	
	return examples, nil
}

// RetrainModel retrains the classifier with fresh data
func (ats *AutomaticTrainingService) RetrainModel() error {
	trainingData, err := ats.CollectTrainingData()
	if err != nil {
		return err
	}
	
	if len(trainingData) < 100 {
		return fmt.Errorf("insufficient training data: %d examples", len(trainingData))
	}
	
	return ats.classifier.TrainModel(trainingData)
}