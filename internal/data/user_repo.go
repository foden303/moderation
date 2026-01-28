package data

import (
	"context"
	"storage/internal/biz"
	"storage/internal/data/postgres/sqlc"

	"github.com/go-kratos/kratos/v2/log"
)

type userRepo struct {
	data *Data
	log  *log.Helper
}

// GetUserByID implements [biz.UserRepo].
func (r *userRepo) GetUserByID(ctx context.Context, id string) (*biz.User, error) {
	user, err := r.data.Queries.GetUserByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return r.toBizUser(user), nil
}

// GetUsers implements [biz.UserRepo].
func (r *userRepo) GetUsers(ctx context.Context) ([]*biz.User, error) {
	// TODO: Add GetUsers query in sqlc
	return nil, nil
}

// UpsertUser implements [biz.UserRepo].
func (r *userRepo) UpsertUser(ctx context.Context, u *biz.User) (*biz.User, error) {
	user, err := r.data.Queries.UpsertUser(ctx, sqlc.UpsertUserParams{
		ID:                  u.ID,
		Nickname:            u.Name,
		UserIdentifier:      u.UserIdentifier,
		StoragePhotosUsed:   u.StoragePhotosUsed,
		StorageVideoUsed:    u.StorageVideoUsed,
		StorageDocumentUsed: u.StorageDocumentUsed,
		StorageAudioUsed:    u.StorageAudioUsed,
		StorageCompressUsed: u.StorageCompressUsed,
		StorageOtherUsed:    u.StorageOtherUsed,
	})
	if err != nil {
		return nil, err
	}
	return r.toBizUser(user), nil
}

// toBizUser converts sqlc.User to biz.User
func (r *userRepo) toBizUser(u sqlc.User) *biz.User {
	return &biz.User{
		ID:                    u.ID,
		Name:                  u.Nickname,
		UserIdentifier:        u.UserIdentifier,
		EffectiveStorageQuota: u.EffectiveStorageQuota,
		StoragePhotosUsed:     u.StoragePhotosUsed,
		StorageVideoUsed:      u.StorageVideoUsed,
		StorageDocumentUsed:   u.StorageDocumentUsed,
		StorageAudioUsed:      u.StorageAudioUsed,
		StorageCompressUsed:   u.StorageCompressUsed,
		StorageOtherUsed:      u.StorageOtherUsed,
		StorageTotalUsed:      u.StorageTotalUsed,
		CreatedAt:             u.CreatedAt.Time,
		UpdatedAt:             u.UpdatedAt.Time,
	}
}

func NewUserRepo(data *Data, logger log.Logger) biz.UserRepo {
	return &userRepo{
		data: data,
		log:  log.NewHelper(logger),
	}
}
