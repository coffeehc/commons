package keylockservice

import (
	"context"
	"sync"

	"github.com/coffeehc/base/log"
	"github.com/coffeehc/boot/plugin"
	"go.uber.org/zap"
)

var service Service
var mutex = new(sync.RWMutex)
var name = "keyLockService"
var scope = zap.String("scope", name)

// GetService 返回已经注册的全局 key 锁服务，服务尚未初始化时 panic。
func GetService() Service {
	mutex.RLock()
	defer mutex.RUnlock()
	if service == nil {
		log.Panic("Service没有初始化", scope)
	}
	return service
}

// EnablePlugin 创建并注册全局 key 锁服务；重复调用不会重复初始化。
func EnablePlugin(ctx context.Context) {
	if name == "" {
		log.Panic("插件名称没有初始化")
	}
	mutex.Lock()
	defer mutex.Unlock()
	if service != nil {
		return
	}
	service = newService(ctx)
	plugin.RegisterPlugin(name, service)
}
