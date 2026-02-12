package data

import (
	"context"
	"moderation/internal/biz"
	"moderation/internal/conf"
	"moderation/internal/pkg/moderator"
	"moderation/internal/pkg/nsfw"
	"moderation/internal/pkg/redis"
	"strconv"

	"github.com/go-kratos/kratos/v2/log"
)

// NewTextModerator creates a new TextModerator.
func NewTextModerator(redisCache redis.Cache, nsfwTextClient *nsfw.TextClient, mc *conf.Moderation, logger log.Logger) *moderator.TextModerator {
	config := moderator.DefaultTextModeratorConfig()

	// Wire text moderation thresholds
	textConf := mc.GetText()
	if textConf != nil {
		if textConf.GetRejectThreshold() > 0 {
			config.RejectThreshold = textConf.GetRejectThreshold()
		}
		if textConf.GetReviewThreshold() > 0 {
			config.ReviewThreshold = textConf.GetReviewThreshold()
		}
	}

	return moderator.NewTextModerator(redisCache, config, nsfwTextClient)
}

// badImageCheckerAdapter adapts ImageCacheRepo to moderator.BadImageChecker interface.
type badImageCheckerAdapter struct {
	repo biz.ImageCacheRepo
}

func (a *badImageCheckerAdapter) FindByPHash(ctx context.Context, phash int64) (bool, error) {
	images, err := a.repo.GetByPHash(ctx, phash)
	if err != nil {
		return false, err
	}
	// Check if any unsafe image found
	for _, img := range images {
		if img.Category != "safe" {
			return true, nil
		}
	}
	return false, nil
}

func (a *badImageCheckerAdapter) SaveBadImage(ctx context.Context, phash int64, category string, nsfwScore float64, sourceURL string) error {
	// Use PHash as basis for FileHash since we don't have Original FileHash here.
	// This is a limitation of the current adapter interface.
	fileHash := "phash:" + strconv.FormatInt(phash, 10)
	return a.repo.Upsert(ctx, &biz.ImageCache{
		FileHash:     fileHash,
		PHash:        phash,
		Category:     category,
		NSFWScore:    nsfwScore,
		SourceURL:    sourceURL,
		AddedBy:      "auto",
		ModelVersion: "nsfw_v1",
		DetectResult: []byte("{}"),
	})
}

// NewNSFWImageClient creates a gRPC NSFW image client from config.
func NewNSFWImageClient(mc *conf.Moderation, logger log.Logger) (*nsfw.ImageClient, func(), error) {
	helper := log.NewHelper(logger)

	nsfwConf := mc.GetNsfwImage()
	if nsfwConf == nil || !nsfwConf.GetEnabled() {
		helper.Info("NSFW image detector disabled, skipping gRPC client")
		return nil, func() {}, nil
	}

	cfg := nsfw.DefaultConfig(nsfwConf.GetAddr())
	if nsfwConf.GetTimeout() != nil {
		cfg.Timeout = nsfwConf.GetTimeout().AsDuration()
	}

	client, err := nsfw.NewImageClient(cfg)
	if err != nil {
		return nil, nil, err
	}

	helper.Infof("NSFW image gRPC client connected to %s", cfg.Address)

	cleanup := func() {
		helper.Info("closing NSFW image gRPC connection")
		client.Close()
	}

	return client, cleanup, nil
}

// NewNSFWTextClient creates a gRPC NSFW text client from config.
func NewNSFWTextClient(mc *conf.Moderation, logger log.Logger) (*nsfw.TextClient, func(), error) {
	helper := log.NewHelper(logger)

	nsfwConf := mc.GetNsfwText()
	if nsfwConf == nil || !nsfwConf.GetEnabled() {
		helper.Info("NSFW text detector disabled, skipping gRPC client")
		return nil, func() {}, nil
	}

	cfg := nsfw.DefaultConfig(nsfwConf.GetAddr())
	if nsfwConf.GetTimeout() != nil {
		cfg.Timeout = nsfwConf.GetTimeout().AsDuration()
	}

	client, err := nsfw.NewTextClient(cfg)
	if err != nil {
		return nil, nil, err
	}

	helper.Infof("NSFW text gRPC client connected to %s", cfg.Address)

	cleanup := func() {
		helper.Info("closing NSFW text gRPC connection")
		client.Close()
	}

	return client, cleanup, nil
}

// NewImageModerator creates a new LocalImageModerator with multi-layer detection.
func NewImageModerator(
	textMod *moderator.TextModerator,
	redisCache redis.Cache,
	nsfwClient *nsfw.ImageClient,
	imageCache biz.ImageCacheRepo,
	logger log.Logger,
) *moderator.LocalImageModerator {
	config := moderator.DefaultImageModeratorConfig()

	// Create adapter for ImageCacheRepo
	checker := &badImageCheckerAdapter{repo: imageCache}

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
