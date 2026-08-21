// Package sqlxdialect implements the dbsource dialect contract with sqlx.
package sqlxdialect

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/coffeehc/commons/dbsource/dialect"
	"github.com/jmoiron/sqlx"
	"github.com/jmoiron/sqlx/reflectx"
)

// Config contains the normalized inputs required to open an sqlx-backed dialect.
type Config struct {
	// Name is the database type reported through the dialect contract.
	Name string
	// DriverName is the database/sql driver registered for the backend.
	DriverName string
	// DataSourceName is the complete driver connection string.
	DataSourceName string
	// IdentifierQuote is the quote token used for identifiers, normally a backtick.
	IdentifierQuote string
	// MaxOpenConns is the maximum number of open connections; zero keeps database/sql defaults.
	MaxOpenConns int
	// MaxIdleConns is the maximum number of idle connections; zero keeps database/sql defaults.
	MaxIdleConns int
	// ConnMaxLifetime is the maximum connection lifetime; zero keeps database/sql defaults.
	ConnMaxLifetime time.Duration
	// Mapper controls struct field mapping and must not be nil.
	Mapper *reflectx.Mapper
}

type serviceImpl struct {
	name            string
	db              *sqlx.DB
	identifierQuote string
	closeOnce       sync.Once
	closeDone       chan struct{}
	closeErr        error
}

var _ dialect.Dialect = (*serviceImpl)(nil)
var _ dialect.Rows = (*rowsImpl)(nil)
var _ dialect.Statement = (*statementImpl)(nil)

type transactionContextKey struct{}

type handler interface {
	sqlx.ExtContext
	PreparexContext(ctx context.Context, query string) (*sqlx.Stmt, error)
}

type transactionFinalizer interface {
	Commit() error
	Rollback() error
}

// New opens and verifies an sqlx-backed dialect without registering plugins or global state.
func New(ctx context.Context, config *Config) (dialect.Dialect, error) {
	if config == nil {
		return nil, fmt.Errorf("sqlxdialect 配置不能为空")
	}
	if config.Name == "" || config.DriverName == "" || config.Mapper == nil {
		return nil, fmt.Errorf("sqlxdialect 名称、驱动和字段映射不能为空")
	}
	db, err := sqlx.Open(config.DriverName, config.DataSourceName)
	if err != nil {
		return nil, err
	}
	db.Mapper = config.Mapper
	if config.MaxOpenConns > 0 {
		db.SetMaxOpenConns(config.MaxOpenConns)
	}
	if config.MaxIdleConns > 0 {
		db.SetMaxIdleConns(config.MaxIdleConns)
	}
	if config.ConnMaxLifetime > 0 {
		db.SetConnMaxLifetime(config.ConnMaxLifetime)
	}
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}
	return &serviceImpl{
		name:            config.Name,
		db:              db,
		identifierQuote: config.IdentifierQuote,
		closeDone:       make(chan struct{}),
	}, nil
}

func (impl *serviceImpl) Name() string {
	return impl.name
}

func (impl *serviceImpl) Rewrite(query string) string {
	return query
}

func (impl *serviceImpl) QuoteIdentifier(identifier string) string {
	quote := impl.identifierQuote
	if quote == "" {
		quote = "`"
	}
	parts := strings.Split(identifier, ".")
	for index, part := range parts {
		if part == "*" {
			continue
		}
		parts[index] = quote + strings.ReplaceAll(part, quote, quote+quote) + quote
	}
	return strings.Join(parts, ".")
}

func (impl *serviceImpl) Exec(ctx context.Context, query string, args ...any) (dialect.Result, error) {
	query = impl.Rewrite(query)
	result, err := impl.handler(ctx).ExecContext(ctx, query, args...)
	if err != nil {
		return dialect.Result{}, err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return dialect.Result{}, err
	}
	lastInsertID, _ := result.LastInsertId()
	return dialect.Result{LastInsertID: lastInsertID, RowsAffected: rowsAffected}, nil
}

func (impl *serviceImpl) Query(ctx context.Context, query string, args ...any) (dialect.Rows, error) {
	query = impl.Rewrite(query)
	rows, err := impl.handler(ctx).QueryxContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	return &rowsImpl{rows: rows}, nil
}

func (impl *serviceImpl) Select(ctx context.Context, dest any, query string, args ...any) error {
	query = impl.Rewrite(query)
	return sqlx.SelectContext(ctx, impl.handler(ctx), dest, query, args...)
}

func (impl *serviceImpl) Get(ctx context.Context, dest any, query string, args ...any) (bool, error) {
	query = impl.Rewrite(query)
	err := sqlx.GetContext(ctx, impl.handler(ctx), dest, query, args...)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	return err == nil, err
}

func (impl *serviceImpl) Prepare(ctx context.Context, query string) (dialect.Statement, error) {
	query = impl.Rewrite(query)
	statement, err := impl.handler(ctx).PreparexContext(ctx, query)
	if err != nil {
		return nil, err
	}
	return &statementImpl{statement: statement}, nil
}

func (impl *serviceImpl) HandleTx(ctx context.Context, options *sql.TxOptions, handle func(context.Context) error) (err error) {
	if impl.transaction(ctx) != nil {
		return handle(ctx)
	}
	transaction, err := impl.db.BeginTxx(ctx, options)
	if err != nil {
		return err
	}
	txContext := context.WithValue(ctx, transactionContextKey{}, transaction)
	return executeTransaction(txContext, transaction, handle)
}

func executeTransaction(ctx context.Context, transaction transactionFinalizer, handle func(context.Context) error) (err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			_ = transaction.Rollback()
			panic(recovered)
		}
		if err != nil {
			if rollbackErr := transaction.Rollback(); rollbackErr != nil && !errors.Is(rollbackErr, sql.ErrTxDone) {
				err = errors.Join(err, rollbackErr)
			}
			return
		}
		if commitErr := transaction.Commit(); commitErr != nil {
			err = commitErr
		}
	}()
	return handle(ctx)
}

func (impl *serviceImpl) Ping(ctx context.Context) error {
	return impl.db.PingContext(ctx)
}

func (impl *serviceImpl) Stats() dialect.PoolStats {
	stats := impl.db.Stats()
	return dialect.PoolStats{
		MaxOpenConnections: stats.MaxOpenConnections,
		OpenConnections:    stats.OpenConnections,
		InUse:              stats.InUse,
		Idle:               stats.Idle,
		WaitCount:          stats.WaitCount,
		WaitDuration:       stats.WaitDuration,
	}
}

func (impl *serviceImpl) Close(ctx context.Context) error {
	impl.closeOnce.Do(func() {
		go func() {
			impl.closeErr = impl.db.Close()
			close(impl.closeDone)
		}()
	})
	select {
	case <-impl.closeDone:
		return impl.closeErr
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (impl *serviceImpl) transaction(ctx context.Context) *sqlx.Tx {
	transaction, _ := ctx.Value(transactionContextKey{}).(*sqlx.Tx)
	return transaction
}

func (impl *serviceImpl) handler(ctx context.Context) handler {
	if transaction := impl.transaction(ctx); transaction != nil {
		return transaction
	}
	return impl.db
}

type rowsImpl struct {
	rows *sqlx.Rows
}

func (impl *rowsImpl) Columns() ([]string, error) {
	return impl.rows.Columns()
}

func (impl *rowsImpl) Next() bool {
	return impl.rows.Next()
}

func (impl *rowsImpl) Scan(dest ...any) error {
	return impl.rows.Scan(dest...)
}

func (impl *rowsImpl) Values() ([]any, error) {
	return impl.rows.SliceScan()
}

func (impl *rowsImpl) Err() error {
	return impl.rows.Err()
}

func (impl *rowsImpl) Close() error {
	return impl.rows.Close()
}

type statementImpl struct {
	statement *sqlx.Stmt
}

func (impl *statementImpl) Exec(ctx context.Context, args ...any) (dialect.Result, error) {
	result, err := impl.statement.ExecContext(ctx, args...)
	if err != nil {
		return dialect.Result{}, err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return dialect.Result{}, err
	}
	lastInsertID, _ := result.LastInsertId()
	return dialect.Result{LastInsertID: lastInsertID, RowsAffected: rowsAffected}, nil
}

func (impl *statementImpl) Close(context.Context) error {
	return impl.statement.Close()
}
