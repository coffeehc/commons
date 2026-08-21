// Package dialect defines the driver-neutral database contract used by dbsource.
package dialect

import (
	"context"
	"database/sql"
	"time"
)

// Result describes the stable outcome of a database write.
type Result struct {
	// LastInsertID is the generated integer identifier when the backend supports it; otherwise it is zero.
	LastInsertID int64
	// RowsAffected is the number of rows changed by the statement.
	RowsAffected int64
}

// PoolStats is the common subset of connection-pool statistics exposed by all dialects.
type PoolStats struct {
	// MaxOpenConnections is the configured upper bound for open connections; zero means backend default.
	MaxOpenConnections int
	// OpenConnections is the number of established connections, including active and idle connections.
	OpenConnections int
	// InUse is the number of connections currently acquired by callers.
	InUse int
	// Idle is the number of connections currently idle in the pool.
	Idle int
	// WaitCount is the number of acquisitions that had to wait for a connection.
	WaitCount int64
	// WaitDuration is the total time spent waiting for a connection.
	WaitDuration time.Duration
}

// Rows provides driver-neutral streaming access to a query result.
// Implementations are not safe for concurrent iteration. Callers must close Rows.
type Rows interface {
	// Columns returns result column names in scan order.
	Columns() ([]string, error)
	// Next advances to the next row and reports whether a row is available.
	Next() bool
	// Scan copies the current row into destination pointers.
	Scan(dest ...any) error
	// Values returns the decoded values of the current row in column order.
	Values() ([]any, error)
	// Err returns the terminal iteration error, if any.
	Err() error
	// Close releases result resources and the underlying connection.
	Close() error
}

// Statement is a driver-neutral executable prepared statement.
// Implementations are safe for the same concurrency level as their owning Dialect.
type Statement interface {
	// Exec executes the statement with positional arguments.
	Exec(ctx context.Context, args ...any) (Result, error)
	// Close releases statement resources. It is a no-op for dialects with automatic statement caching.
	Close(ctx context.Context) error
}

// Executor provides the driver-neutral SQL operations shared by pools, connections, and transactions.
type Executor interface {
	// Exec executes a write statement and returns its stable result.
	Exec(ctx context.Context, query string, args ...any) (Result, error)
	// Query opens a streaming result. The caller must close the returned Rows.
	Query(ctx context.Context, query string, args ...any) (Rows, error)
	// Select scans all returned rows into a non-nil pointer to a slice.
	Select(ctx context.Context, dest any, query string, args ...any) error
	// Get scans the first returned row into a non-nil destination pointer.
	// It returns found=false and a nil error when no row matches.
	Get(ctx context.Context, dest any, query string, args ...any) (found bool, err error)
	// Prepare creates an executable statement.
	Prepare(ctx context.Context, query string) (Statement, error)
}

// Transaction is an explicitly controlled driver-neutral database transaction.
// Callers must finish each transaction with Commit or Rollback.
type Transaction interface {
	Executor
	// Commit makes the transaction changes durable.
	Commit(ctx context.Context) error
	// Rollback discards the transaction changes.
	Rollback(ctx context.Context) error
}

// Connection is one acquired physical database connection.
// Callers must close the connection to return it to its owning pool.
type Connection interface {
	Executor
	// BeginTx starts a transaction on this connection.
	BeginTx(ctx context.Context, options *sql.TxOptions) (Transaction, error)
	// Ping verifies that this connection remains usable.
	Ping(ctx context.Context) error
	// Close returns this connection to its owning pool.
	Close(ctx context.Context) error
}

// Dialect owns database execution, SQL rewriting, transaction propagation, and pool lifecycle.
// Implementations must be safe for concurrent use. Methods return backend errors without logging.
type Dialect interface {
	Executor
	// Name returns the configured database type name.
	Name() string
	// Rewrite converts canonical SQL into backend syntax, including placeholders and identifier quotes.
	Rewrite(query string) string
	// QuoteIdentifier safely quotes one possibly qualified SQL identifier.
	QuoteIdentifier(identifier string) string
	// BeginTx starts an explicitly controlled transaction owned by the active pool.
	BeginTx(ctx context.Context, options *sql.TxOptions) (Transaction, error)
	// Acquire obtains one physical connection from the active pool.
	Acquire(ctx context.Context) (Connection, error)
	// HandleTx executes handle in one transaction and propagates the transaction through its context.
	// Nested calls reuse the current transaction. Callback, commit, and rollback errors are returned.
	HandleTx(ctx context.Context, options *sql.TxOptions, handle func(context.Context) error) error
	// Ping verifies that the backend can acquire and use a connection.
	Ping(ctx context.Context) error
	// Stats returns a snapshot of the connection pool.
	Stats() PoolStats
	// Close starts releasing the connection pool and waits until completion or context cancellation.
	// Pool shutdown continues after context cancellation.
	Close(ctx context.Context) error
}
