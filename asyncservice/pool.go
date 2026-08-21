package asyncservice

import (
	"context"
	"runtime/debug"
	"sync"

	"github.com/coffeehc/base/log"
	"go.uber.org/zap"
)

// taskPool queues accepted tasks and starts them under a dynamic concurrency limit.
type taskPool struct {
	// mutex protects all pool state except taskGroup's internal counters.
	mutex sync.Mutex
	// condition wakes the dispatcher after submissions, completions or limit changes.
	condition *sync.Cond
	// limit is the maximum number of tasks that may run concurrently.
	limit int
	// running is the number of tasks currently executing.
	running int
	// queue retains accepted tasks in FIFO order until they can start.
	queue []func()
	// stopping prevents new submissions while already accepted tasks drain.
	stopping bool
	// taskGroup tracks the dispatcher and every task goroutine it starts.
	taskGroup sync.WaitGroup
	// stopped is closed after the dispatcher drains all accepted tasks and exits.
	stopped chan struct{}
}

// newTaskPool starts the single dispatcher that owns task admission to running state.
func newTaskPool(limit int) *taskPool {
	pool := &taskPool{
		limit:   limit,
		queue:   make([]func(), 0),
		stopped: make(chan struct{}),
	}
	pool.condition = sync.NewCond(&pool.mutex)
	pool.taskGroup.Go(pool.dispatch)
	return pool
}

// Submit retains task unless shutdown has started. Queueing stays non-blocking
// so a running task can safely submit follow-up work at a pool limit of one.
func (pool *taskPool) Submit(task func()) bool {
	if task == nil {
		panic("asyncservice: task must not be nil")
	}
	pool.mutex.Lock()
	defer pool.mutex.Unlock()
	if pool.stopping {
		return false
	}
	pool.queue = append(pool.queue, task)
	pool.condition.Signal()
	return true
}

// ChangeLimit updates the running-task ceiling without interrupting active tasks.
func (pool *taskPool) ChangeLimit(limit int) bool {
	if limit <= 0 {
		return false
	}
	pool.mutex.Lock()
	defer pool.mutex.Unlock()
	if pool.stopping {
		return false
	}
	pool.limit = limit
	pool.condition.Broadcast()
	return true
}

// Status captures all counters under one lock so callers receive a coherent snapshot.
func (pool *taskPool) Status() PoolStatus {
	pool.mutex.Lock()
	defer pool.mutex.Unlock()
	available := pool.limit - pool.running
	if available < 0 {
		available = 0
	}
	return PoolStatus{
		Limit:     pool.limit,
		Running:   pool.running,
		Waiting:   len(pool.queue),
		Available: available,
		Stopped:   pool.stopping,
	}
}

// Stop closes admission and waits for the dispatcher to drain previously accepted tasks.
func (pool *taskPool) Stop(ctx context.Context) error {
	pool.mutex.Lock()
	if !pool.stopping {
		pool.stopping = true
		pool.condition.Broadcast()
	}
	stopped := pool.stopped
	pool.mutex.Unlock()

	select {
	case <-stopped:
		pool.taskGroup.Wait()
		return nil
	default:
	}
	select {
	case <-stopped:
		pool.taskGroup.Wait()
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// dispatch is the only transition path from queued to running. Keeping that
// decision serial avoids overshooting the limit during concurrent resizing.
func (pool *taskPool) dispatch() {
	pool.mutex.Lock()
	defer pool.mutex.Unlock()
	for {
		for len(pool.queue) == 0 || pool.running >= pool.limit {
			if pool.stopping && len(pool.queue) == 0 && pool.running == 0 {
				close(pool.stopped)
				return
			}
			pool.condition.Wait()
		}

		task := pool.queue[0]
		pool.queue[0] = nil
		pool.queue = pool.queue[1:]
		if len(pool.queue) == 0 {
			pool.queue = nil
		}
		pool.running++
		pool.taskGroup.Go(func() {
			pool.run(task)
		})
	}
}

// run isolates task panics so WaitGroup.Go always observes a normal return.
func (pool *taskPool) run(task func()) {
	defer pool.complete()
	defer func() {
		if recovered := recover(); recovered != nil {
			log.Error("异步任务执行异常",
				zap.Any("panic", recovered),
				zap.ByteString("stack", debug.Stack()))
		}
	}()
	task()
}

// complete releases one running slot and wakes the dispatcher after task exit.
func (pool *taskPool) complete() {
	pool.mutex.Lock()
	pool.running--
	pool.condition.Broadcast()
	pool.mutex.Unlock()
}
