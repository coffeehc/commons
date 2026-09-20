package pgdialect

import (
	"context"
	"database/sql"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jmoiron/sqlx/reflectx"
)

type testRecord struct {
	ID int64 `db:"id"`
}

type stubRows struct {
	fields  []pgconn.FieldDescription
	values  [][]any
	index   int
	err     error
	closed  bool
	scanErr error
}

func newStubRows(values ...[]any) *stubRows {
	return &stubRows{
		fields: []pgconn.FieldDescription{{Name: "id"}},
		values: values,
		index:  -1,
	}
}

func (impl *stubRows) Close() {
	impl.closed = true
}

func (impl *stubRows) Err() error {
	return impl.err
}

func (impl *stubRows) CommandTag() pgconn.CommandTag {
	return pgconn.CommandTag{}
}

func (impl *stubRows) FieldDescriptions() []pgconn.FieldDescription {
	return impl.fields
}

func (impl *stubRows) Next() bool {
	if impl.closed || impl.index+1 >= len(impl.values) {
		impl.Close()
		return false
	}
	impl.index++
	return true
}

func (impl *stubRows) Scan(dest ...any) error {
	if impl.scanErr != nil {
		return impl.scanErr
	}
	for index, value := range impl.values[impl.index] {
		reflect.ValueOf(dest[index]).Elem().Set(reflect.ValueOf(value))
	}
	return nil
}

func (impl *stubRows) Values() ([]any, error) {
	if impl.index < 0 || impl.index >= len(impl.values) {
		return nil, errors.New("no current row")
	}
	return impl.values[impl.index], nil
}

func (impl *stubRows) RawValues() [][]byte {
	return nil
}

func (impl *stubRows) Conn() *pgx.Conn {
	return nil
}

type stubTx struct {
	rows        pgx.Rows
	queryErr    error
	commitErr   error
	rollbackErr error
	committed   bool
	rolledBack  bool
	execQuery   string
	rollbackCtx error
}

func (impl *stubTx) Begin(context.Context) (pgx.Tx, error) {
	return impl, nil
}

func (impl *stubTx) Commit(context.Context) error {
	impl.committed = true
	return impl.commitErr
}

func (impl *stubTx) Rollback(ctx context.Context) error {
	impl.rolledBack = true
	impl.rollbackCtx = ctx.Err()
	return impl.rollbackErr
}

func (*stubTx) CopyFrom(context.Context, pgx.Identifier, []string, pgx.CopyFromSource) (int64, error) {
	return 0, nil
}

func (*stubTx) SendBatch(context.Context, *pgx.Batch) pgx.BatchResults {
	return nil
}

func (*stubTx) LargeObjects() pgx.LargeObjects {
	return pgx.LargeObjects{}
}

func (*stubTx) Prepare(context.Context, string, string) (*pgconn.StatementDescription, error) {
	return nil, nil
}

func (impl *stubTx) Exec(_ context.Context, query string, _ ...any) (pgconn.CommandTag, error) {
	impl.execQuery = query
	return pgconn.CommandTag{}, nil
}

func (impl *stubTx) Query(context.Context, string, ...any) (pgx.Rows, error) {
	return impl.rows, impl.queryErr
}

func (*stubTx) QueryRow(context.Context, string, ...any) pgx.Row {
	return nil
}

func (*stubTx) Conn() *pgx.Conn {
	return nil
}

func TestGetClosesRows(t *testing.T) {
	rows := newStubRows([]any{int64(42)})
	transaction := &stubTx{rows: rows}
	ctx := context.WithValue(context.Background(), transactionContextKey{}, pgx.Tx(transaction))
	database := &serviceImpl{mapper: reflectx.NewMapperFunc("db", strings.ToLower)}
	record := testRecord{}

	found, err := database.Get(ctx, &record, "SELECT id")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if !found || record.ID != 42 {
		t.Fatalf("Get() found=%t record=%#v", found, record)
	}
	if !rows.closed {
		t.Fatal("Get() did not close rows")
	}
}

func TestExecRewritesCanonicalSQL(t *testing.T) {
	transaction := &stubTx{}
	ctx := context.WithValue(context.Background(), transactionContextKey{}, pgx.Tx(transaction))
	database := &serviceImpl{}

	if _, err := database.Exec(ctx, "insert into `group` (id) values (?)", 42); err != nil {
		t.Fatalf("Exec() error = %v", err)
	}
	if transaction.execQuery != `insert into "group" (id) values ($1)` {
		t.Fatalf("Exec() query = %q", transaction.execQuery)
	}
}

func TestSelectReturnsRowsError(t *testing.T) {
	wantErr := errors.New("read failed")
	rows := newStubRows()
	rows.err = wantErr
	transaction := &stubTx{rows: rows}
	ctx := context.WithValue(context.Background(), transactionContextKey{}, pgx.Tx(transaction))
	database := &serviceImpl{mapper: reflectx.NewMapperFunc("db", strings.ToLower)}
	records := make([]testRecord, 0)

	err := database.Select(ctx, &records, "SELECT id")
	if !errors.Is(err, wantErr) {
		t.Fatalf("Select() error = %v, want %v", err, wantErr)
	}
}

func TestSelectScansStructSlice(t *testing.T) {
	rows := newStubRows([]any{int64(42)}, []any{int64(43)})
	transaction := &stubTx{rows: rows}
	ctx := context.WithValue(context.Background(), transactionContextKey{}, pgx.Tx(transaction))
	database := &serviceImpl{mapper: reflectx.NewMapperFunc("db", strings.ToLower)}
	records := make([]*testRecord, 0)

	if err := database.Select(ctx, &records, "SELECT id"); err != nil {
		t.Fatalf("Select() error = %v", err)
	}
	if len(records) != 2 || records[0].ID != 42 || records[1].ID != 43 {
		t.Fatalf("Select() records = %#v", records)
	}
}

func TestExecuteTransactionReturnsCommitError(t *testing.T) {
	wantErr := errors.New("commit failed")
	transaction := &stubTx{commitErr: wantErr}

	err := executeTransaction(context.Background(), transaction, func(context.Context) error { return nil })
	if !errors.Is(err, wantErr) {
		t.Fatalf("executeTransaction() error = %v, want %v", err, wantErr)
	}
	if !transaction.committed || transaction.rolledBack {
		t.Fatalf("transaction state committed=%t rolledBack=%t", transaction.committed, transaction.rolledBack)
	}
}

func TestExecuteTransactionJoinsRollbackError(t *testing.T) {
	handleErr := errors.New("handler failed")
	rollbackErr := errors.New("rollback failed")
	transaction := &stubTx{rollbackErr: rollbackErr}

	err := executeTransaction(context.Background(), transaction, func(context.Context) error { return handleErr })
	if !errors.Is(err, handleErr) || !errors.Is(err, rollbackErr) {
		t.Fatalf("executeTransaction() error = %v, want joined handler and rollback errors", err)
	}
	if transaction.committed || !transaction.rolledBack {
		t.Fatalf("transaction state committed=%t rolledBack=%t", transaction.committed, transaction.rolledBack)
	}
}

func TestExecuteTransactionRollsBackAfterRequestCancellation(t *testing.T) {
	transaction := &stubTx{}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := executeTransaction(ctx, transaction, func(context.Context) error { return context.Canceled })
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("executeTransaction() error = %v, want %v", err, context.Canceled)
	}
	if !transaction.rolledBack || transaction.rollbackCtx != nil {
		t.Fatalf("rollback state rolledBack=%t contextError=%v", transaction.rolledBack, transaction.rollbackCtx)
	}
}

func TestExecuteTransactionRollsBackOnPanic(t *testing.T) {
	transaction := &stubTx{}
	defer func() {
		if recover() == nil {
			t.Fatal("executeTransaction() did not propagate panic")
		}
		if transaction.committed || !transaction.rolledBack {
			t.Fatalf("transaction state committed=%t rolledBack=%t", transaction.committed, transaction.rolledBack)
		}
	}()

	_ = executeTransaction(context.Background(), transaction, func(context.Context) error {
		panic("boom")
	})
}

func TestBuildPoolConfigHonorsLimits(t *testing.T) {
	config := &Config{
		DataSourceName:  "postgres://user:password@127.0.0.1:5432/database?sslmode=disable",
		MaxOpenConns:    3,
		ConnMaxLifetime: 30 * time.Second,
		Mapper:          reflectx.NewMapper("db"),
	}

	poolConfig, err := buildPoolConfig(config)
	if err != nil {
		t.Fatalf("buildPoolConfig() error = %v", err)
	}
	if poolConfig.MaxConns != 3 || poolConfig.MaxConnLifetime != 30*time.Second {
		t.Fatalf("pool limits MaxConns=%d MaxConnLifetime=%s", poolConfig.MaxConns, poolConfig.MaxConnLifetime)
	}
}

func TestRewriteSQLHandlesPlaceholdersAndQuotedText(t *testing.T) {
	query := "select ?, '?', E'can\\'t ?', \"?\", `group`, data ?? 'key', data ?| array['a'] /* ? */ -- ?\nwhere id=? and body=$tag$?$tag$"
	want := "select $1, '?', E'can\\'t ?', \"?\", \"group\", data ? 'key', data ?| array['a'] /* ? */ -- ?\nwhere id=$2 and body=$tag$?$tag$"
	if got := rewriteSQL(query); got != want {
		t.Fatalf("rewriteSQL() = %q, want %q", got, want)
	}
}

func TestConvertTxOptions(t *testing.T) {
	options, err := convertTxOptions(&sql.TxOptions{Isolation: sql.LevelSerializable, ReadOnly: true})
	if err != nil {
		t.Fatalf("convertTxOptions() error = %v", err)
	}
	if options.IsoLevel != pgx.Serializable || options.AccessMode != pgx.ReadOnly {
		t.Fatalf("convertTxOptions() = %#v", options)
	}
}

// TestSelectScalarReturnsRowsError preserves a terminal driver error when no columns are available.
func TestSelectScalarReturnsRowsError(t *testing.T) {
	for _, cause := range []error{context.Canceled, context.DeadlineExceeded, &pgconn.PgError{Code: "42601", Message: "syntax error"}} {
		transaction := &stubTx{rows: &stubRows{err: cause}}
		ctx := context.WithValue(t.Context(), transactionContextKey{}, pgx.Tx(transaction))
		database := &serviceImpl{mapper: reflectx.NewMapper("db")}
		var values []int
		if err := database.Select(ctx, &values, "SELECT 1"); !errors.Is(err, cause) {
			t.Fatalf("Select() = %v; want terminal error %v", err, cause)
		}
	}
}

// TestExecuteTransactionKeepsRollbackFailureWithCancellation protects independent cleanup errors.
func TestExecuteTransactionKeepsRollbackFailureWithCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	rollbackErr := errors.New("independent rollback failure")
	transaction := &stubTx{rollbackErr: rollbackErr}
	err := executeTransaction(ctx, transaction, func(context.Context) error { return context.Canceled })
	if !errors.Is(err, context.Canceled) || !errors.Is(err, rollbackErr) {
		t.Fatalf("lost operation or cleanup error: %v", err)
	}
}
