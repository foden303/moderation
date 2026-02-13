package moderator

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"moderation/internal/pkg/bloom"
	"moderation/internal/pkg/hash"
	"moderation/internal/pkg/nsfw"
	"moderation/internal/pkg/redis"

	"github.com/go-kratos/kratos/v2/log"
)

// ImageCategory represents the category of detected content in an image.
type ImageCategory string

const (
	ImageCategoryNSFW     ImageCategory = "nsfw"
	ImageCategoryViolence ImageCategory = "violence"
	ImageCategoryHate     ImageCategory = "hate"
	ImageCategoryText     ImageCategory = "text" // Text detected in image
)

// ImageModerationResult represents the result of image moderation.
type ImageModerationResult struct {
	IsClean      bool
	Categories   map[ImageCategory]float64 // Category -> confidence score
	DetectedText string                    // Text detected via OCR (if applicable)
	TextResult   *TextModerationResult     // Result of text moderation on detected text
	ShouldReject bool
	ShouldReview bool
	PHash        uint64  // Perceptual hash of the image
	NSFWScore    float64 // NSFW detection score
	CacheHit     bool    // Whether result came from cache
}

// ImageModerator interface for image content moderation.
type ImageModerator interface {
	// ModerateImageURL moderates an image from a URL.
	ModerateImageURL(ctx context.Context, ownerID, url string, fileHash *string) (*ImageModerationResult, error)
	// ModerateImageURLs moderates multiple images from URLs.
	ModerateImageURLs(ctx context.Context, urls []string) ([]*ImageModerationResult, error)
}

// ImageModeratorConfig holds configuration for image moderation.
type ImageModeratorConfig struct {
	Workers           int           // Number of workers for parallel processing
	NSFWThreshold     float64       // Threshold for NSFW detection (0-1)
	ViolenceThreshold float64       // Threshold for violence detection (0-1)
	PHashMaxDistance  int32         // Max Hamming distance for pHash similarity (0=exact, 10=similar)
	EnableOCR         bool          // Enable OCR for text-in-image detection
	Timeout           time.Duration // Request timeout
	BloomBits         uint          // Bloom filter size in bits
	BloomHashFuncs    uint          // Number of hash functions for Bloom filter
	BloomKey          string        // Redis key for image Bloom filter
	CacheKeyPrefix    string        // Redis key prefix for image cache
	CacheTTL          time.Duration // TTL for Redis image cache
}

// DefaultImageModeratorConfig returns default configuration.
func DefaultImageModeratorConfig() ImageModeratorConfig {
	return ImageModeratorConfig{
		Workers:           4,
		NSFWThreshold:     0.7,
		ViolenceThreshold: 0.8,
		PHashMaxDistance:  10,
		EnableOCR:         false,
		Timeout:           10 * time.Second,
		BloomBits:         1 << 20, // ~1M bits = 128KB
		BloomHashFuncs:    7,
		BloomKey:          "moderation:bloom:image",
		CacheKeyPrefix:    "moderation:image:",
		CacheTTL:          24 * time.Hour,
	}
}

// ImageCacheResult represents a cached image moderation result from DB.
type ImageCacheResult struct {
	Category  string
	NSFWScore float64
	PHash     int64
}

// BadImageChecker is an interface for checking/storing bad images.
type BadImageChecker interface {
	// FindByPHash searches for similar images within Hamming distance threshold.
	// Returns the closest match or nil if no similar image found.
	FindByPHash(ctx context.Context, phash int64, maxDistance int32) (*ImageCacheResult, error)
	// FindByFileHash looks up a cached result by file hash (SHA256).
	FindByFileHash(ctx context.Context, fileHash string) (*ImageCacheResult, error)
	// SaveBadImage saves a bad image pHash to the database.
	SaveBadImage(ctx context.Context, phash int64, category string, nsfwScore float64, sourceURL string) error
	// SaveImageResult saves any image result (safe or unsafe) to the database.
	SaveImageResult(ctx context.Context, fileHash string, phash int64, category string, nsfwScore float64, sourceURL string) error
}

// imageCacheEntry is the JSON structure stored in Redis cache.
type imageCacheEntry struct {
	Category  string  `json:"category"`
	NSFWScore float64 `json:"nsfw_score"`
	PHash     int64   `json:"phash"`
	IsClean   bool    `json:"is_clean"`
}

// LocalImageModerator implements multi-layer image moderation with pHash + Bloom filter.
type LocalImageModerator struct {
	config          ImageModeratorConfig
	textModerator   *TextModerator
	bloomFilter     *bloom.Filter
	hasher          *hash.PerceptualHasher
	sha256Hasher    *hash.Sha256Hasher
	nsfwClient      *nsfw.ImageClient
	badImageChecker BadImageChecker
	redisCache      redis.Cache
	log             *log.Helper
}

// NewLocalImageModerator creates a new LocalImageModerator with multi-layer detection.
func NewLocalImageModerator(
	config ImageModeratorConfig,
	textMod *TextModerator,
	redisCache redis.Cache,
	nsfwClient *nsfw.ImageClient,
	badImageChecker BadImageChecker,
	logger log.Logger,
) *LocalImageModerator {
	return &LocalImageModerator{
		config:          config,
		textModerator:   textMod,
		bloomFilter:     bloom.NewBloomFilter(redisCache, config.BloomKey, config.BloomBits, config.BloomHashFuncs),
		hasher:          hash.NewPerceptualHasher(),
		sha256Hasher:    hash.NewSha256Hasher(),
		nsfwClient:      nsfwClient,
		badImageChecker: badImageChecker,
		redisCache:      redisCache,
		log:             log.NewHelper(logger),
	}
}

// ModerateImageURL moderates an image from a URL.
// Optimized flow:
//   - If fileHash provided: check cache BEFORE downloading (zero HTTP on cache hit)
//   - If fileHash not provided: download once, compute SHA256 + pHash from same bytes
//   - NSFW detection uses raw bytes (no re-download by NSFW service)
func (m *LocalImageModerator) ModerateImageURL(
	ctx context.Context,
	ownerID, url string,
	fileHash *string,
) (*ImageModerationResult, error) {

	// step 1. Fast cache check when fileHash provided
	if fHash := safeString(fileHash); fHash != "" {
		if res := m.tryCache(ctx, fHash); res != nil {
			return res, nil
		}
	}

	// step 2. Download image once
	imgData, err := m.hasher.DownloadImage(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("failed to download image: %w", err)
	}

	// step 3. Resolve SHA256 fileHash
	fHash := hash.ResolveFileHash(imgData, fileHash)

	// step 4. Re-check cache after computing hash
	if res := m.tryCache(ctx, fHash); res != nil {
		return res, nil
	}

	// step 5. Compute pHash
	imgHash, err := m.hasher.ComputeHashFromBytes(imgData, hash.PHash)
	if err != nil {
		m.log.Warnf("Failed to compute pHash: %v", err)
		return nil, err
	}

	phash := imgHash.Hash
	m.log.Debugf("Image pHash: %016x for URL: %s", phash, url)

	// step 6. Check bloom + DB for known bad image
	if res := m.checkKnownBadImage(ctx, phash, fHash, url); res != nil {
		return res, nil
	}

	// step 7. Run AI detection
	return m.detectNSFWFromBytes(ctx, imgData, url, phash, fHash)
}

// checkKnownBadImage checks if a similar pHash exists in the database using Hamming distance.
func (m *LocalImageModerator) checkKnownBadImage(
	ctx context.Context,
	phash uint64,
	fHash string,
	url string,
) *ImageModerationResult {

	phashBytes := m.phashToBytes(phash)

	maybeExists, err := m.bloomFilter.ExistsWithCtx(ctx, phashBytes)
	if err != nil {
		m.log.Warnf("Bloom filter check failed: %v", err)
		return nil
	}

	if !maybeExists {
		return nil
	}

	m.log.Debugf("Bloom filter hit for pHash %016x", phash)

	match, err := m.badImageChecker.FindByPHash(ctx, int64(phash), m.config.PHashMaxDistance)
	if err != nil {
		m.log.Warnf("DB similarity lookup by pHash failed: %v", err)
		return nil
	}

	if match == nil {
		return nil
	}

	m.log.Infof("Similar bad image detected: pHash=%016x, matched category=%s, nsfw=%.2f", phash, match.Category, match.NSFWScore)

	result := &ImageModerationResult{
		IsClean:      false,
		ShouldReject: true,
		Categories:   map[ImageCategory]float64{ImageCategoryNSFW: match.NSFWScore},
		PHash:        phash,
		CacheHit:     true,
	}
	m.saveImageResult(ctx, fHash, int64(phash), match.Category, match.NSFWScore, url)
	return result
}

// safeString returns the string value of a pointer, or empty string if nil.
func safeString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func (m *LocalImageModerator) tryCache(
	ctx context.Context,
	fHash string,
) *ImageModerationResult {
	// Redis
	if cached, err := m.getFromRedisCache(ctx, fHash); err == nil && cached != nil {
		m.log.Debugf("Redis cache hit for fileHash: %s", fHash)
		return m.cacheEntryToResult(cached, true)
	}
	// DB
	if dbResult, err := m.badImageChecker.FindByFileHash(ctx, fHash); err == nil && dbResult != nil {
		m.log.Debugf("DB cache hit for fileHash: %s", fHash)

		m.setRedisCache(ctx, fHash, dbResult)
		return m.dbCacheToResult(dbResult)
	}
	return nil
}

// detectNSFWFromBytes performs NSFW detection from raw image bytes and saves result.
func (m *LocalImageModerator) detectNSFWFromBytes(ctx context.Context, imgData []byte, url string, phash uint64, fileHash string) (*ImageModerationResult, error) {
	result := &ImageModerationResult{
		IsClean:    true,
		Categories: make(map[ImageCategory]float64),
		PHash:      phash,
	}

	if m.nsfwClient == nil {
		m.log.Warn("NSFW client not configured, skipping NSFW detection")
		if fileHash != "" {
			m.saveImageResult(ctx, fileHash, int64(phash), "safe", 0, url)
		}
		return result, nil
	}

	// Send raw bytes to NSFW service (no re-download by the service)
	nsfwResult, err := m.nsfwClient.Predict(ctx, imgData)
	if err != nil {
		m.log.Warnf("NSFW detection failed: %v", err)
		return result, nil
	}

	result.NSFWScore = nsfwResult.NsfwScore
	result.Categories[ImageCategoryNSFW] = nsfwResult.NsfwScore

	if nsfwResult.IsNsfw || nsfwResult.NsfwScore >= m.config.NSFWThreshold {
		result.IsClean = false
		result.ShouldReject = true

		if phash != 0 {
			if err := m.saveBadImage(ctx, phash, "nsfw", nsfwResult.NsfwScore, url); err != nil {
				m.log.Warnf("Failed to save bad image: %v", err)
			}
		}
		if fileHash != "" {
			m.saveImageResult(ctx, fileHash, int64(phash), "unsafe", nsfwResult.NsfwScore, url)
		}
	} else {
		if fileHash != "" {
			m.saveImageResult(ctx, fileHash, int64(phash), "safe", nsfwResult.NsfwScore, url)
		}
	}

	return result, nil
}

// ModerateImageURLs moderates multiple image URLs using worker pool.
func (m *LocalImageModerator) ModerateImageURLs(ctx context.Context, ownerID string, urls []string, fileHashes []*string) ([]*ImageModerationResult, error) {
	workerCount := m.config.Workers
	if workerCount <= 0 {
		workerCount = 4 // fallback default
	}
	type job struct {
		index    int
		url      string
		fileHash *string
	}
	jobs := make(chan job)
	results := make([]*ImageModerationResult, len(urls))
	var wg sync.WaitGroup
	worker := func() {
		defer wg.Done()
		for j := range jobs {
			// stop if context is cancelled
			select {
			case <-ctx.Done():
				return
			default:
			}
			res, err := m.ModerateImageURL(ctx, ownerID, j.url, j.fileHash)
			if err != nil {
				m.log.Warnf("Failed to moderate image URL %s: %v", j.url, err)
				continue
			}
			results[j.index] = res
		}
	}
	// Start workers
	wg.Add(workerCount)
	for i := 0; i < workerCount; i++ {
		go worker()
	}
	// Push jobs
	for i, url := range urls {
		var fh *string
		if fileHashes != nil && i < len(fileHashes) {
			fh = fileHashes[i]
		}
		jobs <- job{
			index:    i,
			url:      url,
			fileHash: fh,
		}
	}
	close(jobs)
	wg.Wait()
	return results, nil
}

// detectNSFWFromURL performs NSFW detection from URL and saves result.
func (m *LocalImageModerator) detectNSFWFromURL(ctx context.Context, url string, phash uint64, fileHash string) (*ImageModerationResult, error) {
	result := &ImageModerationResult{
		IsClean:    true,
		Categories: make(map[ImageCategory]float64),
		PHash:      phash,
	}

	if m.nsfwClient == nil {
		m.log.Warn("NSFW client not configured, skipping NSFW detection")
		// Save as safe
		if fileHash != "" {
			m.saveImageResult(ctx, fileHash, int64(phash), "safe", 0, url)
		}
		return result, nil
	}

	nsfwResult, err := m.nsfwClient.PredictFromURL(ctx, url)
	if err != nil {
		m.log.Warnf("NSFW detection failed: %v", err)
		return result, nil
	}

	result.NSFWScore = nsfwResult.NsfwScore
	result.Categories[ImageCategoryNSFW] = nsfwResult.NsfwScore

	if nsfwResult.IsNsfw || nsfwResult.NsfwScore >= m.config.NSFWThreshold {
		result.IsClean = false
		result.ShouldReject = true

		if phash != 0 {
			if err := m.saveBadImage(ctx, phash, "nsfw", nsfwResult.NsfwScore, url); err != nil {
				m.log.Warnf("Failed to save bad image: %v", err)
			}
		}
		// Save unsafe result to DB + Redis
		if fileHash != "" {
			m.saveImageResult(ctx, fileHash, int64(phash), "unsafe", nsfwResult.NsfwScore, url)
		}
	} else {
		// Save safe result to DB + Redis
		if fileHash != "" {
			m.saveImageResult(ctx, fileHash, int64(phash), "safe", nsfwResult.NsfwScore, url)
		}
	}

	return result, nil
}

// saveBadImage saves a bad image to DB and updates Bloom filter.
func (m *LocalImageModerator) saveBadImage(ctx context.Context, phash uint64, category string, nsfwScore float64, sourceURL string) error {
	// Save to DB
	if err := m.badImageChecker.SaveBadImage(ctx, int64(phash), category, nsfwScore, sourceURL); err != nil {
		return err
	}

	// Add to Bloom filter
	phashBytes := m.phashToBytes(phash)
	if err := m.bloomFilter.AddWithCtx(ctx, phashBytes); err != nil {
		m.log.Warnf("Failed to add pHash to Bloom filter: %v", err)
	}

	m.log.Infof("Saved bad image: pHash=%016x, category=%s, score=%.2f", phash, category, nsfwScore)
	return nil
}

// saveImageResult saves any image result (safe or unsafe) to DB + Redis cache.
func (m *LocalImageModerator) saveImageResult(ctx context.Context, fileHash string, phash int64, category string, nsfwScore float64, sourceURL string) {
	// Save to DB
	if err := m.badImageChecker.SaveImageResult(ctx, fileHash, phash, category, nsfwScore, sourceURL); err != nil {
		m.log.Warnf("Failed to save image result to DB: %v", err)
	}

	// Save to Redis cache
	m.setRedisCache(ctx, fileHash, &ImageCacheResult{
		Category:  category,
		NSFWScore: nsfwScore,
		PHash:     phash,
	})
}

// Redis cache helpers

func (m *LocalImageModerator) redisCacheKey(fileHash string) string {
	return m.config.CacheKeyPrefix + fileHash
}

func (m *LocalImageModerator) getFromRedisCache(ctx context.Context, fileHash string) (*imageCacheEntry, error) {
	data, err := m.redisCache.GetString(ctx, m.redisCacheKey(fileHash))
	if err != nil {
		return nil, err
	}

	var entry imageCacheEntry
	if err := json.Unmarshal([]byte(data), &entry); err != nil {
		return nil, err
	}
	return &entry, nil
}

func (m *LocalImageModerator) setRedisCache(ctx context.Context, fileHash string, result *ImageCacheResult) {
	entry := imageCacheEntry{
		Category:  result.Category,
		NSFWScore: result.NSFWScore,
		PHash:     result.PHash,
		IsClean:   result.Category == "safe",
	}
	data, err := json.Marshal(entry)
	if err != nil {
		m.log.Warnf("Failed to marshal image cache entry: %v", err)
		return
	}
	if err := m.redisCache.SetString(ctx, m.redisCacheKey(fileHash), string(data), m.config.CacheTTL); err != nil {
		m.log.Warnf("Failed to set Redis image cache: %v", err)
	}
}

func (m *LocalImageModerator) cacheEntryToResult(entry *imageCacheEntry, cacheHit bool) *ImageModerationResult {
	result := &ImageModerationResult{
		IsClean:    entry.IsClean,
		Categories: map[ImageCategory]float64{ImageCategoryNSFW: entry.NSFWScore},
		PHash:      uint64(entry.PHash),
		NSFWScore:  entry.NSFWScore,
		CacheHit:   cacheHit,
	}
	if !entry.IsClean {
		result.ShouldReject = true
	}
	return result
}

func (m *LocalImageModerator) dbCacheToResult(dbResult *ImageCacheResult) *ImageModerationResult {
	isClean := dbResult.Category == "safe"
	result := &ImageModerationResult{
		IsClean:    isClean,
		Categories: map[ImageCategory]float64{ImageCategoryNSFW: dbResult.NSFWScore},
		PHash:      uint64(dbResult.PHash),
		NSFWScore:  dbResult.NSFWScore,
		CacheHit:   true,
	}
	if !isClean {
		result.ShouldReject = true
	}
	return result
}

// AddPHashToBloom adds a pHash directly to the Bloom filter (for rebuilding).
func (m *LocalImageModerator) AddPHashToBloom(ctx context.Context, phash uint64) error {
	phashBytes := m.phashToBytes(phash)
	return m.bloomFilter.AddWithCtx(ctx, phashBytes)
}

// phashToBytes converts a uint64 pHash to bytes for Bloom filter.
func (m *LocalImageModerator) phashToBytes(phash uint64) []byte {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, phash)
	return buf
}

// RebuildBloomFilter rebuilds the Bloom filter from all bad images.
func (m *LocalImageModerator) RebuildBloomFilter(ctx context.Context, phashes []uint64) error {
	for _, phash := range phashes {
		if err := m.AddPHashToBloom(ctx, phash); err != nil {
			m.log.Warnf("Failed to add pHash %016x to Bloom: %v", phash, err)
		}
	}
	m.log.Infof("Rebuilt image Bloom filter with %d pHashes", len(phashes))
	return nil
}

// InvalidateCache removes a fileHash from Redis cache.
func (m *LocalImageModerator) InvalidateCache(ctx context.Context, fileHash string) error {
	_, err := m.redisCache.Del(ctx, m.redisCacheKey(fileHash))
	return err
}

// GetCacheKey returns the Redis cache key for a fileHash (for debugging).
func (m *LocalImageModerator) GetCacheKey(fileHash string) string {
	return fmt.Sprintf("%s%s", m.config.CacheKeyPrefix, fileHash)
}
