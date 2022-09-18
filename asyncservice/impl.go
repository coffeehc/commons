package asyncservice

import (
	"context"
	"time"

	"github.com/RussellLuo/timingwheel"
	"github.com/coffeehc/base/log"
	"github.com/panjf2000/ants/v2"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

const ConfigScopeKey = "sync_config"

type Service interface {
	Schedule(duration time.Duration, do func()) *timingwheel.Timer
	AfterFunc(duration time.Duration, do func()) *timingwheel.Timer
	Submit(do func())
	ChangePoolSize(size int)
}

func NewService(_ context.Context, config *Config) Service {
	log.Debug("异步处理服务配置", zap.Any("config", config))
	pool, err := ants.NewPool(config.PoolSize,
		ants.WithPreAlloc(true),
		ants.WithPanicHandler(func(p interface{}) {
			log.DPanic("出现了不可处理的异常", zap.Any("err", p))
		}),
		ants.WithExpiryDuration(config.ExpiryDuration),
		ants.WithMaxBlockingTasks(config.MaxBlockingTasks),
	)
	if err != nil {
		log.Error("错误", zap.Error(err))
		return nil
	}
	impl := &serviceImpl{
		timingWheel: timingwheel.NewTimingWheel(time.Second, config.WheelSize),
		pool:        pool,
	}
	return impl
}

func newService(ctx context.Context) Service {
	config := &Config{
		PoolSize:         100000,
		ExpiryDuration:   time.Second * 60,
		MaxBlockingTasks: 1000,
		WheelSize:        10000,
	}
	value := viper.Get(ConfigScopeKey)
	if value != nil {
		config = value.(*Config)
	} else {
		err := viper.UnmarshalKey(ConfigScopeKey, config)
		if err != nil {
			log.Error("错误", zap.Error(err))
			return nil
		}
	}
	return NewService(ctx, config)
}

type serviceImpl struct {
	timingWheel *timingwheel.TimingWheel
	pool        *ants.Pool
}

func (impl *serviceImpl) GetPool() *ants.Pool {
	return impl.pool
}

func (impl *serviceImpl) Submit(do func()) {
	impl.pool.Submit(do)
}

func (impl *serviceImpl) ChangePoolSize(size int) {
	impl.pool.Tune(size)
}

func (impl *serviceImpl) Start(ctx context.Context) error {
	impl.timingWheel.Start()

	return nil
}

func (impl *serviceImpl) Stop(ctx context.Context) error {
	impl.timingWheel.Stop()
	return nil
}

func (impl *serviceImpl) Schedule(duration time.Duration, do func()) *timingwheel.Timer {
	return impl.timingWheel.ScheduleFunc(&everyScheduler{duration}, func() {
		impl.pool.Submit(do)
	})
}

func (impl *serviceImpl) AfterFunc(duration time.Duration, do func()) *timingwheel.Timer {
	return impl.timingWheel.AfterFunc(duration, func() {
		impl.pool.Submit(do)
	})
}

type everyScheduler struct {
	Interval time.Duration
}

func (s *everyScheduler) Next(prev time.Time) time.Time {
	return prev.Add(s.Interval)
}
