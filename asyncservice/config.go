package asyncservice

// Config defines asynchronous task execution limits.
type Config struct {
	// PoolSize is the maximum number of concurrently running tasks. Values less
	// than one use the service default and do not preallocate goroutines.
	PoolSize int `mapstructure:"pool_size,omitempty" json:"pool_size,omitempty"`
}
