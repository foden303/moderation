package data

import (
	"context"
	"errors"
	"time"

	"moderation/internal/biz"
	"moderation/internal/data/postgres/sqlc"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/jackc/pgx/v5"
)

type imageCacheRepo struct {
	data *Data
	log  *log.Helper
}

// NewImageCacheRepo creates a new ImageCacheRepo.
func NewImageCacheRepo(data *Data, logger log.Logger) biz.ImageCacheRepo {
	return &imageCacheRepo{
		data: data,
		log:  log.NewHelper(logger),
	}
}

func (r *imageCacheRepo) Upsert(ctx context.Context, cache *biz.ImageCache) error {
	_, err := r.data.Queries.UpsertImageCache(ctx, sqlc.UpsertImageCacheParams{
		FileHash:     cache.FileHash,
		Phash:        cache.PHash,
		DetectResult: cache.DetectResult,
		Category:     cache.Category,
		NsfwScore:    cache.NSFWScore,
		ModelVersion: cache.ModelVersion,
		SourceUrl:    cache.SourceURL,
		AddedBy:      cache.AddedBy,
		ExpiresAt:    toPgTimestamptz(cache.ExpiresAt),
	})
	return err
}

func (r *imageCacheRepo) Get(ctx context.Context, fileHash string) (*biz.ImageCache, error) {
	result, err := r.data.Queries.GetImageCache(ctx, fileHash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil // Not found
		}
		return nil, err
	}
	return toBizImageCache(result), nil
}

func (r *imageCacheRepo) FindSimilarByPHash(ctx context.Context, phash int64, maxDistance int32) ([]*biz.ImageCache, error) {
	results, err := r.data.Queries.FindSimilarByPHash(ctx, sqlc.FindSimilarByPHashParams{
		TargetPhash: phash,
		MaxDistance: int64(maxDistance),
	})
	if err != nil {
		return nil, err
	}
	caches := make([]*biz.ImageCache, len(results))
	for i, result := range results {
		caches[i] = toBizImageCacheFromRow(result)
	}
	return caches, nil
}

func toBizImageCacheFromRow(r sqlc.FindSimilarByPHashRow) *biz.ImageCache {
	var expiresAt *time.Time
	if r.ExpiresAt.Valid {
		t := r.ExpiresAt.Time
		expiresAt = &t
	}
	return &biz.ImageCache{
		FileHash:     r.FileHash,
		PHash:        r.Phash,
		DetectResult: r.DetectResult,
		Category:     r.Category,
		NSFWScore:    r.NsfwScore,
		ModelVersion: r.ModelVersion,
		SourceURL:    r.SourceUrl,
		AddedBy:      r.AddedBy,
		ExpiresAt:    expiresAt,
		CreatedAt:    r.CreatedAt.Time,
		UpdatedAt:    r.UpdatedAt.Time,
	}
}

func (r *imageCacheRepo) Delete(ctx context.Context, fileHash string) error {
	return r.data.Queries.DeleteImageCache(ctx, fileHash)
}

func (r *imageCacheRepo) DeleteExpired(ctx context.Context) (int64, error) {
	return r.data.Queries.DeleteExpiredImageCaches(ctx)
}

func (r *imageCacheRepo) List(ctx context.Context, category string, limit, offset int32) ([]*biz.ImageCache, error) {
	var results []sqlc.ImageCach
	var err error

	if category != "" {
		results, err = r.data.Queries.ListImageCachesByCategory(ctx, sqlc.ListImageCachesByCategoryParams{
			Category: category,
			Limit:    limit,
			Offset:   offset,
		})
	} else {
		results, err = r.data.Queries.ListImageCaches(ctx, sqlc.ListImageCachesParams{
			Limit:  limit,
			Offset: offset,
		})
	}

	if err != nil {
		return nil, err
	}

	caches := make([]*biz.ImageCache, len(results))
	for i, r := range results {
		caches[i] = toBizImageCache(r)
	}
	return caches, nil
}

func (r *imageCacheRepo) ListAll(ctx context.Context) ([]*biz.ImageCache, error) {
	results, err := r.data.Queries.GetAllImageCaches(ctx)
	if err != nil {
		return nil, err
	}
	caches := make([]*biz.ImageCache, len(results))
	for i, r := range results {
		caches[i] = toBizImageCache(r)
	}
	return caches, nil
}

func (r *imageCacheRepo) Count(ctx context.Context, category string) (int64, error) {
	if category != "" {
		return r.data.Queries.CountImageCachesByCategory(ctx, category)
	}

	return r.data.Queries.CountImageCaches(ctx)
}

func toBizImageCache(c sqlc.ImageCach) *biz.ImageCache {
	var expiresAt *time.Time
	if c.ExpiresAt.Valid {
		t := c.ExpiresAt.Time
		expiresAt = &t
	}
	return &biz.ImageCache{
		FileHash:     c.FileHash,
		PHash:        c.Phash,
		DetectResult: c.DetectResult,
		Category:     c.Category,
		NSFWScore:    c.NsfwScore,
		ModelVersion: c.ModelVersion,
		SourceURL:    c.SourceUrl,
		AddedBy:      c.AddedBy,
		ExpiresAt:    expiresAt,
		CreatedAt:    c.CreatedAt.Time,
		UpdatedAt:    c.UpdatedAt.Time,
	}
}
