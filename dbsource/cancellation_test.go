package dbsource

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	baseerrors "github.com/coffeehc/base/errors"
	"github.com/coffeehc/base/log"
	"github.com/coffeehc/commons/dbsource/dialect"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/spf13/viper"
)

// captureDBLogs observes the existing synchronous log subscription without changing
// logger panic behavior or racing the rotating file sink. Tests using it stay serial.
func captureDBLogs(t *testing.T) func() map[string]int {
	t.Helper()
	var previous log.Config
	if err := viper.UnmarshalKey("logger", &previous); err != nil {
		t.Fatal(err)
	}
	viper.Set("logger", log.Config{Level: "debug"})
	log.LoadConfig()
	entries := make(chan []byte, 128)
	id := log.RegisterAccept(entries)
	t.Cleanup(func() { log.UnRegisterAccept(id); viper.Set("logger", previous); log.LoadConfig() })
	// A marker proves the observer is attached; zero errors must not mean a disabled logger.
	log.Debug("数据库测试日志观察器已连接")
	counts := make(map[string]int)
	return func() map[string]int {
		t.Helper()
		for {
			select {
			case entry := <-entries:
				fields := strings.SplitN(string(entry), "\t", 3)
				if len(fields) < 3 {
					t.Fatalf("invalid structured console entry: %q", entry)
				}
				counts[strings.ToLower(fields[1])]++
			default:
				if counts["debug"] == 0 {
					t.Fatal("log observer was not attached")
				}
				return counts
			}
		}
	}
}

// failingDialect returns one deterministic driver error, including races with context cancellation.
type failingDialect struct {
	dialect.Dialect       // Unused methods fail loudly if a test unexpectedly calls them.
	failure         error // Exact backend error returned by every operation under test.
}

func (*failingDialect) Rewrite(query string) string { return query }
func (d *failingDialect) Exec(context.Context, string, ...any) (dialect.Result, error) {
	return dialect.Result{}, d.failure
}
func (d *failingDialect) Select(context.Context, any, string, ...any) error { return d.failure }
func (d *failingDialect) Get(context.Context, any, string, ...any) (bool, error) {
	return false, d.failure
}
func (d *failingDialect) Query(context.Context, string, ...any) (dialect.Rows, error) {
	return nil, d.failure
}
func (d *failingDialect) Prepare(context.Context, string) (dialect.Statement, error) {
	return nil, d.failure
}
func (d *failingDialect) HandleTx(context.Context, *sql.TxOptions, func(context.Context) error) error {
	return d.failure
}

// failingStatement exercises the separate prepared execution error branch.
type failingStatement struct {
	// failure is the exact driver error.
	failure error
}

func (s *failingStatement) Exec(context.Context, ...any) (dialect.Result, error) {
	return dialect.Result{}, s.failure
}
func (*failingStatement) Close(context.Context) error { return nil }

// TestOperationCancellationPolicy locks down all logging entry points and mixed-error boundaries.
func TestOperationCancellationPolicy(t *testing.T) {
	canceled, cancel := context.WithCancel(t.Context())
	cancel()
	expired, stop := context.WithDeadline(t.Context(), time.Now().Add(-time.Second))
	defer stop()
	syntax := &pgconn.PgError{Code: "42601", Message: "syntax error"}
	for _, scenario := range []struct {
		name     string
		ctx      context.Context
		err      error
		expected bool
	}{
		{"cancel", canceled, context.Canceled, true},
		{"wrapped_cancel", canceled, fmt.Errorf("pgx: %w", context.Canceled), true},
		{"deadline", expired, fmt.Errorf("timeout: %w", context.DeadlineExceeded), true},
		{"live_context", t.Context(), context.Canceled, false},
		{"deadline_without_expiry", canceled, context.DeadlineExceeded, false},
		{"syntax_after_cancel", canceled, syntax, false},
		{"constraint_after_cancel", canceled, &pgconn.PgError{Code: "23505", Message: "duplicate key"}, false},
		{"connection_after_cancel", canceled, sql.ErrConnDone, false},
		{"server_cancel", canceled, &pgconn.PgError{Code: "57014", Message: "query canceled"}, false},
		{"database_timeout", t.Context(), &pgconn.PgError{Code: "57014", Message: "statement timeout"}, false},
		{"mixed_errors", canceled, errors.Join(context.Canceled, syntax), false},
		{"wrapped_mixed_errors", canceled, fmt.Errorf("tx: %w", errors.Join(context.Canceled, syntax)), false},
		{"joined_cancellations", canceled, errors.Join(context.Canceled, fmt.Errorf("query: %w", context.Canceled)), true},
	} {
		for _, operation := range []string{"insert", "exec", "query", "row", "rows", "prepare", "statement", "transaction"} {
			t.Run(scenario.name+"/"+operation, func(t *testing.T) {
				counts := captureDBLogs(t)
				d := &failingDialect{failure: scenario.err}
				service := &serviceImpl{dialect: d}
				var err error
				var dest []int
				switch operation {
				case "insert":
					_, _, err = service.InsertContext(scenario.ctx, "test")
				case "exec":
					_, err = service.ExecContext(scenario.ctx, "test")
				case "query":
					err = service.QueryContext(scenario.ctx, &dest, "test")
				case "row":
					_, err = service.QueryRowContext(scenario.ctx, &dest, "test")
				case "rows":
					_, err = service.QueryRowsContext(scenario.ctx, "test")
				case "prepare":
					_, err = service.PrepareContext(scenario.ctx, "test")
				case "statement":
					statement := &statementImpl{service: service, statement: &failingStatement{failure: scenario.err}}
					_, err = statement.Exec(scenario.ctx)
				case "transaction":
					err = service.HandleTx(scenario.ctx, nil)
				}
				if err == nil {
					t.Fatal("operation swallowed error")
				}
				got := counts()
				if scenario.expected {
					if !errors.Is(err, scenario.ctx.Err()) {
						t.Fatalf("lost cancellation identity: %T %v", err, err)
					}
					if got["dpanic"]+got["panic"]+got["fatal"]+got["error"]+got["warn"] != 0 {
						t.Fatalf("cancellation logs: %v", got)
					}
				} else {
					level := "dpanic"
					if operation == "transaction" {
						level = "error"
						if err != scenario.err {
							t.Fatal("transaction changed original error")
						}
					} else {
						converted, ok := err.(baseerrors.Error)
						if !ok || converted.ToError() != scenario.err {
							t.Fatalf("changed real error policy: %T %v", err, err)
						}
					}
					if got[level] != 1 {
						t.Fatalf("real error logs: %v, want one %s", got, level)
					}
				}
			})
		}
	}
}
