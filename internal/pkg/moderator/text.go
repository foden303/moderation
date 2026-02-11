package moderator

import (
	"context"
	"strings"
	"sync"
	"time"

	"moderation/internal/pkg/bloom"
	"moderation/internal/pkg/filter"
	"moderation/internal/pkg/llm"
	"moderation/internal/pkg/redis"
)

// TextModerationResult represents the result of text moderation.
type TextModerationResult struct {
	IsClean         bool
	Matches         []filter.AhoCorasickMatch
	MaxSeverity     int32
	Categories      []string
	ShouldReject    bool
	ShouldReview    bool
	LLMChecked      bool
	LLMCategories   []llm.GuardCategory
	DetectedByLLM   bool
	DetectedPhrases []string // Phrases detected by LLM for feedback
}

// FeedbackCallback is called when LLM detects new bad content.
// This allows saving to DB and updating filters.
type FeedbackCallback func(ctx context.Context, phrase string, categories []llm.GuardCategory) error

// TextModerator provides text content moderation.
type TextModerator struct {
	bloomFilter *bloom.Filter
	ahoCorasick *filter.AhoCorasick
	vllmClient  *llm.VLLMClient
	mu          sync.RWMutex

	// Configuration
	rejectThreshold int32 // Severity threshold for auto-reject
	reviewThreshold int32 // Severity threshold for manual review
	enableLLM       bool  // Whether to use LLM for secondary check

	// Feedback callback for learning
	feedbackCallback FeedbackCallback
}

// TextModeratorConfig holds configuration for TextModerator.
type TextModeratorConfig struct {
	BloomBits          uint
	BloomHashFunctions uint
	BloomKey           string
	RejectThreshold    int32
	ReviewThreshold    int32
	EnableLLM          bool
	VLLMBaseURL        string
	VLLMModel          string
	VLLMTimeout        time.Duration
}

// DefaultTextModeratorConfig returns default configuration.
func DefaultTextModeratorConfig() TextModeratorConfig {
	return TextModeratorConfig{
		BloomBits:          1024 * 1024 * 8, // 8 million bits = 1MB
		BloomHashFunctions: 5,
		BloomKey:           "moderation:bloom:badwords",
		RejectThreshold:    3,
		ReviewThreshold:    2,
		EnableLLM:          true,
		VLLMBaseURL:        "http://localhost:8000",
		VLLMModel:          "Qwen/Qwen3Guard-Gen-0.6B",
		VLLMTimeout:        30 * time.Second,
	}
}

// NewTextModerator creates a new TextModerator.
func NewTextModerator(redisCache redis.Cache, config TextModeratorConfig) *TextModerator {
	tm := &TextModerator{
		bloomFilter:     bloom.NewBloomFilter(redisCache, config.BloomKey, config.BloomBits, config.BloomHashFunctions),
		ahoCorasick:     filter.NewAhoCorasick(),
		rejectThreshold: config.RejectThreshold,
		reviewThreshold: config.ReviewThreshold,
		enableLLM:       config.EnableLLM,
	}

	// Initialize vLLM client if LLM is enabled
	if config.EnableLLM && config.VLLMBaseURL != "" {
		tm.vllmClient = llm.NewVLLMClient(llm.VLLMConfig{
			BaseURL: config.VLLMBaseURL,
			Model:   config.VLLMModel,
			Timeout: config.VLLMTimeout,
		})
	}

	return tm
}

// SetFeedbackCallback sets the callback for learning new bad phrases.
func (tm *TextModerator) SetFeedbackCallback(cb FeedbackCallback) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	tm.feedbackCallback = cb
}

// BadWord represents a bad word with metadata.
type BadWord struct {
	Word     string
	Category string
	Severity int32
}

// RebuildFilters rebuilds both bloom filter and Aho-Corasick from the word list.
func (tm *TextModerator) RebuildFilters(ctx context.Context, words []BadWord) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	// Build Aho-Corasick patterns
	patterns := make([]filter.PatternInfo, len(words))
	for i, w := range words {
		patterns[i] = filter.PatternInfo{
			Word:     w.Word,
			Category: w.Category,
			Severity: w.Severity,
		}
	}
	tm.ahoCorasick.Build(patterns)

	// Add words to bloom filter
	for _, w := range words {
		normalizedWord := filter.NormalizeText(w.Word)
		if err := tm.bloomFilter.AddWithCtx(ctx, []byte(normalizedWord)); err != nil {
			return err
		}
	}

	return nil
}

// AddWord adds a single word to the filters.
func (tm *TextModerator) AddWord(ctx context.Context, word BadWord) error {
	normalizedWord := filter.NormalizeText(word.Word)
	return tm.bloomFilter.AddWithCtx(ctx, []byte(normalizedWord))
}

// Moderate checks text content for bad words with LLM fallback.
func (tm *TextModerator) Moderate(ctx context.Context, text string) (*TextModerationResult, error) {
	result := &TextModerationResult{
		IsClean:         true,
		Matches:         make([]filter.AhoCorasickMatch, 0),
		Categories:      make([]string, 0),
		LLMCategories:   make([]llm.GuardCategory, 0),
		DetectedPhrases: make([]string, 0),
	}

	if text == "" {
		return result, nil
	}

	// step 1: Fast Bloom Filter Check
	words := tokenize(text)
	hasPotentialMatch := false

	for _, word := range words {
		normalizedWord := filter.NormalizeText(word)
		exists, err := tm.bloomFilter.ExistsWithCtx(ctx, []byte(normalizedWord))
		if err != nil {
			return nil, err
		}
		if exists {
			hasPotentialMatch = true
			break
		}
	}

	// step 2: Precise Aho-Corasick Matching
	if hasPotentialMatch {
		matches := tm.ahoCorasick.Search(text)

		if len(matches) > 0 {
			result.IsClean = false
			result.Matches = matches

			// Calculate max severity and collect categories
			categorySet := make(map[string]struct{})
			for _, match := range matches {
				if match.Severity > result.MaxSeverity {
					result.MaxSeverity = match.Severity
				}
				categorySet[match.Category] = struct{}{}
			}

			for cat := range categorySet {
				result.Categories = append(result.Categories, cat)
			}

			// Determine action based on severity
			if result.MaxSeverity >= tm.rejectThreshold {
				result.ShouldReject = true
			} else if result.MaxSeverity >= tm.reviewThreshold {
				result.ShouldReview = true
			}

			return result, nil // Fast path - pattern matched
		}
	}

	// step 3: LLM deep analysis via vLLM + Qwen3Guard (if enabled)
	if tm.enableLLM && tm.vllmClient != nil {
		llmResult, err := tm.vllmClient.ModerateText(ctx, text)
		if err != nil {
			// LLM failed, but we continue with pattern-only result
			return result, nil
		}

		result.LLMChecked = true

		if !llmResult.IsSafe {
			result.IsClean = false
			result.DetectedByLLM = true
			result.LLMCategories = llmResult.ViolatedCategories

			// Qwen3Guard 3-tier severity: Unsafe → reject, Controversial → review
			switch llmResult.Severity {
			case llm.SeverityUnsafe:
				result.ShouldReject = true
			case llm.SeverityControversial:
				result.ShouldReview = true
			}

			// Extract phrases for feedback
			result.DetectedPhrases = append(result.DetectedPhrases, text)

			// step 4: Feedback loop - learn & store
			if tm.feedbackCallback != nil {
				go func() {
					feedbackCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
					defer cancel()
					tm.feedbackCallback(feedbackCtx, text, llmResult.ViolatedCategories)
				}()
			}
		}
	}

	return result, nil
}

// ModerateWithTimeout moderates text with a timeout.
func (tm *TextModerator) ModerateWithTimeout(ctx context.Context, text string, timeout time.Duration) (*TextModerationResult, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	return tm.Moderate(ctx, text)
}

// tokenize splits text into words.
func tokenize(text string) []string {
	// Simple tokenization - split by whitespace and punctuation
	words := make([]string, 0)
	current := strings.Builder{}

	for _, r := range text {
		if isWordChar(r) {
			current.WriteRune(r)
		} else {
			if current.Len() > 0 {
				words = append(words, current.String())
				current.Reset()
			}
		}
	}

	if current.Len() > 0 {
		words = append(words, current.String())
	}

	return words
}

func isWordChar(r rune) bool {
	return (r >= 'a' && r <= 'z') ||
		(r >= 'A' && r <= 'Z') ||
		(r >= '0' && r <= '9') ||
		r == '_' ||
		r >= 0x80 // Unicode characters
}
