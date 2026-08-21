package asyncservice

import "time"

// Timer controls one delayed or periodic callback. Implementations are safe
// for concurrent use and do not wait for an already running callback.
type Timer interface {
	// Stop prevents future callbacks and reports whether the timer was active.
	Stop() bool
}

// PoolStatus is an immutable snapshot of asynchronous task pool state.
type PoolStatus struct {
	// Limit is the current maximum number of concurrently running tasks.
	Limit int
	// Running is the number of tasks currently executing.
	Running int
	// Waiting is the number of accepted tasks waiting to start.
	Waiting int
	// Available is the number of tasks that can start without waiting.
	Available int
	// Stopped reports whether the pool has stopped accepting new tasks.
	Stopped bool
}

// Service owns delayed scheduling and bounded asynchronous task execution.
// All methods are safe for concurrent use. The service retains accepted tasks
// until execution and only rejects submissions after shutdown begins.
type Service interface {
	// Schedule registers task for periodic asynchronous execution. Interval must
	// be positive and Stop prevents future submissions from this schedule.
	Schedule(interval time.Duration, task func()) Timer
	// AfterFunc registers task for one asynchronous execution after delay.
	AfterFunc(delay time.Duration, task func()) Timer
	// Submit accepts task for asynchronous execution. A nil task causes a panic.
	Submit(task func())
	// ChangePoolSize changes the execution concurrency limit. Size must be positive.
	ChangePoolSize(size int)
	// PoolStatus returns a thread-safe snapshot of task pool state.
	PoolStatus() PoolStatus
}
