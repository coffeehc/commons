package redisservice

import (
	"time"

	"github.com/coffeehc/base/errors"
	"github.com/redis/go-redis/v9"
	"github.com/spf13/viper"

	"os"
	"strings"
)

const errScope = "redisService"
const envRedisIsCluster = "ENV_REDIS_CLUSTER"
const envRedisAddr = "ENV_REDIS_ADDR"
const envRedisPassword = "ENV_REDIS_PASSWORD"

// RedisOptions redis 设置
type RedisOptions struct {
	Virtual        bool     `mapstructure:"virtual,omitempty" json:"virtual,omitempty"`
	IsCluster      bool     `mapstructure:"is_cluster,omitempty" json:"is_cluster,omitempty"`
	Addrs          []string `mapstructure:"addrs,omitempty" json:"addrs,omitempty"`
	MaxRedirects   int      `mapstructure:"max_redirects,omitempty" json:"max_redirects,omitempty"`
	ReadOnly       bool     `mapstructure:"read_only,omitempty" json:"read_only,omitempty"`
	RouteByLatency bool     `mapstructure:"route_by_latency,omitempty" json:"route_by_latency,omitempty"`
	Password       string   `mapstructure:"password,omitempty" json:"password,omitempty"`

	DialTimeout  time.Duration `mapstructure:"dial_timeout,omitempty" json:"dial_timeout,omitempty"`
	ReadTimeout  time.Duration `mapstructure:"read_timeout,omitempty" json:"read_timeout,omitempty"`
	WriteTimeout time.Duration `mapstructure:"write_timeout,omitempty" json:"write_timeout,omitempty"`

	PoolSize        int           `mapstructure:"pool_size,omitempty" json:"pool_size,omitempty"`
	PoolTimeout     time.Duration `mapstructure:"pool_timeout,omitempty" json:"pool_timeout,omitempty"`
	ConnMaxIdleTime time.Duration `mapstructure:"conn_max_idle_time,omitempty" json:"conn_max_idle_time,omitempty"`
	MinIdleConns    int           `mapstructure:"min_idle_conns,omitempty" json:"min_idle_conns,omitempty"`
	MaxIdleConns    int           `mapstructure:"max_idle_conns,omitempty" json:"max_idle_conns,omitempty"`
	// 单机版
	DB int `mapstructure:"db,omitempty" json:"db,omitempty"`
}

func (options *RedisOptions) check() error {
	redisAddr := os.Getenv(envRedisAddr)
	if redisAddr != "" {
		options.Addrs = strings.Split(redisAddr, ",")
	}
	if len(options.Addrs) == 0 {
		return errors.SystemError("没有指定 Reids Addrs")
	}
	if os.Getenv(envRedisPassword) != "" {
		options.Password = os.Getenv(envRedisPassword)
	}
	if os.Getenv(envRedisIsCluster) == "true" {
		options.IsCluster = true
	}
	return nil
}

func (options *RedisOptions) adapterClusterOptions() *redis.ClusterOptions {
	return &redis.ClusterOptions{
		Addrs:           options.Addrs,
		MaxRedirects:    options.MaxRedirects,
		ReadOnly:        options.ReadOnly,
		RouteByLatency:  options.RouteByLatency,
		Password:        options.Password,
		DialTimeout:     options.DialTimeout,
		ReadTimeout:     options.ReadTimeout,
		WriteTimeout:    options.WriteTimeout,
		PoolSize:        options.PoolSize,
		PoolTimeout:     options.PoolTimeout,
		ConnMaxIdleTime: options.ConnMaxIdleTime,
		MaxIdleConns:    options.MaxIdleConns,
		MinIdleConns:    options.MinIdleConns,
	}
}

func (options *RedisOptions) adapterOptions() *redis.Options {
	return &redis.Options{
		Network:         "tcp",
		Addr:            options.Addrs[0],
		Password:        options.Password,
		DB:              options.DB,
		MaxRetries:      options.MaxRedirects,
		DialTimeout:     options.DialTimeout * time.Millisecond,
		ReadTimeout:     options.ReadTimeout * time.Millisecond,
		WriteTimeout:    options.WriteTimeout * time.Millisecond,
		PoolSize:        options.PoolSize,
		PoolTimeout:     options.PoolTimeout * time.Millisecond,
		ConnMaxIdleTime: options.ConnMaxIdleTime * time.Millisecond,
		MaxIdleConns:    options.MaxIdleConns,
		MinIdleConns:    options.MinIdleConns,
	}
}

func SetOptions(option *RedisOptions) {
	viper.Set("redis", option)
}
