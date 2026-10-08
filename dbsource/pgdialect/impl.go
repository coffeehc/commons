// Package pgdialect implements the dbsource dialect contract with native pgxpool.
package pgdialect

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/coffeehc/commons/dbsource/dialect"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jmoiron/sqlx/reflectx"
)

// Config contains the normalized inputs required to open a pgxpool-backed dialect.
type Config struct {
	// DataSourceName is the complete PostgreSQL connection string.
	DataSourceName string
	// MaxOpenConns is the maximum number of pool connections; zero keeps pgxpool defaults.
	MaxOpenConns int
	// ConnMaxLifetime is the maximum connection lifetime; zero keeps pgxpool defaults.
	ConnMaxLifetime time.Duration
	// Mapper controls struct field mapping and must not be nil.
	Mapper *reflectx.Mapper
}

type serviceImpl struct {
	pool      *pgxpool.Pool
	mapper    *reflectx.Mapper
	closeOnce sync.Once
	closeDone chan struct{}
}

var _ dialect.Dialect = (*serviceImpl)(nil)
var _ dialect.Connection = (*connectionImpl)(nil)
var _ dialect.Rows = (*rowsImpl)(nil)
var _ dialect.Statement = (*statementImpl)(nil)
var _ dialect.Transaction = (*transactionImpl)(nil)

type transactionContextKey struct{}

type handler interface {
	Exec(ctx context.Context, query string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, query string, args ...any) (pgx.Rows, error)
}

// New opens and verifies a pgxpool-backed dialect without registering plugins or global state.
func New(ctx context.Context, config *Config) (dialect.Dialect, error) {
	if config == nil {
		return nil, fmt.Errorf("pgdialect 配置不能为空")
	}
	if config.DataSourceName == "" || config.Mapper == nil {
		return nil, fmt.Errorf("pgdialect 数据源和字段映射不能为空")
	}
	poolConfig, err := buildPoolConfig(config)
	if err != nil {
		return nil, err
	}
	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return &serviceImpl{
		pool:      pool,
		mapper:    config.Mapper,
		closeDone: make(chan struct{}),
	}, nil
}

func buildPoolConfig(config *Config) (*pgxpool.Config, error) {
	poolConfig, err := ParsePoolConfig(config.DataSourceName)
	if err != nil {
		return nil, err
	}
	poolConfig.HealthCheckPeriod = 2 * time.Minute
	poolConfig.ConnConfig.ConnectTimeout = 5 * time.Second
	if config.MaxOpenConns > 0 {
		poolConfig.MaxConns = int32(config.MaxOpenConns)
	}
	if config.ConnMaxLifetime > 0 {
		poolConfig.MaxConnLifetime = config.ConnMaxLifetime
	}
	return poolConfig, nil
}

func (impl *serviceImpl) Name() string {
	return "postgres"
}

func (impl *serviceImpl) Rewrite(query string) string {
	return rewriteSQL(query)
}

func (impl *serviceImpl) QuoteIdentifier(identifier string) string {
	return quoteIdentifier(identifier)
}

func (impl *serviceImpl) Exec(ctx context.Context, query string, args ...any) (dialect.Result, error) {
	return impl.execWith(ctx, impl.handler(ctx), query, args...)
}

func (impl *serviceImpl) execWith(ctx context.Context, executor handler, query string, args ...any) (dialect.Result, error) {
	query = impl.Rewrite(query)
	commandTag, err := executor.Exec(ctx, query, args...)
	if err != nil {
		return dialect.Result{}, err
	}
	return dialect.Result{RowsAffected: commandTag.RowsAffected()}, nil
}

func (impl *serviceImpl) Query(ctx context.Context, query string, args ...any) (dialect.Rows, error) {
	return impl.queryWith(ctx, impl.handler(ctx), query, args...)
}

func (impl *serviceImpl) queryWith(ctx context.Context, executor handler, query string, args ...any) (dialect.Rows, error) {
	query = impl.Rewrite(query)
	rows, err := executor.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	return &rowsImpl{rows: rows}, nil
}

func (impl *serviceImpl) Select(ctx context.Context, dest any, query string, args ...any) error {
	return impl.selectWith(ctx, impl.handler(ctx), dest, query, args...)
}

func (impl *serviceImpl) selectWith(ctx context.Context, executor handler, dest any, query string, args ...any) error {
	rows, err := impl.queryWith(ctx, executor, query, args...)
	if err != nil {
		return err
	}
	defer rows.Close()
	return dialect.ScanAll(rows, dest, impl.mapper)
}

func (impl *serviceImpl) Get(ctx context.Context, dest any, query string, args ...any) (bool, error) {
	return impl.getWith(ctx, impl.handler(ctx), dest, query, args...)
}

func (impl *serviceImpl) getWith(ctx context.Context, executor handler, dest any, query string, args ...any) (bool, error) {
	rows, err := impl.queryWith(ctx, executor, query, args...)
	if err != nil {
		return false, err
	}
	defer rows.Close()
	err = dialect.ScanOne(rows, dest, impl.mapper)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	return err == nil, err
}

func (impl *serviceImpl) Prepare(ctx context.Context, query string) (dialect.Statement, error) {
	return impl.prepareWith(impl.handler(ctx), query), nil
}

func (impl *serviceImpl) prepareWith(executor handler, query string) dialect.Statement {
	query = impl.Rewrite(query)
	return &statementImpl{handler: executor, query: query}
}

func (impl *serviceImpl) BeginTx(ctx context.Context, options *sql.TxOptions) (dialect.Transaction, error) {
	pgOptions, err := convertTxOptions(options)
	if err != nil {
		return nil, err
	}
	transaction, err := impl.pool.BeginTx(ctx, pgOptions)
	if err != nil {
		return nil, err
	}
	return &transactionImpl{service: impl, transaction: transaction}, nil
}

func (impl *serviceImpl) Acquire(ctx context.Context) (dialect.Connection, error) {
	connection, err := impl.pool.Acquire(ctx)
	if err != nil {
		return nil, err
	}
	return &connectionImpl{service: impl, connection: connection}, nil
}

func (impl *serviceImpl) HandleTx(ctx context.Context, options *sql.TxOptions, handle func(context.Context) error) error {
	if impl.transaction(ctx) != nil {
		return handle(ctx)
	}
	pgOptions, err := convertTxOptions(options)
	if err != nil {
		return err
	}
	transaction, err := impl.pool.BeginTx(ctx, pgOptions)
	if err != nil {
		return err
	}
	return executeTransaction(context.WithValue(ctx, transactionContextKey{}, transaction), transaction, handle)
}

func executeTransaction(ctx context.Context, transaction pgx.Tx, handle func(context.Context) error) (err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			_ = rollbackTransaction(ctx, transaction)
			panic(recovered)
		}
		if err != nil {
			// pgx closes the physical connection on context cancellation, which
			// already aborts the transaction. Still Rollback to release the pool
			// lease, but do not append a second error from that closed connection.
			connection := transaction.Conn()
			canceledConnection := ctx.Err() != nil && errors.Is(err, ctx.Err()) && connection != nil && connection.IsClosed()
			if rollbackErr := rollbackTransaction(ctx, transaction); rollbackErr != nil && !errors.Is(rollbackErr, pgx.ErrTxClosed) && !canceledConnection {
				err = errors.Join(err, rollbackErr)
			}
			return
		}
		if commitErr := transaction.Commit(ctx); commitErr != nil {
			err = commitErr
		}
	}()
	return handle(ctx)
}

const transactionCleanupTimeout = 5 * time.Second

func rollbackTransaction(ctx context.Context, transaction pgx.Tx) error {
	cleanupContext, cancel := context.WithTimeout(context.WithoutCancel(ctx), transactionCleanupTimeout)
	defer cancel()
	return transaction.Rollback(cleanupContext)
}

func convertTxOptions(options *sql.TxOptions) (pgx.TxOptions, error) {
	if options == nil {
		return pgx.TxOptions{}, nil
	}
	converted := pgx.TxOptions{}
	switch options.Isolation {
	case sql.LevelDefault:
	case sql.LevelReadUncommitted:
		converted.IsoLevel = pgx.ReadUncommitted
	case sql.LevelReadCommitted:
		converted.IsoLevel = pgx.ReadCommitted
	case sql.LevelRepeatableRead:
		converted.IsoLevel = pgx.RepeatableRead
	case sql.LevelSerializable:
		converted.IsoLevel = pgx.Serializable
	default:
		return pgx.TxOptions{}, fmt.Errorf("PostgreSQL 不支持事务隔离级别 %s", options.Isolation)
	}
	if options.ReadOnly {
		converted.AccessMode = pgx.ReadOnly
	}
	return converted, nil
}

func (impl *serviceImpl) Ping(ctx context.Context) error {
	return impl.pool.Ping(ctx)
}

func (impl *serviceImpl) Stats() dialect.PoolStats {
	stats := impl.pool.Stat()
	return dialect.PoolStats{
		MaxOpenConnections: int(stats.MaxConns()),
		OpenConnections:    int(stats.TotalConns()),
		InUse:              int(stats.AcquiredConns()),
		Idle:               int(stats.IdleConns()),
		WaitCount:          stats.EmptyAcquireCount(),
		WaitDuration:       stats.EmptyAcquireWaitTime(),
	}
}

func (impl *serviceImpl) Close(ctx context.Context) error {
	impl.closeOnce.Do(func() {
		go func() {
			impl.pool.Close()
			close(impl.closeDone)
		}()
	})
	select {
	case <-impl.closeDone:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (impl *serviceImpl) transaction(ctx context.Context) pgx.Tx {
	transaction, _ := ctx.Value(transactionContextKey{}).(pgx.Tx)
	return transaction
}

func (impl *serviceImpl) handler(ctx context.Context) handler {
	if transaction := impl.transaction(ctx); transaction != nil {
		return transaction
	}
	return impl.pool
}

type transactionImpl struct {
	service     *serviceImpl
	transaction pgx.Tx
}

func (impl *transactionImpl) Exec(ctx context.Context, query string, args ...any) (dialect.Result, error) {
	return impl.service.execWith(ctx, impl.transaction, query, args...)
}

func (impl *transactionImpl) Query(ctx context.Context, query string, args ...any) (dialect.Rows, error) {
	return impl.service.queryWith(ctx, impl.transaction, query, args...)
}

func (impl *transactionImpl) Select(ctx context.Context, dest any, query string, args ...any) error {
	return impl.service.selectWith(ctx, impl.transaction, dest, query, args...)
}

func (impl *transactionImpl) Get(ctx context.Context, dest any, query string, args ...any) (bool, error) {
	return impl.service.getWith(ctx, impl.transaction, dest, query, args...)
}

func (impl *transactionImpl) Prepare(_ context.Context, query string) (dialect.Statement, error) {
	return impl.service.prepareWith(impl.transaction, query), nil
}

func (impl *transactionImpl) Commit(ctx context.Context) error {
	return impl.transaction.Commit(ctx)
}

func (impl *transactionImpl) Rollback(ctx context.Context) error {
	return rollbackTransaction(ctx, impl.transaction)
}

type connectionImpl struct {
	service    *serviceImpl
	connection *pgxpool.Conn
}

func (impl *connectionImpl) Exec(ctx context.Context, query string, args ...any) (dialect.Result, error) {
	return impl.service.execWith(ctx, impl.connection, query, args...)
}

func (impl *connectionImpl) Query(ctx context.Context, query string, args ...any) (dialect.Rows, error) {
	return impl.service.queryWith(ctx, impl.connection, query, args...)
}

func (impl *connectionImpl) Select(ctx context.Context, dest any, query string, args ...any) error {
	return impl.service.selectWith(ctx, impl.connection, dest, query, args...)
}

func (impl *connectionImpl) Get(ctx context.Context, dest any, query string, args ...any) (bool, error) {
	return impl.service.getWith(ctx, impl.connection, dest, query, args...)
}

func (impl *connectionImpl) Prepare(_ context.Context, query string) (dialect.Statement, error) {
	return impl.service.prepareWith(impl.connection, query), nil
}

func (impl *connectionImpl) BeginTx(ctx context.Context, options *sql.TxOptions) (dialect.Transaction, error) {
	pgOptions, err := convertTxOptions(options)
	if err != nil {
		return nil, err
	}
	transaction, err := impl.connection.BeginTx(ctx, pgOptions)
	if err != nil {
		return nil, err
	}
	return &transactionImpl{service: impl.service, transaction: transaction}, nil
}

func (impl *connectionImpl) Ping(ctx context.Context) error {
	return impl.connection.Conn().Ping(ctx)
}

func (impl *connectionImpl) Close(context.Context) error {
	impl.connection.Release()
	return nil
}

type rowsImpl struct {
	rows pgx.Rows
}

func (impl *rowsImpl) Columns() ([]string, error) {
	descriptions := impl.rows.FieldDescriptions()
	columns := make([]string, len(descriptions))
	for index, description := range descriptions {
		columns[index] = description.Name
	}
	return columns, nil
}

func (impl *rowsImpl) Next() bool {
	return impl.rows.Next()
}

func (impl *rowsImpl) Scan(dest ...any) error {
	return impl.rows.Scan(dest...)
}

func (impl *rowsImpl) Values() ([]any, error) {
	return impl.rows.Values()
}

func (impl *rowsImpl) Err() error {
	return impl.rows.Err()
}

func (impl *rowsImpl) Close() error {
	impl.rows.Close()
	return nil
}

type statementImpl struct {
	handler handler
	query   string
}

func (impl *statementImpl) Exec(ctx context.Context, args ...any) (dialect.Result, error) {
	commandTag, err := impl.handler.Exec(ctx, impl.query, args...)
	if err != nil {
		return dialect.Result{}, err
	}
	return dialect.Result{RowsAffected: commandTag.RowsAffected()}, nil
}

func (impl *statementImpl) Close(context.Context) error {
	return nil
}
