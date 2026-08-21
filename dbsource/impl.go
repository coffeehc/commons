package dbsource

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"sync"
	"time"

	"github.com/coffeehc/base/errors"
	"github.com/coffeehc/base/log"
	"github.com/coffeehc/commons/dbsource/dialect"
	"github.com/coffeehc/commons/dbsource/pgdialect"
	"github.com/coffeehc/commons/dbsource/sqlbuilder"
	"github.com/coffeehc/commons/dbsource/sqlxdialect"
	"github.com/jmoiron/sqlx/reflectx"
	"go.uber.org/zap"
)

// ContextKey_EnableLog enables SQL debug logging for one context when it contains any value.
var ContextKey_EnableLog = "__DB_EnableLog"

// EnableLog enables SQL debug logging process-wide.
var EnableLog = false

// NewService opens the configured database and creates the single driver-neutral service.
func NewService(config *Config) Service {
	service, err := Open(context.Background(), config)
	if err != nil {
		log.Panic("初始化数据库失败", zap.Error(err))
	}
	return service
}

// Open creates one independent database service whose lifecycle is owned by the caller.
func Open(ctx context.Context, config *Config) (Service, error) {
	if config == nil {
		return nil, fmt.Errorf("数据库配置不能为空")
	}
	mapper := config.Mapper
	if mapper == nil {
		mapper = JSONMapperFunc
	}
	databaseDialect, err := newDialect(ctx, config, mapper)
	if err != nil {
		return nil, err
	}
	return &serviceImpl{
		databaseType: config.getDBType(),
		dialect:      databaseDialect,
		monitors:     make([]HandleMonitor, 0),
	}, nil
}

func newDialect(ctx context.Context, config *Config, mapper *reflectx.Mapper) (dialect.Dialect, error) {
	lifetime := time.Duration(config.ConnMaxLifetimeSec) * time.Second
	switch config.getDBType() {
	case POSTGRES:
		return pgdialect.New(ctx, &pgdialect.Config{
			DataSourceName:  buildDataSourceNameForPostgresSQL(config),
			MaxOpenConns:    config.MaxOpenConns,
			ConnMaxLifetime: lifetime,
			Mapper:          mapper,
		})
	case MYSQL:
		return sqlxdialect.New(ctx, &sqlxdialect.Config{
			Name:            string(MYSQL),
			DriverName:      string(MYSQL),
			DataSourceName:  buildDataSourceNameForMySQL(config),
			IdentifierQuote: "`",
			MaxOpenConns:    config.MaxOpenConns,
			MaxIdleConns:    config.MaxIdleConns,
			ConnMaxLifetime: lifetime,
			Mapper:          mapper,
		})
	case SQLITE:
		return sqlxdialect.New(ctx, &sqlxdialect.Config{
			Name:            string(SQLITE),
			DriverName:      string(SQLITE),
			DataSourceName:  config.LocalDbPath,
			IdentifierQuote: "`",
			MaxOpenConns:    config.MaxOpenConns,
			MaxIdleConns:    config.MaxIdleConns,
			ConnMaxLifetime: lifetime,
			Mapper:          mapper,
		})
	default:
		return nil, fmt.Errorf("不支持的数据库类型 %q", config.DbType)
	}
}

type serviceImpl struct {
	databaseType  DbType
	dialect       dialect.Dialect
	monitors      []HandleMonitor
	monitorsMutex sync.RWMutex
}

var _ Service = (*serviceImpl)(nil)
var _ Statement = (*statementImpl)(nil)

func (impl *serviceImpl) RegisterHandleMonitor(monitor HandleMonitor) {
	if monitor == nil {
		return
	}
	impl.monitorsMutex.Lock()
	defer impl.monitorsMutex.Unlock()
	for _, current := range impl.monitors {
		if current.Name() == monitor.Name() {
			log.Warn("监视器重复", zap.String("name", monitor.Name()))
			return
		}
	}
	impl.monitors = append(impl.monitors, monitor)
}

func (impl *serviceImpl) addMonitorRecord(query string, delay time.Duration, handleType HandleType) {
	impl.monitorsMutex.RLock()
	monitors := append([]HandleMonitor(nil), impl.monitors...)
	impl.monitorsMutex.RUnlock()
	for _, monitor := range monitors {
		monitor.AddRecord(query, delay, handleType)
	}
}

func (impl *serviceImpl) DeleteById(ctx context.Context, tableName string, id int64) error {
	_, err := impl.ExecContext(ctx, fmt.Sprintf("delete from %s where %s=?", impl.QuoteIdentifier(tableName), impl.QuoteIdentifier("id")), id)
	return err
}

func (impl *serviceImpl) InsertContext(ctx context.Context, query string, args ...any) (int64, int64, error) {
	rewrittenQuery := impl.dialect.Rewrite(query)
	impl.logSQL(ctx, "dbInsert", rewrittenQuery, args)
	startedAt := time.Now()
	result, err := impl.dialect.Exec(ctx, query, args...)
	impl.addMonitorRecord(rewrittenQuery, time.Since(startedAt), HandleTypeExec)
	if err != nil {
		log.DPanic("执行插入失败", zap.String("sql", rewrittenQuery), zap.Error(err))
		return 0, 0, errors.ConverError(err)
	}
	return result.LastInsertID, result.RowsAffected, nil
}

func (impl *serviceImpl) ExecContext(ctx context.Context, query string, args ...any) (int64, error) {
	rewrittenQuery := impl.dialect.Rewrite(query)
	impl.logSQL(ctx, "dbExec", rewrittenQuery, args)
	startedAt := time.Now()
	result, err := impl.dialect.Exec(ctx, query, args...)
	impl.addMonitorRecord(rewrittenQuery, time.Since(startedAt), HandleTypeExec)
	if err != nil {
		log.DPanic("执行 SQL 失败", zap.String("sql", rewrittenQuery), zap.Error(err))
		return 0, errors.ConverError(err)
	}
	return result.RowsAffected, nil
}

func (impl *serviceImpl) QueryContext(ctx context.Context, dest any, query string, args ...any) error {
	if err := validateDestination(dest); err != nil {
		return err
	}
	rewrittenQuery := impl.dialect.Rewrite(query)
	impl.logSQL(ctx, "dbQuery", rewrittenQuery, args)
	startedAt := time.Now()
	err := impl.dialect.Select(ctx, dest, query, args...)
	impl.addMonitorRecord(rewrittenQuery, time.Since(startedAt), HandleTypeQuery)
	if err != nil {
		log.DPanic("执行查询失败", zap.String("sql", rewrittenQuery), zap.Error(err))
		return errors.ConverError(err)
	}
	return nil
}

func (impl *serviceImpl) QueryRowContext(ctx context.Context, dest any, query string, args ...any) (bool, error) {
	if err := validateDestination(dest); err != nil {
		return false, err
	}
	rewrittenQuery := impl.dialect.Rewrite(query)
	impl.logSQL(ctx, "dbQueryRow", rewrittenQuery, args)
	startedAt := time.Now()
	found, err := impl.dialect.Get(ctx, dest, query, args...)
	impl.addMonitorRecord(rewrittenQuery, time.Since(startedAt), HandleTypeQueryRow)
	if err != nil {
		log.DPanic("执行单行查询失败", zap.String("sql", rewrittenQuery), zap.Error(err))
		return false, errors.ConverError(err)
	}
	return found, nil
}

func (impl *serviceImpl) QueryRowsContext(ctx context.Context, query string, args ...any) (Rows, error) {
	rewrittenQuery := impl.dialect.Rewrite(query)
	impl.logSQL(ctx, "dbQueryRows", rewrittenQuery, args)
	startedAt := time.Now()
	rows, err := impl.dialect.Query(ctx, query, args...)
	impl.addMonitorRecord(rewrittenQuery, time.Since(startedAt), HandleTypeQuery)
	if err != nil {
		log.DPanic("打开查询结果失败", zap.String("sql", rewrittenQuery), zap.Error(err))
		return nil, errors.ConverError(err)
	}
	return rows, nil
}

func (impl *serviceImpl) PrepareContext(ctx context.Context, query string) (Statement, error) {
	rewrittenQuery := impl.dialect.Rewrite(query)
	statement, err := impl.dialect.Prepare(ctx, query)
	if err != nil {
		log.DPanic("准备 SQL 失败", zap.String("sql", rewrittenQuery), zap.Error(err))
		return nil, errors.ConverError(err)
	}
	return &statementImpl{service: impl, statement: statement, query: rewrittenQuery}, nil
}

func (impl *serviceImpl) HandleTx(ctx context.Context, handle func(context.Context) error) error {
	return impl.HandleTxWithOptions(ctx, nil, handle)
}

func (impl *serviceImpl) HandleTxWithOptions(ctx context.Context, options *sql.TxOptions, handle func(context.Context) error) error {
	err := impl.dialect.HandleTx(ctx, options, handle)
	if err != nil {
		log.Error("数据库事务执行失败", zap.Error(err))
		return err
	}
	return nil
}

func (impl *serviceImpl) Update(ctx context.Context, tableName string, limitFields map[string]bool, update *sqlbuilder.Update) (*sqlbuilder.UpdateResult, error) {
	sqlContext, err := sqlbuilder.BuildUpdate(tableName, update.GetId(), limitFields, update.GetFields(), update.GetConditions())
	if err != nil {
		return nil, err
	}
	if len(sqlContext.Params) == 1 {
		return nil, nil
	}
	count, err := impl.ExecContext(ctx, sqlContext.Sql, sqlContext.Params...)
	if err != nil {
		return nil, err
	}
	return &sqlbuilder.UpdateResult{Count: count}, nil
}

func (impl *serviceImpl) Query(ctx context.Context, dest any, colNames, tableName string, maxPageSize int64, query *sqlbuilder.Query, joinCondition *sqlbuilder.JoinCondition) (int64, error) {
	pageSQL, totalSQL := sqlbuilder.BuildQuery(colNames, tableName, maxPageSize, query, joinCondition)
	if err := impl.QueryContext(ctx, dest, pageSQL.Sql, pageSQL.Params...); err != nil {
		return 0, err
	}
	if totalSQL == nil {
		return 0, nil
	}
	tableCount := &sqlbuilder.TableCount{}
	if _, err := impl.QueryRowContext(ctx, tableCount, totalSQL.Sql, totalSQL.Params...); err != nil {
		return 0, err
	}
	return tableCount.Count, nil
}

func (impl *serviceImpl) DatabaseType() DbType {
	return impl.databaseType
}

func (impl *serviceImpl) QuoteIdentifier(identifier string) string {
	return impl.dialect.QuoteIdentifier(identifier)
}

func (impl *serviceImpl) Ping(ctx context.Context) error {
	return impl.dialect.Ping(ctx)
}

func (impl *serviceImpl) Stats() dialect.PoolStats {
	return impl.dialect.Stats()
}

func (impl *serviceImpl) Close(ctx context.Context) error {
	return impl.dialect.Close(ctx)
}

func (impl *serviceImpl) logSQL(ctx context.Context, action, query string, args []any) {
	if EnableLog || ctx.Value(ContextKey_EnableLog) != nil {
		log.Debug(action, zap.String("sql", query), zap.Any("params", args))
	}
}

func validateDestination(dest any) error {
	if dest == nil {
		return errors.SystemError("查询目标必须是非空指针")
	}
	value := reflect.ValueOf(dest)
	if value.Kind() != reflect.Ptr || value.IsNil() {
		return errors.SystemError("查询目标必须是非空指针")
	}
	return nil
}

type statementImpl struct {
	service   *serviceImpl
	statement dialect.Statement
	query     string
}

func (impl *statementImpl) Exec(ctx context.Context, args ...any) (dialect.Result, error) {
	impl.service.logSQL(ctx, "dbStatementExec", impl.query, args)
	startedAt := time.Now()
	result, err := impl.statement.Exec(ctx, args...)
	impl.service.addMonitorRecord(impl.query, time.Since(startedAt), HandleTypeExec)
	if err != nil {
		log.DPanic("执行预处理 SQL 失败", zap.String("sql", impl.query), zap.Error(err))
		return dialect.Result{}, errors.ConverError(err)
	}
	return result, nil
}

func (impl *statementImpl) Close(ctx context.Context) error {
	return impl.statement.Close(ctx)
}

// Start verifies that the database remains reachable when the plugin starts.
func (impl *serviceImpl) Start(ctx context.Context) error {
	return impl.Ping(ctx)
}

// Stop closes the active dialect connection pool.
func (impl *serviceImpl) Stop(ctx context.Context) error {
	return impl.Close(ctx)
}
