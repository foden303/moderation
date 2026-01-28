package biz

import (
	"context"
	"time"

	"github.com/go-kratos/kratos/v2/log"
)

// Plan is a Plan model.
type Plan struct {
	CreatedAt             time.Time
	UpdatedAt             time.Time
	ID                    string
	Name                  string
	DisplayName           string
	UserIdentifier        string
	EffectiveStorageQuota int64
	StoragePhotosUsed     int64
	StorageVideoUsed      int64
	StorageDocumentUsed   int64
	StorageAudioUsed      int64
	StorageCompressUsed   int64
	StorageOtherUsed      int64
	StorageTotalUsed      int64
}

// PlanRepo is a Plan repo.
type PlanRepo interface {
	GetPlanByID(context.Context, string) (*Plan, error)
	GetPlans(context.Context) ([]*Plan, error)
	CreatePlan(context.Context, *Plan) (*Plan, error)
}

// PlanUsecase is a Plan usecase.
type PlanUsecase struct {
	repo PlanRepo
}

// NewPlanUsecase new a Plan usecase.
func NewPlanUsecase(repo PlanRepo) *PlanUsecase {
	return &PlanUsecase{repo: repo}
}

// CreatePlan creates a Plan, and returns the new Plan.
func (uc *PlanUsecase) CreatePlan(ctx context.Context, p *Plan) (*Plan, error) {
	log.Infof("CreatePlan: %v", p.Name)
	return uc.repo.CreatePlan(ctx, p)
}

// GetPlanByID retrieves a Plan by ID.
func (uc *PlanUsecase) GetPlanByID(ctx context.Context, id string) (*Plan, error) {
	log.Infof("GetPlanByID: %v", id)
	return uc.repo.GetPlanByID(ctx, id)
}

// GetPlans retrieves all plans.
func (uc *PlanUsecase) GetPlans(ctx context.Context) ([]*Plan, error) {
	log.Info("GetPlans")
	return uc.repo.GetPlans(ctx)
}
