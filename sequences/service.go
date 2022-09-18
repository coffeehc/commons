package sequences

import (
	"context"
	"math/rand"
	"sync"

	"github.com/coffeehc/base/log"
	"github.com/coffeehc/boot/plugin"
	"go.uber.org/zap"
)

var service Service
var mutex = new(sync.RWMutex)
var name = "sequences"
var scope = zap.String("scope", name)

func GetService() Service {
	if service == nil {
		log.Panic("Service没有初始化", scope)
	}
	return service
}

type Service interface {
	SequenceService
}

func EnablePlugin(ctx context.Context) {
	mutex.Lock()
	defer mutex.Unlock()
	if service != nil {
		return
	}
	sequenceService, err := NewSequenceService(rand.Int63n(MaxDCID), rand.Int63n(MaxNodeID))
	if err != nil {
		log.Panic("启动Sequence错误", zap.Error(err))
	}
	service = sequenceService
	plugin.RegisterPlugin(name, service)
}
