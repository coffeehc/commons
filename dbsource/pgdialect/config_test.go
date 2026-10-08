package pgdialect

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"math/big"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

func TestParsePoolConfigExplicitSettings(t *testing.T) {
	password := "spaces \\ quotes ' & = \n 密码"
	u := &url.URL{Scheme: "postgres", User: url.UserPassword("app user", password), Host: "db.example:5544", Path: "/app database"}
	q := url.Values{"sslmode": {"require"}, "search_path": {`"quoted schema"`}, "options": {"-c statement_timeout=4567"}, "application_name": {"explicit app"}, "timezone": {"Asia/Shanghai"}, "connect_timeout": {"9"}, "pool_max_conns": {"7"}}
	u.RawQuery = q.Encode()
	quoted := func(s string) string { return "'" + strings.NewReplacer(`\`, `\\`, `'`, `\'`).Replace(s) + "'" }
	for _, dsn := range []string{u.String(), "host=db.example port=5544 user='app user' dbname='app database' password=" + quoted(password) + " sslmode=require search_path='\"quoted schema\"' options='-c statement_timeout=4567' application_name='explicit app' timezone=Asia/Shanghai connect_timeout=9 pool_max_conns=7"} {
		pc, err := ParsePoolConfig(dsn)
		if err != nil {
			t.Fatal(err)
		}
		c := pc.ConnConfig
		if c.Host != "db.example" || c.Port != 5544 || c.User != "app user" || c.Database != "app database" || c.Password != password {
			t.Fatal("explicit connection values changed")
		}
		wantParams := map[string]string{"search_path": `"quoted schema"`, "options": "-c statement_timeout=4567", "application_name": "explicit app", "timezone": "Asia/Shanghai"}
		if !reflect.DeepEqual(c.RuntimeParams, wantParams) {
			t.Fatalf("runtime parameters = %#v", c.RuntimeParams)
		}
		if c.ConnectTimeout != 9*time.Second || pc.MaxConns != 7 || c.TLSConfig == nil || len(c.Fallbacks) != 0 || c.TLSConfig.ServerName != "db.example" {
			t.Fatal("explicit TLS, timeout or pool values changed")
		}
	}
}

func TestParsePoolConfigHostForms(t *testing.T) {
	for _, tc := range []struct {
		dsn, host    string
		port         uint16
		fallbackHost string
		fallbackPort uint16
	}{
		{"postgres://user:pass@[::1]:5544/db?sslmode=disable", "::1", 5544, "", 0},
		{"postgres://user:pass@[::1]/db?sslmode=disable", "::1", 5432, "", 0},
		{"postgres://user:pass@host1,host2:5544/db?sslmode=disable", "host1", 5432, "host2", 5544},
		{"host=/tmp/pg-socket user=user dbname=db sslmode=disable", "/tmp/pg-socket", 5432, "", 0},
		{`host=localhost user=app\ user dbname=db sslmode=disable`, "localhost", 5432, "", 0},
	} {
		pc, err := ParsePoolConfig(tc.dsn)
		if err != nil {
			t.Fatal(err)
		}
		c := pc.ConnConfig
		if c.Host != tc.host || c.Port != tc.port {
			t.Fatal("host or port changed")
		}
		if tc.fallbackHost != "" && (len(c.Fallbacks) != 1 || c.Fallbacks[0].Host != tc.fallbackHost || c.Fallbacks[0].Port != tc.fallbackPort) {
			t.Fatal("fallback host or port changed")
		}
	}
}

func TestParsePoolConfigRejectsInvalidWithoutSecrets(t *testing.T) {
	const secret = "do-not-disclose-this-password"
	for _, dsn := range []string{
		"", "host=localhost dbname=db", "host=localhost user='' dbname=db", "user=app dbname=db", "host=localhost user=app",
		"host=localhost user=app dbname=db password='" + secret,
		"host=localhost user=app dbname=db password=" + secret + " port=invalid",
		"host=localhost user=app dbname=db password=" + secret + " sslmode=invalid",
		"host=localhost user=app dbname=db password=" + secret + " min_protocol_version=invalid",
		"host=localhost user=app dbname=db password=" + secret + " max_protocol_version=invalid",
		"host=localhost user=app dbname=db password=" + secret + " pool_max_conns=invalid",
		"host=localhost user=app dbname=db service=hidden", "host=localhost user=app dbname=db servicefile=hidden", "host=localhost user=app dbname=db passfile=hidden",
		"host=localhost user=app dbname=db password='" + secret + "'broken", "host=localhost user=app dbname=db password=" + secret + `\`,
		"postgres://user:" + secret + "@localhost/db?port=%xx", "postgres://user:" + secret + "@localhost/db?port=1&port=2",
		"postgres://user:" + secret + "@localhost/db?database=a&dbname=b", "postgres://user:" + secret + "@localhost/db?password=%00",
	} {
		if _, err := ParsePoolConfig(dsn); err == nil || strings.Contains(err.Error(), secret) {
			t.Fatal("expected a sanitized configuration error")
		}
	}
}

// Use an isolated child environment, never Setenv/Unsetenv/Clearenv. That also
// lets the race test call the parser concurrently while checking all variables
// remain byte-for-byte unchanged.
func TestParsePoolConfigPollutedEnvironment(t *testing.T) {
	dir := t.TempDir()
	for name, data := range map[string]string{".pgpass": "*:*:*:*:ambient-password\n", ".pg_service.conf": "[ambient]\nhost=ambient-host\nuser=ambient-user\npassword=ambient-password\n", "bad-cert": "not a certificate"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
	}
	polluted := map[string]string{
		"HOME": dir, "USERPROFILE": dir, "TMPDIR": dir,
		"PGHOST": "ambient-host", "PGPORT": "invalid", "PGUSER": "ambient-user", "PGDATABASE": "ambient-db", "PGPASSWORD": "ambient-password",
		"PGPASSFILE": filepath.Join(dir, ".pgpass"), "PGSERVICE": "ambient", "PGSERVICEFILE": filepath.Join(dir, ".pg_service.conf"),
		"PGAPPNAME": "ambient-app", "PGCONNECT_TIMEOUT": "invalid", "PGSSLMODE": "invalid", "PGSSLSNI": "0",
		"PGSSLKEY": filepath.Join(dir, "bad-cert"), "PGSSLCERT": filepath.Join(dir, "bad-cert"), "PGSSLROOTCERT": filepath.Join(dir, "bad-cert"), "PGSSLPASSWORD": "ambient-password",
		"PGSSLNEGOTIATION": "direct", "PGTARGETSESSIONATTRS": "invalid", "PGTZ": "Invalid/Zone", "PGOPTIONS": "-c search_path=ambient_schema",
		"PGMINPROTOCOLVERSION": "invalid", "PGMAXPROTOCOLVERSION": "invalid",
	}
	cmd := exec.Command(os.Args[0], "-test.run=^TestParsePoolConfigPollutedChild$", "--", "pg-isolation-child")
	for _, entry := range os.Environ() {
		key, _, _ := strings.Cut(entry, "=")
		if _, replaced := polluted[key]; !replaced {
			cmd.Env = append(cmd.Env, entry)
		}
	}
	for key, value := range polluted {
		cmd.Env = append(cmd.Env, key+"="+value)
	}
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("isolated parser regression failed: %v\n%s", err, out)
	}
	files, err := filepath.Glob(filepath.Join(dir, "commons-pg-service-*"))
	if err != nil || len(files) != 0 {
		t.Fatal("temporary service descriptor was not removed")
	}
}

func TestParsePoolConfigPollutedChild(t *testing.T) {
	if os.Args[len(os.Args)-1] != "pg-isolation-child" {
		t.Skip("subprocess helper")
	}
	before := os.Environ()
	var wg sync.WaitGroup
	for i := 0; i < 24; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			dsn := fmt.Sprintf("host=db.example user=explicit-user dbname=explicit-db password='' search_path='\"schema %d\"'", i)
			pc, err := ParsePoolConfig(dsn)
			if err != nil {
				t.Error(err)
				return
			}
			c := pc.ConnConfig
			if c.Host != "db.example" || c.Port != 5432 || c.User != "explicit-user" || c.Database != "explicit-db" || c.Password != "" || c.ConnectTimeout != 5*time.Second {
				t.Error("ambient environment changed connection configuration")
			}
			if c.MinProtocolVersion != "3.0" || c.MaxProtocolVersion != "3.0" || c.ChannelBinding != "prefer" || c.SSLNegotiation != "postgres" || c.ValidateConnect != nil {
				t.Error("ambient protocol settings were used")
			}
			if c.TLSConfig == nil || !c.TLSConfig.InsecureSkipVerify || c.TLSConfig.ServerName != "db.example" || c.TLSConfig.RootCAs != nil || len(c.TLSConfig.Certificates) != 0 || len(c.Fallbacks) != 1 || c.Fallbacks[0].TLSConfig != nil {
				t.Error("ambient TLS settings were used")
			}
			if !reflect.DeepEqual(c.RuntimeParams, map[string]string{"search_path": fmt.Sprintf(`"schema %d"`, i)}) {
				t.Error("ambient runtime parameters were used")
			}
		}(i)
	}
	wg.Wait()
	if !reflect.DeepEqual(before, os.Environ()) {
		t.Fatal("parser modified the process environment")
	}
	if _, err := ParsePoolConfig("host=db.example dbname=db"); err == nil {
		t.Fatal("ambient OS/PGUSER account was accepted")
	}
	// Only a caller-supplied disposable test DSN can enable live verification.
	// Never discover a database from application settings or the polluted PG* values.
	if dsn := os.Getenv("DBSOURCE_TEST_POSTGRES"); dsn != "" {
		pc, err := ParsePoolConfig(dsn)
		if err != nil {
			t.Fatal(err)
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		conn, err := pgx.ConnectConfig(ctx, pc.ConnConfig)
		if err != nil {
			t.Fatal("explicit disposable PostgreSQL connection failed")
		}
		defer conn.Close(context.Background())
		var n int
		if err := conn.QueryRow(ctx, "SELECT 1").Scan(&n); err != nil || n != 1 {
			t.Fatal("explicit disposable PostgreSQL query failed")
		}
	}
}

func TestParsePoolConfigExplicitCertificates(t *testing.T) {
	pub, key, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	template := &x509.Certificate{SerialNumber: big.NewInt(1), NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(time.Hour), IsCA: true, BasicConstraintsValid: true, KeyUsage: x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature}
	der, err := x509.CreateCertificate(rand.Reader, template, template, pub, key)
	if err != nil {
		t.Fatal(err)
	}
	keyDER, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	certPath, keyPath := filepath.Join(dir, "explicit.crt"), filepath.Join(dir, "explicit.key")
	if err := os.WriteFile(certPath, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(keyPath, pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyDER}), 0600); err != nil {
		t.Fatal(err)
	}
	q := url.Values{"sslmode": {"verify-full"}, "sslrootcert": {certPath}, "sslcert": {certPath}, "sslkey": {keyPath}, "channel_binding": {"require"}}
	pc, err := ParsePoolConfig("postgres://user@db.example/db?" + q.Encode())
	if err != nil {
		t.Fatal(err)
	}
	c := pc.ConnConfig
	if c.TLSConfig == nil || c.TLSConfig.InsecureSkipVerify || c.TLSConfig.ServerName != "db.example" || c.TLSConfig.RootCAs == nil || len(c.TLSConfig.Certificates) != 1 || c.ChannelBinding != "require" {
		t.Fatal("explicit TLS configuration was not preserved")
	}
	q.Set("sslkey", filepath.Join(dir, "missing.key"))
	if _, err := ParsePoolConfig("postgres://user@db.example/db?" + q.Encode()); err == nil {
		t.Fatal("invalid explicit TLS file was ignored")
	}
}
