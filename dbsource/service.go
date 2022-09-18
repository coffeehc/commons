package dbsource

import (
	"context"
	"sync"

	"github.com/coffeehc/base/log"
	"github.com/coffeehc/boot/plugin"
	_ "github.com/go-sql-driver/mysql"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

var service Service
var mutex = new(sync.RWMutex)
var name = "dbsource"
var scope = zap.String("scope", name)

func SetConfig(config *Config) {
	viper.Set("dbSource", config)
}

func GetService() Service {
	if service == nil {
		log.Panic("Service没有初始化", scope)
	}
	return service
}

func EnablePlugin(ctx context.Context) {
	if name == "" {
		log.Panic("插件名称没有初始化")
	}
	mutex.Lock()
	defer mutex.Unlock()
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
	plugin.RegisterPlugin(name, service)
}
