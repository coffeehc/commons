package middlewares

import "github.com/gin-gonic/gin"

type Middleware interface {
	GetMiddleware() gin.HandlerFunc
}
