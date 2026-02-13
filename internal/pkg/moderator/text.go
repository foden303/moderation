package moderator

import (
	"context"
	"encoding/hex"
	"encoding/json"
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
	CacheHit        bool     // Whether result came from cache
}

// FeedbackCallback is called when NSFW model detects new bad content.
// This allows saving to DB and updating filters.
type FeedbackCallback func(ctx context.Context, phrase string, categories []string) error

// TextCacheResult represents a cached text moderation result from DB.
type TextCacheResult struct {
	Category  string
	NSFWScore float64
}

// TextCacheChecker is an interface for checking/storing text cache results.
type TextCacheChecker interface {
	// FindByContentHash looks up a cached result by content hash (hex encoded).
	FindByContentHash(ctx context.Context, contentHash string) (*TextCacheResult, error)
	// SaveTextResult saves a text moderation result to the database.
	SaveTextResult(ctx context.Context, contentHash, normalizedContent, category string, nsfwScore float64, detectResult []byte) error
}

// textCacheEntry is the JSON structure stored in Redis cache.
type textCacheEntry struct {
	Category     string   `json:"category"`
	NSFWScore    float64  `json:"nsfw_score"`
	IsClean      bool     `json:"is_clean"`
	ShouldReject bool     `json:"should_reject"`
	ShouldReview bool     `json:"should_review"`
	Categories   []string `json:"categories,omitempty"`
}

// TextModerator provides text content moderation.
type TextModerator struct {
	bloomFilter      *bloom.Filter
	ahoCorasick      *filter.AhoCorasick
	nsfwTextClient   *nsfw.TextClient
	textCacheChecker TextCacheChecker
	redisCache       redis.Cache
	mu               sync.RWMutex

	// Configuration
	rejectThreshold float64 // Severity threshold for auto-reject
	reviewThreshold float64 // Severity threshold for manual review
	cacheKeyPrefix  string
	cacheTTL        time.Duration

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
	CacheKeyPrefix     string
	CacheTTL           time.Duration
}

// DefaultTextModeratorConfig returns default configuration.
func DefaultTextModeratorConfig() TextModeratorConfig {
	return TextModeratorConfig{
		BloomBits:          1024 * 1024 * 8, // 8 million bits = 1MB
		BloomHashFunctions: 5,
		BloomKey:           "moderation:bloom:badwords",
		RejectThreshold:    0.85,
		ReviewThreshold:    0.5,
		CacheKeyPrefix:     "moderation:text:",
		CacheTTL:           24 * time.Hour,
	}
}

// NewTextModerator creates a new TextModerator.
// nsfwTextClient can be nil if text NSFW detection is disabled.
// textCacheChecker can be nil if text caching is disabled.
func NewTextModerator(redisCache redis.Cache, config TextModeratorConfig, nsfwTextClient *nsfw.TextClient, textCacheChecker TextCacheChecker) *TextModerator {
	return &TextModerator{
		bloomFilter:      bloom.NewBloomFilter(redisCache, config.BloomKey, config.BloomBits, config.BloomHashFunctions),
		ahoCorasick:      filter.NewAhoCorasick(),
		nsfwTextClient:   nsfwTextClient,
		textCacheChecker: textCacheChecker,
		redisCache:       redisCache,
		rejectThreshold:  config.RejectThreshold,
		reviewThreshold:  config.ReviewThreshold,
		cacheKeyPrefix:   config.CacheKeyPrefix,
		cacheTTL:         config.CacheTTL,
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
// Flow:
// 1. Compute SHA256 contentHash
// 2. Check Redis cache by contentHash
// 3. Check DB by contentHash
// 4. Bloom filter + Aho-Corasick pattern matching
// 5. NSFW AI model (if needed)
// 6. Save result to DB + Redis
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

	// Step 1: Compute contentHash
	normalizedText := filter.NormalizeText(text)
	contentHash := hash.FastHash(normalizedText)

	// Convert contentHash to hex string for keys and DB lookups
	contentHashHex := hex.EncodeToString(contentHash)

	// Step 2: Check Redis cache by contentHash
	if cached, err := tm.getFromRedisCache(ctx, contentHashHex); err == nil && cached != nil {
		return tm.cacheEntryToResult(cached), nil
	}

	// Step 3: Check DB by contentHash
	if tm.textCacheChecker != nil {
		if dbResult, err := tm.textCacheChecker.FindByContentHash(ctx, contentHashHex); err == nil && dbResult != nil {
			entry := tm.dbCacheToEntry(dbResult)
			// Populate Redis cache for next time
			tm.setRedisCache(ctx, contentHashHex, entry)
			return tm.cacheEntryToResult(entry), nil
		}
	}

	// Step 4: Bloom filter precheck + Aho-Corasick pattern matching
	hasPotentialMatch := false
	if tm.bloomFilter != nil {
		hashed := hash.FastHash(normalizedText)
		exists, err := tm.bloomFilter.ExistsWithCtx(ctx, hashed)
		if err == nil {
			hasPotentialMatch = exists
		}
	}

	// Also check individual tokens against bloom
	if !hasPotentialMatch {
		tokens := tokenize(text)
		for _, token := range tokens {
			normToken := filter.NormalizeText(token)
			hashedToken := hash.HashTextSha256(normToken)
			exists, err := tm.bloomFilter.ExistsWithCtx(ctx, []byte(hashedToken))
			if err == nil && exists {
				hasPotentialMatch = true
				break
			}
		}
	}

	// Aho-Corasick pattern matching (ALWAYS RUN if bloom hit)
	if hasPotentialMatch {
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
	}

	// Step 5: NSFW AI model (if pattern score not high enough to auto-reject)
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

				// Feedback for learning
				tm.enqueueFeedback(text, resp.Categories)
			}
		}
	}

	// Step 6: Final decision from pattern severity
	if result.MaxNsfwScore >= tm.rejectThreshold {
		result.ShouldReject = true
	} else if result.MaxNsfwScore >= tm.reviewThreshold {
		result.ShouldReview = true
	}

	// Step 7: Save result to DB + Redis cache
	category := tm.resultToCategory(result)
	tm.saveTextResult(ctx, contentHashHex, normalizedText, category, result.MaxNsfwScore, result)

	return result, nil
}

// ModerateWithTimeout moderates text with a timeout.
func (tm *TextModerator) ModerateWithTimeout(ctx context.Context, text string, timeout time.Duration) (*TextModerationResult, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	return tm.Moderate(ctx, text)
}

// resultToCategory determines the cache category from a result.
func (tm *TextModerator) resultToCategory(result *TextModerationResult) string {
	if result.ShouldReject {
		return "unsafe"
	}
	if result.ShouldReview {
		return "controversial"
	}
	return "safe"
}

// saveTextResult saves a text result to DB + Redis cache.
func (tm *TextModerator) saveTextResult(ctx context.Context, contentHash, normalizedContent, category string, nsfwScore float64, result *TextModerationResult) {
	// Save to DB
	if tm.textCacheChecker != nil {
		detectResult, _ := json.Marshal(result)
		if err := tm.textCacheChecker.SaveTextResult(ctx, contentHash, normalizedContent, category, nsfwScore, detectResult); err != nil {
			// Log warning but don't fail
			_ = err
		}
	}

	// Save to Redis cache
	entry := &textCacheEntry{
		Category:     category,
		NSFWScore:    nsfwScore,
		IsClean:      result.IsClean,
		ShouldReject: result.ShouldReject,
		ShouldReview: result.ShouldReview,
		Categories:   result.Categories,
	}
	tm.setRedisCache(ctx, contentHash, entry)
}

// mergeCategories merges two category slices, deduplicating.
func mergeCategories(existing, new []string) []string {
	set := make(map[string]struct{})
	for _, c := range existing {
		set[c] = struct{}{}
	}
	for _, c := range new {
		set[c] = struct{}{}
	}
	merged := make([]string, 0, len(set))
	for c := range set {
		merged = append(merged, c)
	}
	return merged
}

// enqueueFeedback triggers the feedback callback asynchronously.
func (tm *TextModerator) enqueueFeedback(text string, categories []string) {
	tm.mu.RLock()
	cb := tm.feedbackCallback
	tm.mu.RUnlock()

	if cb != nil {
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = cb(ctx, text, categories)
		}()
	}
}

// Redis cache helpers

func (tm *TextModerator) redisCacheKey(contentHashHex string) string {
	return tm.cacheKeyPrefix + contentHashHex
}

func (tm *TextModerator) getFromRedisCache(ctx context.Context, contentHashHex string) (*textCacheEntry, error) {
	data, err := tm.redisCache.GetBytes(ctx, tm.redisCacheKey(contentHashHex))
	if err != nil {
		return nil, err
	}
	var entry textCacheEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return nil, err
	}
	return &entry, nil
}

func (tm *TextModerator) setRedisCache(ctx context.Context, contentHashHex string, entry *textCacheEntry) {
	data, err := json.Marshal(entry)
	if err != nil {
		return
	}
	_ = tm.redisCache.SetBytes(ctx, tm.redisCacheKey(contentHashHex), data, tm.cacheTTL)
}

func (tm *TextModerator) cacheEntryToResult(entry *textCacheEntry) *TextModerationResult {
	return &TextModerationResult{
		IsClean:         entry.IsClean,
		MaxNsfwScore:    entry.NSFWScore,
		Categories:      entry.Categories,
		ShouldReject:    entry.ShouldReject,
		ShouldReview:    entry.ShouldReview,
		CacheHit:        true,
		Matches:         make([]filter.AhoCorasickMatch, 0),
		DetectedPhrases: make([]string, 0),
	}
}

func (tm *TextModerator) dbCacheToEntry(dbResult *TextCacheResult) *textCacheEntry {
	isClean := dbResult.Category == "safe"
	shouldReject := dbResult.Category == "unsafe"
	shouldReview := dbResult.Category == "controversial"
	return &textCacheEntry{
		Category:     dbResult.Category,
		NSFWScore:    dbResult.NSFWScore,
		IsClean:      isClean,
		ShouldReject: shouldReject,
		ShouldReview: shouldReview,
	}
}

// InvalidateCache removes a contentHash from Redis cache.
func (tm *TextModerator) InvalidateCache(ctx context.Context, contentHashHex string) error {
	_, err := tm.redisCache.Del(ctx, tm.redisCacheKey(contentHashHex))
	return err
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
