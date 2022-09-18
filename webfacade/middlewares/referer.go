package middlewares

import (
	"net/url"

	"github.com/coffeehc/base/log"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type RefererMiddleware interface {
	Middleware
	SetIgnoreRefererHosts(ignoreRefererHosts []string)
	GetIgnoreRefererHosts() []string
}

func NewRefererMiddleware(ignoreRefererHosts []string) RefererMiddleware {
	impl := &refererMiddlewareImpl{
		ignoreRefererHosts: ignoreRefererHosts,
	}
	return impl
}

type refererMiddlewareImpl struct {
	ignoreRefererHosts []string
	noMatch            gin.HandlerFunc
}

func (impl *refererMiddlewareImpl) SetIgnoreRefererHosts(ignoreRefererHosts []string) {
	impl.ignoreRefererHosts = ignoreRefererHosts
}

func (impl *refererMiddlewareImpl) GetIgnoreRefererHosts() []string {
	return impl.ignoreRefererHosts
}

func (impl *refererMiddlewareImpl) noMatchHandler(c *gin.Context) {
	if impl.noMatch != nil {
		impl.noMatch(c)
	} else {
		c.AbortWithStatus(405)
	}
}

func (impl *refererMiddlewareImpl) GetMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		referer_url := c.GetHeader("Referer")
		if referer_url == "" {
			impl.noMatchHandler(c)
			return
		}
		u, err := url.Parse(referer_url)
		if err != nil {
			log.Warn("无法解析referer", zap.String("referer", referer_url))
			impl.noMatchHandler(c)
			return
		}
		referer := u.Host
		for _, _referer := range impl.ignoreRefererHosts {
			if _referer == referer {
				c.Next()
				return
			}
		}
		impl.noMatchHandler(c)
		return
	}
}
