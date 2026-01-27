package data

import (
	"context"
	"database/sql"
	"storage/internal/biz"
)

type userRepo struct {
	db *sql.DB
	q  *sqlc.Queries
}

func NewUserRepo(data *Data) biz.UserRepo {
	return &userRepo{
		db: data.DB,
		q:  sqlc.New(data.DB),
	}
}

func (r *userRepo) Get(ctx context.Context, id int64) (*biz.User, error) {
	u, err := r.q.GetUser(ctx, id)
	if err != nil {
		return nil, err
	}

	return &biz.User{
		ID:   u.ID,
		Name: u.Name,
	}, nil
}
