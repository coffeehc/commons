package dbsource

import "github.com/jmoiron/sqlx/reflectx"

// PostgresSSLMode selects PostgreSQL transport encryption and certificate verification.
type PostgresSSLMode string

const (
	// PostgresSSLModeDisable disables TLS explicitly.
	PostgresSSLModeDisable PostgresSSLMode = "disable"
	// PostgresSSLModeAllow tries plaintext before TLS.
	PostgresSSLModeAllow PostgresSSLMode = "allow"
	// PostgresSSLModePrefer tries TLS before falling back to plaintext.
	PostgresSSLModePrefer PostgresSSLMode = "prefer"
	// PostgresSSLModeRequire requires TLS without certificate verification.
	PostgresSSLModeRequire PostgresSSLMode = "require"
	// PostgresSSLModeVerifyCA requires TLS and verifies the certificate authority.
	PostgresSSLModeVerifyCA PostgresSSLMode = "verify-ca"
	// PostgresSSLModeVerifyFull requires TLS and verifies the certificate authority and host name.
	PostgresSSLModeVerifyFull PostgresSSLMode = "verify-full"
)

// Config defines one database backend and its connection-pool limits.
type Config struct {
	// DBName is the remote database name; it is unused for SQLite.
	DBName string `mapstructure:"db_name,omitempty" json:"db_name,omitempty"`
	// User is the remote database account name; it is unused for SQLite.
	User string `mapstructure:"user,omitempty" json:"user,omitempty"`
	// Password is the remote database account secret; it is unused for SQLite.
	Password string `mapstructure:"password,omitempty" json:"password,omitempty"`
	// Host is the remote database host name or IP address; it is unused for SQLite.
	Host string `mapstructure:"host,omitempty" json:"host,omitempty"`
	// Port is the remote database TCP port; it is unused for SQLite.
	Port int `mapstructure:"port,omitempty" json:"port,omitempty"`
	// SSLMode selects PostgreSQL transport verification. Empty keeps the pgx default.
	SSLMode PostgresSSLMode `mapstructure:"ssl_mode,omitempty" json:"ssl_mode,omitempty"`
	// SSLRootCert is the PostgreSQL root CA file used by certificate verification.
	SSLRootCert string `mapstructure:"ssl_root_cert,omitempty" json:"ssl_root_cert,omitempty"`
	// SSLCert is the PostgreSQL client certificate file used by mutual TLS.
	SSLCert string `mapstructure:"ssl_cert,omitempty" json:"ssl_cert,omitempty"`
	// SSLKey is the PostgreSQL client private-key file used by mutual TLS.
	SSLKey string `mapstructure:"ssl_key,omitempty" json:"ssl_key,omitempty"`
	// DbType selects the dialect implementation. Empty means PostgreSQL.
	DbType DbType `mapstructure:"db_type,omitempty" json:"db_type,omitempty"`
	// LocalDbPath is the SQLite data source name; it is unused for remote databases.
	LocalDbPath string `mapstructure:"local_db_path,omitempty" json:"local_db_path,omitempty"`
	// MaxOpenConns is the maximum number of open connections; zero keeps backend defaults.
	MaxOpenConns int `mapstructure:"max_open_conns,omitempty" json:"max_open_conns,omitempty"`
	// MaxIdleConns is the maximum number of idle database/sql connections; zero keeps defaults.
	// pgxpool does not have an equivalent maximum-idle-count setting and ignores this field.
	MaxIdleConns int `mapstructure:"max_idle_conns,omitempty" json:"max_idle_conns,omitempty"`
	// ConnMaxLifetimeSec is the maximum connection lifetime in seconds; zero keeps backend defaults.
	ConnMaxLifetimeSec int `mapstructure:"conn_max_lifetime_sec,omitempty" json:"conn_max_lifetime_sec,omitempty"`
	// Mapper controls struct field mapping. Nil uses dbsource.DBMapperFunc.
	Mapper *reflectx.Mapper `mapstructure:"-" json:"-"`
}

func (impl *Config) getDBType() DbType {
	if impl.DbType == "" {
		return POSTGRES
	}
	return impl.DbType
}
