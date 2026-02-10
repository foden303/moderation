package biz

import (
	"context"
	"time"

	"github.com/go-kratos/kratos/v2/log"
)

// Badword is a Badword model.
type Badword struct {
	ID        int64
	Word      string
	Category  string
	Severity  int32
	AddedBy   string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// BadwordRepo is a Badword repository interface.
type BadwordRepo interface {
	Create(context.Context, *Badword) (*Badword, error)
	Update(context.Context, *Badword) (*Badword, error)
	Delete(context.Context, string) error
	FindByID(context.Context, int64) (*Badword, error)
	FindByWord(context.Context, string) (*Badword, error)
	List(ctx context.Context, category string, limit, offset int32) ([]*Badword, error)
	ListAll(context.Context) ([]*Badword, error)
	Count(ctx context.Context, category string) (int64, error)
}

// BadwordUsecase is a Badword usecase.
type BadwordUsecase struct {
	repo BadwordRepo
	log  *log.Helper
}

// NewBadwordUsecase new a Badword usecase.
func NewBadwordUsecase(repo BadwordRepo, logger log.Logger) *BadwordUsecase {
	return &BadwordUsecase{
		repo: repo,
		log:  log.NewHelper(logger),
	}
}

// AddBadword adds a new bad word.
func (uc *BadwordUsecase) AddBadword(ctx context.Context, word, category, addedBy string, severity int32) (*Badword, error) {
	uc.log.Infof("AddBadword: %s, category: %s, severity: %d", word, category, severity)
	return uc.repo.Create(ctx, &Badword{
		Word:     word,
		Category: category,
		Severity: severity,
		AddedBy:  addedBy,
	})
}

// RemoveBadword removes a bad word.
func (uc *BadwordUsecase) RemoveBadword(ctx context.Context, word string) error {
	uc.log.Infof("RemoveBadword: %s", word)
	return uc.repo.Delete(ctx, word)
}

// ListBadwords lists bad words.
func (uc *BadwordUsecase) ListBadwords(ctx context.Context, category string, limit, offset int32) ([]*Badword, int64, error) {
	words, err := uc.repo.List(ctx, category, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	total, err := uc.repo.Count(ctx, category)
	if err != nil {
		return nil, 0, err
	}
	return words, total, nil
}

// GetAllBadwords gets all bad words for building bloom filter.
func (uc *BadwordUsecase) GetAllBadwords(ctx context.Context) ([]*Badword, error) {
	return uc.repo.ListAll(ctx)
}
