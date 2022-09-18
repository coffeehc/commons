package dbsource_test

import (
	"context"
	"database/sql"
	"github.com/coffeehc/commons/dbsource"
	"testing"
	"time"

	"github.com/coffeehc/base/log"
	"github.com/coffeehc/boot/testutils"
	_ "github.com/lib/pq"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

type table_x struct {
	A sql.NullString
	B sql.NullBool
	C sql.NullTime
	D sql.NullInt64
	E sql.NullInt32
	F sql.NullFloat64
}

type dto_x struct {
	A string
	B bool
	C int64
	D int64
	E int32
	F float64
}

func TestConvertTableToDto(t *testing.T) {
	testutils.InitTestConfig()
	log.InitLogger(true)
	table := &table_x{
		A: sql.NullString{
			"haha", true,
		},
		B: sql.NullBool{
			Valid: true,
			Bool:  true,
		},
		C: sql.NullTime{
			Time:  time.Now(),
			Valid: true,
		},
		D: sql.NullInt64{
			Int64: 987,
			Valid: true,
		},
		E: sql.NullInt32{
			Valid: true,
			Int32: 75,
		},
		F: sql.NullFloat64{
			Valid:   true,
			Float64: 0.123456,
		},
	}
	d := &dto_x{}
	err := dbsource.TableToDTOConvert(table, d)
	if err != nil {
		t.Fatal(err)
	}
	log.Debug("对了", zap.Any("dto", d))
	d = &dto_x{
		A: "1234556",
		B: true,
		C: 1596527130129078000,
		D: 123,
		E: 456,
		F: 0.123,
	}
	table = &table_x{}
	dbsource.DTOToTableConvert(d, table)
	log.Debug("对了", zap.Any("table", table))
}

func TestServiceImpl_GetDB(t *testing.T) {
	config := &dbsource.Config{
		DBName:   "douyin",
		User:     "douyin",
		Password: "douyin###123",
		Host:     "192.168.3.2",
		Port:     5433,
	}
	viper.Set("dbSource", config)
	ctx := context.TODO()
	dbsource.EnablePlugin(ctx)
	service := dbsource.GetService()
	db := service.GetXDB()
	err := db.PingContext(context.TODO())
	if err != nil {
		log.Error("ping服务错误", zap.Error(err))
	}
	log.Debug("创建数据源成功")
}
