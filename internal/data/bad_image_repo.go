package data

import (
	"context"
	"errors"
	"moderation/internal/biz"
	"moderation/internal/data/postgres/sqlc"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type badImageRepo struct {
	data *Data
	log  *log.Helper
}

// Create implements biz.BadImageRepo.
func (r *badImageRepo) Create(ctx context.Context, img *biz.BadImage) (*biz.BadImage, error) {
	result, err := r.data.Queries.CreateBadImage(ctx, sqlc.CreateBadImageParams{
		Phash:     img.PHash,
		Category:  pgtype.Text{String: img.Category, Valid: true},
		NsfwScore: pgtype.Float8{Float64: img.NSFWScore, Valid: true},
		SourceUrl: pgtype.Text{String: img.SourceURL, Valid: img.SourceURL != ""},
		AddedBy:   pgtype.Text{String: img.AddedBy, Valid: true},
	})
	if err != nil {
		return nil, err
	}
	return toBizBadImage(result), nil
}

// FindByPHash implements biz.BadImageRepo.
func (r *badImageRepo) FindByPHash(ctx context.Context, phash int64) (*biz.BadImage, error) {
	result, err := r.data.Queries.GetBadImageByPHash(ctx, phash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil // Not found
		}
		return nil, err
	}
	return toBizBadImage(result), nil
}

// FindByPHashes implements biz.BadImageRepo.
func (r *badImageRepo) FindByPHashes(ctx context.Context, phashes []int64) ([]*biz.BadImage, error) {
	results, err := r.data.Queries.GetBadImagesByPHashes(ctx, phashes)
	if err != nil {
		return nil, err
	}
	images := make([]*biz.BadImage, len(results))
	for i, result := range results {
		images[i] = toBizBadImage(result)
	}
	return images, nil
}

// ListAll implements biz.BadImageRepo.
func (r *badImageRepo) ListAll(ctx context.Context) ([]*biz.BadImage, error) {
	results, err := r.data.Queries.ListAllBadImages(ctx)
	if err != nil {
		return nil, err
	}
	images := make([]*biz.BadImage, len(results))
	for i, result := range results {
		images[i] = toBizBadImage(result)
	}
	return images, nil
}

// Count implements biz.BadImageRepo.
func (r *badImageRepo) Count(ctx context.Context) (int64, error) {
	return r.data.Queries.CountBadImages(ctx)
}

// Delete implements biz.BadImageRepo.
func (r *badImageRepo) Delete(ctx context.Context, phash int64) error {
	return r.data.Queries.DeleteBadImage(ctx, phash)
}

// NewBadImageRepo creates a new BadImageRepo.
func NewBadImageRepo(data *Data, logger log.Logger) biz.BadImageRepo {
	return &badImageRepo{
		data: data,
		log:  log.NewHelper(logger),
	}
}

// toBizBadImage converts sqlc.BadImage to biz.BadImage.
func toBizBadImage(img sqlc.BadImage) *biz.BadImage {
	return &biz.BadImage{
		ID:        img.ID,
		PHash:     img.Phash,
		Category:  img.Category.String,
		NSFWScore: img.NsfwScore.Float64,
		SourceURL: img.SourceUrl.String,
		AddedBy:   img.AddedBy.String,
		CreatedAt: img.CreatedAt.Time,
		UpdatedAt: img.UpdatedAt.Time,
	}
}
