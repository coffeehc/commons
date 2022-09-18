package dbsource

import "time"

type HandleType int64

const (
	_ = iota
	HandleTypeQueryRow
	HandleTypeQuery
	HandleTypeExec
)

type HandleMonitor interface {
	Name() string
	AddRecord(sql string, delay time.Duration, handleType HandleType)
}
