package dbsource

import (
	"time"

	"github.com/coffeehc/base/log"
	"go.uber.org/zap"
)

var LogMonitor HandleMonitor = new(baseLogMonitor)

var DefaultLogMonitorSlowQueryDelay = time.Second

type baseLogMonitor struct {
}

func (impl *baseLogMonitor) Name() string {
	return "baseLogMonitor"
}

func (impl *baseLogMonitor) AddRecord(sql string, delay time.Duration, handleType HandleType) {
	if delay > DefaultLogMonitorSlowQueryDelay {
		log.Debug("slow query", zap.Int64("handleType", int64(handleType)), zap.Duration("intreval", delay), zap.String("sql", sql))
	}
}
