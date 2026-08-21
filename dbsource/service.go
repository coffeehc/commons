package dbsource

import (
	"context"
	"database/sql"
	"sync"

	"github.com/coffeehc/base/log"
	"github.com/coffeehc/boot/plugin"
	"github.com/coffeehc/commons/dbsource/dialect"
	"github.com/coffeehc/commons/dbsource/sqlbuilder"
	_ "github.com/go-sql-driver/mysql"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	_ "modernc.org/sqlite"
)

var service Service
var mutex = new(sync.RWMutex)
var name = "dbsource"
var scope = zap.String("scope", name)

// Rows provides driver-neutral streaming access to a query result.
type Rows interface {
	dialect.Rows
}

// Statement is a driver-neutral executable prepared statement.
type Statement interface {
	dialect.Statement
}

// Service owns database access, transaction propagation, monitoring, and backend lifecycle.
// Implementations are safe for concurrent use, convert backend errors, and preserve transaction callback errors.
type Service interface {
	// RegisterHandleMonitor adds one uniquely named SQL execution monitor.
	RegisterHandleMonitor(monitor HandleMonitor)
	// Query builds and executes a paginated query and returns the total count when requested.
	Query(ctx context.Context, dest any, colNames, tableName string, maxPageSize int64, query *sqlbuilder.Query, joinCondition *sqlbuilder.JoinCondition) (int64, error)
	// Update builds and executes one constrained update.
	Update(ctx context.Context, tableName string, limitFields map[string]bool, update *sqlbuilder.Update) (*sqlbuilder.UpdateResult, error)
	// InsertContext executes an insert and returns generated ID and affected rows.
	// Backends without generated integer IDs return zero for the ID.
	InsertContext(ctx context.Context, query string, args ...any) (int64, int64, error)
	// ExecContext executes a write statement and returns affected rows.
	ExecContext(ctx context.Context, query string, args ...any) (int64, error)
	// QueryContext scans all rows into a non-nil pointer to a slice.
	QueryContext(ctx context.Context, dest any, query string, args ...any) error
	// QueryRowContext scans the first row into a non-nil pointer.
	// It returns found=false and nil when no row matches.
	QueryRowContext(ctx context.Context, dest any, query string, args ...any) (bool, error)
	// QueryRowsContext opens a driver-neutral streaming result. The caller must close it.
	QueryRowsContext(ctx context.Context, query string, args ...any) (Rows, error)
	// PrepareContext creates an executable statement bound to the current transaction when present.
	PrepareContext(ctx context.Context, query string) (Statement, error)
	// HandleTx executes handle in one transaction using backend defaults.
	HandleTx(ctx context.Context, handle func(context.Context) error) error
	// HandleTxWithOptions executes handle in one transaction with portable transaction options.
	HandleTxWithOptions(ctx context.Context, options *sql.TxOptions, handle func(context.Context) error) error
	// DeleteById deletes one row by its integer storage ID.
	DeleteById(ctx context.Context, tableName string, id int64) error
	// DatabaseType returns the normalized configured backend type.
	DatabaseType() DbType
	// QuoteIdentifier quotes one possibly qualified identifier for the active backend.
	QuoteIdentifier(identifier string) string
	// Ping verifies that the active backend can acquire and use a connection.
	Ping(ctx context.Context) error
	// Stats returns a backend-neutral pool snapshot.
	Stats() dialect.PoolStats
	// Close starts releasing this service instance and waits until completion or context cancellation.
	// Pool shutdown continues after context cancellation.
	Close(ctx context.Context) error
}

// SetConfig stores the dbsource plugin configuration.
func SetConfig(config *Config) {
	viper.Set("dbSource", config)
}

// GetService returns the initialized global dbsource service.
func GetService() Service {
	if service == nil {
		log.Panic("Service没有初始化", scope)
	}
	return service
}

// EnablePlugin initializes and registers the single dbsource plugin when configured.
func EnablePlugin(ctx context.Context) {
	if name == "" {
		log.Panic("插件名称没有初始化")
	}
	mutex.Lock()
	defer mutex.Unlock()
	if service != nil {
		return
	}
	value := viper.Get("dbSource")
	if value == nil {
		return
	}
	config := &Config{}
	if _, ok := value.(*Config); ok {
		config = value.(*Config)
	} else {
		err := viper.UnmarshalKey("dbSource", config)
		if err != nil {
			log.Panic("没有指定DbSource配置")
		}
	}
	service = NewService(config)
	plugin.RegisterPlugin(name, service)
}
