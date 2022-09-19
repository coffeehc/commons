package middlewares

import (
	"github.com/coffeehc/commons/webfacade"
	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

type LimitService interface {
	Allow() bool
	SetLimit(limit int)
	GetLimit() int
}

func NewLimitService(limit int) LimitService {
	impl := &limitServiceImpl{
		rateLimit: rate.NewLimiter(rate.Limit(limit), limit),
	}
	return impl
}

type limitServiceImpl struct {
	rateLimit *rate.Limiter
}

func (impl *limitServiceImpl) Allow() bool {
	return impl.rateLimit.Allow()
}

func (impl *limitServiceImpl) SetLimit(limit int) {
	impl.rateLimit.SetLimit(rate.Limit(limit))
	impl.rateLimit.SetBurst(limit)
}

func (impl *limitServiceImpl) GetLimit() int {
	return impl.rateLimit.Burst()
}

func LimitMiddleware(limitService LimitService) gin.HandlerFunc {
	return func(c *gin.Context) {
		ok := limitService.Allow()
		if !ok {
			// 丢弃请求
			webfacade.SendError(c, "访问过于频繁", 0, 200)
			return
		}
		c.Next()
	}
}
