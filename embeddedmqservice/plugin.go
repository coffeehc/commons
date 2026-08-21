package embeddedmqservice

import (
	"context"
	"sync"

	"github.com/coffeehc/boot/plugin"
)

var service Service
var serviceMutex = new(sync.RWMutex)
var serviceName = "embeddedMQService"

// GetService 返回嵌入式 MQ 服务实例。
func GetService() Service {
	serviceMutex.RLock()
	defer serviceMutex.RUnlock()
	if service == nil {
		panic("embeddedMQService 未初始化")
	}
	return service
}

// EnablePlugin 启用嵌入式 MQ 服务插件。
func EnablePlugin(ctx context.Context) {
	serviceMutex.Lock()
	defer serviceMutex.Unlock()
	if service != nil {
		return
	}
	service = newService(ctx)
	plugin.RegisterPlugin(serviceName, service)
}
