package redis

import (
	"context"
	"log"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

type Redis struct {
	client *redis.Client
}

const Nil = redis.Nil

func New(url string) Cache {
	// 1. Prepare Redis client configurations
	opts, err := redis.ParseURL(url)
	if err != nil {
		log.Fatal(err)
	}
	// 2. Create a new Redis client
	return &Redis{
		client: redis.NewClient(opts),
	}
}

// NewScript implements Cache.
func NewScript(script string) *redis.Script {
	return redis.NewScript(script)
}

func (r *Redis) SetString(ctx context.Context, key, value string, exp time.Duration) error {
	if err := r.client.Set(ctx, key, value, exp).Err(); err != nil {
		return err
	}
	return nil
}

func (r *Redis) GetString(ctx context.Context, key string) (string, error) {
	val, err := r.client.Get(ctx, key).Result()
	if err != nil {
		return "", err
	}
	return val, nil
}
func (r *Redis) SetUint(ctx context.Context, key string, value uint, exp time.Duration) error {
	if err := r.client.Set(ctx, key, value, exp).Err(); err != nil {
		return err
	}
	return nil
}

func (r *Redis) GetUint(ctx context.Context, key string) (uint, error) {
	valStr, err := r.client.Get(ctx, key).Result()
	if err != nil {
		return 0, err
	}
	val, err := strconv.ParseUint(valStr, 10, 64)
	if err != nil {
		return 0, err
	}
	return uint(val), nil
}

func (r *Redis) SetUint32(ctx context.Context, key string, value uint32, exp time.Duration) error {
	if err := r.client.Set(ctx, key, value, exp).Err(); err != nil {
		return err
	}
	return nil
}

func (r *Redis) GetUint32(ctx context.Context, key string) (uint32, error) {
	valStr, err := r.client.Get(ctx, key).Result()
	if err != nil {
		return 0, err
	}
	val, err := strconv.ParseUint(valStr, 10, 32)
	if err != nil {
		return 0, err
	}
	return uint32(val), nil
}
func (r *Redis) SetUint64(ctx context.Context, key string, value uint64, exp time.Duration) error {
	if err := r.client.Set(ctx, key, value, exp).Err(); err != nil {
		return err
	}
	return nil
}

func (r *Redis) GetUint64(ctx context.Context, key string) (uint64, error) {
	val, err := r.client.Get(ctx, key).Uint64()
	if err != nil {
		return 0, err
	}
	return val, nil
}

func (r *Redis) SetInt64(ctx context.Context, key string, value int64, exp time.Duration) error {
	if err := r.client.Set(ctx, key, value, exp).Err(); err != nil {
		return err
	}
	return nil
}

func (r *Redis) GetInt64(ctx context.Context, key string) (int64, error) {
	val, err := r.client.Get(ctx, key).Int64()
	if err != nil {
		return 0, err
	}
	return val, nil
}

func (r *Redis) SetInt32(ctx context.Context, key string, value int32, exp time.Duration) error {
	if err := r.client.Set(ctx, key, value, exp).Err(); err != nil {
		return err
	}
	return nil
}

func (r *Redis) GetInt32(ctx context.Context, key string) (int32, error) {
	valStr, err := r.client.Get(ctx, key).Result()
	if err != nil {
		return 0, err
	}
	val, err := strconv.ParseInt(valStr, 10, 32)
	if err != nil {
		return 0, err
	}
	return int32(val), nil
}

func (r *Redis) SetInt(ctx context.Context, key string, value int, exp time.Duration) error {
	if err := r.client.Set(ctx, key, value, exp).Err(); err != nil {
		return err
	}
	return nil
}

func (r *Redis) GetInt(ctx context.Context, key string) (int, error) {
	val, err := r.client.Get(ctx, key).Int()
	if err != nil {
		return 0, err
	}
	return val, nil
}

// Expire implements Cache.
func (i *Redis) Expire(ctx context.Context, key string, seconds int) (bool, error) {
	return i.client.Expire(ctx, key, time.Duration(seconds)*time.Second).Result()
}

func (i *Redis) Exists(ctx context.Context, key string) (bool, error) {
	count, err := i.client.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}
	if count > 0 {
		return true, nil
	}
	return false, nil
}

// Del implements Cache.
func (i *Redis) Del(ctx context.Context, keys ...string) (int64, error) {
	return i.client.Del(ctx, keys...).Result()
}

// ScriptRun implements Cache.
func (i *Redis) ScriptRun(ctx context.Context, script *redis.Script, keys []string, args ...any) (any, error) {
	conn := i.client.Conn()
	defer conn.Close()

	return script.Run(ctx, conn, keys, args...).Result()
}
