package redisservice

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

type virtualServiceImpl struct {
	redis.Cmdable
}

func (impl *virtualServiceImpl) GetRedisClient() *redis.Client {
	return nil
}

func (impl *virtualServiceImpl) Del(ctx context.Context, keys ...string) *redis.IntCmd {
	return &redis.IntCmd{}
}

func (impl *virtualServiceImpl) Exists(ctx context.Context, keys ...string) *redis.IntCmd {
	return &redis.IntCmd{}
}
func (impl *virtualServiceImpl) Expire(ctx context.Context, key string, expiration time.Duration) *redis.BoolCmd {
	return &redis.BoolCmd{}
}
func (impl *virtualServiceImpl) ExpireAt(ctx context.Context, key string, tm time.Time) *redis.BoolCmd {
	return &redis.BoolCmd{}
}
func (impl *virtualServiceImpl) Keys(ctx context.Context, pattern string) *redis.StringSliceCmd {
	return &redis.StringSliceCmd{}
}
func (impl *virtualServiceImpl) Get(ctx context.Context, key string) *redis.StringCmd {
	return &redis.StringCmd{}
}
func (impl *virtualServiceImpl) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd {
	return &redis.StatusCmd{}
}
func (impl *virtualServiceImpl) HDel(ctx context.Context, key string, fields ...string) *redis.IntCmd {
	return &redis.IntCmd{}
}
func (impl *virtualServiceImpl) HExists(ctx context.Context, key, field string) *redis.BoolCmd {
	return &redis.BoolCmd{}
}
func (impl *virtualServiceImpl) HGet(ctx context.Context, key, field string) *redis.StringCmd {
	return &redis.StringCmd{}
}
func (impl *virtualServiceImpl) HGetAll(ctx context.Context, key string) *redis.MapStringStringCmd {
	return &redis.MapStringStringCmd{}
}
func (impl *virtualServiceImpl) HIncrBy(ctx context.Context, key, field string, incr int64) *redis.IntCmd {
	return &redis.IntCmd{}
}
func (impl *virtualServiceImpl) Eval(ctx context.Context, script string, keys []string, args ...interface{}) *redis.Cmd {
	return &redis.Cmd{}
}
func (impl *virtualServiceImpl) ScriptLoad(ctx context.Context, script string) *redis.StringCmd {
	return &redis.StringCmd{}
}
