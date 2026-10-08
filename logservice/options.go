package logservice

import (
	"context"
	"database/sql"
	"time"

	"github.com/coffeehc/commons/dbsource"
)

type Options struct {
	DataSource DataSource // Existing PostgreSQL data source; caller owns its lifecycle.
	Config     Config
	Schema     SchemaOptions
}

// DataSource is the existing commons/dbsource.Service method subset needed by the
// built-in PostgreSQL implementation. A dbsource.Service can be passed directly.
// This is database access, not an obligation to implement logservice persistence.
// Explicit transactions ensure Stored is not returned before an ambient host transaction commits.
type DataSource interface {
	Ping(context.Context) error
	BeginTx(context.Context, *sql.TxOptions) (dbsource.Transaction, error)
}

type MigrationMode string

const (
	MigrationAutoSafe   MigrationMode = "auto_safe"   // Default: owned, compatible migrations only.
	MigrationVerifyOnly MigrationMode = "verify_only" // Fail if schema upgrade is required.
)

type SchemaOptions struct {
	Namespace        string // Default "logservice"; validated SQL identifier, never raw SQL.
	MigrationMode    MigrationMode
	LockTimeout      time.Duration
	MigrationTimeout time.Duration
}
