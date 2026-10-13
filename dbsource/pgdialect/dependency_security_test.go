package pgdialect

import (
	"os"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jmoiron/sqlx/reflectx"
)

func TestParsePoolConfigQueryModeCompatibility(t *testing.T) {
	const dsn = "host=localhost user=test dbname=test sslmode=disable"
	for _, test := range []struct {
		name, suffix string
		want         pgx.QueryExecMode
	}{
		{"default", "", pgx.QueryExecModeCacheStatement},
		{"explicit simple", " default_query_exec_mode=simple_protocol", pgx.QueryExecModeSimpleProtocol},
	} {
		t.Run(test.name, func(t *testing.T) {
			config, err := ParsePoolConfig(dsn + test.suffix)
			if err != nil {
				t.Fatal(err)
			}
			if config.ConnConfig.DefaultQueryExecMode != test.want {
				t.Fatalf("query mode=%v, want %v", config.ConnConfig.DefaultQueryExecMode, test.want)
			}
		})
	}
}

// GO-2026-5004: dollar-quoted text must never cause parameter substitution
// inside a literal, including when callers explicitly choose simple protocol.
// This test is read-only and uses only the dedicated, explicitly supplied DSN.
func TestPostgresDollarQuotedParameterSafety(t *testing.T) {
	dsn := os.Getenv("DBSOURCE_TEST_POSTGRES")
	if dsn == "" {
		t.Skip("set DBSOURCE_TEST_POSTGRES to an isolated PostgreSQL test database")
	}
	for _, mode := range []pgx.QueryExecMode{pgx.QueryExecModeCacheStatement, pgx.QueryExecModeSimpleProtocol} {
		t.Run(mode.String(), func(t *testing.T) {
			config, err := ParsePoolConfig(dsn)
			if err != nil {
				t.Fatal(err)
			}
			config.ConnConfig.DefaultQueryExecMode = mode
			config.MaxConns = 1
			pool, err := pgxpool.NewWithConfig(t.Context(), config)
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(pool.Close)
			database := &serviceImpl{pool: pool, mapper: reflectx.NewMapperFunc("db", strings.ToLower)}
			for _, literal := range []string{"$tag$ $1 $tag$", "$$ $1 $$"} {
				for _, input := range []string{"plain", "quote ' and backslash \\", "$tag$, 'escaped'; SELECT 'injected' --", "$$, 'escaped'; SELECT 'injected' --"} {
					var got struct {
						Quoted string `db:"quoted"`
						Value  string `db:"value"`
					}
					found, err := database.Get(t.Context(), &got, "SELECT "+literal+" AS quoted, ?::text AS value", input)
					if err != nil || !found || got.Quoted != " $1 " || got.Value != input {
						t.Fatalf("literal=%q found=%t quoted=%q value=%q err=%v", literal, found, got.Quoted, got.Value, err)
					}
				}
			}
		})
	}
}
