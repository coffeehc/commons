package dbsource

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	baseerrors "github.com/coffeehc/base/errors"
	"github.com/coffeehc/commons/dbsource/pgdialect"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// postgresFixture opens real connections and owns only a uniquely named test schema.
// DBSOURCE_TEST_POSTGRES must explicitly select a disposable database; no application config is read.
func postgresFixture(t *testing.T) (*serviceImpl, *pgxpool.Pool, string) {
	t.Helper()
	dsn := os.Getenv("DBSOURCE_TEST_POSTGRES")
	if dsn == "" {
		t.Skip("set DBSOURCE_TEST_POSTGRES to an isolated PostgreSQL test database")
	}
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	admin, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(admin.Close)
	schema := fmt.Sprintf("dbsource_cancel_%d", time.Now().UnixNano())
	quoted := pgx.Identifier{schema}.Sanitize()
	if _, err := admin.Exec(ctx, "CREATE SCHEMA "+quoted); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		cleanup, stop := context.WithTimeout(context.Background(), 10*time.Second)
		defer stop()
		if _, err := admin.Exec(cleanup, "DROP SCHEMA "+quoted+" CASCADE"); err != nil {
			t.Error(err)
		}
		var exists bool
		if err := admin.QueryRow(cleanup, "SELECT EXISTS(SELECT 1 FROM pg_namespace WHERE nspname=$1)", schema).Scan(&exists); err != nil || exists {
			t.Errorf("schema cleanup exists=%t err=%v", exists, err)
		}
	})
	d, err := pgdialect.New(ctx, &pgdialect.Config{DataSourceName: dsn, MaxOpenConns: 3, Mapper: DBMapperFunc})
	if err != nil {
		t.Fatal(err)
	}
	service := &serviceImpl{databaseType: POSTGRES, dialect: d}
	t.Cleanup(func() {
		cleanup, stop := context.WithTimeout(context.Background(), 10*time.Second)
		defer stop()
		if err := service.Close(cleanup); err != nil {
			t.Error(err)
		}
	})
	return service, admin, quoted
}

// waitPostgresQuery verifies server-side execution before cancellation, avoiding timer-only races.
func waitPostgresQuery(t *testing.T, admin *pgxpool.Pool, query string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		var active bool
		err := admin.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM pg_stat_activity WHERE query=$1 AND state='active' AND pid<>pg_backend_pid())", query).Scan(&active)
		if err != nil {
			t.Fatal(err)
		}
		if active {
			return
		}
		select {
		case <-ticker.C:
		case <-ctx.Done():
			t.Fatal("query never became active")
		}
	}
}

// TestPostgresInFlightCancellation exercises pgx's real cancellation wrapping at each SQL entry point.
func TestPostgresInFlightCancellation(t *testing.T) {
	service, admin, schema := postgresFixture(t)
	if _, err := admin.Exec(t.Context(), "CREATE TABLE "+schema+".pending(id text PRIMARY KEY)"); err != nil {
		t.Fatal(err)
	}
	for _, operation := range []string{"query", "query_struct", "row", "rows", "exec", "insert", "statement", "transaction", "handle_tx", "deadline"} {
		t.Run(operation, func(t *testing.T) {
			counts := captureDBLogs(t)
			ctx, cancel := context.WithCancel(t.Context())
			want := context.Canceled
			if operation == "deadline" {
				cancel()
				ctx, cancel = context.WithTimeout(t.Context(), 1500*time.Millisecond)
				want = context.DeadlineExceeded
			}
			defer cancel()
			query := fmt.Sprintf("SELECT 1 AS value FROM pg_sleep(30) /*dbsource_%s_%d*/", operation, time.Now().UnixNano())
			finished := make(chan error, 1)
			go func() {
				var values []int
				var err error
				switch operation {
				case "query", "deadline":
					err = service.QueryContext(ctx, &values, query)
				case "query_struct":
					var records []struct {
						Value int `db:"value"`
					}
					err = service.QueryContext(ctx, &records, query)
				case "row":
					var value int
					_, err = service.QueryRowContext(ctx, &value, query)
				case "rows":
					var rows Rows
					rows, err = service.QueryRowsContext(ctx, query)
					if rows != nil {
						for rows.Next() {
						}
						err = rows.Err()
						_ = rows.Close()
					}
				case "exec":
					_, err = service.ExecContext(ctx, query)
				case "insert":
					_, _, err = service.InsertContext(ctx, query)
				case "statement":
					var statement Statement
					statement, err = service.PrepareContext(ctx, query)
					if err == nil {
						_, err = statement.Exec(ctx)
						_ = statement.Close(context.Background())
					}
				case "transaction":
					var tx Transaction
					tx, err = service.BeginTx(ctx, nil)
					if err == nil {
						_, err = tx.ExecContext(ctx, "INSERT INTO "+schema+".pending VALUES (?)", operation)
						if err == nil {
							err = tx.QueryContext(ctx, &values, query)
						}
						_ = tx.Rollback(context.Background())
					}
				case "handle_tx":
					err = service.HandleTx(ctx, func(txctx context.Context) error {
						if _, err := service.ExecContext(txctx, "INSERT INTO "+schema+".pending VALUES (?)", operation); err != nil {
							return err
						}
						return service.QueryContext(txctx, &values, query)
					})
				}
				finished <- err
			}()
			waitPostgresQuery(t, admin, query)
			if operation != "deadline" {
				cancel()
			}
			select {
			case err := <-finished:
				if !errors.Is(err, want) {
					t.Errorf("error %T %v, want %v", err, err, want)
				}
				t.Logf("returned %T: %v; errors.Is(%v)=%t", err, err, want, errors.Is(err, want))
			case <-time.After(8 * time.Second):
				t.Fatal("operation failed to exit")
			}
			if operation == "transaction" || operation == "handle_tx" {
				var remaining int
				if err := admin.QueryRow(t.Context(), "SELECT count(*) FROM "+schema+".pending").Scan(&remaining); err != nil || remaining != 0 {
					t.Fatalf("canceled transaction persisted rows: count=%d err=%v", remaining, err)
				}
				// pgxpool destroys canceled connections asynchronously after Release.
				released, stop := context.WithTimeout(t.Context(), 5*time.Second)
				defer stop()
				ticker := time.NewTicker(10 * time.Millisecond)
				defer ticker.Stop()
				for service.Stats().InUse != 0 {
					select {
					case <-ticker.C:
					case <-released.Done():
						t.Fatal("canceled transaction leaked its pool lease")
					}
				}
			}
			logs := counts()
			if logs["warn"]+logs["error"]+logs["dpanic"]+logs["panic"]+logs["fatal"] != 0 {
				t.Fatalf("unexpected cancellation logs: %v", logs)
			}
		})
	}
}

// TestPostgresRealFailures keeps SQL, constraint, server timeout, and connection failure policy intact.
func TestPostgresRealFailures(t *testing.T) {
	service, admin, schema := postgresFixture(t)
	if _, err := admin.Exec(t.Context(), "CREATE TABLE "+schema+".records(id integer PRIMARY KEY); INSERT INTO "+schema+".records VALUES (1)"); err != nil {
		t.Fatal(err)
	}
	for _, scenario := range []struct{ name, query, code string }{
		{"invalid_sql", "SELEC 1", "42601"},
		{"constraint", "INSERT INTO " + schema + ".records VALUES (1)", "23505"},
		{"server_timeout", "SELECT pg_sleep(30)", "57014"},
		{"connection_failure", "SELECT 1", ""},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			counts := captureDBLogs(t)
			ctx, stop := context.WithTimeout(t.Context(), 10*time.Second)
			defer stop()
			connection, err := service.AcquireConnection(ctx)
			if err != nil {
				t.Fatal(err)
			}
			defer connection.Close(context.Background())
			if scenario.name == "server_timeout" {
				if _, err := connection.ExecContext(ctx, "SET statement_timeout='50ms'"); err != nil {
					t.Fatal(err)
				}
				defer connection.ExecContext(context.Background(), "SET statement_timeout=0")
			}
			if scenario.name == "connection_failure" {
				var pid int
				if _, err := connection.QueryRowContext(ctx, &pid, "SELECT pg_backend_pid()"); err != nil {
					t.Fatal(err)
				}
				if _, err := admin.Exec(ctx, "SELECT pg_terminate_backend($1)", pid); err != nil {
					t.Fatal(err)
				}
			}
			_, err = connection.ExecContext(ctx, scenario.query)
			converted, ok := err.(baseerrors.Error)
			if !ok {
				t.Fatalf("real error lost framework wrapper: %T %v", err, err)
			}
			if ctx.Err() != nil {
				t.Fatalf("request context unexpectedly done: %v", ctx.Err())
			}
			if scenario.code != "" {
				var pgErr *pgconn.PgError
				if !errors.As(converted.ToError(), &pgErr) || pgErr.Code != scenario.code {
					t.Fatalf("wrong driver error: %v", err)
				}
			}
			if got := counts(); got["dpanic"] != 1 {
				t.Fatalf("real failure logs: %v", got)
			}
		})
	}
}
