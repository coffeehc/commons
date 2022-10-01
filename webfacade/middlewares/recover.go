package middlewares

import (
	"context"
	"github.com/coffeehc/commons/webfacade"
	"net/http"
	"strings"
	"time"

	"github.com/coffeehc/base/errors"
	"github.com/coffeehc/base/log"
	"github.com/coffeehc/httpx"
	"github.com/gin-gonic/gin"
)

func RecoverMiddleware(timeout time.Duration) gin.HandlerFunc {
	log.Debug("构建异常处理中间件")
	return func(c *gin.Context) {
		ctx := context.WithValue(c.Request.Context(), "path", c.Request.RequestURI)
		cancelFunc := func() {}
		if timeout > 0 {
			ctx, cancelFunc = context.WithTimeout(ctx, timeout)
		}
		c.Request = c.Request.WithContext(ctx)
		defer func() {
			if err := recover(); err != nil {
				if errStr, ok := err.(string); ok {
					log.DPanic(errStr, httpx.LogKeyHTTPContext(c)...)
				} else {
					e := errors.ConverUnknowError(err)
					if errors.IsMessageError(e) {
						log.Error(e.Error(), e.GetFields(httpx.LogKeyHTTPContext(c)...)...)
					} else {
						if strings.HasPrefix(e.Error(), "context ") || strings.HasPrefix(e.Error(), "rpc error") {
							log.Error(e.Error(), e.GetFields(httpx.LogKeyHTTPContext(c)...)...)
						} else {
							log.DPanic(e.Error(), e.GetFields(httpx.LogKeyHTTPContext(c)...)...)
						}
					}
				}
				c.Redirect(http.StatusFound, "/")
			}
			cancelFunc()
		}()
		c.Next()
	}
}

func RecoverJsonMiddleware(timeout time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := context.WithValue(c.Request.Context(), "path", c.Request.RequestURI)
		cancelFunc := func() {}
		if timeout > 0 {
			ctx, cancelFunc = context.WithTimeout(ctx, timeout)
		}
		c.Request = c.Request.WithContext(ctx)
		defer func() {
			if err := recover(); err != nil {
				e := errors.ConverUnknowError(err)
				if errors.IsMessageError(e) {
					log.Warn(e.Error(), e.GetFields(httpx.LogKeyHTTPContext(c)...)...)
					webfacade.SendError(c, e.Error(), 0, 200)
				} else {
					if strings.HasPrefix(e.Error(), "context ") || strings.HasPrefix(e.Error(), "rpc error") {
						log.Error(e.Error(), e.GetFields(httpx.LogKeyHTTPContext(c)...)...)
						webfacade.SendError(c, "系统很忙,请稍后重试", 0, 200)
						return
					}
					log.DPanic(e.Error(), e.GetFields(httpx.LogKeyHTTPContext(c)...)...)
					webfacade.SendError(c, "系统开小差了,请稍后重试", 0, 200)
				}
			}
		}()
		c.Next()
		cancelFunc()
	}
}
