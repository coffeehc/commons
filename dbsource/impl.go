package dbsource

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/coffeehc/commons/dbsource/sqlbuilder"
	"reflect"
	"strings"
	"time"

	"github.com/coffeehc/base/errors"
	"github.com/coffeehc/base/log"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

var ContextKey_EnableLog = "__DB_EnableLog"

var EnableLog = false

type Handler interface {
	sqlx.ExecerContext
	sqlx.PreparerContext
	sqlx.QueryerContext
}

type Service interface {
	GetXDB() *sqlx.DB
	RegisterHandleMonitor(monitor HandleMonitor)
	GetOrCreateContext(ctx context.Context) context.Context
	GetOrCreateHandler(ctx context.Context) (context.Context, bool, Handler)
	GetOrCreateTxHandler(ctx context.Context, options *sql.TxOptions) (context.Context, bool, Handler, error)
	Query(ctx context.Context, ts interface{}, colNames, tableName string, maxPageSize int64, query *sqlbuilder.Query, joinCondition *sqlbuilder.JoinCondition) (int64, error)
	Update(ctx context.Context, tableName string, limitFields map[string]bool, update *sqlbuilder.Update) (*sqlbuilder.UpdateResult, error)
	InsertContext(ctx context.Context, sql string, args ...interface{}) (int64, int64, error)
	ExecContext(ctx context.Context, sql string, args ...interface{}) (int64, error)
	QueryContext(ctx context.Context, ts interface{}, query string, args ...interface{}) error
	QueryRowContext(ctx context.Context, t interface{}, query string, args ...interface{}) (bool, error)
	HandleTx(ctx context.Context, txHanle func(ctx context.Context) error) error
	DeleteById(ctx context.Context, tableName string, id int64) error
	PgFormat() bool
}

func NewService(config *Config) Service {
	db := newDBSource(config)
	impl := &serviceImpl{db: db, config: config, monitors: make([]HandleMonitor, 0)}
	return impl
}

type serviceImpl struct {
	db       *sqlx.DB
	config   *Config
	monitors []HandleMonitor
}

func (impl *serviceImpl) PgFormat() bool {
	return impl.config.DbType == POSTGRES
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

func (impl *serviceImpl) addMonitorRecord(sql string, delay time.Duration, handleType HandleType) {
	for _, m := range impl.monitors {
		m.AddRecord(sql, delay, handleType)
	}
}

func (impl *serviceImpl) DeleteById(ctx context.Context, tableName string, id int64) error {
	sql := fmt.Sprintf("delete from %s where id=?", tableName)
	_, err := impl.execContext(ctx, sql, id)
	if err != nil {
		log.DPanic("SQL错误", zap.Error(err))
	}
	return err
}

func (impl *serviceImpl) InsertContext(ctx context.Context, sql string, args ...interface{}) (int64, int64, error) {
	ctx = impl.GetOrCreateContext(ctx)
	result, err := impl.execContext(ctx, sql, args...)
	if err != nil {
		log.DPanic("SQL错误", zap.Error(err))
		return 0, 0, err
	}
	lastInsertId, _ := result.LastInsertId()
	rowsAffected, _ := result.RowsAffected()
	return lastInsertId, rowsAffected, nil
}

func (impl *serviceImpl) ExecContext(ctx context.Context, sql string, args ...interface{}) (int64, error) {
	if impl.PgFormat() {
		sql = strings.ReplaceAll(sql, "`", "")
	}
	ctx = impl.GetOrCreateContext(ctx)
	result, _err := impl.execContext(ctx, sql, args...)
	if _err != nil {
		log.DPanic("SQL错误", zap.Error(_err))
		return 0, _err
	}
	count, err := result.RowsAffected()
	if err != nil {
		log.DPanic("获取执行行数错误", zap.Error(err))
		return 0, errors.ConverError(err)
	}
	return count, nil
}

func (impl *serviceImpl) QueryContext(ctx context.Context, ts interface{}, query string, args ...interface{}) error {
	if impl.PgFormat() {
		query = strings.ReplaceAll(query, "`", "")
	}
	ctx = impl.GetOrCreateContext(ctx)
	tType := reflect.TypeOf(ts)
	if tType.Kind() != reflect.Ptr {
		log.DPanic("需要填充的对象必须是指针地址")
		return errors.SystemError("需要填充的对象必须是指针地址")
	}
	handler := GetHandler(ctx)
	config := GetConfig(ctx)
	if config.EnableRebind {
		xdb := GetXDB(ctx)
		if xdb == nil {
			log.DPanic("上下文中没有Xdb对象")
		} else {
			query = xdb.Rebind(query)
		}
	}
	if EnableLog || ctx.Value(ContextKey_EnableLog) != nil {
		log.Debug("dbQuery", zap.String("sql", query), zap.Any("params", args))
	}
	t := time.Now()
	rows, err := handler.QueryxContext(ctx, query, args...)
	if err != nil {
		log.DPanic("执行sql错误", zap.Error(err), zap.String("sql", query))
		return errors.ConverError(err)
	}
	defer rows.Close()
	go impl.addMonitorRecord(query, time.Now().Sub(t), HandleTypeExec)
	return scanAll(rows, ts, false)
}

func (impl *serviceImpl) QueryRowContext(ctx context.Context, t interface{}, query string, args ...interface{}) (bool, error) {
	if impl.PgFormat() {
		query = strings.ReplaceAll(query, "`", "")
	}
	ctx = impl.GetOrCreateContext(ctx)
	tType := reflect.TypeOf(t)
	if tType.Kind() != reflect.Ptr {
		log.DPanic("需要填充的对象必须是指针地址")
		return false, errors.SystemError("需要填充的对象必须是指针地址")
	}
	handler := GetHandler(ctx)
	config := GetConfig(ctx)
	if config.EnableRebind {
		xdb := GetXDB(ctx)
		if xdb == nil {
			log.DPanic("上下文中没有Xdb对象")
		} else {
			query = xdb.Rebind(query)
		}
	}
	if EnableLog || ctx.Value(ContextKey_EnableLog) != nil {
		log.Debug("dbQueryRow", zap.String("sql", query), zap.Any("params", args))
	}
	ti := time.Now()
	row := handler.QueryRowxContext(ctx, query, args...)
	go impl.addMonitorRecord(query, time.Now().Sub(ti), HandleTypeExec)
	var err error
	switch tType.Elem().Kind() {
	case reflect.Struct:
		err = row.StructScan(t)
		break
	case reflect.Map:
		tmap, ok := t.(map[string]interface{})
		if !ok {
			return false, errors.SystemError("需要填充的目标参数不是map[string]interface{}")
		}
		err = row.MapScan(tmap)
		break
	default:
		err = row.Scan(t)
	}
	if err == nil {
		return true, nil
	}
	switch {
	case err == sql.ErrNoRows:
		if ctx.Value("__DB_EnableLog") != nil {
			log.Error("没有查到记录", zap.Error(err))
		}
		return false, nil
	case err == ctx.Err():
		log.DPanic("上下文结束", zap.Error(err))
		return false, nil
	default:
		log.DPanic("数据处理异常", zap.Error(err))
		return false, errors.ConverError(err)
	}
}

func (impl *serviceImpl) HandleTx(ctx context.Context, txHanle func(ctx context.Context) error) error {
	ctx, create, headler, err := impl.GetOrCreateTxHandler(ctx, nil)
	if err != nil {
		return err
	}
	tx := headler.(*sqlx.Tx)
	defer func() {
		if create {
			if err != nil {
				tx.Rollback()
			} else {
				tx.Commit()
			}
		}
	}()
	err = txHanle(ctx)
	if err != nil {
		log.DPanic("SQL错误", zap.Error(err))
	}
	return err
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

func (impl *serviceImpl) Query(ctx context.Context, ts interface{}, colNames, tableName string, maxPageSize int64, query *sqlbuilder.Query, joinCondition *sqlbuilder.JoinCondition) (int64, error) {
	ctx = impl.GetOrCreateContext(ctx)
	pageSqlContext, totalSqlContext := sqlbuilder.BuildQuery(colNames, tableName, maxPageSize, query, joinCondition, impl.config.DbType == POSTGRES)
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

func (impl *serviceImpl) GetXDB() *sqlx.DB {
	return impl.db
}

func (impl *serviceImpl) GetOrCreateContext(ctx context.Context) context.Context {
	ctx = impl.warpContext(ctx)
	headler := GetHandler(ctx)
	if headler != nil {
		return ctx
	}
	headler = impl.GetXDB()
	return SetHandler(ctx, headler)
}

func (impl *serviceImpl) GetOrCreateHandler(ctx context.Context) (context.Context, bool, Handler) {
	ctx = impl.warpContext(ctx)
	headler := GetHandler(ctx)
	if headler != nil {
		return ctx, false, headler
	}
	headler = impl.GetXDB()
	ctx = SetHandler(ctx, headler)
	return ctx, true, headler
}

func (impl *serviceImpl) GetOrCreateTxHandler(ctx context.Context, options *sql.TxOptions) (context.Context, bool, Handler, error) {
	ctx = impl.warpContext(ctx)
	handler := GetHandler(ctx)
	if handler != nil {
		return ctx, false, handler, nil
	}
	handler, err := impl.GetXDB().BeginTxx(ctx, options)
	if err != nil {
		log.DPanic("开始事务失败", zap.Error(err))
		return ctx, false, nil, errors.SystemError("开启事务失败")
	}
	ctx = SetHandler(ctx, handler)
	return ctx, true, handler, nil
}

func (impl *serviceImpl) execContext(ctx context.Context, sql string, args ...interface{}) (sql.Result, error) {
	handler := GetHandler(ctx)
	config := GetConfig(ctx)
	if config.EnableRebind {
		xdb := GetXDB(ctx)
		if xdb == nil {
			log.Panic("上下文中没有Xdb对象")
		}
		sql = xdb.Rebind(sql)
	}
	if EnableLog || ctx.Value(ContextKey_EnableLog) != nil {
		log.Debug("dbExec", zap.String("sql", sql), zap.Any("params", args))
	}
	t := time.Now()
	result, err := handler.ExecContext(ctx, sql, args...)
	if err != nil {
		log.DPanic("执行sql错误", zap.String("sql", sql), zap.Error(err))
		return nil, errors.ConverError(err)
	}
	go impl.addMonitorRecord(sql, time.Now().Sub(t), HandleTypeExec)
	return result, nil
}
