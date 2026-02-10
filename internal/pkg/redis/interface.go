package redis

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

type Cache interface {
	SetString(ctx context.Context, key, value string, exp time.Duration) error
	GetString(ctx context.Context, key string) (string, error)

	SetInt(ctx context.Context, key string, value int, exp time.Duration) error
	GetInt(ctx context.Context, key string) (int, error)

	SetInt32(ctx context.Context, key string, value int32, exp time.Duration) error
	GetInt32(ctx context.Context, key string) (int32, error)

	SetInt64(ctx context.Context, key string, value int64, exp time.Duration) error
	GetInt64(ctx context.Context, key string) (int64, error)

	SetUint(ctx context.Context, key string, value uint, exp time.Duration) error
	GetUint(ctx context.Context, key string) (uint, error)

	SetUint32(ctx context.Context, key string, value uint32, exp time.Duration) error
	GetUint32(ctx context.Context, key string) (uint32, error)

	SetUint64(ctx context.Context, key string, value uint64, exp time.Duration) error
	GetUint64(ctx context.Context, key string) (uint64, error)

	Exists(ctx context.Context, key string) (bool, error)

	ScriptRun(ctx context.Context, script *redis.Script, keys []string,
		args ...any) (any, error)

	Del(ctx context.Context, keys ...string) (int64, error)

	Expire(ctx context.Context, key string, seconds int) (bool, error)
}
