package moderator

import (
	"bytes"
	"context"
	"encoding/binary"
	"io"
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
	CacheHit     bool    // Whether result came from cache (Bloom filter hit)
}

// ImageModerator interface for image content moderation.
type ImageModerator interface {
	// ModerateImage moderates an image from a reader.
	ModerateImage(ctx context.Context, reader io.Reader) (*ImageModerationResult, error)

	// ModerateImageURL moderates an image from a URL.
	ModerateImageURL(ctx context.Context, url string) (*ImageModerationResult, error)
}

// ImageModeratorConfig holds configuration for image moderation.
type ImageModeratorConfig struct {
	NSFWThreshold     float64       // Threshold for NSFW detection (0-1)
	ViolenceThreshold float64       // Threshold for violence detection (0-1)
	EnableOCR         bool          // Enable OCR for text-in-image detection
	Timeout           time.Duration // Request timeout
	BloomBits         uint          // Bloom filter size in bits
	BloomHashFuncs    uint          // Number of hash functions for Bloom filter
	BloomKey          string        // Redis key for image Bloom filter
}

// DefaultImageModeratorConfig returns default configuration.
func DefaultImageModeratorConfig() ImageModeratorConfig {
	return ImageModeratorConfig{
		NSFWThreshold:     0.7,
		ViolenceThreshold: 0.8,
		EnableOCR:         true,
		Timeout:           10 * time.Second,
		BloomBits:         1 << 20, // ~1M bits = 128KB
		BloomHashFuncs:    7,
		BloomKey:          "moderation:image:bloom",
	}
}

// BadImageChecker is an interface for checking/storing bad images.
type BadImageChecker interface {
	// FindByPHash checks if a pHash exists in the database.
	FindByPHash(ctx context.Context, phash int64) (bool, error)
	// SaveBadImage saves a bad image pHash to the database.
	SaveBadImage(ctx context.Context, phash int64, category string, nsfwScore float64, sourceURL string) error
}

// LocalImageModerator implements multi-layer image moderation with pHash + Bloom filter.
type LocalImageModerator struct {
	config          ImageModeratorConfig
	textModerator   *TextModerator
	bloomFilter     *bloom.Filter
	hasher          *hash.PerceptualHasher
	nsfwClient      *nsfw.Client
	badImageChecker BadImageChecker
	log             *log.Helper
}

// NewLocalImageModerator creates a new LocalImageModerator with multi-layer detection.
func NewLocalImageModerator(
	config ImageModeratorConfig,
	textMod *TextModerator,
	redisCache redis.Cache,
	nsfwClient *nsfw.Client,
	badImageChecker BadImageChecker,
	logger log.Logger,
) *LocalImageModerator {
	return &LocalImageModerator{
		config:          config,
		textModerator:   textMod,
		bloomFilter:     bloom.NewBloomFilter(redisCache, config.BloomKey, config.BloomBits, config.BloomHashFuncs),
		hasher:          hash.NewPerceptualHasher(),
		nsfwClient:      nsfwClient,
		badImageChecker: badImageChecker,
		log:             log.NewHelper(logger),
	}
}

// ModerateImage moderates an image from a reader using multi-layer approach:
// 1. Generate pHash
// 2. Check Bloom filter
// 3. If Bloom hit -> DB lookup to confirm
// 4. If not cached -> NSFW detector
// 5. If NSFW -> Save to DB + Bloom
func (m *LocalImageModerator) ModerateImage(ctx context.Context, reader io.Reader) (*ImageModerationResult, error) {
	// Read image bytes
	imageData, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	// Step 1: Generate pHash
	imgHash, err := m.hasher.ComputeHashFromBytes(imageData, hash.PHash)
	if err != nil {
		m.log.Warnf("Failed to compute pHash: %v, falling back to NSFW detection", err)
		// Fallback to direct NSFW detection
		return m.detectNSFW(ctx, imageData, 0, "")
	}

	phash := imgHash.Hash
	m.log.Debugf("Image pHash: %016x", phash)

	// Step 2: Check Bloom filter
	phashBytes := m.phashToBytes(phash)
	maybeExists, err := m.bloomFilter.ExistsWithCtx(ctx, phashBytes)
	if err != nil {
		m.log.Warnf("Bloom filter check failed: %v, continuing with NSFW detection", err)
	}

	if maybeExists {
		m.log.Debugf("Bloom filter hit for pHash %016x, checking DB", phash)

		// Step 3: DB lookup to confirm
		exists, err := m.badImageChecker.FindByPHash(ctx, int64(phash))
		if err != nil {
			m.log.Warnf("DB lookup failed: %v", err)
		}

		if exists {
			m.log.Infof("Cached bad image detected: pHash=%016x", phash)
			return &ImageModerationResult{
				IsClean:      false,
				ShouldReject: true,
				Categories:   map[ImageCategory]float64{ImageCategoryNSFW: 1.0},
				PHash:        phash,
				CacheHit:     true,
			}, nil
		}
		// False positive from Bloom filter, continue to NSFW detection
	}

	// Step 4: NSFW detection
	return m.detectNSFW(ctx, imageData, phash, "")
}

// ModerateImageURL moderates an image from a URL.
func (m *LocalImageModerator) ModerateImageURL(ctx context.Context, url string) (*ImageModerationResult, error) {
	// Step 1: Generate pHash from URL
	imgHash, err := m.hasher.ComputeHashFromURL(ctx, url, hash.PHash)
	if err != nil {
		m.log.Warnf("Failed to compute pHash from URL: %v, using direct detection", err)
		return m.detectNSFWFromURL(ctx, url, 0)
	}

	phash := imgHash.Hash
	m.log.Debugf("Image pHash: %016x for URL: %s", phash, url)

	// Step 2: Check Bloom filter
	phashBytes := m.phashToBytes(phash)
	maybeExists, err := m.bloomFilter.ExistsWithCtx(ctx, phashBytes)
	if err != nil {
		m.log.Warnf("Bloom filter check failed: %v", err)
	}

	if maybeExists {
		m.log.Debugf("Bloom filter hit for pHash %016x", phash)

		// Step 3: DB lookup
		exists, err := m.badImageChecker.FindByPHash(ctx, int64(phash))
		if err != nil {
			m.log.Warnf("DB lookup failed: %v", err)
		}

		if exists {
			m.log.Infof("Cached bad image detected: pHash=%016x", phash)
			return &ImageModerationResult{
				IsClean:      false,
				ShouldReject: true,
				Categories:   map[ImageCategory]float64{ImageCategoryNSFW: 1.0},
				PHash:        phash,
				CacheHit:     true,
			}, nil
		}
	}

	// Step 4: NSFW detection from URL
	return m.detectNSFWFromURL(ctx, url, phash)
}

// detectNSFW performs NSFW detection and updates cache if NSFW detected.
func (m *LocalImageModerator) detectNSFW(ctx context.Context, imageData []byte, phash uint64, sourceURL string) (*ImageModerationResult, error) {
	result := &ImageModerationResult{
		IsClean:    true,
		Categories: make(map[ImageCategory]float64),
		PHash:      phash,
	}

	if m.nsfwClient == nil {
		m.log.Warn("NSFW client not configured, skipping NSFW detection")
		return result, nil
	}

	// Call NSFW detector
	nsfwResult, err := m.nsfwClient.DetectFromBytes(ctx, imageData)
	if err != nil {
		m.log.Warnf("NSFW detection failed: %v", err)
		return result, nil // Return clean on error to avoid blocking
	}

	result.NSFWScore = nsfwResult.NSFWScore
	result.Categories[ImageCategoryNSFW] = nsfwResult.NSFWScore

	if nsfwResult.IsNSFW || nsfwResult.NSFWScore >= m.config.NSFWThreshold {
		result.IsClean = false
		result.ShouldReject = true

		// Step 5: Save to DB and Bloom filter for future fast detection
		if phash != 0 {
			if err := m.saveBadImage(ctx, phash, "nsfw", nsfwResult.NSFWScore, sourceURL); err != nil {
				m.log.Warnf("Failed to save bad image: %v", err)
			}
		}
	}

	return result, nil
}

// detectNSFWFromURL performs NSFW detection from URL.
func (m *LocalImageModerator) detectNSFWFromURL(ctx context.Context, url string, phash uint64) (*ImageModerationResult, error) {
	result := &ImageModerationResult{
		IsClean:    true,
		Categories: make(map[ImageCategory]float64),
		PHash:      phash,
	}

	if m.nsfwClient == nil {
		m.log.Warn("NSFW client not configured, skipping NSFW detection")
		return result, nil
	}

	nsfwResult, err := m.nsfwClient.DetectFromURL(ctx, url)
	if err != nil {
		m.log.Warnf("NSFW detection failed: %v", err)
		return result, nil
	}

	result.NSFWScore = nsfwResult.NSFWScore
	result.Categories[ImageCategoryNSFW] = nsfwResult.NSFWScore

	if nsfwResult.IsNSFW || nsfwResult.NSFWScore >= m.config.NSFWThreshold {
		result.IsClean = false
		result.ShouldReject = true

		if phash != 0 {
			if err := m.saveBadImage(ctx, phash, "nsfw", nsfwResult.NSFWScore, url); err != nil {
				m.log.Warnf("Failed to save bad image: %v", err)
			}
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

// ModerateImageBytes is a convenience method for moderating image bytes.
func (m *LocalImageModerator) ModerateImageBytes(ctx context.Context, imageData []byte) (*ImageModerationResult, error) {
	return m.ModerateImage(ctx, bytes.NewReader(imageData))
}
