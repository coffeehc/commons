package redisservice

import (
	"context"

	"github.com/coffeehc/base/errors"
	"github.com/coffeehc/base/log"
	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"
)

var scope = zap.String("scope", "redis")

// NewRedisService 创建一个 Redis 服务
func NewRedisService(config *RedisOptions) (Service, error) {
	if config == nil {
		config = &RedisOptions{
			IsCluster:          false,
			MaxRedirects:       3,
			DialTimeout:        300,
			ReadTimeout:        8,
			WriteTimeout:       8,
			PoolSize:           10,
			PoolTimeout:        15,
			DB:                 1,
			IdleCheckFrequency: 10,
		}
	}
	if config.Virtual {
		log.Warn("创建了一个虚拟的Redis服务")
		return &virtualServiceImpl{}, nil
	}
	err := config.check()
	if err != nil {
		return nil, err
	}
	redisService := &redisServiceImpl{}
	if config.IsCluster {
		client := redis.NewClusterClient(config.adapterClusterOptions())
		redisService.Cmdable = client
		redisService.clusterClient = client
		redisService.isCluster = true
	} else {
		client := redis.NewClient(config.adapterOptions())
		redisService.Cmdable = client
		redisService.singleClient = client

	}
	cmd := redisService.Ping(context.TODO())
	if cmd.Err() != nil {
		log.Error("Redis ping失败", zap.Error(cmd.Err()), scope)
		return nil, errors.MessageError("redis Ping失败")
	}
	return redisService, nil
}

type redisServiceImpl struct {
	redis.Cmdable
	singleClient  *redis.Client
	clusterClient *redis.ClusterClient
	isCluster     bool
}

func (impl *redisServiceImpl) GetRedisClient() *redis.Client {
	return impl.singleClient
}
