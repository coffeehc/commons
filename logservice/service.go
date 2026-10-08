// Package logservice records structured errors and delivers durable post-commit events.
package logservice

import (
	"context"
	"errors"
	"time"
)

type RecordID string
type GroupID string
type EventID string
type HandlerID string
type DeliveryID string

// Scope is bound by trusted application composition, never a public query parameter.
type Scope struct{ Application, Environment, Tenant string }
type Ref struct{ Kind, ID string }

type Severity string

const (
	SeverityInfo     Severity = "info"
	SeverityWarning  Severity = "warning"
	SeverityError    Severity = "error"
	SeverityCritical Severity = "critical"
)

type Nature string

const (
	NatureSystem             Nature = "system"
	NatureBusiness           Nature = "business"
	NatureUserInput          Nature = "user_input"
	NatureExternalDependency Nature = "external_dependency"
	NatureDataConsistency    Nature = "data_consistency"
	NatureSecurity           Nature = "security"
	NatureResource           Nature = "resource"
	NatureUnknown            Nature = "unknown"
)

type Facts struct {
	Source                     string // Where the failure was observed.
	Nature                     Nature // General cause dimension.
	Category                   string // Application-defined category, independent of Nature.
	Severity                   Severity
	Component, Operation, Code string
	Summary, Detail, Stack     string
	Retryable, UserFixable     *bool // nil is unknown; independent of Severity.
	RequestID, TraceID         string
	Subject                    Ref
	Related                    []Ref
	Attributes                 map[string]string // Allowlisted, bounded and redacted.
}

// RecordInput represents one observed failure, not an unresolved business issue.
type RecordInput struct {
	ID         RecordID // Stable for retries of this submission; new observation => new ID.
	OccurredAt time.Time
	Facts      Facts
	GroupKey   string // Optional stable error-template key, without sensitive values.
}

type RecordDisposition string

const (
	RecordStored     RecordDisposition = "stored"
	RecordDuplicate  RecordDisposition = "duplicate"
	RecordSuppressed RecordDisposition = "suppressed"
	RecordDisabled   RecordDisposition = "disabled"
)

type RecordReceipt struct {
	Disposition RecordDisposition
	RecordID    RecordID
	GroupID     GroupID
	CommittedAt time.Time
	ReasonCode  string
}
type ErrorRecord struct {
	ID                     RecordID
	GroupID                GroupID
	OccurredAt, RecordedAt time.Time
	Facts                  Facts // Immutable observed facts; detailed payload can expire by policy.
	DetailsRetained        bool
}
type ErrorGroup struct {
	ID                      GroupID
	KeyVersion              string
	FirstSeenAt, LastSeenAt time.Time
	EntryCount              uint64 // Deduplicated observed failures, not incident or retry-attempt counts.
	Representative          Facts  // Summary/routing metadata only; no Detail or Stack.
}
type Recorder interface {
	// Stored means record, dedup, group counters and outbox committed atomically.
	Record(context.Context, RecordInput) (RecordReceipt, error)
}

type RecordFilter struct {
	Since, Until    time.Time // Occurred-at interval [Since, Until); zero is unbounded.
	Sources         []string
	Natures         []Nature
	Categories      []string
	Severities      []Severity
	Component, Code string
	Subject         *Ref
	Text            string // Searches allowlisted redacted fields only.
}
type PageRequest struct {
	Size   int
	Cursor string
}
type PageInfo struct {
	NextCursor string
	AsOf       time.Time
}
type QueryRequest struct {
	Filter RecordFilter
	Page   PageRequest
}
type GroupRow struct {
	Group             ErrorGroup
	MatchedEntryCount uint64
}
type QueryResult struct {
	Items                         []GroupRow
	MatchedGroups, MatchedEntries uint64 // Same filter and per-request snapshot.
	Page                          PageInfo
}
type RecordPage struct {
	Items []ErrorRecord
	Page  PageInfo
}
type DetailRequest struct {
	GroupID GroupID
	Filter  RecordFilter
	Page    PageRequest
}
type DetailResult struct {
	Group   ErrorGroup
	Records []ErrorRecord
	Page    PageInfo
}
type Reader interface {
	Query(context.Context, QueryRequest) (QueryResult, error)
	QueryRecords(context.Context, QueryRequest) (RecordPage, error)
	Detail(context.Context, DetailRequest) (DetailResult, error)
	GetRecord(context.Context, RecordID) (ErrorRecord, error)
}

// Service is scope-bound by trusted host composition. Close never closes the host data source.
type Service interface {
	Recorder
	Reader
	Scope() Scope
	QueryDeliveries(context.Context, DeliveryQuery) (DeliveryPage, error)
	// PruneDetails physically clears expired details. Reads hide them immediately at expiry.
	PruneDetails(context.Context) (int64, error)
	Close(context.Context) error
}

var (
	ErrInvalid            = errors.New("logservice: invalid input")
	ErrNotFound           = errors.New("logservice: not found in scope")
	ErrConflict           = errors.New("logservice: version or idempotency conflict")
	ErrUnavailable        = errors.New("logservice: persistence unavailable")
	ErrHookFailed         = errors.New("logservice: synchronous extension failed")
	ErrRegistrationFrozen = errors.New("logservice: registration is frozen")
	ErrDuplicateHandler   = errors.New("logservice: duplicate handler ID")
	ErrCausationLimit     = errors.New("logservice: causation limit exceeded")
	ErrClosed             = errors.New("logservice: closed")
	ErrBackpressure       = errors.New("logservice: delivery backlog limit reached")
	ErrPermanentHandler   = errors.New("logservice: permanent handler failure")
)
