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
	Save(context.Context, *User) (*User, error)
	Update(context.Context, *User) (*User, error)
	FindByID(context.Context, int64) (*User, error)
	ListByHello(context.Context, string) ([]*User, error)
	ListAll(context.Context) ([]*User, error)
}

// UserUsecase is a User usecase.
type UserUsecase struct {
	repo UserRepo
}

// NewUserUsecase new a User usecase.
func NewUserUsecase(repo UserRepo) *UserUsecase {
	return &UserUsecase{repo: repo}
}

// CreateUser creates a User, and returns the new User.
func (uc *UserUsecase) CreateUser(ctx context.Context, u *User) (*User, error) {
	log.Infof("CreateUser: %v", u.Name)
	return uc.repo.Save(ctx, u)
}
