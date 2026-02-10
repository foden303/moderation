package data

import (
	"context"
	"moderation/internal/biz"
	"moderation/internal/pkg/moderator"
	"moderation/internal/pkg/nsfw"
	"moderation/internal/pkg/redis"

	"github.com/go-kratos/kratos/v2/log"
)

// NewTextModerator creates a new TextModerator.
func NewTextModerator(redisCache redis.Cache, logger log.Logger) *moderator.TextModerator {
	config := moderator.DefaultTextModeratorConfig()
	return moderator.NewTextModerator(redisCache, config)
}

// badImageCheckerAdapter adapts BadImageRepo to moderator.BadImageChecker interface.
type badImageCheckerAdapter struct {
	repo biz.BadImageRepo
}

func (a *badImageCheckerAdapter) FindByPHash(ctx context.Context, phash int64) (bool, error) {
	img, err := a.repo.FindByPHash(ctx, phash)
	if err != nil {
		return false, err
	}
	return img != nil, nil
}

func (a *badImageCheckerAdapter) SaveBadImage(ctx context.Context, phash int64, category string, nsfwScore float64, sourceURL string) error {
	_, err := a.repo.Create(ctx, &biz.BadImage{
		PHash:     phash,
		Category:  category,
		NSFWScore: nsfwScore,
		SourceURL: sourceURL,
		AddedBy:   "auto",
	})
	return err
}

// NewImageModerator creates a new LocalImageModerator with multi-layer detection.
func NewImageModerator(
	textMod *moderator.TextModerator,
	redisCache redis.Cache,
	badImageRepo biz.BadImageRepo,
	logger log.Logger,
) *moderator.LocalImageModerator {
	config := moderator.DefaultImageModeratorConfig()

	// Create NSFW client (defaults to localhost:8081)
	nsfwClient := nsfw.NewClient(nsfw.DefaultConfig())

	// Create adapter for BadImageRepo
	checker := &badImageCheckerAdapter{repo: badImageRepo}

	return moderator.NewLocalImageModerator(
		config,
		textMod,
		redisCache,
		nsfwClient,
		checker,
		logger,
	)
}

// NewVideoModerator creates a new LocalVideoModerator.
func NewVideoModerator(imgMod *moderator.LocalImageModerator, textMod *moderator.TextModerator, logger log.Logger) *moderator.LocalVideoModerator {
	config := moderator.DefaultVideoModeratorConfig()
	return moderator.NewLocalVideoModerator(config, imgMod, textMod)
}
