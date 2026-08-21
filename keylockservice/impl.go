package keylockservice

import (
	"container/list"
	"context"
	"reflect"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Service 提供进程内按 key 互斥的锁能力。
//
// 所有方法均可并发调用。key 必须是 Go 可比较类型；相同 key 串行执行，
// 不同 key 可以并发执行。成功获取锁后，调用方必须使用返回的 token 解锁。
type Service interface {
	// TryLock 尝试立即获取 key 对应的锁，不等待其他持锁者释放。
	TryLock(key interface{}) (string, bool)
	// LockWithTimeout 等待获取 key 对应的锁，并在成功持锁 lockTimeout 后自动释放。
	// ctx 只控制等待获取锁的过程，不改变已经取得的锁。
	LockWithTimeout(ctx context.Context, key interface{}, lockTimeout time.Duration) (string, error)
	// Lock 等待获取 key 对应的锁，ctx 取消时停止等待。
	Lock(ctx context.Context, key interface{}) (string, error)
	// UnLock 使用获取锁时返回的 token 释放 key；无效或过期 token 不产生影响。
	UnLock(key interface{}, token string)
}

// lockWaiter 表示一个已经进入指定 key 等待队列的锁请求。
type lockWaiter struct {
	// token 是请求成功取得锁后用于释放所有权的 UUID。
	token string
	// lockTimeout 是成功持锁后的自动释放时长。
	lockTimeout time.Duration
	// timed 表示请求是否需要在成功持锁后自动释放。
	timed bool
	// ready 在锁所有权按队列顺序移交给当前请求时关闭。
	ready chan struct{}
	// element 标识请求仍在等待队列中；所有权移交或取消时置空。
	element *list.Element
}

// lockState 表示一个 key 当前的持锁者和等待队列。
type lockState struct {
	// mutex 串行化当前 key 的所有权移交和等待取消。
	mutex sync.Mutex
	// retired 表示当前状态已经从 service 中删除，迟到的调用方必须重新查找。
	retired bool
	// token 是当前持锁者的 UUID。
	token string
	// timer 负责当前持锁者的可选自动释放。
	timer *time.Timer
	// waiters 按请求到达顺序保存等待者。
	waiters list.List
}

// serviceImpl 维护进程内所有 key 的锁状态。
type serviceImpl struct {
	// locks 只保存当前已持有或仍有等待者的 key，不同 key 使用独立状态锁。
	locks sync.Map
}

var _ Service = (*serviceImpl)(nil)

func newService(_ context.Context) Service {
	return &serviceImpl{}
}

func (impl *serviceImpl) LockWithTimeout(ctx context.Context, key interface{}, lockTimeout time.Duration) (string, error) {
	return impl.lock(ctx, key, lockTimeout, true)
}

func (impl *serviceImpl) Lock(ctx context.Context, key interface{}) (string, error) {
	return impl.lock(ctx, key, 0, false)
}

// lock 立即取得空闲锁，或按到达顺序等待当前持锁者直接移交所有权。
func (impl *serviceImpl) lock(ctx context.Context, key interface{}, lockTimeout time.Duration, timed bool) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	validateKey(key)
	waiter := &lockWaiter{
		token:       uuid.New().String(),
		lockTimeout: lockTimeout,
		timed:       timed,
		ready:       make(chan struct{}),
	}

	var state *lockState
	for {
		value, _ := impl.locks.LoadOrStore(key, &lockState{})
		state = value.(*lockState)
		state.mutex.Lock()
		if state.retired {
			state.mutex.Unlock()
			continue
		}
		if state.token == "" {
			state.token = waiter.token
			impl.startUnlockTimerLocked(key, state, waiter)
			state.mutex.Unlock()
			return waiter.token, nil
		}
		waiter.element = state.waiters.PushBack(waiter)
		state.mutex.Unlock()
		break
	}

	select {
	case <-waiter.ready:
		return waiter.token, nil
	case <-ctx.Done():
		state.mutex.Lock()
		if waiter.element != nil {
			state.waiters.Remove(waiter.element)
			waiter.element = nil
			state.mutex.Unlock()
			return "", ctx.Err()
		}
		state.mutex.Unlock()
		// 所有权已经移交时优先返回锁，保持原实现可能在取消竞争中成功获取的语义。
		<-waiter.ready
		return waiter.token, nil
	}
}

func (impl *serviceImpl) TryLock(key interface{}) (string, bool) {
	validateKey(key)
	token := uuid.New().String()
	for {
		value, _ := impl.locks.LoadOrStore(key, &lockState{})
		state := value.(*lockState)
		state.mutex.Lock()
		if state.retired {
			state.mutex.Unlock()
			continue
		}
		if state.token != "" {
			state.mutex.Unlock()
			return token, false
		}
		state.token = token
		state.mutex.Unlock()
		return token, true
	}
}

func (impl *serviceImpl) UnLock(key interface{}, token string) {
	validateKey(key)
	value, exists := impl.locks.Load(key)
	if !exists {
		return
	}
	state := value.(*lockState)
	state.mutex.Lock()
	defer state.mutex.Unlock()
	if state.retired || state.token != token {
		return
	}
	if state.timer != nil {
		state.timer.Stop()
		state.timer = nil
	}

	front := state.waiters.Front()
	if front == nil {
		state.token = ""
		state.retired = true
		impl.locks.CompareAndDelete(key, state)
		return
	}
	waiter := front.Value.(*lockWaiter)
	state.waiters.Remove(front)
	waiter.element = nil
	state.token = waiter.token
	impl.startUnlockTimerLocked(key, state, waiter)
	close(waiter.ready)
}

// startUnlockTimerLocked 为已经取得所有权的定时锁注册自动释放任务。
func (impl *serviceImpl) startUnlockTimerLocked(key interface{}, state *lockState, waiter *lockWaiter) {
	if !waiter.timed {
		return
	}
	state.timer = time.AfterFunc(waiter.lockTimeout, func() {
		impl.UnLock(key, waiter.token)
	})
}

// validateKey 保留原 sync.Map 对不可比较 key 的 panic 契约，但避免在持锁后 panic。
func validateKey(key interface{}) {
	keyType := reflect.TypeOf(key)
	if keyType != nil && !keyType.Comparable() {
		panic("keylockservice: key is not comparable")
	}
}
