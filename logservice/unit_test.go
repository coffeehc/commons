package logservice

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/coffeehc/commons/dbsource"
)

type absentDataSource struct{}

func (*absentDataSource) Ping(context.Context) error { return nil }
func (*absentDataSource) BeginTx(context.Context, *sql.TxOptions) (dbsource.Transaction, error) {
	return nil, errors.New("unused")
}
func unitOptions() Options {
	return Options{DataSource: &absentDataSource{}, Config: Config{Scope: Scope{"app", "test", "tenant"}}}
}
func TestNewValidationAndRegistration(t *testing.T) {
	var nilDB *absentDataSource
	o := unitOptions()
	o.DataSource = nilDB
	if _, err := New(o); !errors.Is(err, ErrInvalid) {
		t.Fatalf("typed nil: %v", err)
	}
	for _, namespace := range []string{"public;DROP SCHEMA public", "public.schema", "../schema", "A"} {
		o = unitOptions()
		o.Schema.Namespace = namespace
		if _, err := New(o); !errors.Is(err, ErrInvalid) {
			t.Fatalf("namespace %q: %v", namespace, err)
		}
	}
	b, err := New(unitOptions())
	if err != nil {
		t.Fatal(err)
	}
	if err := b.RegisterHandler(HandlerRegistration{ID: "handler"}, HandlerFunc(func(context.Context, Delivery) error { return nil })); !errors.Is(err, ErrInvalid) {
		t.Fatalf("implicit all: %v", err)
	}
	reg := HandlerRegistration{ID: "handler", Filter: EventFilter{Kinds: []EventKind{EventErrorRecorded}}}
	var nilHandler HandlerFunc
	if err := b.RegisterHandler(reg, nilHandler); !errors.Is(err, ErrInvalid) {
		t.Fatalf("typed nil handler: %v", err)
	}
	if err := b.RegisterHandler(reg, HandlerFunc(func(context.Context, Delivery) error { return nil })); err != nil {
		t.Fatal(err)
	}
	reg.Filter.Kinds[0] = EventGroupCreated
	original := b.(*builder).handlers["handler"].Registration.Filter.Kinds[0]
	if original != EventErrorRecorded {
		t.Fatal("registration retained caller-owned slice")
	}
	if err := b.RegisterHandler(reg, HandlerFunc(func(context.Context, Delivery) error { return nil })); !errors.Is(err, ErrDuplicateHandler) {
		t.Fatal(err)
	}
}
func TestRedactionCoversTextAndBoundaries(t *testing.T) {
	for _, raw := range []string{
		`password=fixtureSecret`, `{"password":"fixtureSecret"}`, `{'access_token':'fixtureSecret'}`, `Authorization: Bearer fixtureSecret`, `Authorization: Basic fixtureSecret`, `https://user:fixtureSecret@example.test/path`, `token=fixtureSecret&ok=1`, `-----BEGIN PRIVATE KEY-----fixtureSecret-----END PRIVATE KEY-----`,
	} {
		got := redactString(raw, 1024)
		if strings.Contains(got, "fixtureSecret") {
			t.Errorf("leaked %q => %q", raw, got)
		}
	}
	s := &serviceImpl{allowed: map[string]bool{"region": true, "api_key": true}}
	f := Facts{Source: "source", Summary: "password=fixtureSecret", Detail: `{"token":"fixtureSecret"}`, Stack: "Authorization: Bearer fixtureSecret", Subject: Ref{ID: "token=fixtureSecret"}, Related: []Ref{{ID: "password=fixtureSecret"}}, Attributes: map[string]string{"region": "token=fixtureSecret", "api_key": "fixtureSecret", "other": "secret"}}
	got := s.redact(cloneFacts(f))
	if len(got.Attributes) != 1 || got.Attributes["region"] == f.Attributes["region"] || got.Subject.ID == f.Subject.ID || got.Related[0].ID == f.Related[0].ID {
		t.Fatalf("redaction failed: %+v", got)
	}
	if f.Related[0].ID != "password=fixtureSecret" || f.Attributes["region"] != "token=fixtureSecret" {
		t.Fatal("caller input was mutated")
	}
	unicode := redactString(strings.Repeat("中", 1000), 100)
	if len(unicode) > 100 || strings.ContainsRune(unicode, '�') {
		t.Fatal("broken UTF-8 truncation")
	}
}
func TestInputBoundaries(t *testing.T) {
	in := RecordInput{ID: "stable-id", Facts: Facts{Source: "source", Summary: "failure"}}
	if err := validateInput(in); err != nil {
		t.Fatal(err)
	}
	variants := []RecordInput{in, in, in, in, in, in, in}
	variants[0].ID = "unsafe\nid"
	variants[5].ID = "token:fixtureSecret"
	variants[6].ID = "eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiIxMjMifQ.fixtureSignature"
	variants[1].Facts.Source = strings.Repeat("x", 4097)
	variants[2].Facts.Attributes = map[string]string{"key": strings.Repeat("x", 16385)}
	variants[3].OccurredAt = time.Date(10000, 1, 1, 0, 0, 0, 0, time.UTC)
	variants[4].Facts.Related = []Ref{{ID: strings.Repeat("x", 4097)}}
	for i, v := range variants {
		if err := validateInput(v); !errors.Is(err, ErrInvalid) {
			t.Fatalf("variant %d: %v", i, err)
		}
	}
}
func TestClassifierAndHookIsolation(t *testing.T) {
	s := &serviceImpl{options: unitOptions(), allowed: map[string]bool{}}
	s.options.Config.SynchronousHookBudget = time.Second
	s.classifiers = []namedClassifier{{"rule", ClassifierFunc(func(context.Context, Facts) (*Classification, error) {
		return &Classification{Nature: NatureExternalDependency, Category: "dependency", Severity: SeverityCritical}, nil
	})}}
	in := RecordInput{ID: "record", Facts: Facts{Source: "source", Summary: "failure", Severity: SeverityWarning}}
	got, suppressed, err := s.prepare(context.Background(), in)
	if err != nil || suppressed || got.Facts.Nature != NatureExternalDependency || got.Facts.Severity != SeverityWarning {
		t.Fatalf("classification %+v %v %v", got, suppressed, err)
	}
	s.normalizers = []namedNormalizer{{"panic", NormalizerFunc(func(context.Context, Facts) (Facts, error) { panic("sensitive-value") })}}
	if _, _, err := s.prepare(context.Background(), in); !errors.Is(err, ErrHookFailed) || strings.Contains(err.Error(), "sensitive") {
		t.Fatalf("unsafe hook panic: %v", err)
	}
	s.normalizers = []namedNormalizer{{"timeout", NormalizerFunc(func(ctx context.Context, f Facts) (Facts, error) { <-ctx.Done(); return f, nil })}}
	s.options.Config.SynchronousHookBudget = time.Millisecond
	if _, _, err := s.prepare(context.Background(), in); !errors.Is(err, ErrHookFailed) {
		t.Fatalf("budget %v", err)
	}
}
func TestFiltersAndSafeFailures(t *testing.T) {
	yes, no := true, false
	e := Event{Kind: EventErrorRecorded, Facts: &Facts{Source: "task", Nature: NatureSystem, Severity: SeverityError}}
	if matches(EventFilter{Kinds: []EventKind{EventErrorRecorded}, Retryable: &no}, e) {
		t.Fatal("unknown retryability matched false")
	}
	e.Facts.Retryable = &yes
	if !matches(EventFilter{Kinds: []EventKind{EventErrorRecorded}, Sources: []string{"task"}, Retryable: &yes}, e) {
		t.Fatal("matching predicate rejected")
	}
	if matches(EventFilter{Kinds: []EventKind{EventGroupCreated}}, e) {
		t.Fatal("kind mismatch accepted")
	}
	raw := errors.New("password=fixtureSecret")
	if err := storageError(raw); !errors.Is(err, ErrUnavailable) || strings.Contains(err.Error(), "fixtureSecret") {
		t.Fatalf("unsafe storage failure: %v", err)
	}
	if err := storageError(context.DeadlineExceeded); !errors.Is(err, ErrUnavailable) || !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("lost cancellation classification: %v", err)
	}
	if code := handlerErrorCode(raw); strings.Contains(code, "fixtureSecret") {
		t.Fatal("unsafe handler diagnostics")
	}
}
