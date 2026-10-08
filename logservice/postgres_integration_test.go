package logservice

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/coffeehc/commons/dbsource"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// postgresFixture exercises the real commons/dbsource PostgreSQL implementation.
// The caller must explicitly opt in with a disposable, plaintext database URL;
// each fixture owns only its random schema, and never reads application config.
func postgresFixture(t *testing.T) (dbsource.Service, *pgxpool.Pool, string) {
	t.Helper()
	dsn := os.Getenv("LOGSERVICE_TEST_POSTGRES")
	if dsn == "" {
		t.Skip("set LOGSERVICE_TEST_POSTGRES to an isolated PostgreSQL test database (sslmode=disable)")
	}
	connection, err := pgx.ParseConfig(dsn)
	if err != nil {
		t.Fatal(err)
	}
	if connection.TLSConfig != nil {
		t.Fatal("integration fixtures require explicit sslmode=disable; use a disposable test database")
	}
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	admin, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(admin.Close)
	if err := admin.Ping(ctx); err != nil {
		t.Fatal(err)
	}
	schema := fmt.Sprintf("logservice_test_%d", time.Now().UnixNano())
	quoted := pgx.Identifier{schema}.Sanitize()
	t.Cleanup(func() {
		cleanup, stop := context.WithTimeout(context.Background(), 10*time.Second)
		defer stop()
		if _, err := admin.Exec(cleanup, "DROP SCHEMA IF EXISTS "+quoted+" CASCADE"); err != nil {
			t.Error(err)
		}
	})
	db, err := dbsource.Open(ctx, &dbsource.Config{
		DbType: dbsource.POSTGRES,
		Host:   connection.Host, Port: int(connection.Port), DBName: connection.Database,
		User: connection.User, Password: connection.Password,
		SSLMode: dbsource.PostgresSSLModeDisable, MaxOpenConns: 16,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		cleanup, stop := context.WithTimeout(context.Background(), 10*time.Second)
		defer stop()
		if err := db.Close(cleanup); err != nil {
			t.Error(err)
		}
	})
	return db, admin, schema
}

func postgresOptions(db DataSource, schema string) Options {
	return Options{DataSource: db, Schema: SchemaOptions{Namespace: schema}, Config: Config{
		Scope:         Scope{Application: "integration", Environment: "test", Tenant: "a"},
		RecordTimeout: 10 * time.Second, PollInterval: 10 * time.Millisecond,
		AllowedAttributes: []string{"region"},
	}}
}

func startPostgresService(t *testing.T, options Options, registrations ...struct {
	registration HandlerRegistration
	handler      EventHandler
}) Service {
	t.Helper()
	builder, err := New(options)
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range registrations {
		if err := builder.RegisterHandler(item.registration, item.handler); err != nil {
			t.Fatal(err)
		}
	}
	svc, err := builder.Start(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := svc.Close(ctx); err != nil {
			t.Error(err)
		}
	})
	return svc
}

func postgresInput(id string) RecordInput {
	return RecordInput{ID: RecordID(id), GroupKey: "fetch-failed", OccurredAt: time.Date(2026, 10, 7, 10, 0, 0, 0, time.UTC), Facts: Facts{
		Source: "integration", Nature: NatureExternalDependency, Category: "fetch", Severity: SeverityError,
		Component: "collector", Operation: "fetch", Code: "UPSTREAM_TIMEOUT", Summary: "upstream timed out",
		Detail: "synthetic diagnostic detail", Stack: "synthetic stack", Subject: Ref{Kind: "job", ID: "job-1"},
		Attributes: map[string]string{"region": "test-zone", "password": "synthetic-secret", "unapproved": "discard-me"},
	}}
}

func postgresCount(t *testing.T, admin *pgxpool.Pool, schema, table string) int64 {
	t.Helper()
	var count int64
	if err := admin.QueryRow(t.Context(), "SELECT count(*) FROM "+pgx.Identifier{schema, table}.Sanitize()).Scan(&count); err != nil {
		t.Fatal(err)
	}
	return count
}

func waitPostgres(t *testing.T, condition func() bool) {
	t.Helper()
	deadline := time.NewTimer(8 * time.Second)
	defer deadline.Stop()
	tick := time.NewTicker(10 * time.Millisecond)
	defer tick.Stop()
	for !condition() {
		select {
		case <-tick.C:
		case <-deadline.C:
			t.Fatal("PostgreSQL condition did not become true")
		}
	}
}

func TestPostgresRecordRoundTripConcurrentDedupAndScope(t *testing.T) {
	db, admin, schema := postgresFixture(t)
	options := postgresOptions(db, schema)
	svc := startPostgresService(t, options)
	input := postgresInput("repeated")
	first, err := svc.Record(t.Context(), input)
	if err != nil || first.Disposition != RecordStored {
		t.Fatalf("initial record: %+v, %v", first, err)
	}
	if first.CommittedAt.IsZero() || first.GroupID == "" {
		t.Fatalf("incomplete receipt: %+v", first)
	}
	const unique = 24
	var wg sync.WaitGroup
	errs := make(chan error, unique*2)
	for i := 0; i < unique; i++ {
		wg.Add(2)
		go func(i int) {
			defer wg.Done()
			r, err := svc.Record(t.Context(), postgresInput(fmt.Sprintf("unique-%02d", i)))
			if err == nil && (r.Disposition != RecordStored || r.GroupID != first.GroupID) {
				err = fmt.Errorf("unexpected unique receipt: %+v", r)
			}
			errs <- err
		}(i)
		go func() {
			defer wg.Done()
			r, err := svc.Record(t.Context(), input)
			if err == nil && (r.Disposition != RecordDuplicate || r.GroupID != first.GroupID || !r.CommittedAt.Equal(first.CommittedAt)) {
				err = fmt.Errorf("unexpected duplicate receipt: %+v", r)
			}
			errs <- err
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Error(err)
		}
	}
	changed := input
	changed.Facts.Summary = "different observation"
	if _, err := svc.Record(t.Context(), changed); !errors.Is(err, ErrConflict) {
		t.Fatalf("same ID changed payload: %v", err)
	}
	groups, err := svc.Query(t.Context(), QueryRequest{})
	if err != nil || groups.MatchedGroups != 1 || groups.MatchedEntries != unique+1 || len(groups.Items) != 1 || groups.Items[0].Group.EntryCount != unique+1 {
		t.Fatalf("aggregates: %+v, %v", groups, err)
	}
	if groups.Items[0].Group.Representative.Detail != "" || groups.Items[0].Group.Representative.Stack != "" {
		t.Fatal("group retained detailed payload")
	}
	detail, err := svc.Detail(t.Context(), DetailRequest{GroupID: first.GroupID})
	if err != nil || len(detail.Records) != unique+1 {
		t.Fatalf("detail: records=%d err=%v", len(detail.Records), err)
	}
	record, err := svc.GetRecord(t.Context(), input.ID)
	if err != nil || record.Facts.Detail != input.Facts.Detail || !record.DetailsRetained {
		t.Fatalf("record roundtrip: %+v, %v", record, err)
	}
	if len(record.Facts.Attributes) != 1 || record.Facts.Attributes["region"] != "test-zone" {
		t.Fatalf("attribute allowlist: %+v", record.Facts.Attributes)
	}
	if got := postgresCount(t, admin, schema, "records"); got != unique+1 {
		t.Fatalf("physical records=%d", got)
	}
	if got := postgresCount(t, admin, schema, "events"); got != unique+2 {
		t.Fatalf("physical events=%d, want one group event plus one per record", got)
	}

	otherOptions := options
	otherOptions.Config.Scope.Tenant = "b"
	other := startPostgresService(t, otherOptions)
	if _, err := other.GetRecord(t.Context(), input.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("cross-scope record leaked: %v", err)
	}
	if _, err := other.Detail(t.Context(), DetailRequest{GroupID: first.GroupID}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("cross-scope detail leaked: %v", err)
	}
	if groups, err := other.Query(t.Context(), QueryRequest{}); err != nil || groups.MatchedEntries != 0 {
		t.Fatalf("cross-scope counts: %+v %v", groups, err)
	}
	if r, err := other.Record(t.Context(), input); err != nil || r.Disposition != RecordStored {
		t.Fatalf("independent scope record: %+v %v", r, err)
	}
	if err := svc.Close(t.Context()); err != nil {
		t.Fatal(err)
	}
	if err := db.Ping(t.Context()); err != nil {
		t.Fatalf("Close closed borrowed data source: %v", err)
	}
	if _, err := svc.Record(t.Context(), postgresInput("after-close")); !errors.Is(err, ErrClosed) {
		t.Fatalf("record after Close: %v", err)
	}
}

func TestPostgresQueryPaginationFiltersAndExpiry(t *testing.T) {
	db, admin, schema := postgresFixture(t)
	svc := startPostgresService(t, postgresOptions(db, schema))
	for i := 0; i < 5; i++ {
		input := postgresInput(fmt.Sprintf("record-%d", i))
		input.GroupKey = fmt.Sprintf("group-%d", i)
		input.OccurredAt = input.OccurredAt.Add(time.Duration(i) * time.Second)
		if _, err := svc.Record(t.Context(), input); err != nil {
			t.Fatal(err)
		}
	}
	seen := map[RecordID]bool{}
	request := QueryRequest{Page: PageRequest{Size: 2}}
	for pageNumber := 0; ; pageNumber++ {
		if pageNumber > 5 {
			t.Fatal("pagination did not terminate")
		}
		page, err := svc.QueryRecords(t.Context(), request)
		if err != nil {
			t.Fatal(err)
		}
		if len(page.Items) > 2 || page.Page.AsOf.IsZero() {
			t.Fatalf("invalid page: %+v", page)
		}
		for _, record := range page.Items {
			if seen[record.ID] {
				t.Fatalf("duplicate page record %s", record.ID)
			}
			seen[record.ID] = true
		}
		if page.Page.NextCursor == "" {
			break
		}
		request.Page.Cursor = page.Page.NextCursor
	}
	if len(seen) != 5 {
		t.Fatalf("paginated records=%d", len(seen))
	}
	first, err := svc.Query(t.Context(), QueryRequest{Page: PageRequest{Size: 2}})
	if err != nil || len(first.Items) != 2 || first.MatchedGroups != 5 || first.MatchedEntries != 5 || first.Page.NextCursor == "" {
		t.Fatalf("group page: %+v %v", first, err)
	}
	if _, err := svc.Query(t.Context(), QueryRequest{Filter: RecordFilter{Sources: []string{"different"}}, Page: PageRequest{Cursor: first.Page.NextCursor}}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("cursor accepted changed filter: %v", err)
	}
	base := postgresInput("ignored").OccurredAt
	filtered, err := svc.Query(t.Context(), QueryRequest{Filter: RecordFilter{Since: base.Add(time.Second), Until: base.Add(3 * time.Second), Sources: []string{"integration"}, Categories: []string{"fetch"}, Natures: []Nature{NatureExternalDependency}, Severities: []Severity{SeverityError}, Component: "collector", Code: "UPSTREAM_TIMEOUT", Subject: &Ref{Kind: "job", ID: "job-1"}}})
	if err != nil || filtered.MatchedEntries != 2 || filtered.MatchedGroups != 2 {
		t.Fatalf("filtered counts: %+v %v", filtered, err)
	}
	literal, err := svc.Query(t.Context(), QueryRequest{Filter: RecordFilter{Text: "%"}})
	if err != nil || literal.MatchedEntries != 0 {
		t.Fatalf("LIKE wildcard should be literal: %+v %v", literal, err)
	}
	before, err := svc.GetRecord(t.Context(), "record-0")
	if err != nil || !before.DetailsRetained || before.Facts.Detail == "" {
		t.Fatalf("unexpired details: %+v %v", before, err)
	}
	if _, err := admin.Exec(t.Context(), "UPDATE "+pgx.Identifier{schema, "records"}.Sanitize()+" SET details_expire_at=now()-interval '1 second' WHERE id=$1", "record-0"); err != nil {
		t.Fatal(err)
	}
	after, err := svc.GetRecord(t.Context(), "record-0")
	if err != nil || after.DetailsRetained || after.Facts.Detail != "" || after.Facts.Stack != "" || after.Facts.Summary == "" {
		t.Fatalf("expired details visible: %+v %v", after, err)
	}
	if count, err := svc.PruneDetails(t.Context()); err != nil || count != 1 {
		t.Fatalf("physical pruning: %d, %v", count, err)
	}
	var payload string
	if err := admin.QueryRow(t.Context(), "SELECT facts::text FROM "+pgx.Identifier{schema, "records"}.Sanitize()+" WHERE id=$1", "record-0").Scan(&payload); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(payload, "synthetic diagnostic detail") || strings.Contains(payload, "synthetic stack") {
		t.Fatalf("expired detail retained in storage: %s", payload)
	}
	expired := postgresInput("record-0")
	expired.GroupKey = "group-0"
	if receipt, err := svc.Record(t.Context(), expired); err != nil || receipt.Disposition != RecordDuplicate {
		t.Fatalf("detail expiry removed dedup protection: %+v %v", receipt, err)
	}
	if receipt, err := svc.Record(t.Context(), postgresInput("record-0")); err == nil || receipt.Disposition == RecordStored { // Original record used a distinct group key.
		t.Fatalf("expired record unexpectedly accepted changed identity: %+v %v", receipt, err)
	}
}

// Faults are injected at the real transaction boundary: every non-faulted
// statement and commit still executes through dbsource against PostgreSQL.
type postgresFaultSource struct {
	DataSource
	mode atomic.Int32 // 1: reject before commit; 2: commit, then lose the acknowledgement.
}

func (s *postgresFaultSource) BeginTx(ctx context.Context, options *sql.TxOptions) (dbsource.Transaction, error) {
	tx, err := s.DataSource.BeginTx(ctx, options)
	if err != nil {
		return nil, err
	}
	return &postgresFaultTransaction{Transaction: tx, source: s}, nil
}

type postgresFaultTransaction struct {
	dbsource.Transaction
	source        *postgresFaultSource
	recordWritten bool
}

func (t *postgresFaultTransaction) ExecContext(ctx context.Context, query string, args ...any) (int64, error) {
	if strings.HasPrefix(strings.ToLower(strings.TrimSpace(query)), "insert") && strings.Contains(query, "records(") {
		t.recordWritten = true
	}
	return t.Transaction.ExecContext(ctx, query, args...)
}
func (t *postgresFaultTransaction) Commit(ctx context.Context) error {
	var mode int32
	if t.recordWritten {
		mode = t.source.mode.Swap(0)
	}
	if mode == 1 {
		return errors.New("synthetic pre-commit failure")
	}
	if err := t.Transaction.Commit(ctx); err != nil {
		return err
	}
	if mode == 2 {
		return errors.New("synthetic lost commit acknowledgement")
	}
	return nil
}

func TestPostgresRollbackAndAmbiguousCommitRecovery(t *testing.T) {
	db, admin, schema := postgresFixture(t)
	fault := &postgresFaultSource{DataSource: db}
	svc := startPostgresService(t, postgresOptions(fault, schema))
	input := postgresInput("uncertain")
	fault.mode.Store(1)
	if receipt, err := svc.Record(t.Context(), input); err == nil || receipt.Disposition == RecordStored {
		t.Fatalf("pre-commit error reported stored: %+v %v", receipt, err)
	}
	for _, table := range []string{"records", "groups", "events", "deliveries"} {
		if got := postgresCount(t, admin, schema, table); got != 0 {
			t.Fatalf("rollback left %d %s", got, table)
		}
	}
	fault.mode.Store(2)
	if receipt, err := svc.Record(t.Context(), input); err == nil || receipt.Disposition == RecordStored {
		t.Fatalf("lost acknowledgement reported unqualified success: %+v %v", receipt, err)
	}
	if record, err := svc.GetRecord(t.Context(), input.ID); err != nil || record.ID != input.ID {
		t.Fatalf("committed record missing after acknowledgement loss: %+v %v", record, err)
	}
	if receipt, err := svc.Record(t.Context(), input); err != nil || receipt.Disposition != RecordDuplicate {
		t.Fatalf("same-ID reconciliation: %+v %v", receipt, err)
	}
	if got := postgresCount(t, admin, schema, "records"); got != 1 {
		t.Fatalf("ambiguous commit duplicated records: %d", got)
	}
	if got := postgresCount(t, admin, schema, "events"); got != 2 {
		t.Fatalf("ambiguous commit duplicated events: %d", got)
	}

	abort := errors.New("synthetic host transaction rollback")
	if err := db.HandleTx(t.Context(), func(ctx context.Context) error {
		if receipt, err := svc.Record(ctx, postgresInput("host-transaction")); err != nil || receipt.Disposition != RecordStored {
			t.Fatalf("record from ambient transaction: %+v %v", receipt, err)
		}
		return abort
	}); !errors.Is(err, abort) {
		t.Fatalf("host rollback: %v", err)
	}
	if _, err := svc.GetRecord(t.Context(), "host-transaction"); err != nil {
		t.Fatalf("Stored was tied to rolled-back host transaction: %v", err)
	}
}

func postgresRegistration(id HandlerID) HandlerRegistration {
	return HandlerRegistration{ID: id, Filter: EventFilter{Kinds: []EventKind{EventErrorRecorded}}, Delivery: DeliveryPolicy{
		Timeout: time.Second, MaxAttempts: 3, InitialBackoff: 20 * time.Millisecond, MaxBackoff: 100 * time.Millisecond, MaxAge: time.Minute, Concurrency: 1,
	}}
}

func postgresWithHandler(registration HandlerRegistration, handler EventHandler) struct {
	registration HandlerRegistration
	handler      EventHandler
} {
	return struct {
		registration HandlerRegistration
		handler      EventHandler
	}{registration, handler}
}

func postgresDelivery(t *testing.T, svc Service, id HandlerID) (DeliveryRecord, bool) {
	t.Helper()
	page, err := svc.QueryDeliveries(t.Context(), DeliveryQuery{HandlerID: id})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Items) == 0 {
		return DeliveryRecord{}, false
	}
	if len(page.Items) != 1 {
		t.Fatalf("handler %s has %d deliveries, want one", id, len(page.Items))
	}
	return page.Items[0], true
}

func TestPostgresHandlerFanoutRetryAndPermanentFailure(t *testing.T) {
	db, admin, schema := postgresFixture(t)
	var mu sync.Mutex
	keys := make(map[HandlerID]string)
	calls := make(map[HandlerID]int)
	handler := func(id HandlerID) HandlerFunc {
		return func(_ context.Context, delivery Delivery) error {
			mu.Lock()
			defer mu.Unlock()
			calls[id]++
			if previous := keys[id]; previous != "" && previous != delivery.IdempotencyKey {
				t.Errorf("idempotency key changed for %s", id)
			}
			keys[id] = delivery.IdempotencyKey
			if delivery.Event.Facts == nil || delivery.Event.Facts.Detail != "" || delivery.Event.Facts.Stack != "" {
				t.Errorf("handler received detailed payload: %+v", delivery.Event.Facts)
			}
			if delivery.IdempotencyKey == "" || delivery.ID == "" || delivery.Event.CommittedAt.IsZero() {
				t.Errorf("incomplete durable delivery: %+v", delivery)
			}
			switch id {
			case "transient":
				if calls[id] == 1 {
					return errors.New("synthetic transient password=do-not-persist")
				}
			case "permanent":
				return fmt.Errorf("synthetic failure: %w", ErrPermanentHandler)
			}
			return nil
		}
	}
	groupRegistration := postgresRegistration("first-group")
	groupRegistration.Filter.Kinds = []EventKind{EventGroupCreated}
	options := postgresOptions(db, schema)
	svc := startPostgresService(t, options,
		postgresWithHandler(postgresRegistration("transient"), handler("transient")),
		postgresWithHandler(postgresRegistration("permanent"), handler("permanent")),
		postgresWithHandler(groupRegistration, handler("first-group")),
	)
	if receipt, err := svc.Record(t.Context(), postgresInput("fanout")); err != nil || receipt.Disposition != RecordStored {
		t.Fatalf("record: %+v %v", receipt, err)
	}
	waitPostgres(t, func() bool {
		transient, a := postgresDelivery(t, svc, "transient")
		permanent, b := postgresDelivery(t, svc, "permanent")
		group, c := postgresDelivery(t, svc, "first-group")
		return a && b && c && transient.State == DeliverySucceeded && permanent.State == DeliveryFailed && group.State == DeliverySucceeded
	})
	transient, _ := postgresDelivery(t, svc, "transient")
	permanent, _ := postgresDelivery(t, svc, "permanent")
	group, _ := postgresDelivery(t, svc, "first-group")
	if transient.Attempts != 2 || permanent.Attempts != 1 || group.Attempts != 1 {
		t.Fatalf("attempt counts: transient=%d permanent=%d group=%d", transient.Attempts, permanent.Attempts, group.Attempts)
	}
	if strings.Contains(transient.LastErrorDetail, "do-not-persist") || strings.Contains(permanent.LastErrorDetail, "synthetic") {
		t.Fatal("handler error text leaked into delivery diagnostics")
	}
	if postgresCount(t, admin, schema, "records") != 1 || postgresCount(t, admin, schema, "deliveries") != 3 {
		t.Fatal("handler outcome mutated record or fanout count")
	}
	mu.Lock()
	defer mu.Unlock()
	if keys["transient"] == keys["permanent"] {
		t.Fatal("different handler deliveries shared an idempotency key")
	}
}

func TestPostgresHandlerRestartRetainsPendingDelivery(t *testing.T) {
	db, _, schema := postgresFixture(t)
	options := postgresOptions(db, schema)
	registration := postgresRegistration("restart")
	registration.Delivery.InitialBackoff = 2 * time.Second
	registration.Delivery.MaxBackoff = 2 * time.Second
	var firstKey string
	var mu sync.Mutex
	first := startPostgresService(t, options, postgresWithHandler(registration, HandlerFunc(func(_ context.Context, d Delivery) error {
		mu.Lock()
		firstKey = d.IdempotencyKey
		mu.Unlock()
		return errors.New("synthetic retry after restart")
	})))
	if _, err := first.Record(t.Context(), postgresInput("restart")); err != nil {
		t.Fatal(err)
	}
	waitPostgres(t, func() bool {
		d, ok := postgresDelivery(t, first, "restart")
		return ok && d.State == DeliveryPending && d.Attempts == 1
	})
	if err := first.Close(t.Context()); err != nil {
		t.Fatal(err)
	}
	second := startPostgresService(t, options, postgresWithHandler(registration, HandlerFunc(func(_ context.Context, d Delivery) error {
		mu.Lock()
		defer mu.Unlock()
		if d.IdempotencyKey != firstKey || d.Attempt != 2 {
			t.Errorf("restart changed delivery identity/attempt: %+v", d)
		}
		return nil
	})))
	waitPostgres(t, func() bool {
		d, ok := postgresDelivery(t, second, "restart")
		return ok && d.State == DeliverySucceeded && d.Attempts == 2
	})
	builder, err := New(options)
	if err != nil {
		t.Fatal(err)
	}
	changed := registration
	changed.Filter.Sources = []string{"different"}
	if err := builder.RegisterHandler(changed, HandlerFunc(func(context.Context, Delivery) error { return nil })); err != nil {
		t.Fatal(err)
	}
	if svc, err := builder.Start(t.Context()); err == nil {
		_ = svc.Close(t.Context())
		t.Fatal("changed durable registry silently replaced pending-event routing policy")
	}
}

func TestPostgresLeaseFencesLateHandlerAcknowledgement(t *testing.T) {
	db, admin, schema := postgresFixture(t)
	options := postgresOptions(db, schema)
	registration := postgresRegistration("lease")
	registration.Delivery.Timeout = 5 * time.Second
	entered := make(chan struct{})
	release := make(chan struct{})
	var releaseOnce sync.Once
	defer releaseOnce.Do(func() { close(release) })
	first := startPostgresService(t, options, postgresWithHandler(registration, HandlerFunc(func(context.Context, Delivery) error {
		select {
		case <-entered:
		default:
			close(entered)
		}
		<-release // Deliberately violates cooperative cancellation, exercising fencing.
		return ErrPermanentHandler
	})))
	if _, err := first.Record(t.Context(), postgresInput("lease")); err != nil {
		t.Fatal(err)
	}
	select {
	case <-entered:
	case <-time.After(5 * time.Second):
		t.Fatal("first handler did not start")
	}
	if _, err := admin.Exec(t.Context(), "UPDATE "+pgx.Identifier{schema, "deliveries"}.Sanitize()+" SET lease_until=now()-interval '1 second' WHERE handler_id='lease'"); err != nil {
		t.Fatal(err)
	}
	second := startPostgresService(t, options, postgresWithHandler(registration, HandlerFunc(func(context.Context, Delivery) error { return nil })))
	waitPostgres(t, func() bool {
		d, ok := postgresDelivery(t, second, "lease")
		return ok && d.State == DeliverySucceeded && d.Attempts == 2
	})
	before, _ := postgresDelivery(t, second, "lease")
	releaseOnce.Do(func() { close(release) })
	if err := first.Close(t.Context()); err != nil {
		t.Fatal(err)
	}
	after, _ := postgresDelivery(t, second, "lease")
	if after.State != DeliverySucceeded || after.Attempts != 2 || after.Version != before.Version {
		t.Fatalf("stale handler acknowledgement overwrote new lease: before=%+v after=%+v", before, after)
	}
}

func TestPostgresBackpressureIsAtomic(t *testing.T) {
	db, admin, schema := postgresFixture(t)
	options := postgresOptions(db, schema)
	options.Config.MaxPendingDeliveries = 1
	entered := make(chan struct{})
	svc := startPostgresService(t, options, postgresWithHandler(postgresRegistration("slow"), HandlerFunc(func(ctx context.Context, _ Delivery) error {
		select {
		case <-entered:
		default:
			close(entered)
		}
		<-ctx.Done()
		return ctx.Err()
	})))
	if _, err := svc.Record(t.Context(), postgresInput("accepted")); err != nil {
		t.Fatal(err)
	}
	select {
	case <-entered:
	case <-time.After(5 * time.Second):
		t.Fatal("slow handler did not start")
	}
	if receipt, err := svc.Record(t.Context(), postgresInput("rejected")); !errors.Is(err, ErrBackpressure) || receipt.Disposition == RecordStored {
		t.Fatalf("backpressure result: %+v %v", receipt, err)
	}
	if _, err := svc.GetRecord(t.Context(), "rejected"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("rejected record persisted: %v", err)
	}
	for table, want := range map[string]int64{"records": 1, "groups": 1, "events": 2, "deliveries": 1} {
		if got := postgresCount(t, admin, schema, table); got != want {
			t.Fatalf("backpressure %s=%d want %d", table, got, want)
		}
	}
	if receipt, err := svc.Record(t.Context(), postgresInput("accepted")); err != nil || receipt.Disposition != RecordDuplicate {
		t.Fatalf("backpressure prevented safe duplicate reconciliation: %+v %v", receipt, err)
	}
}

func TestPostgresMigrationOwnershipAndVerifyOnly(t *testing.T) {
	t.Run("verify_missing", func(t *testing.T) {
		db, admin, schema := postgresFixture(t)
		options := postgresOptions(db, schema)
		options.Schema.MigrationMode = MigrationVerifyOnly
		builder, err := New(options)
		if err != nil {
			t.Fatal(err)
		}
		if svc, err := builder.Start(t.Context()); err == nil {
			_ = svc.Close(t.Context())
			t.Fatal("verify-only initialized missing schema")
		}
		var exists bool
		if err := admin.QueryRow(t.Context(), "SELECT EXISTS(SELECT 1 FROM pg_namespace WHERE nspname=$1)", schema).Scan(&exists); err != nil || exists {
			t.Fatalf("verify-only changed schema exists=%t err=%v", exists, err)
		}
	})
	t.Run("foreign_namespace", func(t *testing.T) {
		db, admin, schema := postgresFixture(t)
		quoted := pgx.Identifier{schema}.Sanitize()
		if _, err := admin.Exec(t.Context(), "CREATE SCHEMA "+quoted+"; CREATE TABLE "+quoted+".foreign_data(id int PRIMARY KEY); INSERT INTO "+quoted+".foreign_data VALUES (7)"); err != nil {
			t.Fatal(err)
		}
		builder, err := New(postgresOptions(db, schema))
		if err != nil {
			t.Fatal(err)
		}
		if svc, err := builder.Start(t.Context()); err == nil {
			_ = svc.Close(t.Context())
			t.Fatal("migration adopted unknown populated namespace")
		}
		if got := postgresCount(t, admin, schema, "foreign_data"); got != 1 {
			t.Fatal("migration modified unrelated data")
		}
	})
	t.Run("existing_compatible", func(t *testing.T) {
		db, _, schema := postgresFixture(t)
		options := postgresOptions(db, schema)
		first := startPostgresService(t, options)
		if _, err := first.Record(t.Context(), postgresInput("existing")); err != nil {
			t.Fatal(err)
		}
		if err := first.Close(t.Context()); err != nil {
			t.Fatal(err)
		}
		options.Schema.MigrationMode = MigrationVerifyOnly
		second := startPostgresService(t, options)
		if _, err := second.GetRecord(t.Context(), "existing"); err != nil {
			t.Fatalf("verify-only lost existing record: %v", err)
		}
	})
}

func TestPostgresConcurrentStartupAndIncompatibleVersion(t *testing.T) {
	db, admin, schema := postgresFixture(t)
	options := postgresOptions(db, schema)
	const count = 6
	type result struct {
		svc Service
		err error
	}
	results := make(chan result, count)
	for i := 0; i < count; i++ {
		go func() {
			builder, err := New(options)
			if err != nil {
				results <- result{err: err}
				return
			}
			svc, err := builder.Start(t.Context())
			results <- result{svc, err}
		}()
	}
	for i := 0; i < count; i++ {
		r := <-results
		if r.err != nil {
			t.Errorf("concurrent startup: %v", r.err)
			continue
		}
		if err := r.svc.Close(t.Context()); err != nil {
			t.Error(err)
		}
	}
	if got := postgresCount(t, admin, schema, "schema_version"); got != 1 {
		t.Fatalf("migration version rows=%d", got)
	}
	if _, err := admin.Exec(t.Context(), "UPDATE "+pgx.Identifier{schema, "schema_version"}.Sanitize()+" SET version=999"); err != nil {
		t.Fatal(err)
	}
	builder, err := New(options)
	if err != nil {
		t.Fatal(err)
	}
	if svc, err := builder.Start(t.Context()); err == nil {
		_ = svc.Close(t.Context())
		t.Fatal("future schema version accepted")
	}
	var version int
	if err := admin.QueryRow(t.Context(), "SELECT version FROM "+pgx.Identifier{schema, "schema_version"}.Sanitize()).Scan(&version); err != nil || version != 999 {
		t.Fatalf("startup rewrote incompatible version: %d %v", version, err)
	}
}

func TestPostgresStartContextIsStartupOnly(t *testing.T) {
	db, _, schema := postgresFixture(t)
	builder, err := New(postgresOptions(db, schema))
	if err != nil {
		t.Fatal(err)
	}
	startup, cancel := context.WithCancel(t.Context())
	svc, err := builder.Start(startup)
	if err != nil {
		cancel()
		t.Fatal(err)
	}
	defer svc.Close(context.Background())
	cancel()
	if _, err := svc.Record(t.Context(), postgresInput("after-start-cancel")); err != nil {
		t.Fatalf("startup context canceled service lifetime: %v", err)
	}
	if err := builder.RegisterHandler(postgresRegistration("late"), HandlerFunc(func(context.Context, Delivery) error { return nil })); !errors.Is(err, ErrRegistrationFrozen) {
		t.Fatalf("post-start registration: %v", err)
	}
	if _, err := builder.Start(t.Context()); !errors.Is(err, ErrRegistrationFrozen) {
		t.Fatalf("second Start: %v", err)
	}
}

func TestPostgresCanceledRecordDoesNotPersist(t *testing.T) {
	db, admin, schema := postgresFixture(t)
	svc := startPostgresService(t, postgresOptions(db, schema))
	blocker, err := admin.Begin(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	defer blocker.Rollback(context.Background())
	if _, err := blocker.Exec(t.Context(), "LOCK TABLE "+pgx.Identifier{schema, "scopes"}.Sanitize()+" IN ACCESS EXCLUSIVE MODE"); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(t.Context(), 100*time.Millisecond)
	defer cancel()
	if receipt, err := svc.Record(ctx, postgresInput("canceled")); !errors.Is(err, context.DeadlineExceeded) || receipt.Disposition == RecordStored {
		t.Fatalf("canceled SQL: %+v %v", receipt, err)
	}
	if err := blocker.Rollback(t.Context()); err != nil {
		t.Fatal(err)
	}
	for _, table := range []string{"records", "groups", "events", "deliveries"} {
		if got := postgresCount(t, admin, schema, table); got != 0 {
			t.Fatalf("canceled transaction persisted %d %s", got, table)
		}
	}
	if _, err := svc.Record(t.Context(), postgresInput("after-cancel")); err != nil {
		t.Fatalf("pool unusable after cancellation: %v", err)
	}
}

func TestPostgresMigrationRejectsIncompleteOwnedSchema(t *testing.T) {
	for _, name := range []string{"missing_table", "missing_column"} {
		t.Run(name, func(t *testing.T) {
			db, admin, schema := postgresFixture(t)
			options := postgresOptions(db, schema)
			first := startPostgresService(t, options)
			if err := first.Close(t.Context()); err != nil {
				t.Fatal(err)
			}
			query := "DROP TABLE " + pgx.Identifier{schema, "deliveries"}.Sanitize()
			if name == "missing_column" {
				query = "ALTER TABLE " + pgx.Identifier{schema, "records"}.Sanitize() + " DROP COLUMN digest"
			}
			if _, err := admin.Exec(t.Context(), query); err != nil {
				t.Fatal(err)
			}
			builder, err := New(options)
			if err != nil {
				t.Fatal(err)
			}
			if svc, err := builder.Start(t.Context()); err == nil {
				_ = svc.Close(t.Context())
				t.Fatal("startup accepted structurally incomplete owned schema")
			}
		})
	}
}

func TestPostgresStatementFailureRollsBackWholeRecord(t *testing.T) {
	db, admin, schema := postgresFixture(t)
	svc := startPostgresService(t, postgresOptions(db, schema))
	events := pgx.Identifier{schema, "events"}.Sanitize()
	if _, err := admin.Exec(t.Context(), "ALTER TABLE "+events+" ADD CONSTRAINT integration_reject CHECK (false)"); err != nil {
		t.Fatal(err)
	}
	input := postgresInput("constraint-failure")
	if receipt, err := svc.Record(t.Context(), input); !errors.Is(err, ErrUnavailable) || receipt.Disposition == RecordStored {
		t.Fatalf("outbox SQL failure: %+v %v", receipt, err)
	}
	for _, table := range []string{"records", "groups", "events", "deliveries"} {
		if got := postgresCount(t, admin, schema, table); got != 0 {
			t.Fatalf("SQL failure left %d %s", got, table)
		}
	}
	if _, err := admin.Exec(t.Context(), "ALTER TABLE "+events+" DROP CONSTRAINT integration_reject"); err != nil {
		t.Fatal(err)
	}
	if receipt, err := svc.Record(t.Context(), input); err != nil || receipt.Disposition != RecordStored {
		t.Fatalf("same-ID retry after SQL rollback: %+v %v", receipt, err)
	}
}

func TestPostgresHandlerTimeoutPanicAndRetryExhaustion(t *testing.T) {
	db, _, schema := postgresFixture(t)
	registration := func(id HandlerID) HandlerRegistration {
		r := postgresRegistration(id)
		r.Delivery.Timeout = 30 * time.Millisecond
		r.Delivery.MaxAttempts = 2
		return r
	}
	svc := startPostgresService(t, postgresOptions(db, schema),
		postgresWithHandler(registration("timeout"), HandlerFunc(func(ctx context.Context, _ Delivery) error { <-ctx.Done(); return ctx.Err() })),
		postgresWithHandler(registration("panic"), HandlerFunc(func(context.Context, Delivery) error { panic("synthetic secret must not be persisted") })),
		postgresWithHandler(registration("exhausted"), HandlerFunc(func(context.Context, Delivery) error { return errors.New("synthetic repeated transient") })),
	)
	if _, err := svc.Record(t.Context(), postgresInput("failure-policies")); err != nil {
		t.Fatal(err)
	}
	for id, code := range map[HandlerID]string{"timeout": "handler_timeout", "panic": "handler_panic", "exhausted": "handler_failed"} {
		waitPostgres(t, func() bool { d, ok := postgresDelivery(t, svc, id); return ok && d.State == DeliveryFailed })
		d, _ := postgresDelivery(t, svc, id)
		if d.Attempts != 2 || d.LastErrorCode != code || d.LastErrorDetail != "" {
			t.Fatalf("%s final delivery: %+v", id, d)
		}
	}
}

func TestPostgresCausationBlocksSelfDelivery(t *testing.T) {
	db, _, schema := postgresFixture(t)
	var svc Service
	ready := make(chan struct{})
	derived := make(chan RecordReceipt, 1)
	svc = startPostgresService(t, postgresOptions(db, schema), postgresWithHandler(postgresRegistration("loop"), HandlerFunc(func(ctx context.Context, d Delivery) error {
		<-ready
		r, err := svc.Record(ctx, postgresInput("derived"))
		if err == nil {
			derived <- r
		}
		return err
	})))
	close(ready)
	if _, err := svc.Record(t.Context(), postgresInput("root")); err != nil {
		t.Fatal(err)
	}
	select {
	case r := <-derived:
		if r.Disposition != RecordStored {
			t.Fatalf("derived receipt: %+v", r)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("derived record never committed")
	}
	waitPostgres(t, func() bool {
		page, err := svc.QueryDeliveries(t.Context(), DeliveryQuery{HandlerID: "loop"})
		if err != nil {
			t.Fatal(err)
		}
		if len(page.Items) != 2 {
			return false
		}
		var blocked, succeeded int
		for _, d := range page.Items {
			if d.State == DeliveryBlocked && d.LastErrorCode == "causation_loop" && d.Attempts == 0 {
				blocked++
			}
			if d.State == DeliverySucceeded {
				succeeded++
			}
		}
		return blocked == 1 && succeeded == 1
	})
}

func TestPostgresDuplicateSurvivesSynchronousExtensionChanges(t *testing.T) {
	db, _, schema := postgresFixture(t)
	options := postgresOptions(db, schema)
	input := postgresInput("stable-producer-id")
	start := func(normalizedSummary string, suppress bool) Service {
		builder, err := New(options)
		if err != nil {
			t.Fatal(err)
		}
		if normalizedSummary != "" {
			if err := builder.RegisterNormalizer("versioned-normalizer", NormalizerFunc(func(_ context.Context, f Facts) (Facts, error) { f.Summary = normalizedSummary; return f, nil })); err != nil {
				t.Fatal(err)
			}
		}
		if suppress {
			if err := builder.RegisterClassifier("new-suppression-policy", ClassifierFunc(func(context.Context, Facts) (*Classification, error) { return &Classification{Suppress: true}, nil })); err != nil {
				t.Fatal(err)
			}
		}
		svc, err := builder.Start(t.Context())
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			if err := svc.Close(context.Background()); err != nil {
				t.Error(err)
			}
		})
		return svc
	}
	first := start("normalizer-version-one", false)
	original, err := first.Record(t.Context(), input)
	if err != nil || original.Disposition != RecordStored {
		t.Fatalf("original: %+v %v", original, err)
	}
	if err := first.Close(t.Context()); err != nil {
		t.Fatal(err)
	}
	for _, suppress := range []bool{false, true} {
		next := start("normalizer-version-two", suppress)
		replay, err := next.Record(t.Context(), input)
		if err != nil || replay.Disposition != RecordDuplicate || replay.GroupID != original.GroupID || !replay.CommittedAt.Equal(original.CommittedAt) {
			t.Fatalf("identical producer input after extension change (suppress=%t): %+v %v", suppress, replay, err)
		}
		record, err := next.GetRecord(t.Context(), input.ID)
		if err != nil || record.Facts.Summary != "normalizer-version-one" {
			t.Fatalf("replay changed immutable facts: %+v %v", record, err)
		}
		if err := next.Close(t.Context()); err != nil {
			t.Fatal(err)
		}
	}
}

func TestPostgresJSONCredentialsNeverPersist(t *testing.T) {
	db, admin, schema := postgresFixture(t)
	svc := startPostgresService(t, postgresOptions(db, schema))
	input := postgresInput("json-redaction")
	input.Facts.Summary = `upstream returned {"password":"fixturePasswordSecret","access_token": "fixtureAccessTokenSecret"}`
	input.Facts.Detail = `{"nested":{"api_key":"fixtureApiKeySecret"},"refresh_token":"fixtureRefreshTokenSecret","password":"prefix\"fixtureEscapedSecret"}`
	input.Facts.Stack = "request Authorization: Bearer fixtureBearerSecret\nrequest Authorization: Basic fixtureBasicSecret\nCookie: session=fixtureCookieOne; csrf=fixtureCookieTwo"
	input.Facts.Attributes["region"] = `{"secret": "fixtureAttributeSecret"}`
	if _, err := svc.Record(t.Context(), input); err != nil {
		t.Fatal(err)
	}
	for table, column := range map[string]string{"records": "facts", "groups": "representative", "events": "payload"} {
		var rows []string
		if err := db.QueryContext(t.Context(), &rows, "SELECT "+column+"::text FROM "+pgx.Identifier{schema, table}.Sanitize()); err != nil {
			t.Fatal(err)
		}
		for _, row := range rows {
			for _, secret := range []string{"fixturePasswordSecret", "fixtureAccessTokenSecret", "fixtureApiKeySecret", "fixtureRefreshTokenSecret", "fixtureBearerSecret", "fixtureBasicSecret", "fixtureEscapedSecret", "fixtureCookieOne", "fixtureCookieTwo", "fixtureAttributeSecret"} {
				if strings.Contains(row, secret) {
					t.Errorf("synthetic secret %s persisted in %s", secret, table)
				}
			}
		}
	}
	if postgresCount(t, admin, schema, "records") != 1 {
		t.Fatal("redaction discarded the observation")
	}
}

func TestPostgresInvalidMachineIdentifiersRejectedBeforePersistence(t *testing.T) {
	db, admin, schema := postgresFixture(t)
	svc := startPostgresService(t, postgresOptions(db, schema))
	for _, id := range []RecordID{"contains\x00nul", "contains\nnewline", "token:fixtureSecret", "eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiJmaXh0dXJlIn0.fixtureSignature"} {
		input := postgresInput(string(id))
		if _, err := svc.Record(t.Context(), input); !errors.Is(err, ErrInvalid) {
			t.Errorf("invalid record identifier %q: %v", id, err)
		}
	}
	if postgresCount(t, admin, schema, "records") != 0 {
		t.Fatal("invalid machine ID persisted")
	}
	builder, err := New(postgresOptions(db, schema))
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []HandlerID{"contains\nnewline", "token:fixtureSecret", "eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiJmaXh0dXJlIn0.fixtureSignature"} {
		if err := builder.RegisterHandler(postgresRegistration(id), HandlerFunc(func(context.Context, Delivery) error { return nil })); !errors.Is(err, ErrInvalid) {
			t.Errorf("unsafe handler ID %q: %v", id, err)
		}
	}
}

func TestPostgresCursorExcludesLaterRecordsAndScopes(t *testing.T) {
	db, _, schema := postgresFixture(t)
	options := postgresOptions(db, schema)
	svc := startPostgresService(t, options)
	for _, id := range []string{"a", "c", "e"} {
		input := postgresInput(id)
		input.GroupKey = "group-" + id
		if _, err := svc.Record(t.Context(), input); err != nil {
			t.Fatal(err)
		}
	}
	first, err := svc.QueryRecords(t.Context(), QueryRequest{Page: PageRequest{Size: 1}})
	if err != nil || len(first.Items) != 1 || first.Items[0].ID != "a" || first.Page.NextCursor == "" {
		t.Fatalf("first page: %+v %v", first, err)
	}
	groups, err := svc.Query(t.Context(), QueryRequest{Page: PageRequest{Size: 1}})
	if err != nil || groups.Page.NextCursor == "" {
		t.Fatalf("first group page: %+v %v", groups, err)
	}
	for _, id := range []string{"b", "d"} {
		input := postgresInput(id)
		input.GroupKey = "group-c"
		if _, err := svc.Record(t.Context(), input); err != nil {
			t.Fatal(err)
		}
	}
	remaining, err := svc.QueryRecords(t.Context(), QueryRequest{Page: PageRequest{Cursor: first.Page.NextCursor}})
	if err != nil || len(remaining.Items) != 2 || remaining.Items[0].ID != "c" || remaining.Items[1].ID != "e" {
		t.Fatalf("late inserts changed cursor record set: %+v %v", remaining, err)
	}
	remainingGroups, err := svc.Query(t.Context(), QueryRequest{Page: PageRequest{Cursor: groups.Page.NextCursor}})
	if err != nil || remainingGroups.MatchedGroups != 3 || remainingGroups.MatchedEntries != 3 {
		t.Fatalf("late inserts changed cursor aggregate set: %+v %v", remainingGroups, err)
	}
	for _, row := range remainingGroups.Items {
		if row.Group.EntryCount != 1 || row.MatchedEntryCount != 1 {
			t.Fatalf("late record leaked into cursor totals: %+v", row)
		}
	}
	otherOptions := options
	otherOptions.Config.Scope.Tenant = "other"
	other := startPostgresService(t, otherOptions)
	if _, err := other.QueryRecords(t.Context(), QueryRequest{Page: PageRequest{Cursor: first.Page.NextCursor}}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("cross-scope cursor accepted: %v", err)
	}
}

func TestPostgresRegistryUpgradeFencesOldWriterAndPreservesPolicy(t *testing.T) {
	db, admin, schema := postgresFixture(t)
	options := postgresOptions(db, schema)
	oldRegistration := postgresRegistration("old")
	oldRegistration.Delivery.InitialBackoff = time.Hour
	oldRegistration.Delivery.MaxBackoff = time.Hour
	oldRegistration.Delivery.MaxAge = 2 * time.Hour
	first := startPostgresService(t, options, postgresWithHandler(oldRegistration, HandlerFunc(func(context.Context, Delivery) error { return errors.New("synthetic pending at upgrade") })))
	input := postgresInput("before-upgrade")
	original, err := first.Record(t.Context(), input)
	if err != nil {
		t.Fatal(err)
	}
	waitPostgres(t, func() bool {
		d, ok := postgresDelivery(t, first, "old")
		return ok && d.State == DeliveryPending && d.Attempts == 1
	})
	pending, _ := postgresDelivery(t, first, "old")

	version2 := options
	version2.Config.RegistryVersion = 2
	version2.Config.PreviousRegistryVersion = 1
	newRegistration := postgresRegistration("new")
	second := startPostgresService(t, version2, postgresWithHandler(newRegistration, HandlerFunc(func(context.Context, Delivery) error { return nil })))
	blocked, ok := postgresDelivery(t, second, "old")
	if !ok || blocked.ID != pending.ID || blocked.State != DeliveryBlocked || blocked.LastErrorCode != "registry_missing_handler" || blocked.Attempts != 1 {
		t.Fatalf("removed handler delivery: %+v", blocked)
	}
	if _, err := first.Record(t.Context(), postgresInput("stale-writer")); !errors.Is(err, ErrConflict) {
		t.Fatalf("old registry writer was not fenced: %v", err)
	}
	if r, err := second.Record(t.Context(), input); err != nil || r.Disposition != RecordDuplicate || r.GroupID != original.GroupID {
		t.Fatalf("registry upgrade lost dedup: %+v %v", r, err)
	}
	if _, err := second.Record(t.Context(), postgresInput("after-upgrade")); err != nil {
		t.Fatal(err)
	}
	waitPostgres(t, func() bool { d, ok := postgresDelivery(t, second, "new"); return ok && d.State == DeliverySucceeded })

	version3 := options
	version3.Config.RegistryVersion = 3
	version3.Config.PreviousRegistryVersion = 2
	changedOldPolicy := postgresRegistration("old")
	changedOldPolicy.Delivery.MaxAttempts = 1
	third := startPostgresService(t, version3,
		postgresWithHandler(newRegistration, HandlerFunc(func(context.Context, Delivery) error { return nil })),
		postgresWithHandler(changedOldPolicy, HandlerFunc(func(_ context.Context, d Delivery) error {
			if d.Attempt != 2 || d.ID != pending.ID || d.IdempotencyKey != string(pending.ID) {
				t.Errorf("restored delivery changed identity/attempt: %+v", d)
			}
			return nil
		})),
	)
	if _, err := admin.Exec(t.Context(), "UPDATE "+pgx.Identifier{schema, "deliveries"}.Sanitize()+" SET next_attempt_at=now() WHERE handler_id='old'"); err != nil {
		t.Fatal(err)
	}
	waitPostgres(t, func() bool {
		d, ok := postgresDelivery(t, third, "old")
		return ok && d.State == DeliverySucceeded && d.Attempts == 2
	})
	if groups, err := third.Query(t.Context(), QueryRequest{}); err != nil || groups.MatchedEntries != 2 {
		t.Fatalf("registry upgrades changed immutable records: %+v %v", groups, err)
	}
}

func TestPostgresRegistryUpgradeRejectsActiveLease(t *testing.T) {
	db, admin, schema := postgresFixture(t)
	options := postgresOptions(db, schema)
	registration := postgresRegistration("active")
	registration.Delivery.Timeout = 5 * time.Second
	entered := make(chan struct{})
	first := startPostgresService(t, options, postgresWithHandler(registration, HandlerFunc(func(ctx context.Context, _ Delivery) error { close(entered); <-ctx.Done(); return ctx.Err() })))
	if _, err := first.Record(t.Context(), postgresInput("active-at-upgrade")); err != nil {
		t.Fatal(err)
	}
	select {
	case <-entered:
	case <-time.After(5 * time.Second):
		t.Fatal("handler did not acquire lease")
	}
	version2 := options
	version2.Config.RegistryVersion = 2
	version2.Config.PreviousRegistryVersion = 1
	builder, err := New(version2)
	if err != nil {
		t.Fatal(err)
	}
	if svc, err := builder.Start(t.Context()); !errors.Is(err, ErrConflict) {
		if svc != nil {
			_ = svc.Close(t.Context())
		}
		t.Fatalf("active-lease upgrade error: %v", err)
	}
	var revision int
	if err := admin.QueryRow(t.Context(), "SELECT registry_version FROM "+pgx.Identifier{schema, "scopes"}.Sanitize()).Scan(&revision); err != nil || revision != 1 {
		t.Fatalf("rejected upgrade changed revision=%d err=%v", revision, err)
	}
	if err := first.Close(t.Context()); err != nil {
		t.Fatal(err)
	}
	second := startPostgresService(t, version2)
	delivery, ok := postgresDelivery(t, second, "active")
	if !ok || delivery.State != DeliveryBlocked || delivery.LastErrorCode != "registry_missing_handler" {
		t.Fatalf("upgrade after graceful stop lost pending delivery: %+v", delivery)
	}
}

func TestPostgresConcurrentRegistryUpgradeHasOneWinner(t *testing.T) {
	db, admin, schema := postgresFixture(t)
	options := postgresOptions(db, schema)
	first := startPostgresService(t, options)
	if err := first.Close(t.Context()); err != nil {
		t.Fatal(err)
	}
	options.Config.RegistryVersion = 2
	options.Config.PreviousRegistryVersion = 1
	type result struct {
		svc Service
		err error
	}
	results := make(chan result, 2)
	for _, id := range []HandlerID{"candidate-a", "candidate-b"} {
		builder, err := New(options)
		if err != nil {
			t.Fatal(err)
		}
		if err := builder.RegisterHandler(postgresRegistration(id), HandlerFunc(func(context.Context, Delivery) error { return nil })); err != nil {
			t.Fatal(err)
		}
		go func() { svc, err := builder.Start(t.Context()); results <- result{svc, err} }()
	}
	wins, conflicts := 0, 0
	for i := 0; i < 2; i++ {
		r := <-results
		if r.err == nil {
			wins++
			if err := r.svc.Close(t.Context()); err != nil {
				t.Error(err)
			}
		} else if errors.Is(r.err, ErrConflict) {
			conflicts++
		} else {
			t.Error(r.err)
		}
	}
	if wins != 1 || conflicts != 1 {
		t.Fatalf("concurrent registry CAS wins=%d conflicts=%d", wins, conflicts)
	}
	var revision int
	if err := admin.QueryRow(t.Context(), "SELECT registry_version FROM "+pgx.Identifier{schema, "scopes"}.Sanitize()).Scan(&revision); err != nil || revision != 2 {
		t.Fatalf("registry revision=%d err=%v", revision, err)
	}
}

func TestPostgresCloseDeadlineDoesNotPretendHandlerStopped(t *testing.T) {
	db, _, schema := postgresFixture(t)
	registration := postgresRegistration("uncooperative")
	registration.Delivery.Timeout = 5 * time.Second
	entered := make(chan struct{})
	release := make(chan struct{})
	var once sync.Once
	defer once.Do(func() { close(release) })
	svc := startPostgresService(t, postgresOptions(db, schema), postgresWithHandler(registration, HandlerFunc(func(context.Context, Delivery) error {
		close(entered)
		<-release
		return nil
	})))
	if _, err := svc.Record(t.Context(), postgresInput("close-timeout")); err != nil {
		t.Fatal(err)
	}
	select {
	case <-entered:
	case <-time.After(5 * time.Second):
		t.Fatal("handler did not start")
	}
	closing, cancel := context.WithTimeout(t.Context(), 20*time.Millisecond)
	defer cancel()
	if err := svc.Close(closing); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Close with live uncooperative callback: %v", err)
	}
	if _, err := svc.Record(t.Context(), postgresInput("while-closing")); !errors.Is(err, ErrClosed) {
		t.Fatalf("closing service still accepts writes: %v", err)
	}
	once.Do(func() { close(release) })
	finished, stop := context.WithTimeout(t.Context(), 5*time.Second)
	defer stop()
	if err := svc.Close(finished); err != nil {
		t.Fatalf("shutdown did not finish after callback returned: %v", err)
	}
	if err := db.Ping(t.Context()); err != nil {
		t.Fatalf("shutdown closed host data source: %v", err)
	}
}
