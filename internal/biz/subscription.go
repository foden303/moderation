package biz

import (
	"context"
	"time"

	"github.com/go-kratos/kratos/v2/log"
)

// Subscription is a Subscription model.
type Subscription struct {
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

// SubscriptionRepo is a Subscription repo.
type SubscriptionRepo interface {
	GetSubscriptionByID(context.Context, string) (*Subscription, error)
	GetSubscriptions(context.Context) ([]*Subscription, error)
	CreateSubscription(context.Context, *Subscription) (*Subscription, error)
	ExpireSubscription(context.Context, string) error
	ExpireSubscriptions(context.Context) error
	AutoResetQuotaDefault(context.Context) error
}

// SubscriptionUsecase is a Subscription usecase.
type SubscriptionUsecase struct {
	repo SubscriptionRepo
}

// NewSubscriptionUsecase new a Subscription usecase.
func NewSubscriptionUsecase(repo SubscriptionRepo) *SubscriptionUsecase {
	return &SubscriptionUsecase{repo: repo}
}

// CreateSubscription creates a Subscription, and returns the new Subscription.
func (uc *SubscriptionUsecase) CreateSubscription(ctx context.Context, p *Subscription) (*Subscription, error) {
	log.Infof("CreateSubscription: %v", p.Name)
	return uc.repo.CreateSubscription(ctx, p)
}

// GetSubscriptionByID retrieves a Subscription by ID.
func (uc *SubscriptionUsecase) GetSubscriptionByID(ctx context.Context, id string) (*Subscription, error) {
	log.Infof("GetSubscriptionByID: %v", id)
	return uc.repo.GetSubscriptionByID(ctx, id)
}

// GetSubscriptions retrieves all subscriptions.
func (uc *SubscriptionUsecase) GetSubscriptions(ctx context.Context) ([]*Subscription, error) {
	log.Info("GetSubscriptions")
	return uc.repo.GetSubscriptions(ctx)
}

// ExpireSubscription expires a Subscription by ID.
func (uc *SubscriptionUsecase) ExpireSubscription(ctx context.Context, id string) error {
	log.Infof("ExpireSubscription: %v", id)
	return uc.repo.ExpireSubscription(ctx, id)
}

// ExpireSubscriptions expires all due subscriptions.
func (uc *SubscriptionUsecase) ExpireSubscriptions(ctx context.Context) error {
	log.Info("ExpireSubscriptions")
	return uc.repo.ExpireSubscriptions(ctx)
}

// AutoResetQuotaDefault resets the quota for default subscriptions.
func (uc *SubscriptionUsecase) AutoResetQuotaDefault(ctx context.Context) error {
	log.Info("AutoResetQuotaDefault")
	return uc.repo.AutoResetQuotaDefault(ctx)
}
