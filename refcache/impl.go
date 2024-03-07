package refcache

import (
	"context"
	"sync"
	"time"
)

type Service interface {
	Set(key string, value interface{}, timeout time.Duration)
	Del(key string)
	Get(key string) (interface{}, bool)
}

func newService(ctx context.Context) Service {
	impl := &serviceImpl{}
	return impl
}

type serviceImpl struct {
	cache    sync.Map
	expTimes sync.Map
}

func (impl *serviceImpl) Set(key string, value interface{}, timeout time.Duration) {
	impl.cache.Store(key, value)
	impl.expTimes.Store(key, time.Now().Add(timeout).UnixMilli())
}

func (impl *serviceImpl) Del(key string) {
	impl.cache.Delete(key)
	impl.expTimes.Delete(key)
}

func (impl *serviceImpl) Get(key string) (interface{}, bool) {
	return impl.cache.Load(key)
}

func (impl *serviceImpl) Start(ctx context.Context) error {
	go impl.monit()
	return nil
}

func (impl *serviceImpl) Stop(ctx context.Context) error {
	return nil
}

func (impl *serviceImpl) monit() {
	for {
		time.Sleep(time.Second * 10)
		now := time.Now().UnixMilli()
		impl.expTimes.Range(func(key, value any) bool {
			if now > value.(int64) {
				impl.Del(key.(string))
			}
			return true
		})
	}
}
