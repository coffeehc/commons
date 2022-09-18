package memcache

import (
	"context"
	"sync"

	"github.com/coffeehc/base/log"
	"github.com/coffeehc/boot/plugin"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

var service Service
var mutex = new(sync.RWMutex)
var name = "memcache"
var scope = zap.String("scope", name)

var DefaultMemcacheSize = 1024 * 1024 * 8
var DefaultConfigKey_MemcacheSize = "memcache.size"
var DefaultConfigKey_MemcacheDisable = "memcache.disable"

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
	viper.SetDefault(DefaultConfigKey_MemcacheSize, DefaultMemcacheSize)
	service = newService()
	plugin.RegisterPlugin(name, service)
}

func SetMemcacheSize(size int) {
	viper.Set("memcache.size", size)
}
