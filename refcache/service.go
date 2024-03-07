package refcache

import (
	"context"
	"github.com/coffeehc/base/log"
	"github.com/coffeehc/boot/plugin"
	"go.uber.org/zap"
	"sync"
)

var service Service
var _serviceMutex = new(sync.RWMutex)
var _serviceName = "Ref_cache"

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
	service = newService(ctx)
	plugin.RegisterPlugin(_serviceName, service)
}
