package dbsource

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/jackc/pgx/v5/pgxpool"
)

func newSQLiteTestService(t *testing.T) *serviceImpl {
	t.Helper()
	databaseName := strings.NewReplacer("/", "_", " ", "_").Replace(t.Name())
	service := NewService(&Config{
		DbType:             SQLITE,
		LocalDbPath:        fmt.Sprintf("file:%s?mode=memory&cache=shared", databaseName),
		MaxOpenConns:       3,
		MaxIdleConns:       1,
		ConnMaxLifetimeSec: 30,
	}).(*serviceImpl)
	t.Cleanup(func() {
		_ = service.Stop(context.Background())
	})
	return service
}

func createTestRecordsTable(t *testing.T, service *serviceImpl) {
	t.Helper()
	if _, err := service.ExecContext(context.Background(), `CREATE TABLE test_records (id INTEGER PRIMARY KEY, name TEXT NOT NULL)`); err != nil {
		t.Fatalf("create test table: %v", err)
	}
}

type recordingMonitor struct {
	count atomic.Int64
}

func (*recordingMonitor) Name() string {
	return "recordingMonitor"
}

func (impl *recordingMonitor) AddRecord(string, time.Duration, HandleType) {
	impl.count.Add(1)
}

func testRecordCount(t *testing.T, service *serviceImpl) int {
	t.Helper()
	var count int
	found, err := service.QueryRowContext(context.Background(), &count, `SELECT COUNT(*) FROM test_records`)
	if err != nil {
		t.Fatalf("count test records: %v", err)
	}
	if !found {
		t.Fatal("count test records: no result")
	}
	return count
}

func TestDeleteByIDAcceptsRawContext(t *testing.T) {
	service := newSQLiteTestService(t)
	createTestRecordsTable(t, service)
	if _, err := service.ExecContext(context.Background(), `INSERT INTO test_records (id, name) VALUES (1, 'one')`); err != nil {
		t.Fatalf("insert test record: %v", err)
	}

	if err := service.DeleteById(context.Background(), "test_records", 1); err != nil {
		t.Fatalf("DeleteById() error = %v", err)
	}
	if count := testRecordCount(t, service); count != 0 {
		t.Fatalf("record count = %d, want 0", count)
	}
}

func TestDefaultMapperUsesDBTags(t *testing.T) {
	service := NewService(&Config{
		DbType:      SQLITE,
		LocalDbPath: "file:default_mapper_uses_db_tags?mode=memory&cache=shared",
	}).(*serviceImpl)
	t.Cleanup(func() { _ = service.Stop(context.Background()) })
	if _, err := service.ExecContext(context.Background(), `CREATE TABLE mapper_records (record_name TEXT NOT NULL)`); err != nil {
		t.Fatalf("create mapper table: %v", err)
	}
	if _, err := service.ExecContext(context.Background(), `INSERT INTO mapper_records(record_name) VALUES (?)`, "db-value"); err != nil {
		t.Fatalf("insert mapper record: %v", err)
	}
	record := struct {
		Name string `json:"json_name" db:"record_name"`
	}{}
	found, err := service.QueryRowContext(context.Background(), &record, `SELECT record_name FROM mapper_records`)
	if err != nil {
		t.Fatalf("QueryRowContext() error = %v", err)
	}
	if !found || record.Name != "db-value" {
		t.Fatalf("record = %+v, found = %v", record, found)
	}
}

func TestExplicitJSONMapperRemainsSupported(t *testing.T) {
	service := NewService(&Config{
		DbType:      SQLITE,
		LocalDbPath: "file:explicit_json_mapper?mode=memory&cache=shared",
		Mapper:      JSONMapperFunc,
	}).(*serviceImpl)
	t.Cleanup(func() { _ = service.Stop(context.Background()) })
	if _, err := service.ExecContext(context.Background(), `CREATE TABLE mapper_records (record_name TEXT NOT NULL)`); err != nil {
		t.Fatalf("create mapper table: %v", err)
	}
	if _, err := service.ExecContext(context.Background(), `INSERT INTO mapper_records(record_name) VALUES (?)`, "json-value"); err != nil {
		t.Fatalf("insert mapper record: %v", err)
	}
	record := struct {
		Name string `json:"record_name" db:"db_name"`
	}{}
	found, err := service.QueryRowContext(context.Background(), &record, `SELECT record_name FROM mapper_records`)
	if err != nil {
		t.Fatalf("QueryRowContext() error = %v", err)
	}
	if !found || record.Name != "json-value" {
		t.Fatalf("record = %+v, found = %v", record, found)
	}
}

func TestHandleTxStartsTransactionWithInitializedContext(t *testing.T) {
	service := newSQLiteTestService(t)
	createTestRecordsTable(t, service)
	wantErr := errors.New("abort transaction")
	err := service.HandleTx(context.Background(), func(txCtx context.Context) error {
		if _, execErr := service.ExecContext(txCtx, `INSERT INTO test_records (id, name) VALUES (?, ?)`, 1, "one"); execErr != nil {
			return execErr
		}
		return wantErr
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("HandleTx() error = %v, want %v", err, wantErr)
	}
	if count := testRecordCount(t, service); count != 0 {
		t.Fatalf("record count after rollback = %d, want 0", count)
	}
}

func TestHandleTxRollsBackOnPanic(t *testing.T) {
	service := newSQLiteTestService(t)
	createTestRecordsTable(t, service)

	func() {
		defer func() {
			if recover() == nil {
				t.Fatal("HandleTx() did not propagate panic")
			}
		}()
		_ = service.HandleTx(context.Background(), func(txCtx context.Context) error {
			if _, err := service.ExecContext(txCtx, `INSERT INTO test_records (id, name) VALUES (?, ?)`, 1, "one"); err != nil {
				return err
			}
			panic("boom")
		})
	}()

	if count := testRecordCount(t, service); count != 0 {
		t.Fatalf("record count after panic rollback = %d, want 0", count)
	}
}

func TestQueryRowContextReturnsCancellation(t *testing.T) {
	service := newSQLiteTestService(t)
	createTestRecordsTable(t, service)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	record := struct {
		ID int64 `db:"id"`
	}{}

	found, err := service.QueryRowContext(ctx, &record, `SELECT id FROM test_records WHERE id = ?`, 1)
	if err == nil {
		t.Fatal("QueryRowContext() error = nil, want cancellation error")
	}
	if found {
		t.Fatal("QueryRowContext() found = true, want false")
	}
}

func TestPreparedStatementAndStreamingRowsUseUnifiedInterfaces(t *testing.T) {
	service := newSQLiteTestService(t)
	createTestRecordsTable(t, service)
	statement, err := service.PrepareContext(context.Background(), `INSERT INTO test_records (id, name) VALUES (?, ?)`)
	if err != nil {
		t.Fatalf("PrepareContext() error = %v", err)
	}
	defer statement.Close(context.Background())
	result, err := statement.Exec(context.Background(), 1, "one")
	if err != nil {
		t.Fatalf("Statement.Exec() error = %v", err)
	}
	if result.RowsAffected != 1 {
		t.Fatalf("RowsAffected = %d, want 1", result.RowsAffected)
	}

	rows, err := service.QueryRowsContext(context.Background(), `SELECT id, name FROM test_records WHERE id = ?`, 1)
	if err != nil {
		t.Fatalf("QueryRowsContext() error = %v", err)
	}
	defer rows.Close()
	columns, err := rows.Columns()
	if err != nil {
		t.Fatalf("Rows.Columns() error = %v", err)
	}
	if len(columns) != 2 || columns[0] != "id" || columns[1] != "name" {
		t.Fatalf("Rows.Columns() = %#v", columns)
	}
	if !rows.Next() {
		t.Fatal("Rows.Next() = false, want one row")
	}
	values, err := rows.Values()
	if err != nil {
		t.Fatalf("Rows.Values() error = %v", err)
	}
	if len(values) != 2 || values[1] != "one" {
		t.Fatalf("Rows.Values() = %#v", values)
	}
}

func TestExplicitTransactionUsesUnifiedInterface(t *testing.T) {
	service := newSQLiteTestService(t)
	createTestRecordsTable(t, service)

	transaction, err := service.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatalf("BeginTx() error = %v", err)
	}
	if _, err = transaction.ExecContext(context.Background(), `INSERT INTO test_records (id, name) VALUES (?, ?)`, 1, "one"); err != nil {
		t.Fatalf("Transaction.ExecContext() error = %v", err)
	}
	if err = transaction.Rollback(context.Background()); err != nil {
		t.Fatalf("Transaction.Rollback() error = %v", err)
	}
	if count := testRecordCount(t, service); count != 0 {
		t.Fatalf("record count after rollback = %d, want 0", count)
	}
}

func TestAcquiredConnectionKeepsConnectionLocalState(t *testing.T) {
	service := newSQLiteTestService(t)
	connection, err := service.AcquireConnection(context.Background())
	if err != nil {
		t.Fatalf("AcquireConnection() error = %v", err)
	}
	defer connection.Close(context.Background())
	if _, err = connection.ExecContext(context.Background(), `CREATE TEMP TABLE connection_records (id INTEGER PRIMARY KEY)`); err != nil {
		t.Fatalf("Connection.ExecContext() error = %v", err)
	}
	transaction, err := connection.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatalf("Connection.BeginTx() error = %v", err)
	}
	if _, err = transaction.ExecContext(context.Background(), `INSERT INTO connection_records (id) VALUES (?)`, 1); err != nil {
		t.Fatalf("Transaction.ExecContext() error = %v", err)
	}
	if err = transaction.Commit(context.Background()); err != nil {
		t.Fatalf("Transaction.Commit() error = %v", err)
	}
	var count int
	found, err := connection.QueryRowContext(context.Background(), &count, `SELECT COUNT(*) FROM connection_records`)
	if err != nil {
		t.Fatalf("Connection.QueryRowContext() error = %v", err)
	}
	if !found || count != 1 {
		t.Fatalf("connection count = %d, found = %v, want 1 and true", count, found)
	}
}

func TestDatabaseTypeUsesDefaultPostgreSQL(t *testing.T) {
	service := &serviceImpl{databaseType: (&Config{}).getDBType()}
	if service.DatabaseType() != POSTGRES {
		t.Fatalf("DatabaseType() = %q, want %q", service.DatabaseType(), POSTGRES)
	}
}

func TestOpenRejectsUnsupportedDatabaseType(t *testing.T) {
	service, err := Open(context.Background(), &Config{DbType: DbType("oracle")})
	if err == nil {
		t.Fatal("Open() error = nil")
	}
	if service != nil {
		t.Fatal("Open() returned a service for an unsupported database type")
	}
}

func TestSetValueRemovesAllNumericZeroTypes(t *testing.T) {
	values := []any{int(0), int8(0), int16(0), int32(0), int64(0), float32(0), float64(0)}
	for _, value := range values {
		params := make(map[string]any)
		SetValue(params, "value", value, true)
		if len(params) != 0 {
			t.Fatalf("SetValue() retained zero value %T", value)
		}
	}
}

func TestStopClosesDatabase(t *testing.T) {
	service := newSQLiteTestService(t)
	if err := service.Stop(context.Background()); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}
	if err := service.Ping(context.Background()); err == nil {
		t.Fatal("database remained open after Stop()")
	}
}

func TestConnectionPoolHonorsConfiguredMaximum(t *testing.T) {
	service := newSQLiteTestService(t)
	if maximum := service.Stats().MaxOpenConnections; maximum != 3 {
		t.Fatalf("MaxOpenConnections = %d, want 3", maximum)
	}
}

func TestDatabaseOperationsRecordMonitorsBeforeReturning(t *testing.T) {
	service := newSQLiteTestService(t)
	monitor := &recordingMonitor{}
	service.RegisterHandleMonitor(monitor)

	if _, err := service.ExecContext(context.Background(), `CREATE TABLE monitored_records (id INTEGER PRIMARY KEY)`); err != nil {
		t.Fatalf("ExecContext() error = %v", err)
	}
	if count := monitor.count.Load(); count != 1 {
		t.Fatalf("monitor record count = %d, want 1", count)
	}
}

func TestPostgresDataSourceNameEscapesCredentials(t *testing.T) {
	config := &Config{
		DBName:   "database name",
		User:     "user name",
		Password: "quote' and space",
		Host:     "127.0.0.1",
		Port:     5432,
		SSLMode:  PostgresSSLModeVerifyFull,
	}

	poolConfig, err := pgxpool.ParseConfig(buildDataSourceNameForPostgresSQL(config))
	if err != nil {
		t.Fatalf("ParseConfig() error = %v", err)
	}
	if poolConfig.ConnConfig.Database != config.DBName || poolConfig.ConnConfig.User != config.User || poolConfig.ConnConfig.Password != config.Password {
		t.Fatalf("parsed PostgreSQL credentials = %#v", poolConfig.ConnConfig)
	}
	if poolConfig.ConnConfig.TLSConfig == nil || poolConfig.ConnConfig.TLSConfig.ServerName != config.Host {
		t.Fatalf("parsed PostgreSQL TLS config = %#v", poolConfig.ConnConfig.TLSConfig)
	}
}

func TestPostgresDataSourceNameIncludesTLSCertificates(t *testing.T) {
	config := &Config{
		DBName:      "database",
		User:        "user",
		Host:        "database.internal",
		Port:        5432,
		SSLMode:     PostgresSSLModeVerifyFull,
		SSLRootCert: "/certificates/root ca.pem",
		SSLCert:     "/certificates/client.pem",
		SSLKey:      "/certificates/client key.pem",
	}
	parsed, err := url.Parse(buildDataSourceNameForPostgresSQL(config))
	if err != nil {
		t.Fatalf("url.Parse() error = %v", err)
	}
	query := parsed.Query()
	if query.Get("sslmode") != string(config.SSLMode) || query.Get("sslrootcert") != config.SSLRootCert || query.Get("sslcert") != config.SSLCert || query.Get("sslkey") != config.SSLKey {
		t.Fatalf("PostgreSQL TLS parameters = %#v", query)
	}
}

func TestPostgresDataSourceNameDoesNotDisableTLSByDefault(t *testing.T) {
	dataSourceName := buildDataSourceNameForPostgresSQL(&Config{
		DBName: "database",
		User:   "user",
		Host:   "127.0.0.1",
		Port:   5432,
	})
	if strings.Contains(dataSourceName, "sslmode=disable") {
		t.Fatalf("PostgreSQL data source disabled TLS: %s", dataSourceName)
	}
}

func TestMySQLDataSourceNameEscapesCredentials(t *testing.T) {
	config := &Config{
		DBName:   "database/name",
		User:     "username",
		Password: "password@tcp(elsewhere):/",
		Host:     "127.0.0.1",
		Port:     3306,
	}

	parsed, err := mysql.ParseDSN(buildDataSourceNameForMySQL(config))
	if err != nil {
		t.Fatalf("ParseDSN() error = %v", err)
	}
	if parsed.DBName != config.DBName || parsed.User != config.User || parsed.Passwd != config.Password {
		t.Fatalf("parsed MySQL credentials = %#v", parsed)
	}
	if parsed.Loc != time.Local || !parsed.ParseTime || !parsed.InterpolateParams {
		t.Fatalf("parsed MySQL options = %#v", parsed)
	}
}
