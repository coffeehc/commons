package middlewares

import (
	"github.com/coffeehc/base/log"
	"github.com/coffeehc/commons/sequences"
	"github.com/coffeehc/commons/webfacade/internal"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func RequestIdMiddleware() gin.HandlerFunc {
	service, err := sequences.NewSequenceService(0, 0)
	if err != nil {
		log.Panic("创建Sequence服务失败", zap.Error(err))
	}
	return func(c *gin.Context) {
		c.Set(internal.ContextKey_RequestId, service.NextID())
		c.Next()
	}
}

func GetRequestId(c *gin.Context) int64 {
	return c.GetInt64(internal.ContextKey_RequestId)
}
