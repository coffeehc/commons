package dbmonitor

import (
	"context"
	"github.com/coffeehc/commons/dbsource"
	"github.com/coffeehc/commons/webfacade"
	"github.com/gin-gonic/gin"
	"time"
)

type SqlCountMonitor interface {
	RegisterWebEndpoint(engine *gin.Engine)
	dbsource.HandleMonitor
}

func NewSqlCountMonitor(ctx context.Context) SqlCountMonitor {
	impl := &countMonitor{
		sqlMap:     make(map[string]int64, 200),
		sqlChannel: make(chan string, 5000),
	}
	go func() {
		for sql := range impl.sqlChannel {
			impl.sqlMap[sql] += 1
		}
	}()
	return impl
}

type countMonitor struct {
	sqlMap     map[string]int64
	sqlChannel chan string
}

func (impl *countMonitor) RegisterWebEndpoint(engine *gin.Engine) {
	engine.GET("/api/v1/db/monitor/count", impl.monitorCount())
}

func (impl *countMonitor) Name() string {
	return "countMonitor"
}

func (impl *countMonitor) AddRecord(sql string, delay time.Duration, handleType dbsource.HandleType) {
	impl.sqlChannel <- sql
}

func (impl *countMonitor) monitorCount() gin.HandlerFunc {
	return func(c *gin.Context) {
		webfacade.SendSuccess(c, impl.sqlMap, 0)
	}
}
