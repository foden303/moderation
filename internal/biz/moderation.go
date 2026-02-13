package biz

import (
	"context"
	"time"

	"moderation/internal/pkg/hash"
	"moderation/internal/pkg/moderator"

	"github.com/go-kratos/kratos/v2/log"
)

// ModerationResult represents the complete moderation result.
type ModerationResult struct {
	RequestID   string
	IsClean     bool
	Action      ModerationAction
	Verdict     Verdict
	Reason      string
	Categories  []string
	Scores      map[string]float64
	ProcessedAt time.Time

	// Detailed results
	TextResult  *moderator.TextModerationResult
	ImageResult *moderator.ImageModerationResult
	VideoResult *moderator.VideoModerationResult
}

// ModerationAction represents the action to take.
type ModerationAction int

const (
	ModerationActionUnspecified ModerationAction = iota
	ModerationActionAutoApprove
	ModerationActionAutoReject
	ModerationActionPendingReview
)

// Verdict represents the moderation verdict.
type Verdict int

const (
	VerdictUnspecified Verdict = iota
	VerdictClean
	VerdictReject
	VerdictReview
)

type TextCacheCategory string

const (
	TextCacheCategorySafe          TextCacheCategory = "safe"
	TextCacheCategoryUnsafe        TextCacheCategory = "unsafe"
	TextCacheCategoryControversial TextCacheCategory = "controversial"
)

func (c TextCacheCategory) String() string {
	return string(c)
}

type ImageCacheCategory string

const (
	ImageCacheCategorySafe   ImageCacheCategory = "safe"
	ImageCacheCategoryUnsafe ImageCacheCategory = "unsafe"
)

func (i ImageCacheCategory) String() string {
	return string(i)
}

type ModelVersion string

const (
	ModelVersionText  ModelVersion = "Qwen/Qwen3Guard-Gen-0.6B"
	ModelVersionImage ModelVersion = "Falconsai/nsfw_image_detection"
)

func (m ModelVersion) String() string {
	return string(m)
}

// ModerationUsecase orchestrates content moderation.
type ModerationUsecase struct {
	textModerator  *moderator.TextModerator
	imageModerator *moderator.LocalImageModerator
	videoModerator *moderator.LocalVideoModerator
	textCache      TextCacheRepo
	imageCache     ImageCacheRepo
	log            *log.Helper
}

// NewModerationUsecase creates a new ModerationUsecase.
func NewModerationUsecase(
	textMod *moderator.TextModerator,
	imgMod *moderator.LocalImageModerator,
	videoMod *moderator.LocalVideoModerator,
	textCache TextCacheRepo,
	imageCache ImageCacheRepo,
	logger log.Logger,
) *ModerationUsecase {
	return &ModerationUsecase{
		textModerator:  textMod,
		imageModerator: imgMod,
		videoModerator: videoMod,
		textCache:      textCache,
		imageCache:     imageCache,
		log:            log.NewHelper(logger),
	}
}

// RebuildFilters rebuilds all moderation filters from database.
func (uc *ModerationUsecase) RebuildFilters(ctx context.Context) (int, error) {
	uc.log.Info("Rebuilding moderation filters from database")

	// Get all bad words from database (TextCache)
	caches, err := uc.textCache.ListAll(ctx)
	if err != nil {
		return 0, err
	}

	// Filter for only unsafe items if cache contains mixed results
	var words []moderator.BadWord
	for _, c := range caches {
		if c.Category != "safe" {
			words = append(words, moderator.BadWord{
				Word:      c.NormalizedContent,
				Category:  c.Category,
				NsfwScore: c.NSFWScore,
			})
		}
	}

	// Rebuild text moderator filters
	if err := uc.textModerator.RebuildFilters(ctx, words); err != nil {
		return 0, err
	}

	uc.log.Infof("Rebuilt moderation filters with %d words", len(words))
	return len(words), nil
}

// AddBadWord adds a new bad word and updates filters.
func (uc *ModerationUsecase) AddBadWord(ctx context.Context, word, category string, nsfwScore float64, addedBy, modelVersion *string) error {
	if nsfwScore < 0 || nsfwScore > 1 {
		nsfwScore = 1.0
	}
	// Add to text moderator bloom filter
	// Map nsfwScore to severity (int32). Assuming user provides compatible values or we trunacate.
	if err := uc.textModerator.AddWord(ctx, moderator.BadWord{
		Word:      word,
		Category:  category,
		NsfwScore: nsfwScore,
	}); err != nil {
		return err
	}

	// Add to TextCache (permanent storage via nil expiry)
	return uc.textCache.Upsert(ctx, &TextCache{
		ContentHash:       hash.HashTextSha256(word),
		NormalizedContent: word,
		Category:          category,
		NSFWScore:         nsfwScore,
		ModelVersion:      uc.defaultModelVersion(modelVersion),
		AddedBy:           uc.defaultAddedBy(addedBy),
		ExpiresAt:         nil, // Permanent
		DetectResult:      []byte("{}"),
	})
}

// ListBadWords lists bad words from cache.
func (uc *ModerationUsecase) ListBadWords(ctx context.Context, category string, limit, offset int32) ([]*TextCache, int64, error) {
	caches, err := uc.textCache.List(ctx, category, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	total, err := uc.textCache.Count(ctx, category)
	if err != nil {
		return nil, 0, err
	}
	return caches, total, nil
}

// RemoveBadWord removes a bad word.
func (uc *ModerationUsecase) RemoveBadWord(ctx context.Context, word string) error {
	return uc.textCache.Delete(ctx, hash.HashTextSha256(word))
}

// ModerateText moderates text content.
func (uc *ModerationUsecase) ModerateText(ctx context.Context, requestID, text string) (*ModerationResult, error) {
	uc.log.Debugf("ModerateText: requestID=%s, contentLen=%d", requestID, len(text))

	textResult, err := uc.textModerator.Moderate(ctx, text)
	if err != nil {
		return nil, err
	}

	result := &ModerationResult{
		RequestID:   requestID,
		IsClean:     textResult.IsClean,
		Categories:  textResult.Categories,
		ProcessedAt: time.Now(),
		TextResult:  textResult,
		Scores:      make(map[string]float64),
	}

	uc.fillVerdict(result, textResult.ShouldReject, textResult.ShouldReview)
	uc.fillScores(result, textResult, nil, nil)

	return result, nil
}

// ModerateImage moderates an image URL.
func (uc *ModerationUsecase) ModerateImage(ctx context.Context, requestID, ownerID, imageURL string, fileHash *string) (*ModerationResult, error) {
	uc.log.Debugf("ModerateImage: requestID=%s, ownerID=%s", requestID, ownerID)

	imgResult, err := uc.imageModerator.ModerateImageURL(ctx, ownerID, imageURL, fileHash)
	if err != nil {
		return nil, err
	}

	result := &ModerationResult{
		RequestID:   requestID,
		IsClean:     imgResult.IsClean,
		ProcessedAt: time.Now(),
		ImageResult: imgResult,
		Scores:      make(map[string]float64),
	}

	uc.fillVerdict(result, imgResult.ShouldReject, imgResult.ShouldReview)
	uc.fillScores(result, nil, imgResult, nil)

	return result, nil
}

// ModerateAudio moderates an audio URL.
func (uc *ModerationUsecase) ModerateAudio(ctx context.Context, requestID, audioURL string) (*ModerationResult, error) {
	uc.log.Debugf("ModerateAudio: requestID=%s (Not Implemented)", requestID)
	// Audio moderation is temporarily disabled/not implemented
	return &ModerationResult{
		RequestID:   requestID,
		IsClean:     true,
		ProcessedAt: time.Now(),
		Scores:      make(map[string]float64),
		Action:      ModerationActionAutoApprove,
		Verdict:     VerdictClean,
	}, nil
}

// ModerateVideo moderates a video URL.
func (uc *ModerationUsecase) ModerateVideo(ctx context.Context, requestID, videoURL string) (*ModerationResult, error) {
	uc.log.Debugf("ModerateVideo: requestID=%s", requestID)

	videoResult, err := uc.videoModerator.ModerateVideoURL(ctx, videoURL)
	if err != nil {
		return nil, err
	}

	result := &ModerationResult{
		RequestID:   requestID,
		IsClean:     videoResult.IsClean,
		ProcessedAt: time.Now(),
		VideoResult: videoResult,
		Scores:      make(map[string]float64),
	}

	uc.fillVerdict(result, videoResult.ShouldReject, videoResult.ShouldReview)
	uc.fillScores(result, nil, nil, videoResult)

	return result, nil
}

// Moderate moderates a generic request with mixed content (text, images, video).
// Audio is currently not implemented.
func (uc *ModerationUsecase) Moderate(ctx context.Context, requestID, content string, imageURLs, audioURLs, videoURLs []string) (*ModerationResult, error) {
	uc.log.Debugf("Moderate: requestID=%s, contentLen=%d, images=%d, videos=%d",
		requestID, len(content), len(imageURLs), len(videoURLs))

	result := &ModerationResult{
		RequestID:   requestID,
		IsClean:     true,
		Scores:      make(map[string]float64),
		ProcessedAt: time.Now(),
	}

	// 1. Moderate Text
	if content != "" {
		textResult, err := uc.textModerator.Moderate(ctx, content)
		if err != nil {
			return nil, err
		}
		result.TextResult = textResult
		if !textResult.IsClean {
			result.IsClean = false
			result.Categories = append(result.Categories, textResult.Categories...)
		}
	}

	// 2. Moderate Images
	if len(imageURLs) > 0 {
		// For simplicity, we just take the first bad result or the last result
		// In a real system, we might want a list of image results
		for _, url := range imageURLs {
			imgResult, err := uc.imageModerator.ModerateImageURL(ctx, "", url, nil)
			if err != nil {
				uc.log.Errorf("Failed to moderate image %s: %v", url, err)
				continue
			}
			if !imgResult.IsClean {
				result.IsClean = false
				// Append categories...
			}
			// Keep the last result for detail returning (simplified)
			result.ImageResult = imgResult
			if imgResult.ShouldReject {
				break // Fail fast on images? Or collect all? Let's generic fillVerdict handle it
			}
		}
	}

	// if len(audioURLs) > 0 {
	// for _, url := range audioURLs {
	// 	audioResult, err := uc.audioModerator.ModerateAudioURL(ctx, url)
	// 	if err != nil {
	// 		uc.log.Errorf("Failed to moderate audio %s: %v", url, err)
	// 		continue
	// 	}
	// 	if !audioResult.IsClean {
	// 		result.IsClean = false
	// 	}
	// 	result.AudioResult = audioResult
	// 	if audioResult.ShouldReject {
	// 		break
	// 	}
	// }
	// }

	// 4. Moderate Video
	if len(videoURLs) > 0 {
		for _, url := range videoURLs {
			vidResult, err := uc.videoModerator.ModerateVideoURL(ctx, url)
			if err != nil {
				uc.log.Errorf("Failed to moderate video %s: %v", url, err)
				continue
			}
			if !vidResult.IsClean {
				result.IsClean = false
			}
			result.VideoResult = vidResult
			if vidResult.ShouldReject {
				break
			}
		}
	}

	// Calculate final verdict
	shouldReject := (result.TextResult != nil && result.TextResult.ShouldReject) ||
		(result.ImageResult != nil && result.ImageResult.ShouldReject) ||
		(result.VideoResult != nil && result.VideoResult.ShouldReject)

	shouldReview := (result.TextResult != nil && result.TextResult.ShouldReview) ||
		(result.ImageResult != nil && result.ImageResult.ShouldReview) ||
		(result.VideoResult != nil && result.VideoResult.ShouldReview)

	uc.fillVerdict(result, shouldReject, shouldReview)
	uc.fillScores(result, result.TextResult, result.ImageResult, result.VideoResult)

	return result, nil
}

// Helper to fill verdict and action
func (uc *ModerationUsecase) fillVerdict(result *ModerationResult, shouldReject, shouldReview bool) {
	if shouldReject {
		result.Action = ModerationActionAutoReject
		result.Verdict = VerdictReject
		result.Reason = "Content rejected due to policy violation"
	} else if shouldReview {
		result.Action = ModerationActionPendingReview
		result.Verdict = VerdictReview
		result.Reason = "Content flagged for manual review"
	} else {
		result.Action = ModerationActionAutoApprove
		result.Verdict = VerdictClean
	}
}

// Helper to fill scores
func (uc *ModerationUsecase) fillScores(result *ModerationResult,
	textRes *moderator.TextModerationResult,
	imgRes *moderator.ImageModerationResult,
	vidRes *moderator.VideoModerationResult) {

	if textRes != nil && textRes.MaxNsfwScore > 0 {
		result.Scores["text_nsfw"] = textRes.MaxNsfwScore
	}
	if imgRes != nil {
		for cat, score := range imgRes.Categories {
			result.Scores[string(cat)] = score
		}
	}
	if vidRes != nil {
		if vidRes.MaxNSFWScore > 0 {
			result.Scores["video_nsfw"] = vidRes.MaxNSFWScore
		}
		if vidRes.MaxViolenceScore > 0 {
			result.Scores["video_violence"] = vidRes.MaxViolenceScore
		}
	}
}

// defaultAddedBy returns the default addedBy value if nil
func (uc *ModerationUsecase) defaultAddedBy(addedBy *string) string {
	if addedBy == nil {
		return "manual"
	}
	return *addedBy
}

// defaultModelVersion returns the default model version if nil
func (uc *ModerationUsecase) defaultModelVersion(modelVersion *string) string {
	if modelVersion == nil {
		return ModelVersionText.String()
	}
	return *modelVersion
}
