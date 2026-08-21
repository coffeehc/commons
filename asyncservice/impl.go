package asyncservice

import (
	"context"
	"time"

	"github.com/coffeehc/base/log"
	"github.com/coffeehc/boot/plugin"
	"github.com/coffeehc/commons/timingwheel"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

const defaultPoolSize = 100000

// ConfigScopeKey is the viper scope used to load async service configuration.
const ConfigScopeKey = "sync_config"

// NewService creates an independent asynchronous execution service. A nil
// configuration or non-positive pool size uses the default concurrency limit.
func NewService(_ context.Context, config *Config) Service {
	poolSize := defaultPoolSize
	if config != nil && config.PoolSize > 0 {
		poolSize = config.PoolSize
	}
	return &serviceImpl{
		timingWheel: timingwheel.New(),
		pool:        newTaskPool(poolSize),
	}
}

func newService(ctx context.Context) Service {
	config := &Config{PoolSize: defaultPoolSize}
	if err := viper.UnmarshalKey(ConfigScopeKey, config); err != nil {
		log.Error("加载异步服务配置失败", zap.Error(err))
		return nil
	}
	return NewService(ctx, config)
}

// serviceImpl owns asynchronous task admission, concurrency and timer lifecycle.
type serviceImpl struct {
	// timingWheel owns all delayed and periodic callback registrations.
	timingWheel *timingwheel.Wheel
	// pool owns task admission, execution concurrency and shutdown draining.
	pool *taskPool
}

var _ plugin.Plugin = (*serviceImpl)(nil)

// Submit accepts task for asynchronous execution. Tasks accepted before
// shutdown are retained in FIFO order until the concurrency limit allows them
// to start. A nil task causes a panic.
func (impl *serviceImpl) Submit(task func()) {
	impl.pool.Submit(task)
}

// ChangePoolSize changes the maximum number of concurrently running tasks.
// Shrinking the limit does not cancel tasks that are already running.
func (impl *serviceImpl) ChangePoolSize(size int) {
	if !impl.pool.ChangeLimit(size) {
		log.Warn("修改异步任务并发上限失败", zap.Int("size", size))
	}
}

// PoolStatus returns a thread-safe snapshot of current task pool state.
func (impl *serviceImpl) PoolStatus() PoolStatus {
	return impl.pool.Status()
}

// Start starts the plugin lifecycle. The task dispatcher is available from
// construction so independent services can submit tasks without plugin setup.
func (impl *serviceImpl) Start(_ context.Context) error {
	return nil
}

// Stop prevents future timer triggers and drains every task accepted before
// shutdown. It returns the context error when the caller stops waiting early.
func (impl *serviceImpl) Stop(ctx context.Context) error {
	impl.timingWheel.Stop()
	return impl.pool.Stop(ctx)
}

// Schedule registers a periodic callback. The callback only submits the task;
// actual execution remains controlled by the pool concurrency limit.
func (impl *serviceImpl) Schedule(interval time.Duration, task func()) Timer {
	return impl.timingWheel.Schedule(interval, func() {
		impl.Submit(task)
	})
}

// AfterFunc registers a one-shot delayed callback. The callback only submits
// the task; actual execution remains controlled by the pool concurrency limit.
func (impl *serviceImpl) AfterFunc(delay time.Duration, task func()) Timer {
	return impl.timingWheel.AfterFunc(delay, func() {
		impl.Submit(task)
	})
}
