package data

import (
	"context"
	"moderation/internal/biz"
	"moderation/internal/conf"
	"moderation/internal/pkg/moderator"
	"moderation/internal/pkg/nsfw"
	"moderation/internal/pkg/redis"

	"github.com/go-kratos/kratos/v2/log"
)

// NewTextModerator creates a new TextModerator.
func NewTextModerator(redisCache redis.Cache, mc *conf.Moderation, logger log.Logger) *moderator.TextModerator {
	config := moderator.DefaultTextModeratorConfig()

	// Wire vLLM config from proto config
	vllmConf := mc.GetVllm()
	if vllmConf != nil {
		config.EnableLLM = vllmConf.GetEnabled()
		if vllmConf.GetBaseUrl() != "" {
			config.VLLMBaseURL = vllmConf.GetBaseUrl()
		}
		if vllmConf.GetModel() != "" {
			config.VLLMModel = vllmConf.GetModel()
		}
		if vllmConf.GetTimeout() != nil {
			config.VLLMTimeout = vllmConf.GetTimeout().AsDuration()
		}
	}

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

// NewNSFWClient creates a gRPC NSFW client from config.
func NewNSFWClient(mc *conf.Moderation, logger log.Logger) (*nsfw.GRPCClient, func(), error) {
	helper := log.NewHelper(logger)

	nsfwConf := mc.GetNsfw()
	if nsfwConf == nil || !nsfwConf.GetEnabled() {
		helper.Info("NSFW detector disabled, skipping gRPC client creation")
		return nil, func() {}, nil
	}

	cfg := nsfw.DefaultGRPCConfig()
	if nsfwConf.GetAddr() != "" {
		cfg.Address = nsfwConf.GetAddr()
	}
	if nsfwConf.GetTimeout() != nil {
		cfg.Timeout = nsfwConf.GetTimeout().AsDuration()
	}

	client, err := nsfw.NewGRPCClient(cfg)
	if err != nil {
		return nil, nil, err
	}

	helper.Infof("NSFW gRPC client connected to %s", cfg.Address)

	cleanup := func() {
		helper.Info("closing NSFW gRPC connection")
		client.Close()
	}

	return client, cleanup, nil
}

// NewImageModerator creates a new LocalImageModerator with multi-layer detection.
func NewImageModerator(
	textMod *moderator.TextModerator,
	redisCache redis.Cache,
	nsfwClient *nsfw.GRPCClient,
	badImageRepo biz.BadImageRepo,
	logger log.Logger,
) *moderator.LocalImageModerator {
	config := moderator.DefaultImageModeratorConfig()

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
