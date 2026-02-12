package data

import (
	"context"
	"errors"
	"time"

	"moderation/internal/biz"
	"moderation/internal/data/postgres/sqlc"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type textCacheRepo struct {
	data *Data
	log  *log.Helper
}

// NewTextCacheRepo creates a new TextCacheRepo.
func NewTextCacheRepo(data *Data, logger log.Logger) biz.TextCacheRepo {
	return &textCacheRepo{
		data: data,
		log:  log.NewHelper(logger),
	}
}

func (r *textCacheRepo) Upsert(ctx context.Context, cache *biz.TextCache) error {
	_, err := r.data.Queries.UpsertTextCache(ctx, sqlc.UpsertTextCacheParams{
		ContentHash:       cache.ContentHash,
		NormalizedContent: cache.NormalizedContent,
		DetectResult:      cache.DetectResult,
		Category:          cache.Category,
		NsfwScore:         cache.NSFWScore,
		ModelVersion:      cache.ModelVersion,
		AddedBy:           cache.AddedBy,
		ExpiresAt:         toPgTimestamptz(cache.ExpiresAt),
	})
	return err
}

func (r *textCacheRepo) Get(ctx context.Context, contentHash string) (*biz.TextCache, error) {
	result, err := r.data.Queries.GetTextCache(ctx, contentHash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil // Not found
		}
		return nil, err
	}
	return toBizTextCache(result), nil
}

func (r *textCacheRepo) Delete(ctx context.Context, contentHash string) error {
	return r.data.Queries.DeleteTextCache(ctx, contentHash)
}

func (r *textCacheRepo) DeleteExpired(ctx context.Context) (int64, error) {
	return r.data.Queries.DeleteExpiredTextCaches(ctx)
}

func (r *textCacheRepo) List(ctx context.Context, category string, limit, offset int32) ([]*biz.TextCache, error) {
	var results []sqlc.TextCach
	var err error

	if category != "" {
		results, err = r.data.Queries.ListTextCachesByCategory(ctx, sqlc.ListTextCachesByCategoryParams{
			Category: category,
			Limit:    limit,
			Offset:   offset,
		})
	} else {
		results, err = r.data.Queries.ListTextCaches(ctx, sqlc.ListTextCachesParams{
			Limit:  limit,
			Offset: offset,
		})
	}

	if err != nil {
		return nil, err
	}

	caches := make([]*biz.TextCache, len(results))
	for i, r := range results {
		caches[i] = toBizTextCache(r)
	}
	return caches, nil
}

func (r *textCacheRepo) ListAll(ctx context.Context) ([]*biz.TextCache, error) {
	results, err := r.data.Queries.GetAllTextCaches(ctx)
	if err != nil {
		return nil, err
	}
	caches := make([]*biz.TextCache, len(results))
	for i, r := range results {
		caches[i] = toBizTextCache(r)
	}
	return caches, nil
}

func (r *textCacheRepo) Count(ctx context.Context, category string) (int64, error) {
	if category != "" {
		return r.data.Queries.CountTextCachesByCategory(ctx, category)
	}
	return r.data.Queries.CountTextCaches(ctx)
}

func toBizTextCache(c sqlc.TextCach) *biz.TextCache {
	var expiresAt *time.Time
	if c.ExpiresAt.Valid {
		t := c.ExpiresAt.Time
		expiresAt = &t
	}
	return &biz.TextCache{
		ContentHash:       c.ContentHash,
		NormalizedContent: c.NormalizedContent,
		DetectResult:      c.DetectResult,
		Category:          c.Category,
		NSFWScore:         c.NsfwScore,
		ModelVersion:      c.ModelVersion,
		AddedBy:           c.AddedBy,
		ExpiresAt:         expiresAt,
		CreatedAt:         c.CreatedAt.Time,
		UpdatedAt:         c.UpdatedAt.Time,
	}
}

func toPgTimestamptz(t *time.Time) pgtype.Timestamptz {
	if t == nil {
		return pgtype.Timestamptz{Valid: false}
	}
	return pgtype.Timestamptz{Time: *t, Valid: true}
}
