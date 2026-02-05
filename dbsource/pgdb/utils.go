package pgdb

import (
	"context"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/coffeehc/base/log"
	"github.com/georgysavva/scany/v2/dbscan"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

func MultiTagMapper(field reflect.StructField) string {
	// 1. db tag
	if tag := field.Tag.Get("db"); tag != "" && tag != "-" {
		return tag
	}
	// 2. json tag
	if tag := field.Tag.Get("json"); tag != "" && tag != "-" {
		return strings.Split(tag, ",")[0]
	}
	return dbscan.SnakeCaseMapper(field.Name)
}

func newPool(config *Config) (*pgxpool.Pool, error) {
	params := make([]string, 0)
	params = append(params, fmt.Sprintf("dbname='%s'", config.DBName))
	params = append(params, fmt.Sprintf("user='%s'", config.User))
	params = append(params, fmt.Sprintf("password='%s'", config.Password))
	params = append(params, fmt.Sprintf("host='%s'", config.Host))
	params = append(params, fmt.Sprintf("port='%d'", config.Port))
	params = append(params, "sslmode=disable")
	poolConfig, err := pgxpool.ParseConfig(strings.Join(params, " "))
	if err != nil {
		return nil, err
	}
	poolConfig.HealthCheckPeriod = 2 * time.Minute
	poolConfig.ConnConfig.ConnectTimeout = 5 * time.Second
	if config.ConnMaxLifetimeSec > 60 {
		poolConfig.MaxConnLifetime = time.Second * time.Duration(config.ConnMaxLifetimeSec)
	} else {
		poolConfig.MaxConnLifetime = time.Second * 60
	}
	if config.MaxIdleConns > 5 {
		poolConfig.MinIdleConns = int32(config.MaxIdleConns)
	} else {
		poolConfig.MinIdleConns = int32(5)
	}
	if config.MaxOpenConns > 15 {
		poolConfig.MaxConns = int32(config.MaxOpenConns)
	} else {
		poolConfig.MaxConns = int32(15)
	}
	pool, err := pgxpool.NewWithConfig(context.Background(), poolConfig)
	if err != nil {
		log.Error("创建连接池失败", zap.Error(err))
		return nil, err
	}
	ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
	if err := pool.Ping(ctx); err != nil {
		log.Error("无法连接到数据库", zap.Error(err))
		return nil, err
	}
	return pool, nil
}

func rebind(query string) string {
	rqb := make([]byte, 0, len(query)+10)
	var i, j int
	for i = strings.Index(query, "?"); i != -1; i = strings.Index(query, "?") {
		rqb = append(rqb, query[:i]...)
		rqb = append(rqb, '$')
		j++
		rqb = strconv.AppendInt(rqb, int64(j), 10)
		query = query[i+1:]
	}
	return string(append(rqb, query...))
}
