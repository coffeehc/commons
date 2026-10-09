package logservice

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/coffeehc/commons/dbsource"
)

type registeredHandler struct {
	Registration HandlerRegistration
	handler      EventHandler
}
type namedNormalizer struct {
	name string
	hook Normalizer
}
type namedClassifier struct {
	name string
	hook Classifier
}
type namedRedactor struct {
	name string
	hook Redactor
}
type builder struct {
	mu          sync.Mutex
	options     Options
	frozen      bool
	started     bool
	prepared    *serviceImpl
	normalizers []namedNormalizer
	classifiers []namedClassifier
	redactors   []namedRedactor
	handlers    map[HandlerID]registeredHandler
}
type serviceImpl struct {
	options     Options
	scopeKey    string
	table       string
	normalizers []namedNormalizer
	classifiers []namedClassifier
	redactors   []namedRedactor
	handlers    map[HandlerID]registeredHandler
	allowed     map[string]bool
	ctx         context.Context
	cancel      context.CancelFunc
	mu          sync.Mutex
	closed      bool
	work        sync.WaitGroup
	done        chan struct{}
}

var _ Builder = (*builder)(nil)
var _ Service = (*serviceImpl)(nil)
var opaqueID = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.:/-]{0,255}$`)
var identifier = regexp.MustCompile(`^[a-z][a-z0-9_]{0,47}$`)

// New constructs logservice with its built-in PostgreSQL persistence. It performs
// no I/O; Start verifies/migrates the schema before accepting records.
func New(options Options) (Builder, error) {
	c := &options.Config
	if nilInterface(options.DataSource) {
		return nil, fmt.Errorf("%w: DataSource is required", ErrInvalid)
	}
	for _, v := range []string{c.Scope.Application, c.Scope.Environment, c.Scope.Tenant} {
		if strings.TrimSpace(v) == "" || len(v) > 128 || redactString(v, len(v)) != v {
			return nil, fmt.Errorf("%w: all scope fields must be 1..128 bytes", ErrInvalid)
		}
	}
	if options.Schema.Namespace == "" {
		options.Schema.Namespace = "logservice"
	}
	if !identifier.MatchString(options.Schema.Namespace) {
		return nil, fmt.Errorf("%w: invalid namespace", ErrInvalid)
	}
	if options.Schema.MigrationMode == "" {
		options.Schema.MigrationMode = MigrationAutoSafe
	}
	if options.Schema.MigrationMode != MigrationAutoSafe && options.Schema.MigrationMode != MigrationVerifyOnly {
		return nil, fmt.Errorf("%w: invalid migration mode", ErrInvalid)
	}
	if options.Schema.LockTimeout == 0 {
		options.Schema.LockTimeout = 5 * time.Second
	}
	if options.Schema.MigrationTimeout == 0 {
		options.Schema.MigrationTimeout = 30 * time.Second
	}
	if c.RegistryVersion == 0 {
		c.RegistryVersion = 1
	}
	if c.RegistryVersion > 1<<63-1 || c.PreviousRegistryVersion > 1<<63-1 {
		return nil, ErrInvalid
	}
	if c.GroupingVersion == "" {
		c.GroupingVersion = "v1"
	}
	if len(c.GroupingVersion) > 64 {
		return nil, fmt.Errorf("%w: grouping version too long", ErrInvalid)
	}
	if c.RecordTimeout == 0 {
		c.RecordTimeout = 2 * time.Second
	}
	if c.SynchronousHookBudget == 0 {
		c.SynchronousHookBudget = 10 * time.Millisecond
	}
	if c.DetailRetention == 0 {
		c.DetailRetention = 7 * 24 * time.Hour
	}
	if c.MaxCausationDepth == 0 {
		c.MaxCausationDepth = 8
	}
	if c.MaxPendingDeliveries == 0 {
		c.MaxPendingDeliveries = 10000
	}
	if c.PollInterval == 0 {
		c.PollInterval = 250 * time.Millisecond
	}
	if c.RecordTimeout < 0 || c.SynchronousHookBudget < 0 || c.DetailRetention < 0 || c.PollInterval < time.Millisecond || c.MaxCausationDepth < 1 || c.MaxCausationDepth > 64 || c.MaxPendingDeliveries < 1 || options.Schema.LockTimeout < time.Millisecond || options.Schema.MigrationTimeout < time.Millisecond {
		return nil, fmt.Errorf("%w: invalid limits", ErrInvalid)
	}
	if len(c.AllowedAttributes) > 128 {
		return nil, ErrInvalid
	}
	for _, key := range c.AllowedAttributes {
		if key == "" || len(key) > 64 {
			return nil, ErrInvalid
		}
	}
	c.AllowedAttributes = append([]string(nil), c.AllowedAttributes...)
	return &builder{options: options, handlers: map[HandlerID]registeredHandler{}}, nil
}
func nilInterface(v any) bool {
	if v == nil {
		return true
	}
	r := reflect.ValueOf(v)
	switch r.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return r.IsNil()
	}
	return false
}
func (b *builder) RegisterNormalizer(name string, hook Normalizer) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.frozen {
		return ErrRegistrationFrozen
	}
	if name == "" || nilInterface(hook) {
		return ErrInvalid
	}
	for _, h := range b.normalizers {
		if h.name == name {
			return ErrConflict
		}
	}
	b.normalizers = append(b.normalizers, namedNormalizer{name, hook})
	return nil
}
func (b *builder) RegisterClassifier(name string, hook Classifier) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.frozen {
		return ErrRegistrationFrozen
	}
	if name == "" || nilInterface(hook) {
		return ErrInvalid
	}
	for _, h := range b.classifiers {
		if h.name == name {
			return ErrConflict
		}
	}
	b.classifiers = append(b.classifiers, namedClassifier{name, hook})
	return nil
}
func (b *builder) RegisterRedactor(name string, hook Redactor) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.frozen {
		return ErrRegistrationFrozen
	}
	if name == "" || nilInterface(hook) {
		return ErrInvalid
	}
	for _, h := range b.redactors {
		if h.name == name {
			return ErrConflict
		}
	}
	b.redactors = append(b.redactors, namedRedactor{name, hook})
	return nil
}
func (b *builder) RegisterHandler(reg HandlerRegistration, hook EventHandler) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.frozen {
		return ErrRegistrationFrozen
	}
	if !safeID(string(reg.ID)) || len(reg.ID) > 128 || nilInterface(hook) {
		return ErrInvalid
	}
	if _, ok := b.handlers[reg.ID]; ok {
		return ErrDuplicateHandler
	}
	if reg.Filter.AllKinds == (len(reg.Filter.Kinds) > 0) {
		return fmt.Errorf("%w: select Kinds or AllKinds", ErrInvalid)
	}
	for _, k := range reg.Filter.Kinds {
		if k != EventErrorRecorded && k != EventGroupCreated {
			return ErrInvalid
		}
	}
	p := &reg.Delivery
	if p.Timeout == 0 {
		p.Timeout = 30 * time.Second
	}
	if p.MaxAttempts == 0 {
		p.MaxAttempts = 5
	}
	if p.InitialBackoff == 0 {
		p.InitialBackoff = time.Second
	}
	if p.MaxBackoff == 0 {
		p.MaxBackoff = time.Minute
	}
	if p.MaxAge == 0 {
		p.MaxAge = 24 * time.Hour
	}
	if p.Concurrency == 0 {
		p.Concurrency = 1
	}
	if p.Timeout < time.Millisecond || p.Timeout > time.Hour || p.MaxBackoff > 24*time.Hour || p.MaxAge > 365*24*time.Hour || p.InitialBackoff < time.Millisecond || p.MaxBackoff < p.InitialBackoff || p.MaxAge < time.Millisecond || p.Concurrency > 32 || p.MaxAttempts > 100 {
		return ErrInvalid
	}
	// Registration slices are copied so callers cannot change routing after Start.
	data, _ := json.Marshal(reg)
	var copied HandlerRegistration
	_ = json.Unmarshal(data, &copied)
	reg = copied
	b.handlers[reg.ID] = registeredHandler{reg, hook}
	return nil
}

// Prepare performs only the schema/registry transaction. It starts no delivery,
// maintenance, or cancellation goroutine and never owns the caller's data source.
func (b *builder) Prepare(ctx context.Context) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	_, err := b.prepareLocked(ctx)
	return err
}
func (b *builder) prepareLocked(ctx context.Context) (*serviceImpl, error) {
	if ctx == nil {
		return nil, ErrInvalid
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s := b.prepared
	if s == nil {
		s = &serviceImpl{options: b.options, scopeKey: hash(b.options.Config.Scope), table: `"` + b.options.Schema.Namespace + `".`, normalizers: b.normalizers, classifiers: b.classifiers, redactors: b.redactors, handlers: b.handlers, allowed: map[string]bool{}, done: make(chan struct{})}
		for _, key := range b.options.Config.AllowedAttributes {
			s.allowed[key] = true
		}
	}
	// Revalidate on repeated calls, including Start after an offline Prepare, so
	// intervening schema drift or an unavailable data source cannot be hidden.
	if err := s.migrate(ctx); err != nil {
		return nil, err
	}
	b.prepared = s
	b.frozen = true
	return s, nil
}
func (b *builder) Start(ctx context.Context) (Service, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.started {
		return nil, ErrRegistrationFrozen
	}
	s, err := b.prepareLocked(ctx)
	if err != nil {
		return nil, err
	}
	b.started = true
	s.ctx, s.cancel = context.WithCancel(context.Background())
	b.frozen = true
	for _, h := range s.handlers {
		for n := uint32(0); n < h.Registration.Delivery.Concurrency; n++ {
			s.work.Add(1)
			go s.worker(h)
		}
	}
	s.work.Add(1)
	go s.maintenance()
	go func() {
		<-s.ctx.Done()
		s.mu.Lock()
		s.closed = true
		s.mu.Unlock()
		s.work.Wait()
		close(s.done)
	}()
	return s, nil
}
func (s *serviceImpl) Scope() Scope { return s.options.Config.Scope }
func (s *serviceImpl) beginOperation() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed || s.ctx.Err() != nil {
		return ErrClosed
	}
	s.work.Add(1)
	return nil
}
func (s *serviceImpl) Close(ctx context.Context) error {
	if ctx == nil {
		return ErrInvalid
	}
	s.mu.Lock()
	s.closed = true
	s.cancel()
	s.mu.Unlock()
	select {
	case <-s.done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
func hash(v any) string {
	data, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
func (s *serviceImpl) tx(ctx context.Context, options *sql.TxOptions, fn func(dbsource.Transaction) error) error {
	tx, err := s.options.DataSource.BeginTx(ctx, options)
	if err != nil {
		return storageError(err)
	}
	defer func() {
		cleanup, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = tx.Rollback(cleanup)
	}()
	if err = fn(tx); err != nil {
		return err
	}
	if err = tx.Commit(ctx); err != nil {
		return storageError(err)
	}
	return nil
}
func storageError(err error) error {
	if err == nil {
		return nil
	}
	// Never expose database error text: it can contain row values and SQL parameters.
	if errors.Is(err, context.Canceled) {
		return errors.Join(ErrUnavailable, context.Canceled)
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return errors.Join(ErrUnavailable, context.DeadlineExceeded)
	}
	return ErrUnavailable
}
func (s *serviceImpl) registry() string {
	items := []HandlerRegistration{}
	for _, h := range s.handlers {
		items = append(items, h.Registration)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })
	return hash(items)
}
func (s *serviceImpl) migrate(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, s.options.Schema.MigrationTimeout)
	defer cancel()
	if err := s.options.DataSource.Ping(ctx); err != nil {
		return storageError(err)
	}
	return s.tx(ctx, nil, func(tx dbsource.Transaction) error {
		if _, err := tx.ExecContext(ctx, `SET LOCAL search_path TO pg_catalog`); err != nil {
			return storageError(err)
		}
		if _, err := tx.ExecContext(ctx, `SELECT set_config('lock_timeout',$1,true)`, fmt.Sprintf("%dms", s.options.Schema.LockTimeout.Milliseconds())); err != nil {
			return storageError(err)
		}
		if _, err := tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(hashtext($1))`, "commons.logservice:"+s.options.Schema.Namespace); err != nil {
			return storageError(err)
		}
		var exists bool
		if _, err := tx.QueryRowContext(ctx, &exists, `SELECT to_regclass($1) IS NOT NULL`, s.options.Schema.Namespace+".schema_version"); err != nil {
			return storageError(err)
		}
		if !exists && s.options.Schema.MigrationMode == MigrationVerifyOnly {
			return fmt.Errorf("%w: logservice schema missing", ErrConflict)
		}
		if !exists {
			var namespaceExists bool
			if _, err := tx.QueryRowContext(ctx, &namespaceExists, `SELECT EXISTS(SELECT 1 FROM pg_namespace WHERE nspname=$1)`, s.options.Schema.Namespace); err != nil {
				return storageError(err)
			}
			if namespaceExists {
				return fmt.Errorf("%w: namespace is not owned by logservice", ErrConflict)
			}
			statements := []string{
				`CREATE SCHEMA IF NOT EXISTS "` + s.options.Schema.Namespace + `"`,
				`CREATE TABLE ` + s.table + `schema_version (singleton boolean PRIMARY KEY DEFAULT true CHECK(singleton), version integer NOT NULL,fingerprint text NOT NULL DEFAULT '')`,
				`INSERT INTO ` + s.table + `schema_version(version) VALUES(1)`,
				`CREATE TABLE ` + s.table + `scopes(scope text PRIMARY KEY, registry text NOT NULL,registry_version bigint NOT NULL,record_sequence bigint NOT NULL DEFAULT 0)`,
				`CREATE TABLE ` + s.table + `groups(scope text NOT NULL,id text NOT NULL,key_version text NOT NULL,first_seen_at timestamptz NOT NULL,last_seen_at timestamptz NOT NULL,entry_count bigint NOT NULL,representative jsonb NOT NULL,PRIMARY KEY(scope,id))`,
				`CREATE TABLE ` + s.table + `records(scope text NOT NULL,id text NOT NULL,sequence bigint NOT NULL,group_id text NOT NULL,occurred_at timestamptz NOT NULL,recorded_at timestamptz NOT NULL,facts jsonb NOT NULL,details_expire_at timestamptz NOT NULL,details_retained boolean NOT NULL,digest text NOT NULL,PRIMARY KEY(scope,id),FOREIGN KEY(scope,group_id) REFERENCES ` + s.table + `groups(scope,id))`,
				`CREATE INDEX ON ` + s.table + `records(scope,group_id,recorded_at,id)`,
				`CREATE INDEX ON ` + s.table + `records(scope,occurred_at,id)`,
				`CREATE TABLE ` + s.table + `events(scope text NOT NULL,id text NOT NULL,payload jsonb NOT NULL,PRIMARY KEY(scope,id))`,
				`CREATE TABLE ` + s.table + `deliveries(scope text NOT NULL,id text NOT NULL,event_id text NOT NULL,handler_id text NOT NULL,policy jsonb NOT NULL,state text NOT NULL,attempts integer NOT NULL DEFAULT 0,next_attempt_at timestamptz NOT NULL,created_at timestamptz NOT NULL,lease_until timestamptz,lease_token text NOT NULL DEFAULT '',last_error_code text NOT NULL DEFAULT '',version bigint NOT NULL DEFAULT 1,PRIMARY KEY(scope,id),UNIQUE(scope,event_id,handler_id),FOREIGN KEY(scope,event_id) REFERENCES ` + s.table + `events(scope,id))`,
				`CREATE INDEX ON ` + s.table + `deliveries(scope,handler_id,state,next_attempt_at,lease_until)`,
			}
			for _, query := range statements {
				if _, err := tx.ExecContext(ctx, query); err != nil {
					return storageError(err)
				}
			}
		}
		if !exists {
			fingerprint, err := s.schemaFingerprint(ctx, tx)
			if err != nil {
				return err
			}
			if _, err := tx.ExecContext(ctx, `UPDATE `+s.table+`schema_version SET fingerprint=$1 WHERE singleton`, fingerprint); err != nil {
				return storageError(err)
			}
		}
		var savedFingerprint string
		if _, err := tx.QueryRowContext(ctx, &savedFingerprint, `SELECT fingerprint FROM `+s.table+`schema_version WHERE singleton`); err != nil {
			return storageError(err)
		}
		fingerprint, err := s.schemaFingerprint(ctx, tx)
		if err != nil {
			return err
		}
		if savedFingerprint != fingerprint {
			return fmt.Errorf("%w: logservice schema structure changed", ErrConflict)
		}
		var version int
		found, err := tx.QueryRowContext(ctx, &version, `SELECT version FROM `+s.table+`schema_version WHERE singleton`)
		if err != nil {
			return storageError(err)
		}
		if !found || version != 1 {
			return fmt.Errorf("%w: incompatible schema version", ErrConflict)
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO `+s.table+`scopes(scope,registry,registry_version) VALUES($1,$2,$3) ON CONFLICT DO NOTHING`, s.scopeKey, s.registry(), s.options.Config.RegistryVersion); err != nil {
			return storageError(err)
		}
		return s.reconcileRegistry(ctx, tx)
	})
}
func (s *serviceImpl) maintenance() {
	defer s.work.Done()
	timer := time.NewTicker(time.Minute)
	defer timer.Stop()
	for {
		_, _ = s.pruneDetails(s.ctx)
		select {
		case <-s.ctx.Done():
			return
		case <-timer.C:
		}
	}
}
func (s *serviceImpl) PruneDetails(ctx context.Context) (int64, error) {
	if ctx == nil {
		return 0, ErrInvalid
	}
	if err := s.beginOperation(); err != nil {
		return 0, err
	}
	defer s.work.Done()
	return s.pruneDetails(ctx)
}
func (s *serviceImpl) pruneDetails(ctx context.Context) (int64, error) {
	var count int64
	err := s.tx(ctx, nil, func(tx dbsource.Transaction) error {
		var err error
		count, err = tx.ExecContext(ctx, `UPDATE `+s.table+`records SET facts=facts - 'Detail' - 'Stack', details_retained=false WHERE scope=$1 AND details_retained AND details_expire_at<=clock_timestamp()`, s.scopeKey)
		return storageError(err)
	})
	return count, err
}

// Keep errors.Is available to callers while suppressing handler error text from persistence.
func handlerErrorCode(err error) string {
	if errors.Is(err, ErrPermanentHandler) {
		return "handler_permanent"
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return "handler_timeout"
	}
	if errors.Is(err, context.Canceled) {
		return "handler_canceled"
	}
	return "handler_failed"
}

func (s *serviceImpl) schemaFingerprint(ctx context.Context, tx dbsource.Transaction) (string, error) {
	var shape string
	_, err := tx.QueryRowContext(ctx, &shape, `SELECT coalesce(string_agg(item,E'\n' ORDER BY item),'') FROM (
 SELECT 'column:'||c.relname||':'||a.attname||':'||format_type(a.atttypid,a.atttypmod)||':'||a.attnotnull::text AS item FROM pg_attribute a JOIN pg_class c ON c.oid=a.attrelid JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname=$1 AND c.relkind='r' AND a.attnum>0 AND NOT a.attisdropped
 UNION ALL SELECT 'constraint:'||c.relname||':'||k.conname||':'||pg_get_constraintdef(k.oid) FROM pg_constraint k JOIN pg_class c ON c.oid=k.conrelid JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname=$1
 UNION ALL SELECT 'index:'||indexname||':'||indexdef FROM pg_indexes WHERE schemaname=$1
 ) definitions`, s.options.Schema.Namespace)
	if err != nil {
		return "", storageError(err)
	}
	return hash(shape), nil
}

func (s *serviceImpl) reconcileRegistry(ctx context.Context, tx dbsource.Transaction) error {
	var current struct {
		Registry string `db:"registry"`
		Version  uint64 `db:"registry_version"`
	}
	if _, err := tx.QueryRowContext(ctx, &current, `SELECT registry,registry_version FROM `+s.table+`scopes WHERE scope=$1 FOR UPDATE`, s.scopeKey); err != nil {
		return storageError(err)
	}
	c := s.options.Config
	if current.Registry == s.registry() && current.Version == c.RegistryVersion {
		return nil
	}
	if c.PreviousRegistryVersion != current.Version || c.RegistryVersion != current.Version+1 {
		return fmt.Errorf("%w: handler changes require next RegistryVersion and matching PreviousRegistryVersion", ErrConflict)
	}
	var running int
	if _, err := tx.QueryRowContext(ctx, &running, `SELECT count(*) FROM `+s.table+`deliveries WHERE scope=$1 AND state='running' AND lease_until>clock_timestamp()`, s.scopeKey); err != nil {
		return storageError(err)
	}
	if running > 0 {
		return fmt.Errorf("%w: close old service and wait for active deliveries before registry upgrade", ErrConflict)
	}
	if _, err := tx.ExecContext(ctx, `UPDATE `+s.table+`scopes SET registry=$2,registry_version=$3 WHERE scope=$1`, s.scopeKey, s.registry(), c.RegistryVersion); err != nil {
		return storageError(err)
	}
	args := []any{s.scopeKey}
	parts := []string{}
	for id := range s.handlers {
		args = append(args, string(id))
		parts = append(parts, fmt.Sprintf("$%d", len(args)))
	}
	missing := "true"
	present := "false"
	if len(parts) > 0 {
		missing = "handler_id NOT IN (" + strings.Join(parts, ",") + ")"
		present = "handler_id IN (" + strings.Join(parts, ",") + ")"
	}
	if _, err := tx.ExecContext(ctx, `UPDATE `+s.table+`deliveries SET state='blocked',last_error_code='registry_missing_handler',lease_until=NULL,lease_token='',version=version+1 WHERE scope=$1 AND state IN ('pending','running') AND `+missing, args...); err != nil {
		return storageError(err)
	}
	if _, err := tx.ExecContext(ctx, `UPDATE `+s.table+`deliveries SET state='pending',last_error_code='',version=version+1 WHERE scope=$1 AND state='blocked' AND last_error_code='registry_missing_handler' AND `+present, args...); err != nil {
		return storageError(err)
	}
	return nil
}
func (s *serviceImpl) checkRegistry(ctx context.Context, tx dbsource.Transaction, lock string) error {
	var current struct {
		Registry string `db:"registry"`
		Version  uint64 `db:"registry_version"`
	}
	found, err := tx.QueryRowContext(ctx, &current, `SELECT registry,registry_version FROM `+s.table+`scopes WHERE scope=$1 `+lock, s.scopeKey)
	if err != nil {
		return storageError(err)
	}
	if !found || current.Registry != s.registry() || current.Version != s.options.Config.RegistryVersion {
		return ErrConflict
	}
	return nil
}
