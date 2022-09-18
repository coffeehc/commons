package dbsource

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/coffeehc/base/errors"
	"github.com/coffeehc/base/log"
	"github.com/jmoiron/sqlx"
	"github.com/jmoiron/sqlx/reflectx"
	"go.uber.org/zap"
)

type DbType string

const (
	MYSQL    DbType = "mysql"
	POSTGRES DbType = "postgres"
	SQLITE   DbType = "sqlite3"
)

func buildDataSourceNameForMySql(config *Config) string {
	values := make(url.Values)
	values.Set("charset", "utf8mb4")
	values.Set("interpolateParams", "true")
	values.Set("parseTime", "true")
	values.Set("loc", "Local")
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?%s", config.User, config.Password, config.Host, config.Port, config.DBName, values.Encode())
}

func buildDataSourceNameForPostgresSQL(config *Config) string {
	params := make([]string, 0)
	params = append(params, fmt.Sprintf("dbname='%s'", config.DBName))
	params = append(params, fmt.Sprintf("user='%s'", config.User))
	params = append(params, fmt.Sprintf("password='%s'", config.Password))
	params = append(params, fmt.Sprintf("host='%s'", config.Host))
	params = append(params, fmt.Sprintf("port='%d'", config.Port))
	params = append(params, "sslmode=disable")
	return strings.Join(params, " ")
}

func budilDataSourcwNameForSqlit(config *Config) string {
	return config.LocalDbPath
}

func newDBSource(config *Config) *sqlx.DB {
	var db *sqlx.DB
	var err error
	var dataSource = ""
	switch config.getDBType() {
	case MYSQL:
		dataSource = buildDataSourceNameForMySql(config)
	case POSTGRES:
		dataSource = buildDataSourceNameForPostgresSQL(config)
	case SQLITE:
		dataSource = budilDataSourcwNameForSqlit(config)
	}
	db, err = sqlx.Open(string(config.getDBType()), dataSource)
	if err != nil {
		log.Panic("打开数据库失败", zap.Error(err))
	}
	// log.Debug("打开数据库", zap.String("dataSource", dataSource))
	if config.ConnMaxLifetimeSec > 60 {
		db.SetConnMaxLifetime(time.Second * time.Duration(config.ConnMaxLifetimeSec))
	} else {
		db.SetConnMaxLifetime(time.Second * 60)
	}
	if config.MaxIdleConns > 3 {
		db.SetMaxIdleConns(config.MaxIdleConns)
	} else {
		db.SetMaxIdleConns(3)
	}
	if config.MaxOpenConns > 10 {
		db.SetMaxOpenConns(config.MaxOpenConns)
	} else {
		db.SetMaxOpenConns(10)
	}

	db.Mapper = reflectx.NewMapperFunc("json", strings.ToLower)
	err = db.Ping()
	if err != nil {
		log.Panic("db连接失败", zap.Error(err))
	}
	return db
}

var ErrorCountDiff = errors.MessageError("变更数据量不符合预期")

func CheckRowsAffected(result sql.Result, okCount int64) error {
	count, err := result.RowsAffected()
	if err != nil {
		return errors.ConverError(err)
	}
	if count != okCount {
		return ErrorCountDiff
	}
	return nil
}

func SetValue(params map[string]interface{}, name string, value interface{}, removeNull bool) {
	if value == nil {
		return
	}
	if removeNull {
		switch value.(type) {
		case string:
			if value == "" {
				return
			}
			break
		case int, int8, int64, int16, int32, float32, float64:
			if value == 0 {
				return
			}
			break
		case []byte:
			if len(value.([]byte)) == 0 {
				return
			}
			break
		case time.Time:
			if value.(time.Time).IsZero() {
				return
			}
			break
		}
	}
	params[name] = value
}

const contextXDBKey = "__dbsource_xdb"
const contextConfigKey = "__dbsource_config"

func (impl *serviceImpl) warpContext(ctx context.Context) context.Context {
	return impl.setXDB(impl.setConfig(ctx))
}

func (impl *serviceImpl) setXDB(ctx context.Context) context.Context {
	xdb := GetXDB(ctx)
	if xdb != nil {
		return ctx
	}
	return context.WithValue(ctx, contextXDBKey, impl.db)
}

func (impl *serviceImpl) setConfig(ctx context.Context) context.Context {
	xdb := GetConfig(ctx)
	if xdb != nil {
		return ctx
	}
	return context.WithValue(ctx, contextConfigKey, impl.config)
}

func GetXDB(ctx context.Context) *sqlx.DB {
	v := ctx.Value(contextXDBKey)
	if v == nil {
		return nil
	}
	return v.(*sqlx.DB)
}

func GetConfig(ctx context.Context) *Config {
	v := ctx.Value(contextConfigKey)
	if v == nil {
		return nil
	}
	return v.(*Config)
}

var ContextHandlerKey = "__dbsource_headler"

func SetHandler(ctx context.Context, handler Handler) context.Context {
	_handler := GetHandler(ctx)
	if _handler != nil {
		// log.Warn("上下文中已经存在了数据库处理对象", zap.Any("handler", handler))
		return ctx
	}
	return context.WithValue(ctx, ContextHandlerKey, handler)
}

func GetHandler(ctx context.Context) Handler {
	v := ctx.Value(ContextHandlerKey)
	if v == nil {
		return nil
	}
	return v.(Handler)
}
