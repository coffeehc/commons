package logservice

import (
	"context"
	"time"
)

// Builder is obtained from logservice.New using built-in PostgreSQL persistence.
// Registration is frozen at Start; no remote plugin loading is implied.
type Builder interface {
	RegisterNormalizer(name string, normalizer Normalizer) error
	RegisterClassifier(name string, classifier Classifier) error
	RegisterRedactor(name string, redactor Redactor) error
	RegisterHandler(HandlerRegistration, EventHandler) error
	// Prepare freezes registrations and validates/prepares owned schema without starting workers.
	// It is idempotent, and the same builder may subsequently Start exactly once.
	Prepare(context.Context) error
	Start(context.Context) (Service, error)
}
type Config struct {
	Scope                   Scope
	RegistryVersion         uint64 // Default 1; change registration only through the next revision.
	PreviousRegistryVersion uint64 // Explicit compare-and-swap upgrade from the stored revision.
	GroupingVersion         string
	RecordTimeout           time.Duration // Default 2s; bounded by caller deadline, includes persistence.
	SynchronousHookBudget   time.Duration // Cooperative budget, default 10ms.
	DetailRetention         time.Duration // Default seven days; immutable metadata/dedup remain retained.
	MaxCausationDepth       int           // Default 8.
	MaxPendingDeliveries    int           // Default 10000; explicit ErrBackpressure, never silent loss.
	PollInterval            time.Duration // Default 250ms.
	AllowedAttributes       []string      // Deny-by-default; sensitive key names are always removed.
	Disabled                bool          // Static host switch; Record returns disabled without persistence.
}

// Synchronous extensions are deterministic, bounded and side-effect-free.
// Scope, record identity, timestamps and causal lineage are outside their API.
type Normalizer interface {
	Normalize(context.Context, Facts) (Facts, error)
}
type Classification struct {
	Nature                 Nature
	Category               string
	Severity               Severity
	GroupKey               string
	Retryable, UserFixable *bool
	Suppress               bool
}
type Classifier interface {
	// First non-nil match wins; fills unspecified explicit fields by default.
	Classify(context.Context, Facts) (*Classification, error)
}
type Redactor interface {
	Redact(context.Context, Facts) (Facts, error)
}

type EventKind string

const (
	EventErrorRecorded EventKind = "error.recorded"
	EventGroupCreated  EventKind = "group.created"
)

type Causation struct {
	RootEventID, ParentEventID EventID
	Depth                      int
	HandlerPath                []HandlerID
}

// Event is an immutable post-commit snapshot. Go maps/slices require isolated copies.
// Record/group events carry RecordID, GroupID and redacted summary Facts.
type Event struct {
	ID            EventID
	SchemaVersion uint32
	Scope         Scope
	Kind          EventKind
	CommittedAt   time.Time
	RecordID      RecordID
	GroupID       GroupID
	Facts         *Facts // No Detail/Stack.
	Causation     Causation
}
type EventFilter struct {
	Kinds             []EventKind // Nonempty unless AllKinds; mutually exclusive.
	AllKinds          bool        // Explicit opt-in to every kind, never the zero-value default.
	Sources           []string
	Natures           []Nature
	Categories, Codes []string
	Severities        []Severity
	Retryable         *bool // nil: no predicate; false/true exclude unknown values.
}
type DeliveryPolicy struct {
	Timeout                            time.Duration
	MaxAttempts                        uint32 // Includes the initial attempt.
	InitialBackoff, MaxBackoff, MaxAge time.Duration
	Concurrency                        uint32
}
type HandlerRegistration struct {
	ID       HandlerID
	Filter   EventFilter
	Delivery DeliveryPolicy
}
type Delivery struct {
	ID             DeliveryID
	IdempotencyKey string // Stable scope+handler+event key across retries and restarts.
	Attempt        uint32 // Total attempts, never reset.
	Event          Event
}
type EventHandler interface {
	// nil acknowledges consumption or an intentional no-op, not recovery.
	// At-least-once delivery; handlers still check current business conditions.
	// Wrap ErrPermanentHandler to stop retrying; other errors are transient by default.
	// Pass ctx to derived Record calls to propagate causation.
	Handle(context.Context, Delivery) error
}
type DeliveryState string

const (
	DeliveryPending   DeliveryState = "pending"
	DeliveryRunning   DeliveryState = "running"
	DeliverySucceeded DeliveryState = "succeeded"
	DeliveryFailed    DeliveryState = "failed"
	DeliveryBlocked   DeliveryState = "blocked"
)

type DeliveryRecord struct {
	ID              DeliveryID
	EventID         EventID
	HandlerID       HandlerID
	State           DeliveryState
	Attempts        uint32
	NextAttemptAt   time.Time
	LastErrorCode   string // Safe machine code; never re-ingested.
	LastErrorDetail string // Always empty in v1; raw handler errors are not persisted.
	Version         uint64
}
type DeliveryQuery struct {
	HandlerID HandlerID
	EventID   EventID
	States    []DeliveryState
	Page      PageRequest
}
type DeliveryPage struct {
	Items []DeliveryRecord
	Page  PageInfo
}

// HandlerFunc adapts a function to EventHandler.
type HandlerFunc func(context.Context, Delivery) error

func (f HandlerFunc) Handle(ctx context.Context, d Delivery) error { return f(ctx, d) }

type ClassifierFunc func(context.Context, Facts) (*Classification, error)

func (f ClassifierFunc) Classify(ctx context.Context, v Facts) (*Classification, error) {
	return f(ctx, v)
}

type NormalizerFunc func(context.Context, Facts) (Facts, error)

func (f NormalizerFunc) Normalize(ctx context.Context, v Facts) (Facts, error) { return f(ctx, v) }

type RedactorFunc func(context.Context, Facts) (Facts, error)

func (f RedactorFunc) Redact(ctx context.Context, v Facts) (Facts, error) { return f(ctx, v) }
