package moderator

import (
	"context"
	"strings"
	"sync"
	"time"

	"moderation/internal/pkg/bloom"
	"moderation/internal/pkg/filter"
	"moderation/internal/pkg/hash"
	"moderation/internal/pkg/nsfw"
	"moderation/internal/pkg/redis"
)

// TextModerationResult represents the result of text moderation.
type TextModerationResult struct {
	IsClean         bool
	Matches         []filter.AhoCorasickMatch
	MaxNsfwScore    float64
	Categories      []string
	ShouldReject    bool
	ShouldReview    bool
	NSFWChecked     bool     // Whether NSFW text model was called
	DetectedByNSFW  bool     // Whether NSFW model flagged it
	DetectedPhrases []string // Phrases detected for feedback
}

// FeedbackCallback is called when NSFW model detects new bad content.
// This allows saving to DB and updating filters.
type FeedbackCallback func(ctx context.Context, phrase string, categories []string) error

// TextModerator provides text content moderation.
type TextModerator struct {
	bloomFilter    *bloom.Filter
	ahoCorasick    *filter.AhoCorasick
	nsfwTextClient *nsfw.TextClient
	mu             sync.RWMutex

	// Configuration
	rejectThreshold float64 // Severity threshold for auto-reject
	reviewThreshold float64 // Severity threshold for manual review

	// Feedback callback for learning
	feedbackCallback FeedbackCallback
}

// TextModeratorConfig holds configuration for TextModerator.
type TextModeratorConfig struct {
	BloomBits          uint
	BloomHashFunctions uint
	BloomKey           string
	RejectThreshold    float64
	ReviewThreshold    float64
}

// DefaultTextModeratorConfig returns default configuration.
func DefaultTextModeratorConfig() TextModeratorConfig {
	return TextModeratorConfig{
		BloomBits:          1024 * 1024 * 8, // 8 million bits = 1MB
		BloomHashFunctions: 5,
		BloomKey:           "moderation:bloom:badwords",
		RejectThreshold:    0.85,
		ReviewThreshold:    0.5,
	}
}

// NewTextModerator creates a new TextModerator.
// nsfwTextClient can be nil if text NSFW detection is disabled.
func NewTextModerator(redisCache redis.Cache, config TextModeratorConfig, nsfwTextClient *nsfw.TextClient) *TextModerator {
	return &TextModerator{
		bloomFilter:     bloom.NewBloomFilter(redisCache, config.BloomKey, config.BloomBits, config.BloomHashFunctions),
		ahoCorasick:     filter.NewAhoCorasick(),
		nsfwTextClient:  nsfwTextClient,
		rejectThreshold: config.RejectThreshold,
		reviewThreshold: config.ReviewThreshold,
	}
}

// SetFeedbackCallback sets the callback for learning new bad phrases.
func (tm *TextModerator) SetFeedbackCallback(cb FeedbackCallback) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	tm.feedbackCallback = cb
}

// BadWord represents a bad word with metadata.
type BadWord struct {
	Word      string
	Category  string
	NsfwScore float64
}

// RebuildFilters rebuilds both bloom filter and Aho-Corasick from the word list.
func (tm *TextModerator) RebuildFilters(ctx context.Context, words []BadWord) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	// Build Aho-Corasick patterns
	patterns := make([]filter.PatternInfo, len(words))
	for i, w := range words {
		patterns[i] = filter.PatternInfo{
			Word:      w.Word,
			Category:  w.Category,
			NsfwScore: w.NsfwScore,
		}
	}
	tm.ahoCorasick.Build(patterns)

	// Add words to bloom filter
	for _, w := range words {
		// Add full phrase
		normalizedWord := filter.NormalizeText(w.Word)
		hashedWord := hash.HashTextSha256(normalizedWord)
		if err := tm.bloomFilter.AddWithCtx(ctx, []byte(hashedWord)); err != nil {
			return err
		}

		// Add constituent tokens if it's a phrase
		tokens := tokenize(w.Word)
		if len(tokens) > 1 {
			for _, token := range tokens {
				normToken := filter.NormalizeText(token)
				hashedToken := hash.HashTextSha256(normToken)
				if err := tm.bloomFilter.AddWithCtx(ctx, []byte(hashedToken)); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// AddWord adds a single word to the filters.
func (tm *TextModerator) AddWord(ctx context.Context, word BadWord) error {
	// Add full phrase
	normalizedWord := filter.NormalizeText(word.Word)
	hashedWord := hash.HashTextSha256(normalizedWord)
	if err := tm.bloomFilter.AddWithCtx(ctx, []byte(hashedWord)); err != nil {
		return err
	}

	// Add constituent tokens if it's a phrase
	tokens := tokenize(word.Word)
	if len(tokens) > 1 {
		for _, token := range tokens {
			normToken := filter.NormalizeText(token)
			hashedToken := hash.HashTextSha256(normToken)
			if err := tm.bloomFilter.AddWithCtx(ctx, []byte(hashedToken)); err != nil {
				return err
			}
		}
	}
	return nil
}

// Moderate checks text content for bad words with NSFW model fallback.
func (tm *TextModerator) Moderate(ctx context.Context, text string) (*TextModerationResult, error) {
	result := &TextModerationResult{
		IsClean:         true,
		Matches:         make([]filter.AhoCorasickMatch, 0),
		Categories:      make([]string, 0),
		DetectedPhrases: make([]string, 0),
	}

	if text == "" {
		return result, nil
	}
	// step 1: normalize text
	normalizedText := filter.NormalizeText(text)
	// step 2: Bloom Fast Prefilter (optional)

	hasPotentialMatch := false
	if tm.bloomFilter != nil {
		hashed := hash.FastHash(normalizedText)
		exists, err := tm.bloomFilter.ExistsWithCtx(ctx, hashed)
		if err != nil {
			return nil, err
		}
		hasPotentialMatch = exists
	}
	// step 3: Pattern Matching (ALWAYS RUN)
	matches := tm.ahoCorasick.Search(normalizedText)

	if len(matches) > 0 {
		result.IsClean = false
		result.Matches = matches

		categorySet := make(map[string]struct{})

		for _, m := range matches {
			if m.NsfwScore > result.MaxNsfwScore {
				result.MaxNsfwScore = m.NsfwScore
			}
			categorySet[m.Category] = struct{}{}
		}

		for c := range categorySet {
			result.Categories = append(result.Categories, c)
		}
	}

	needModel := true
	if result.MaxNsfwScore >= tm.rejectThreshold {
		needModel = false
	}

	if tm.nsfwTextClient != nil && needModel {

		resp, err := tm.nsfwTextClient.Predict(ctx, text)
		if err == nil {

			result.NSFWChecked = true

			if resp.IsNsfw {

				result.IsClean = false
				result.DetectedByNSFW = true

				result.Categories = mergeCategories(result.Categories, resp.Categories)

				switch strings.ToLower(resp.SafetyLabel) {
				case "unsafe":
					result.ShouldReject = true
				case "controversial":
					result.ShouldReview = true
				}

				result.DetectedPhrases = append(result.DetectedPhrases, text)

				tm.enqueueFeedback(text, resp.Categories)
			}
		}
	}
	// step 4: Final decision from pattern severity
	if result.MaxNsfwScore >= tm.rejectThreshold {
		result.ShouldReject = true
	} else if result.MaxNsfwScore >= tm.reviewThreshold {
		result.ShouldReview = true
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
