package biz

import (
	"context"
	"time"

	v1 "storage/api/helloworld/v1"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
)

var (
	// ErrUserNotFound is user not found.
	ErrUserNotFound = errors.NotFound(v1.ErrorReason_USER_NOT_FOUND.String(), "user not found")
)

// User is a User model.
type User struct {
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

// UserRepo is a User repo.
type UserRepo interface {
	GetUserByID(context.Context, string) (*User, error)
	GetUsers(context.Context) ([]*User, error)
	UpsertUser(context.Context, *User) (*User, error)
}

// UserUsecase is a User usecase.
type UserUsecase struct {
	repo UserRepo
}

// NewUserUsecase new a User usecase.
func NewUserUsecase(repo UserRepo) *UserUsecase {
	return &UserUsecase{repo: repo}
}

// UpsertUser creates or updates a User, and returns the new or updated User.
func (uc *UserUsecase) UpsertUser(ctx context.Context, u *User) (*User, error) {
	log.Infof("UpsertUser: %v", u.Name)
	return uc.repo.UpsertUser(ctx, u)
}

// GetUserByID retrieves a User by ID.
func (uc *UserUsecase) GetUserByID(ctx context.Context, id string) (*User, error) {
	log.Infof("GetUserByID: %v", id)
	return uc.repo.GetUserByID(ctx, id)
}

// GetUsers retrieves all users.
func (uc *UserUsecase) GetUsers(ctx context.Context) ([]*User, error) {
	log.Info("GetUsers")
	return uc.repo.GetUsers(ctx)
}
