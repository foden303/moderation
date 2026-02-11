package biz

import (
	"context"
	"time"

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

// ModerationUsecase orchestrates content moderation.
type ModerationUsecase struct {
	textModerator  *moderator.TextModerator
	imageModerator *moderator.LocalImageModerator
	videoModerator *moderator.LocalVideoModerator
	badwordRepo    BadwordRepo
	log            *log.Helper
}

// NewModerationUsecase creates a new ModerationUsecase.
func NewModerationUsecase(
	textMod *moderator.TextModerator,
	imgMod *moderator.LocalImageModerator,
	videoMod *moderator.LocalVideoModerator,
	repo BadwordRepo,
	logger log.Logger,
) *ModerationUsecase {
	return &ModerationUsecase{
		textModerator:  textMod,
		imageModerator: imgMod,
		videoModerator: videoMod,
		badwordRepo:    repo,
		log:            log.NewHelper(logger),
	}
}

// ModerateText moderates text content.
func (uc *ModerationUsecase) ModerateText(ctx context.Context, requestID, content string) (*ModerationResult, error) {
	uc.log.Debugf("ModerateText: requestID=%s, contentLen=%d", requestID, len(content))

	textResult, err := uc.textModerator.Moderate(ctx, content)
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
func (uc *ModerationUsecase) ModerateImage(ctx context.Context, requestID, imageURL string) (*ModerationResult, error) {
	uc.log.Debugf("ModerateImage: requestID=%s", requestID)

	imgResult, err := uc.imageModerator.ModerateImageURL(ctx, imageURL)
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
			imgResult, err := uc.imageModerator.ModerateImageURL(ctx, url)
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

	if textRes != nil && textRes.MaxSeverity > 0 {
		result.Scores["text_severity"] = float64(textRes.MaxSeverity)
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

// RebuildFilters rebuilds all moderation filters from database.
func (uc *ModerationUsecase) RebuildFilters(ctx context.Context) (int, error) {
	uc.log.Info("Rebuilding moderation filters from database")

	// Get all bad words from database
	badwords, err := uc.badwordRepo.ListAll(ctx)
	if err != nil {
		return 0, err
	}

	// Convert to moderator.BadWord
	words := make([]moderator.BadWord, len(badwords))
	for i, bw := range badwords {
		words[i] = moderator.BadWord{
			Word:     bw.Word,
			Category: bw.Category,
			Severity: bw.Severity,
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
func (uc *ModerationUsecase) AddBadWord(ctx context.Context, word, category, addedBy string, severity int32) error {
	// Add to text moderator bloom filter
	return uc.textModerator.AddWord(ctx, moderator.BadWord{
		Word:     word,
		Category: category,
		Severity: severity,
	})
}
