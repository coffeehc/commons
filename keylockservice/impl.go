package keylockservice

import (
	"context"
	"sync"
	"time"
)

var baseStruct = &struct{}{}

type Service interface {
	TryLock(key interface{}) bool
	Lock(key interface{}, timeout time.Duration) error
	UnLock(key interface{})
}

func newService(ctx context.Context) Service {
	impl := &serviceImpl{}
	return impl
}

type serviceImpl struct {
	locks sync.Map
}

func (impl *serviceImpl) Lock(key interface{}, timeout time.Duration) error {
	ctx, cancelFunc := context.WithTimeout(context.TODO(), timeout)
	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		ok := impl.TryLock(key)
		if ok {
			cancelFunc()
			return nil
		}
		time.Sleep(time.Millisecond * 50)
	}
}

func (impl *serviceImpl) TryLock(key interface{}) bool {
	_, loaded := impl.locks.LoadOrStore(key, baseStruct)
	return !loaded
}

func (impl *serviceImpl) UnLock(key interface{}) {
	impl.locks.Delete(key)
}
