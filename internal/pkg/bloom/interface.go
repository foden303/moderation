package bloom

import "context"

type bitSetProvider interface {
	check(ctx context.Context, offsets []uint) (bool, error)
	set(ctx context.Context, offsets []uint) error
	del(ctx context.Context) error
	expire(ctx context.Context, seconds int) (bool, error)
}
