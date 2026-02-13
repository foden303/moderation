package data

import (
	"context"
	"fmt"
	"time"

	"moderation/internal/conf"
	pkgredis "moderation/internal/pkg/redis"

	"github.com/go-kratos/kratos/v2/log"
	redis "github.com/redis/go-redis/v9"
)

// NewRedisCache creates a new Redis cache from configuration.
func NewRedisCache(c *conf.Data, logger log.Logger) (pkgredis.Cache, func(), error) {
	helper := log.NewHelper(logger)

	// Build connection options from config
	opts := &redis.Options{
		Addr:    c.Redis.Addr,
		Network: c.Redis.Network,
	}

	if c.Redis.ReadTimeout != nil {
		opts.ReadTimeout = c.Redis.ReadTimeout.AsDuration()
	}
	if c.Redis.WriteTimeout != nil {
		opts.WriteTimeout = c.Redis.WriteTimeout.AsDuration()
	}

	client := redis.NewClient(opts)

	// Test connection with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		helper.Errorf("failed to connect to Redis at %s: %v", c.Redis.Addr, err)
		return nil, nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	helper.Infof("connected to Redis at %s", c.Redis.Addr)

	// Create cache wrapper
	cache := NewRedisWrapper(client)
	cleanup := func() {
		helper.Info("closing Redis connection")
		client.Close()
	}

	return cache, cleanup, nil
}

// RedisWrapper wraps redis.Client to implement pkgredis.Cache interface.
type RedisWrapper struct {
	client *redis.Client
}

// NewRedisWrapper creates a new RedisWrapper.
func NewRedisWrapper(client *redis.Client) *RedisWrapper {
	return &RedisWrapper{client: client}
}

// Implement all Cache interface methods by delegating to the underlying Redis struct
func (r *RedisWrapper) SetString(ctx context.Context, key, value string, exp time.Duration) error {
	return r.client.Set(ctx, key, value, exp).Err()
}

func (r *RedisWrapper) GetString(ctx context.Context, key string) (string, error) {
	return r.client.Get(ctx, key).Result()
}

func (r *RedisWrapper) SetBytes(ctx context.Context, key string, value []byte, exp time.Duration) error {
	return r.client.Set(ctx, key, value, exp).Err()
}

func (r *RedisWrapper) GetBytes(ctx context.Context, key string) ([]byte, error) {
	return r.client.Get(ctx, key).Bytes()
}

func (r *RedisWrapper) SetInt(ctx context.Context, key string, value int, exp time.Duration) error {
	return r.client.Set(ctx, key, value, exp).Err()
}

func (r *RedisWrapper) GetInt(ctx context.Context, key string) (int, error) {
	return r.client.Get(ctx, key).Int()
}

func (r *RedisWrapper) SetInt32(ctx context.Context, key string, value int32, exp time.Duration) error {
	return r.client.Set(ctx, key, value, exp).Err()
}

func (r *RedisWrapper) GetInt32(ctx context.Context, key string) (int32, error) {
	val, err := r.client.Get(ctx, key).Int64()
	return int32(val), err
}

func (r *RedisWrapper) SetInt64(ctx context.Context, key string, value int64, exp time.Duration) error {
	return r.client.Set(ctx, key, value, exp).Err()
}

func (r *RedisWrapper) GetInt64(ctx context.Context, key string) (int64, error) {
	return r.client.Get(ctx, key).Int64()
}

func (r *RedisWrapper) SetUint(ctx context.Context, key string, value uint, exp time.Duration) error {
	return r.client.Set(ctx, key, value, exp).Err()
}

func (r *RedisWrapper) GetUint(ctx context.Context, key string) (uint, error) {
	val, err := r.client.Get(ctx, key).Uint64()
	return uint(val), err
}

func (r *RedisWrapper) SetUint32(ctx context.Context, key string, value uint32, exp time.Duration) error {
	return r.client.Set(ctx, key, value, exp).Err()
}

func (r *RedisWrapper) GetUint32(ctx context.Context, key string) (uint32, error) {
	val, err := r.client.Get(ctx, key).Uint64()
	return uint32(val), err
}

func (r *RedisWrapper) SetUint64(ctx context.Context, key string, value uint64, exp time.Duration) error {
	return r.client.Set(ctx, key, value, exp).Err()
}

func (r *RedisWrapper) GetUint64(ctx context.Context, key string) (uint64, error) {
	return r.client.Get(ctx, key).Uint64()
}

func (r *RedisWrapper) Exists(ctx context.Context, key string) (bool, error) {
	count, err := r.client.Exists(ctx, key).Result()
	return count > 0, err
}

func (r *RedisWrapper) ScriptRun(ctx context.Context, script *redis.Script, keys []string, args ...any) (any, error) {
	conn := r.client.Conn()
	defer conn.Close()
	return script.Run(ctx, conn, keys, args...).Result()
}

func (r *RedisWrapper) Del(ctx context.Context, keys ...string) (int64, error) {
	return r.client.Del(ctx, keys...).Result()
}

func (r *RedisWrapper) Expire(ctx context.Context, key string, seconds int) (bool, error) {
	return r.client.Expire(ctx, key, time.Duration(seconds)*time.Second).Result()
}
