package data

import (
	"context"
	"moderation/internal/biz"
	"moderation/internal/data/postgres/sqlc"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/jackc/pgx/v5/pgtype"
)

type badwordRepo struct {
	data *Data
	log  *log.Helper
}

// Create implements biz.BadwordRepo.
func (r *badwordRepo) Create(ctx context.Context, bw *biz.Badword) (*biz.Badword, error) {
	result, err := r.data.Queries.CreateBadWord(ctx, sqlc.CreateBadWordParams{
		Word:     bw.Word,
		Category: pgtype.Text{String: bw.Category, Valid: true},
		Severity: pgtype.Int4{Int32: bw.Severity, Valid: true},
		AddedBy:  pgtype.Text{String: bw.AddedBy, Valid: true},
	})
	if err != nil {
		return nil, err
	}
	return toBizBadword(result), nil
}

// Update implements biz.BadwordRepo.
func (r *badwordRepo) Update(ctx context.Context, bw *biz.Badword) (*biz.Badword, error) {
	result, err := r.data.Queries.UpdateBadWord(ctx, sqlc.UpdateBadWordParams{
		ID:       bw.ID,
		Word:     bw.Word,
		Category: pgtype.Text{String: bw.Category, Valid: true},
		Severity: pgtype.Int4{Int32: bw.Severity, Valid: true},
	})
	if err != nil {
		return nil, err
	}
	return toBizBadword(result), nil
}

// Delete implements biz.BadwordRepo.
func (r *badwordRepo) Delete(ctx context.Context, word string) error {
	return r.data.Queries.DeleteBadWord(ctx, word)
}

// FindByID implements biz.BadwordRepo.
func (r *badwordRepo) FindByID(ctx context.Context, id int64) (*biz.Badword, error) {
	result, err := r.data.Queries.GetBadWordByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return toBizBadword(result), nil
}

// FindByWord implements biz.BadwordRepo.
func (r *badwordRepo) FindByWord(ctx context.Context, word string) (*biz.Badword, error) {
	result, err := r.data.Queries.GetBadWordByWord(ctx, word)
	if err != nil {
		return nil, err
	}
	return toBizBadword(result), nil
}

// List implements biz.BadwordRepo.
func (r *badwordRepo) List(ctx context.Context, category string, limit, offset int32) ([]*biz.Badword, error) {
	var results []sqlc.BadWord
	var err error

	if category != "" {
		results, err = r.data.Queries.ListBadWordsByCategory(ctx, sqlc.ListBadWordsByCategoryParams{
			Category: pgtype.Text{String: category, Valid: true},
			Limit:    limit,
			Offset:   offset,
		})
	} else {
		results, err = r.data.Queries.ListBadWords(ctx, sqlc.ListBadWordsParams{
			Limit:  limit,
			Offset: offset,
		})
	}
	if err != nil {
		return nil, err
	}

	badwords := make([]*biz.Badword, len(results))
	for i, result := range results {
		badwords[i] = toBizBadword(result)
	}
	return badwords, nil
}

// ListAll implements biz.BadwordRepo.
func (r *badwordRepo) ListAll(ctx context.Context) ([]*biz.Badword, error) {
	results, err := r.data.Queries.ListAllBadWords(ctx)
	if err != nil {
		return nil, err
	}

	badwords := make([]*biz.Badword, len(results))
	for i, result := range results {
		badwords[i] = toBizBadword(result)
	}
	return badwords, nil
}

// Count implements biz.BadwordRepo.
func (r *badwordRepo) Count(ctx context.Context, category string) (int64, error) {
	if category != "" {
		return r.data.Queries.CountBadWordsByCategory(ctx, pgtype.Text{String: category, Valid: true})
	}
	return r.data.Queries.CountBadWords(ctx)
}

// NewBadwordRepo creates a new BadwordRepo.
func NewBadwordRepo(data *Data, logger log.Logger) biz.BadwordRepo {
	return &badwordRepo{
		data: data,
		log:  log.NewHelper(logger),
	}
}

// toBizBadword converts sqlc.BadWord to biz.Badword.
func toBizBadword(bw sqlc.BadWord) *biz.Badword {
	return &biz.Badword{
		ID:        bw.ID,
		Word:      bw.Word,
		Category:  bw.Category.String,
		Severity:  bw.Severity.Int32,
		AddedBy:   bw.AddedBy.String,
		CreatedAt: bw.CreatedAt.Time,
		UpdatedAt: bw.UpdatedAt.Time,
	}
}
