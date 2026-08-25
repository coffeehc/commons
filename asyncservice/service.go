package asyncservice

import (
	"context"
	"time"
)

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
	// Paused reports whether accepted tasks are temporarily prevented from starting.
	Paused bool
}

// Controller 在不扩展稳定 Service 接口的前提下提供进程级任务准入控制。
type Controller interface {
	// Suspend 允许已接收任务排空，并挂起此后接收的新任务。
	Suspend()
	// WaitIdle 等待挂起前已经接收的任务全部退出。
	WaitIdle(ctx context.Context) error
	// Resume 恢复执行挂起期间接收的任务。
	Resume()
	// SubmitControl 在可挂起任务池之外启动一个服务生命周期控制任务。
	// 服务开始停止后返回 false。
	SubmitControl(task func()) bool
	// SubmitIfRunning 仅在任务池未挂起时接收任务。
	SubmitIfRunning(task func()) bool
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
