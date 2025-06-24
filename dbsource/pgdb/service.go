package pgdb

import (
	"context"
	"github.com/coffeehc/base/log"
	"github.com/coffeehc/boot/plugin"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"sync"
)

var service Service
var _serviceMutex = new(sync.RWMutex)
var _serviceName = "pg_db_source"

func GetService() Service {
	if service == nil {
		log.Panic("Service没有初始化", zap.String("serviceName", _serviceName))
	}
	return service
}

func EnablePlugin(ctx context.Context) {
	if _serviceName == "" {
		log.Panic("插件名称没有初始化")
	}
	_serviceMutex.Lock()
	defer _serviceMutex.Unlock()
	if service != nil {
		return
	}
	value := viper.Get("dbSource")
	if value == nil {
		return
	}
	config := &Config{}
	if _, ok := value.(*Config); ok {
		config = value.(*Config)
	} else {
		err := viper.UnmarshalKey("dbSource", config)
		if err != nil {
			log.Panic("没有指定DbSource配置")
		}
	}
	service = NewService(config)
	plugin.RegisterPlugin(_serviceName, service)
}
