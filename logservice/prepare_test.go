package logservice

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

func TestPrepareFailureDoesNotFreeze(t *testing.T) {
	b, err := New(unitOptions())
	if err != nil {
		t.Fatal(err)
	}
	if err := b.Prepare(nil); !errors.Is(err, ErrInvalid) {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if err := b.Prepare(ctx); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if err := b.Prepare(t.Context()); err == nil {
		t.Fatal("unavailable datasource accepted")
	}
	if b.(*builder).frozen || b.(*builder).prepared != nil {
		t.Fatal("failed preparation published state")
	}
	if err := b.RegisterHandler(HandlerRegistration{ID: "retry", Filter: EventFilter{Kinds: []EventKind{EventErrorRecorded}}}, HandlerFunc(func(context.Context, Delivery) error { return nil })); err != nil {
		t.Fatal(err)
	}
}

func TestPostgresPrepareLifecycle(t *testing.T) {
	db, admin, schema := postgresFixture(t)
	b, err := New(postgresOptions(db, schema))
	if err != nil {
		t.Fatal(err)
	}
	reg := HandlerRegistration{ID: "prepared", Filter: EventFilter{Kinds: []EventKind{EventErrorRecorded}}}
	deliveries := make(chan Delivery, 8)
	if err := b.RegisterHandler(reg, HandlerFunc(func(_ context.Context, d Delivery) error { deliveries <- d; return nil })); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		if err := b.Prepare(t.Context()); err != nil {
			t.Fatal(err)
		}
	}
	p := b.(*builder).prepared
	if p.ctx != nil || p.cancel != nil || b.(*builder).started {
		t.Fatal("Prepare started runtime")
	}
	select {
	case <-p.done:
		t.Fatal("Prepare launched lifecycle")
	default:
	}
	if err := b.RegisterHandler(reg, HandlerFunc(func(context.Context, Delivery) error { return nil })); !errors.Is(err, ErrRegistrationFrozen) {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	results := make(chan Service, 2)
	failures := make(chan error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s, e := b.Start(t.Context())
			if e != nil {
				failures <- e
			} else {
				results <- s
			}
		}()
	}
	wg.Wait()
	if len(results) != 1 || len(failures) != 1 {
		t.Fatalf("start ownership: %d/%d", len(results), len(failures))
	}
	if err := <-failures; !errors.Is(err, ErrRegistrationFrozen) {
		t.Fatal(err)
	}
	svc := <-results
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
		defer cancel()
		if err := svc.Close(ctx); err != nil {
			t.Error(err)
		}
	})
	if _, err := svc.Record(t.Context(), postgresInput("prepared-record")); err != nil {
		t.Fatal(err)
	}
	select {
	case <-deliveries:
	case <-time.After(5 * time.Second):
		t.Fatal("Start after Prepare did not deliver")
	}
	if postgresCount(t, admin, schema, "records") != 1 {
		t.Fatal("record missing")
	}
}

func TestPostgresPrepareRevalidatesSchema(t *testing.T) {
	db, admin, schema := postgresFixture(t)
	opts := postgresOptions(db, schema)
	b, err := New(opts)
	if err != nil {
		t.Fatal(err)
	}
	if err := b.Prepare(t.Context()); err != nil {
		t.Fatal(err)
	}
	if _, err := admin.Exec(t.Context(), "ALTER TABLE "+pgx.Identifier{schema, "records"}.Sanitize()+" DROP COLUMN digest"); err != nil {
		t.Fatal(err)
	}
	if err := b.Prepare(t.Context()); err == nil {
		t.Fatal("Prepare accepted drift")
	}
	if _, err := b.Start(t.Context()); err == nil {
		t.Fatal("Start accepted drift after Prepare")
	}
	if b.(*builder).started {
		t.Fatal("failed start published runtime")
	}
}
