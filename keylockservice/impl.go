package keylockservice

import (
	"context"
	"github.com/google/uuid"
	"sync"
	"time"
)

type lockBody struct {
	token string
}

type Service interface {
	TryLock(key interface{}) (string, bool)
	LockWithTimeout(ctx context.Context, key interface{}, lockTimeout time.Duration) (string, error)
	Lock(ctx context.Context, key interface{}) (string, error)
	UnLock(key interface{}, token string)
}

func newService(ctx context.Context) Service {
	impl := &serviceImpl{}
	return impl
}

type serviceImpl struct {
	locks sync.Map
}

func (impl *serviceImpl) LockWithTimeout(ctx context.Context, key interface{}, lockTimeout time.Duration) (string, error) {
	token, err := impl.Lock(ctx, key)
	if err == nil {
		go func() {
			time.AfterFunc(lockTimeout, func() {
				impl.UnLock(key, token)
			})
		}()
	}
	return token, err
}

func (impl *serviceImpl) Lock(ctx context.Context, key interface{}) (string, error) {
	for {
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		token, ok := impl.TryLock(key)
		if ok {
			return token, nil
		}
		time.Sleep(time.Millisecond * 50)
	}
}

func (impl *serviceImpl) TryLock(key interface{}) (string, bool) {
	body := &lockBody{
		token: uuid.New().String(),
	}
	_, loaded := impl.locks.LoadOrStore(key, body)
	return body.token, !loaded
}

func (impl *serviceImpl) UnLock(key interface{}, token string) {
	value, loaded := impl.locks.Load(key)
	if !loaded {
		return
	}
	if value.(*lockBody).token == token {
		impl.locks.Delete(key)
	}
}
