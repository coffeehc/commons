package asyncservice

import (
	"context"
	"sync"

	"github.com/coffeehc/base/log"
	"github.com/coffeehc/boot/plugin"
	"go.uber.org/zap"
)

var service Service
var serviceMutex = new(sync.RWMutex)
var serviceName = "syncService"
var serviceScope = zap.String("scope", serviceName)

// GetService returns the registered asynchronous execution service and panics
// when EnablePlugin has not completed.
func GetService() Service {
	serviceMutex.RLock()
	defer serviceMutex.RUnlock()
	if service == nil {
		log.Panic("异步服务没有初始化", serviceScope)
	}
	return service
}

// EnablePlugin creates and registers the singleton asynchronous execution service.
func EnablePlugin(ctx context.Context) {
	serviceMutex.Lock()
	defer serviceMutex.Unlock()
	if service != nil {
		return
	}
	service = newService(ctx)
	plugin.RegisterPlugin(serviceName, service)
}
