package memcache

import (
	"context"

	"github.com/coffeehc/base/log"
	"github.com/coocood/freecache"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

type Service interface {
	Get(ctx context.Context, key string, target *[]byte) bool
	Set(ctx context.Context, key string, target []byte, expireSeconds int64)
	// 内部使用json编码
	GetWithCoder(ctx context.Context, key string, target interface{}, coder Coder) bool
	// 内部使用json编码
	SetWithCoder(ctx context.Context, key string, target interface{}, coder Coder, expireSeconds int64)
	Del(ctx context.Context, key string)
	GetCache() *freecache.Cache
}

func newService() Service {
	size := viper.GetInt(DefaultConfigKey_MemcacheSize)
	disableCache := viper.GetBool(DefaultConfigKey_MemcacheDisable)
	return NewService(size, disableCache)
}

func NewService(memcacheSize int, disableCache bool) Service {
	impl := &serviceImpl{
		cache:        freecache.NewCache(memcacheSize),
		disableCache: disableCache,
	}
	return impl
}

type serviceImpl struct {
	cache        *freecache.Cache
	disableCache bool
}

func (impl *serviceImpl) GetWithCoder(ctx context.Context, key string, target interface{}, coder Coder) bool {
	data, err := impl.cache.Get([]byte(key))
	if err == nil {
		err = coder.Unmarshal(data, target)
		if err == nil {
			return true
		}
	}
	return false
}

func (impl *serviceImpl) SetWithCoder(ctx context.Context, key string, target interface{}, coder Coder, expireSeconds int64) {
	if impl.disableCache {
		return
	}
	data, err := coder.Marshal(target)
	if err != nil {
		log.Warn("序列化失败，无法缓存", zap.Error(err))
		return
	}
	err = impl.cache.Set([]byte(key), data, int(expireSeconds))
	if err != nil {
		log.Warn("存入缓存失败", zap.Error(err))
		return
	}
}

func (impl *serviceImpl) GetCache() *freecache.Cache {
	return impl.cache
}

func (impl *serviceImpl) Get(ctx context.Context, key string, target *[]byte) bool {
	data, err := impl.cache.Get([]byte(key))
	if err == nil {
		*target = data
	}
	return false
}

func (impl *serviceImpl) Set(ctx context.Context, key string, target []byte, expireSeconds int64) {
	if impl.disableCache {
		return
	}
	err := impl.cache.Set([]byte(key), target, int(expireSeconds))
	if err != nil {
		log.Warn("存入缓存失败", zap.Error(err))
		return
	}
}

func (impl *serviceImpl) Del(ctx context.Context, key string) {
	impl.cache.Del([]byte(key))
}
