package bufpool

import (
	"context"
	"sync"
)

type Service interface {
	GetByte(size int) []byte
	Put([]byte)
}

func newService(_ context.Context) Service {
	impl := &serviceImpl{}
	return impl
}

type serviceImpl struct {
	poolFactory sync.Map
}

func (impl *serviceImpl) GetByte(size int) []byte {
	pool := impl.getPool(size)
	return pool.Get().([]byte)
}

func (impl *serviceImpl) getPool(size int) *sync.Pool {
	v, loaded := impl.poolFactory.Load(size)
	if !loaded {
		pool := &sync.Pool{}
		pool.New = func() any {
			return make([]byte, size)
		}
		v, loaded = impl.poolFactory.LoadOrStore(size, pool)
		if loaded {
			return v.(*sync.Pool)
		}
		return pool
	}
	return v.(*sync.Pool)
}

func (impl *serviceImpl) Put(buf []byte) {
	size := len(buf)
	pool := impl.getPool(size)
	pool.Put(buf)
}

func (impl *serviceImpl) Start(_ context.Context) error {
	return nil
}

func (impl *serviceImpl) Stop(_ context.Context) error {
	return nil
}
