package dbsource

import (
	"database/sql"
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/coffeehc/base/errors"
	"github.com/go-sql-driver/mysql"
	"github.com/jackc/pgx/v5"
	"github.com/jmoiron/sqlx/reflectx"
)

// NewMapperFunc creates a sqlx field mapper using tag and lowercase field-name fallback.
func NewMapperFunc(tag string) *reflectx.Mapper {
	return reflectx.NewMapperFunc(tag, strings.ToLower)
}

// JSONMapperFunc maps database columns through json struct tags.
var JSONMapperFunc = NewMapperFunc("json")

// DBMapperFunc maps database columns through db struct tags.
var DBMapperFunc = NewMapperFunc("db")

// DbType identifies one supported database dialect.
type DbType string

const (
	// MYSQL selects the sqlx-backed MySQL dialect.
	MYSQL DbType = "mysql"
	// POSTGRES selects the native pgxpool-backed PostgreSQL dialect.
	POSTGRES DbType = "postgres"
	// SQLITE selects the sqlx-backed pure-Go SQLite dialect.
	SQLITE DbType = "sqlite"
)

func buildDataSourceNameForMySQL(config *Config) string {
	mysqlConfig := mysql.NewConfig()
	mysqlConfig.User = config.User
	mysqlConfig.Passwd = config.Password
	mysqlConfig.Net = "tcp"
	mysqlConfig.Addr = net.JoinHostPort(config.Host, strconv.Itoa(config.Port))
	mysqlConfig.DBName = config.DBName
	mysqlConfig.Params = map[string]string{"charset": "utf8mb4"}
	mysqlConfig.InterpolateParams = true
	mysqlConfig.ParseTime = true
	mysqlConfig.Loc = time.Local
	return mysqlConfig.FormatDSN()
}

func buildDataSourceNameForPostgresSQL(config *Config) string {
	databaseURL := &url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(config.User, config.Password),
		Host:   net.JoinHostPort(config.Host, strconv.Itoa(config.Port)),
		Path:   config.DBName,
	}
	query := databaseURL.Query()
	if config.SearchPath != "" {
		query.Set("search_path", pgx.Identifier{config.SearchPath}.Sanitize())
	}
	if config.SSLMode != "" {
		query.Set("sslmode", string(config.SSLMode))
	}
	if config.SSLRootCert != "" {
		query.Set("sslrootcert", config.SSLRootCert)
	}
	if config.SSLCert != "" {
		query.Set("sslcert", config.SSLCert)
	}
	if config.SSLKey != "" {
		query.Set("sslkey", config.SSLKey)
	}
	databaseURL.RawQuery = query.Encode()
	return databaseURL.String()
}

// ErrorCountDiff reports that a write changed a different number of rows than required.
var ErrorCountDiff = errors.MessageError("变更数据量不符合预期")

// CheckRowsAffected verifies that result changed exactly okCount rows.
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

// SetValue writes one non-nil value into params and can omit supported zero values.
func SetValue(params map[string]interface{}, name string, value interface{}, removeNull bool) {
	if value == nil {
		return
	}
	if removeNull {
		switch typedValue := value.(type) {
		case string:
			if typedValue == "" {
				return
			}
		case int:
			if typedValue == 0 {
				return
			}
		case int8:
			if typedValue == 0 {
				return
			}
		case int16:
			if typedValue == 0 {
				return
			}
		case int32:
			if typedValue == 0 {
				return
			}
		case int64:
			if typedValue == 0 {
				return
			}
		case float32:
			if typedValue == 0 {
				return
			}
		case float64:
			if typedValue == 0 {
				return
			}
		case []byte:
			if len(typedValue) == 0 {
				return
			}
		case time.Time:
			if typedValue.IsZero() {
				return
			}
		}
	}
	params[name] = value
}
