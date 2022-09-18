package dbsource

// DatabaseConfig 数据库配置
type Config struct {
	DBName             string `mapstructure:"db_name,omitempty" json:"db_name,omitempty"`
	User               string `mapstructure:"user,omitempty" json:"user,omitempty"`
	Password           string `mapstructure:"password,omitempty" json:"password,omitempty"`
	Host               string `mapstructure:"host,omitempty" json:"host,omitempty"`
	Port               int    `mapstructure:"port,omitempty" json:"port,omitempty"`
	DbType             DbType `mapstructure:"db_type,omitempty" json:"db_type,omitempty"`
	EnableRebind       bool   `mapstructure:"enable_rebind,omitempty" json:"enable_rebind,omitempty"`
	LocalDbPath        string `mapstructure:"local_db_path,omitempty" json:"local_db_path,omitempty"`
	MaxOpenConns       int    `mapstructure:"max_open_conns,omitempty" json:"max_open_conns,omitempty"`
	MaxIdleConns       int    `mapstructure:"max_idle_conns,omitempty" json:"max_idle_conns,omitempty"`
	ConnMaxLifetimeSec int    `mapstructure:"conn_max_lifetime_sec,omitempty" json:"conn_max_lifetime_sec,omitempty"`
}

func (impl *Config) getDBType() DbType {
	if impl.DbType == "" {
		return POSTGRES
	}
	return impl.DbType
}
