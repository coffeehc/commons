package dbsource

import "time"

// HandleType identifies the database operation observed by a monitor.
type HandleType int64

const (
	_ = iota
	// HandleTypeQueryRow identifies a single-row query.
	HandleTypeQueryRow
	// HandleTypeQuery identifies a multi-row or streaming query.
	HandleTypeQuery
	// HandleTypeExec identifies a write statement.
	HandleTypeExec
)

// HandleMonitor observes completed database operations and must be safe for concurrent calls.
// AddRecord runs synchronously on the database operation path and must return promptly.
type HandleMonitor interface {
	// Name returns the unique monitor registration name.
	Name() string
	// AddRecord records one operation and its elapsed duration.
	AddRecord(sql string, delay time.Duration, handleType HandleType)
}
