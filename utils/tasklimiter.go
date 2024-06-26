package utils

import (
	"sync"
	"sync/atomic"
)

type TaskLimiter interface {
	Take()
	Recycle()
	SetSize(size int64)
	Size() int64
	WaitGroupAdd(delta int)
	WaitGroupDone()
	WaitGroupWait()
	Count() int64
}

func NewTaskLimiter(size int64) TaskLimiter {
	lock := &sync.Mutex{}
	impl := &taskLimiter{
		size: size,
		cond: sync.NewCond(lock),
	}
	return impl
}

type taskLimiter struct {
	size      int64
	count     int64
	cond      *sync.Cond
	waitGroup sync.WaitGroup
}

func (impl *taskLimiter) Count() int64 {
	return atomic.LoadInt64(&impl.count)
}

func (impl *taskLimiter) WaitGroupDone() {
	impl.waitGroup.Done()
}

func (impl *taskLimiter) WaitGroupWait() {
	impl.waitGroup.Wait()
}

func (impl *taskLimiter) WaitGroupAdd(delta int) {
	impl.waitGroup.Add(delta)
}

func (impl *taskLimiter) Size() int64 {
	return atomic.LoadInt64(&impl.size)
}

func (impl *taskLimiter) SetSize(size int64) {
	atomic.StoreInt64(&impl.size, size)
}

func (impl *taskLimiter) Take() {
	for impl.Count() >= impl.Size() {
		impl.wait()
	}
	atomic.AddInt64(&impl.count, 1)
	impl.cond.Signal()
}

func (impl *taskLimiter) Recycle() {
	for impl.Count() <= 0 {
		impl.wait()
	}
	atomic.AddInt64(&impl.count, -1)
	impl.cond.Signal()
}

func (impl *taskLimiter) wait() {
	impl.cond.L.Lock()
	impl.cond.Wait()
	impl.cond.L.Unlock()
}
