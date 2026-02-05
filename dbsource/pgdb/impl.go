package pgdb

import (
	"context"
	"database/sql"
	_errors "errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/coffeehc/base/errors"
	"github.com/coffeehc/base/log"
	"github.com/coffeehc/commons/dbsource/sqlbuilder"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

type Handler interface {
	Exec(ctx context.Context, sql string, arguments ...any) (commandTag pgconn.CommandTag, err error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

var ContextKey_EnableLog = "__DB_EnableLog"

var EnableLog = false

type Service interface {
	GetPool() *pgxpool.Pool
	RegisterHandleMonitor(monitor HandleMonitor)
	GetOrCreateContext(ctx context.Context) context.Context
	GetOrCreateHandler(ctx context.Context) (context.Context, bool, Handler)
	GetOrCreateTxHandler(ctx context.Context, options pgx.TxOptions) (context.Context, bool, Handler, error)
	Query(ctx context.Context, ts interface{}, colNames, tableName string, maxPageSize int64, query *sqlbuilder.Query, joinCondition *sqlbuilder.JoinCondition) (int64, error)
	Update(ctx context.Context, tableName string, limitFields map[string]bool, update *sqlbuilder.Update) (*sqlbuilder.UpdateResult, error)
	InsertContext(ctx context.Context, sql string, args ...interface{}) (int64, int64, error)
	ExecContext(ctx context.Context, sql string, args ...interface{}) (int64, error)
	QueryContext(ctx context.Context, ts interface{}, query string, args ...interface{}) error
	QueryRowContext(ctx context.Context, t interface{}, query string, args ...interface{}) (bool, error)
	HandleTx(ctx context.Context, txHanle func(ctx context.Context) error) error
	DeleteById(ctx context.Context, tableName string, id int64) error
}

func NewService(config *Config) Service {
	pool, err := newPool(config)
	if err != nil {
		log.Error("初始化连接池失败", zap.Error(err))
		return nil
	}
	impl := &serviceImpl{
		pool:      pool,
		monitors:  make([]HandleMonitor, 0),
		pgscanAPI: &ScanAPI{},
	}
	return impl
}

type serviceImpl struct {
	pool      *pgxpool.Pool
	monitors  []HandleMonitor
	pgscanAPI *ScanAPI
}

func (impl *serviceImpl) GetPool() *pgxpool.Pool {
	return impl.pool
}

func (impl *serviceImpl) addMonitorRecord(sql string, delay time.Duration, handleType HandleType) {
	for _, m := range impl.monitors {
		m.AddRecord(sql, delay, handleType)
	}
}

func (impl *serviceImpl) RegisterHandleMonitor(monitor HandleMonitor) {
	for _, m := range impl.monitors {
		if m.Name() == monitor.Name() {
			log.Warn("监视器重复", zap.String("name", monitor.Name()))
			return
		}
	}
	impl.monitors = append(impl.monitors, monitor)
}

func (impl *serviceImpl) GetOrCreateContext(ctx context.Context) context.Context {
	tx := GetTxFormContext(ctx)
	if tx != nil {
		return ctx
	}
	pool := GetPoolFormContext(ctx)
	if pool != nil {
		return ctx
	}
	return setPoolToContext(ctx, impl.GetPool())
}

func (impl *serviceImpl) GetOrCreateHandler(ctx context.Context) (context.Context, bool, Handler) {
	tx := GetTxFormContext(ctx)
	if tx != nil {
		return ctx, false, tx
	}
	pool := GetPoolFormContext(ctx)
	if pool != nil {
		return ctx, false, pool
	}
	return setPoolToContext(ctx, impl.GetPool()), true, impl.GetPool()
}

func (impl *serviceImpl) GetOrCreateTxHandler(ctx context.Context, options pgx.TxOptions) (context.Context, bool, Handler, error) {
	tx := GetTxFormContext(ctx)
	if tx != nil {
		return ctx, false, tx, nil
	}
	ctx = impl.GetOrCreateContext(ctx)
	txHandler, err := impl.GetPool().BeginTx(ctx, options)
	if err != nil {
		log.DPanic("开始事务失败", zap.Error(err))
		return ctx, false, nil, errors.SystemError("开启事务失败")
	}
	return setTxToContext(ctx, txHandler), true, txHandler, nil
}

func (impl *serviceImpl) Query(ctx context.Context, ts interface{}, colNames, tableName string, maxPageSize int64, query *sqlbuilder.Query, joinCondition *sqlbuilder.JoinCondition) (int64, error) {
	pageSqlContext, totalSqlContext := sqlbuilder.BuildQuery(colNames, tableName, maxPageSize, query, joinCondition, true)
	err := impl.QueryContext(ctx, ts, pageSqlContext.Sql, pageSqlContext.Params...)
	if err != nil {
		log.DPanic("SQL错误", zap.Error(err))
		return 0, err
	}
	count := int64(0)
	if totalSqlContext != nil {
		tc := &sqlbuilder.TableCount{}
		_, err = impl.QueryRowContext(ctx, tc, totalSqlContext.Sql, totalSqlContext.Params...)
		if err != nil {
			log.DPanic("SQL错误", zap.Error(err))
			return 0, err
		}
		count = tc.Count
	}
	return count, nil
}

func (impl *serviceImpl) Update(ctx context.Context, tableName string, limitFields map[string]bool, update *sqlbuilder.Update) (*sqlbuilder.UpdateResult, error) {
	sqlContext, err := sqlbuilder.BuildUpdate(tableName, update.GetId(), limitFields, update.GetFields(), update.GetConditions())
	if err != nil {
		log.DPanic("SQL错误", zap.Error(err))
		return nil, err
	}
	if len(sqlContext.Params) == 1 {
		return nil, nil
	}
	count, err := impl.ExecContext(impl.GetOrCreateContext(ctx), sqlContext.Sql, sqlContext.Params...)
	if err != nil {
		log.DPanic("SQL错误", zap.Error(err))
		return nil, err
	}
	return &sqlbuilder.UpdateResult{
		Count: count,
	}, nil
}

func (impl *serviceImpl) InsertContext(ctx context.Context, sql string, args ...interface{}) (int64, int64, error) {
	sql = strings.ReplaceAll(sql, "`", "")
	ctx = impl.GetOrCreateContext(ctx)
	result, err := impl.execContext(ctx, sql, args...)
	if err != nil {
		log.DPanic("SQL错误", zap.Error(err))
		return 0, 0, err
	}
	// lastInsertId, _ := result.LastInsertId()
	rowsAffected := result.RowsAffected()
	return 0, rowsAffected, nil
}

func (impl *serviceImpl) ExecContext(ctx context.Context, sql string, args ...interface{}) (int64, error) {
	sql = strings.ReplaceAll(sql, "`", "")
	ctx = impl.GetOrCreateContext(ctx)
	result, _err := impl.execContext(ctx, sql, args...)
	if _err != nil {
		log.DPanic("SQL错误", zap.Error(_err))
		return 0, _err
	}
	count := result.RowsAffected()
	return count, nil
}

func (impl *serviceImpl) QueryContext(ctx context.Context, ts interface{}, query string, args ...interface{}) error {
	query = strings.ReplaceAll(query, "`", "")
	tType := reflect.TypeOf(ts)
	if tType.Kind() != reflect.Ptr {
		log.DPanic("需要填充的对象必须是指针地址")
		return errors.SystemError("需要填充的对象必须是指针地址")
	}
	var handler Handler = nil
	handler = GetTxFormContext(ctx)
	if handler == nil {
		handler = impl.GetPool()
	}
	query = rebind(query)
	if EnableLog || ctx.Value(ContextKey_EnableLog) != nil {
		log.Debug("dbQuery", zap.String("sql", query), zap.Any("params", args))
	}
	t := time.Now()
	rows, err := handler.Query(ctx, query, args...)
	// rows, err := handler.QueryxContext(ctx, query, args...)
	if err != nil {
		log.DPanic("执行sql错误", zap.Error(err), zap.String("sql", query))
		return errors.ConverError(err)
	}
	// defer rows.Close()
	go impl.addMonitorRecord(query, time.Now().Sub(t), HandleTypeExec)
	defer rows.Close()
	return impl.pgscanAPI.ScanAll(ts, rows)
	// return rows.Scan(ts)
}

func (impl *serviceImpl) QueryRowContext(ctx context.Context, t interface{}, query string, args ...interface{}) (bool, error) {
	query = strings.ReplaceAll(query, "`", "")
	tType := reflect.TypeOf(t)
	if tType.Kind() != reflect.Ptr {
		log.DPanic("需要填充的对象必须是指针地址")
		return false, errors.SystemError("需要填充的对象必须是指针地址")
	}
	var handler Handler = nil
	handler = GetTxFormContext(ctx)
	if handler == nil {
		handler = impl.GetPool()
	}
	query = rebind(query)
	if EnableLog || ctx.Value(ContextKey_EnableLog) != nil {
		log.Debug("dbQueryRow", zap.String("sql", query), zap.Any("params", args))
	}
	ti := time.Now()
	row, err := handler.Query(ctx, query, args...)
	if err != nil {
		return false, err
	}
	// row := handler.QueryRowxContext(ctx, query, args...)
	go impl.addMonitorRecord(query, time.Now().Sub(ti), HandleTypeExec)
	err = impl.pgscanAPI.ScanOne(t, row)
	if err != nil {
		if _errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (impl *serviceImpl) HandleTx(ctx context.Context, txHanle func(ctx context.Context) error) error {
	ctx, create, headler, err := impl.GetOrCreateTxHandler(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	tx := headler.(pgx.Tx)
	defer func() {
		if create {
			if err != nil {
				tx.Rollback(ctx)
			} else {
				tx.Commit(ctx)
			}
		}
	}()
	err = txHanle(ctx)
	if err != nil {
		log.Error("SQL错误", zap.Error(err))
	}
	return err
}

func (impl *serviceImpl) DeleteById(ctx context.Context, tableName string, id int64) error {
	sql := fmt.Sprintf("delete from %s where id=?", tableName)
	_, err := impl.execContext(ctx, sql, id)
	if err != nil {
		log.DPanic("SQL错误", zap.Error(err))
	}
	return err
}

func (impl *serviceImpl) Start(ctx context.Context) error {
	return nil
}

func (impl *serviceImpl) Stop(ctx context.Context) error {
	return nil
}

func (impl *serviceImpl) execContext(ctx context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error) {
	var handler Handler = nil
	handler = GetTxFormContext(ctx)
	if handler == nil {
		handler = impl.GetPool()
	}
	sql = rebind(sql)
	if EnableLog || ctx.Value(ContextKey_EnableLog) != nil {
		log.Debug("dbExec", zap.String("sql", sql), zap.Any("params", args))
	}
	t := time.Now()
	commandTag, err := handler.Exec(ctx, sql, args...)
	if err != nil {
		log.DPanic("执行sql错误", zap.String("sql", sql), zap.Error(err))
		return pgconn.CommandTag{}, errors.ConverError(err)
	}
	go impl.addMonitorRecord(sql, time.Now().Sub(t), HandleTypeExec)
	return commandTag, nil
}
