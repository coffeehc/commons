package dbmonitor

import (
	"sync"
	"time"

	"github.com/coffeehc/commons/dbsource"
	"github.com/coffeehc/httpx/httpxcommons"
	"github.com/gofiber/fiber/v3"
)

// SqlCountMonitor tracks SQL execution counts and exposes the current snapshot over HTTP.
type SqlCountMonitor interface {
	// RegisterWebEndpoint registers the count snapshot endpoint.
	RegisterWebEndpoint(app *fiber.App)
	dbsource.HandleMonitor
}

// NewSqlCountMonitor creates an in-memory synchronous SQL counter.
func NewSqlCountMonitor() SqlCountMonitor {
	return &countMonitor{sqlMap: make(map[string]int64, 200)}
}

type countMonitor struct {
	sqlMap map[string]int64
	mutex  sync.RWMutex
}

func (impl *countMonitor) RegisterWebEndpoint(app *fiber.App) {
	app.Get("/api/v1/db/monitor/count", impl.monitorCount())
}

func (impl *countMonitor) Name() string {
	return "countMonitor"
}

func (impl *countMonitor) AddRecord(sql string, delay time.Duration, handleType dbsource.HandleType) {
	impl.mutex.Lock()
	impl.sqlMap[sql]++
	impl.mutex.Unlock()
}

func (impl *countMonitor) monitorCount() fiber.Handler {
	return func(c fiber.Ctx) error {
		impl.mutex.RLock()
		counts := make(map[string]int64, len(impl.sqlMap))
		for sql, count := range impl.sqlMap {
			counts[sql] = count
		}
		impl.mutex.RUnlock()
		return httpxcommons.SendSuccess(c, counts, 0)
	}
}
