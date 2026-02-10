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

	// Determine action and verdict based on text moderation result
	if textResult.IsClean {
		result.Action = ModerationActionAutoApprove
		result.Verdict = VerdictClean
	} else if textResult.ShouldReject {
		result.Action = ModerationActionAutoReject
		result.Verdict = VerdictReject
		result.Reason = "Content contains prohibited words"
	} else if textResult.ShouldReview {
		result.Action = ModerationActionPendingReview
		result.Verdict = VerdictReview
		result.Reason = "Content flagged for review"
	} else {
		result.Action = ModerationActionAutoApprove
		result.Verdict = VerdictClean
	}

	// Add severity score
	if textResult.MaxSeverity > 0 {
		result.Scores["text_severity"] = float64(textResult.MaxSeverity)
	}

	return result, nil
}

// ModeratePost moderates a complete post (text + images).
func (uc *ModerationUsecase) ModeratePost(ctx context.Context, requestID, content string, imageURLs []string) (*ModerationResult, error) {
	uc.log.Debugf("ModeratePost: requestID=%s, contentLen=%d, images=%d", requestID, len(content), len(imageURLs))

	// First moderate text
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

	// Add text severity score
	if textResult.MaxSeverity > 0 {
		result.Scores["text_severity"] = float64(textResult.MaxSeverity)
	}

	// Moderate images if any
	for i, url := range imageURLs {
		imgResult, err := uc.imageModerator.ModerateImageURL(ctx, url)
		if err != nil {
			uc.log.Warnf("Failed to moderate image %d: %v", i, err)
			continue
		}

		if !imgResult.IsClean {
			result.IsClean = false
			// Merge image categories with result
			for cat, score := range imgResult.Categories {
				result.Scores[string(cat)] = score
			}
		}

		if imgResult.ShouldReject {
			result.IsClean = false
		}

		// Store the last image result (in production, would aggregate all)
		result.ImageResult = imgResult
	}

	// Determine final action and verdict
	if textResult.ShouldReject || (result.ImageResult != nil && result.ImageResult.ShouldReject) {
		result.Action = ModerationActionAutoReject
		result.Verdict = VerdictReject
		result.Reason = "Content rejected due to policy violation"
	} else if textResult.ShouldReview || (result.ImageResult != nil && result.ImageResult.ShouldReview) {
		result.Action = ModerationActionPendingReview
		result.Verdict = VerdictReview
		result.Reason = "Content flagged for manual review"
	} else if result.IsClean {
		result.Action = ModerationActionAutoApprove
		result.Verdict = VerdictClean
	} else {
		result.Action = ModerationActionPendingReview
		result.Verdict = VerdictReview
	}

	return result, nil
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
