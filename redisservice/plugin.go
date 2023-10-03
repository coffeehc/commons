package redisservice

import (
	"context"
	"sync"

	"github.com/coffeehc/base/log"
	"github.com/coffeehc/boot/plugin"
	"github.com/redis/go-redis/v9"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

var service Service
var mutex = new(sync.Mutex)

// Service redis 服务接口定义
type Service interface {
	redis.Cmdable
	GetRedisClient() *redis.Client
}

func GetService() Service {
	if service == nil {
		log.Panic("Service没有初始化", scope)
	}
	return service
}

func EnablePlugin(ctx context.Context) {
	mutex.Lock()
	defer mutex.Unlock()
	if service != nil {
		return
	}
	if !viper.IsSet("redis") {
		log.Panic("没有找到Redis的配置", zap.String("configKey", "redis"), scope)
	}
	config := &RedisOptions{}
	err := viper.UnmarshalKey("redis", config)
	if err != nil {
		log.Panic("加载redis配置失败", zap.Error(err), scope)
	}
	_Service, err := NewRedisService(config)
	if err != nil {
		log.Panic("初始化Redis失败", zap.Error(err))
	}
	service = _Service
	plugin.RegisterPlugin("redis", service)
}
