package asyncservice

import "time"

type Config struct {
	PoolSize         int           `mapstructure:"pool_size,omitempty" json:"pool_size,omitempty"`
	ExpiryDuration   time.Duration `mapstructure:"expiry_duration,omitempty" json:"expiry_duration,omitempty"`
	MaxBlockingTasks int           `mapstructure:"max_blocking_tasks,omitempty" json:"max_blocking_tasks,omitempty"`
	WheelSize        int64         `mapstructure:"wheel_size,omitempty" json:"wheel_size,omitempty"`
}
