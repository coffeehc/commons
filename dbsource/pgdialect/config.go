package pgdialect

import (
	"errors"
	"net"
	"net/url"
	"os"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

var errInvalidConnectionConfig = errors.New("pgdialect: invalid PostgreSQL connection configuration")

// ParsePoolConfig parses an explicit PostgreSQL URL or keyword/value connection
// string without allowing PG* variables, OS account defaults, .pgpass, service
// files, or default client certificates to supply configuration. Host, user and
// database are required; port defaults to 5432 and sslmode to prefer. Explicit
// TLS certificate paths and PostgreSQL runtime/pool parameters are preserved.
// Passwords belong in the supplied configuration, not a separate passfile.
//
// pgx still reads its environment internally, but every such input is overridden
// before pgx validates it. The process environment is never changed.
func ParsePoolConfig(connString string) (*pgxpool.Config, error) {
	explicit, err := parseConnectionSettings(connString)
	if err != nil {
		return nil, errInvalidConnectionConfig
	}
	for _, key := range []string{"host", "user", "database"} {
		if strings.TrimSpace(explicit[key]) == "" {
			return nil, errors.New("pgdialect: PostgreSQL host, user and database must be explicit and nonempty")
		}
	}
	for _, host := range strings.Split(explicit["host"], ",") {
		if strings.TrimSpace(host) == "" {
			return nil, errInvalidConnectionConfig
		}
	}
	for _, key := range []string{"service", "servicefile", "passfile"} {
		if _, ok := explicit[key]; ok {
			return nil, errors.New("pgdialect: PostgreSQL service and password files are not supported; supply connection settings directly")
		}
	}

	// pgx treats even service='' as a request to load a service file. Supply a
	// real, empty service instead. This temporary descriptor contains no user
	// configuration or credentials and is removed before this function returns.
	service, err := os.CreateTemp("", "commons-pg-service-*")
	if err != nil {
		return nil, errors.New("pgdialect: cannot initialize isolated PostgreSQL configuration")
	}
	defer os.Remove(service.Name())
	if _, err = service.WriteString("[commons_config_only]\n"); err != nil {
		service.Close()
		return nil, errors.New("pgdialect: cannot initialize isolated PostgreSQL configuration")
	}
	if err = service.Close(); err != nil {
		return nil, errors.New("pgdialect: cannot initialize isolated PostgreSQL configuration")
	}

	// Keep this list in sync with pgconn.parseEnvSettings and defaultSettings
	// when upgrading pgx. Empty runtime defaults are removed after parsing so
	// the server retains its own application_name, timezone and options defaults.
	settings := map[string]string{
		"host": "", "port": "5432", "user": "", "database": "", "password": "",
		"passfile": os.DevNull, "service": "commons_config_only", "servicefile": service.Name(),
		"connect_timeout": "5", "application_name": "", "timezone": "", "options": "",
		"sslmode": "prefer", "sslcert": "", "sslkey": "", "sslrootcert": "", "sslpassword": "",
		"sslsni": "1", "sslnegotiation": "postgres", "target_session_attrs": "any",
		"min_protocol_version": "3.0", "max_protocol_version": "3.0", "channel_binding": "prefer",
	}
	for key, value := range explicit {
		settings[key] = value
	}
	keys := make([]string, 0, len(settings))
	for key := range settings {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	var dsn strings.Builder
	for _, key := range keys {
		dsn.WriteString(key)
		dsn.WriteString("='")
		dsn.WriteString(strings.NewReplacer(`\`, `\\`, `'`, `\'`).Replace(settings[key]))
		dsn.WriteString("' ")
	}
	config, err := pgxpool.ParseConfig(strings.TrimSpace(dsn.String()))
	if err != nil {
		// Driver parse errors contain the DSN and arbitrary parameter values.
		// Do not propagate either through startup logs or settings-test endpoints.
		return nil, errInvalidConnectionConfig
	}
	for _, key := range []string{"application_name", "timezone", "options"} {
		if _, ok := explicit[key]; !ok {
			delete(config.ConnConfig.RuntimeParams, key)
		}
	}
	return config, nil
}

func parseConnectionSettings(dsn string) (map[string]string, error) {
	settings := make(map[string]string)
	set := func(key, value string) error {
		if key == "dbname" {
			key = "database"
		}
		if key == "" || strings.IndexByte(value, 0) >= 0 {
			return errInvalidConnectionConfig
		}
		for _, ch := range key {
			if !(ch >= 'a' && ch <= 'z' || ch >= 'A' && ch <= 'Z' || ch >= '0' && ch <= '9' || ch == '_' || ch == '.') {
				return errInvalidConnectionConfig
			}
		}
		settings[key] = value
		return nil
	}
	dsn = strings.TrimSpace(dsn)
	if strings.HasPrefix(dsn, "postgres://") || strings.HasPrefix(dsn, "postgresql://") {
		u, err := url.Parse(dsn)
		if err != nil || u.Fragment != "" {
			return nil, errInvalidConnectionConfig
		}
		if u.User != nil {
			settings["user"] = u.User.Username()
			if password, present := u.User.Password(); present {
				settings["password"] = password
			}
		}
		if u.Host != "" {
			var hosts, ports []string
			for _, address := range strings.Split(u.Host, ",") {
				host, port := strings.Trim(address, "[]"), "5432"
				if strings.Contains(address, ":") && net.ParseIP(host) == nil {
					host, port, err = net.SplitHostPort(address)
					if err != nil || port == "" {
						return nil, errInvalidConnectionConfig
					}
				}
				hosts, ports = append(hosts, host), append(ports, port)
			}
			settings["host"], settings["port"] = strings.Join(hosts, ","), strings.Join(ports, ",")
		}
		if u.Path != "" {
			settings["database"] = strings.TrimLeft(u.Path, "/")
		}
		query, err := url.ParseQuery(u.RawQuery)
		if err != nil {
			return nil, errInvalidConnectionConfig
		}
		if _, database := query["database"]; database && query.Has("dbname") {
			return nil, errInvalidConnectionConfig
		}
		for key, values := range query {
			if len(values) != 1 || set(key, values[0]) != nil {
				return nil, errInvalidConnectionConfig
			}
		}
		for key, value := range settings {
			if set(key, value) != nil {
				return nil, errInvalidConnectionConfig
			}
		}
		return settings, nil
	}

	for dsn != "" {
		equal := strings.IndexByte(dsn, '=')
		if equal < 0 {
			return nil, errInvalidConnectionConfig
		}
		key := strings.TrimSpace(dsn[:equal])
		dsn = strings.TrimLeft(dsn[equal+1:], " \t\r\n\v\f")
		quoted := strings.HasPrefix(dsn, "'")
		if quoted {
			dsn = dsn[1:]
		}
		var value strings.Builder
		closed := !quoted
		i := 0
		for i < len(dsn) {
			ch := dsn[i]
			if ch == '\\' {
				i++
				if i == len(dsn) {
					return nil, errInvalidConnectionConfig
				}
				value.WriteByte(dsn[i])
				i++
				continue
			}
			if quoted && ch == '\'' {
				i++
				closed = true
				break
			}
			if !quoted && strings.ContainsRune(" \t\r\n\v\f", rune(ch)) {
				break
			}
			value.WriteByte(ch)
			i++
		}
		if !closed || i < len(dsn) && !strings.ContainsRune(" \t\r\n\v\f", rune(dsn[i])) {
			return nil, errInvalidConnectionConfig
		}
		if err := set(key, value.String()); err != nil {
			return nil, err
		}
		dsn = strings.TrimLeft(dsn[i:], " \t\r\n\v\f")
	}
	return settings, nil
}
