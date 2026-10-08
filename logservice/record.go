package logservice

import (
	"context"
	"encoding/json"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/coffeehc/commons/dbsource"
)

var sensitiveKey = regexp.MustCompile(`(?i)(password|passwd|secret|token|authorization|cookie|api[-_]?key|private[-_]?key|credential)`)
var credentialPair = regexp.MustCompile(`(?i)((?:password|passwd|secret|token|access_token|refresh_token|api[-_]?key|authorization|cookie)["']?\s*[=:]\s*)(?:"(?:\\.|[^"\\])*"|'(?:\\.|[^'\\])*'|[^\s,;&]+)`)
var cookieHeader = regexp.MustCompile(`(?im)\b(?:cookie|set-cookie)\s*:\s*[^\r\n]*`)
var bearer = regexp.MustCompile(`(?i)\b(?:bearer|basic)\s+[a-z0-9+/_.=:-]+`)
var urlPassword = regexp.MustCompile(`(?i)(https?://)[^/@\s]+:[^/@\s]+@`)
var jwt = regexp.MustCompile(`\beyJ[a-zA-Z0-9_-]+\.[a-zA-Z0-9_-]+\.[a-zA-Z0-9_-]+\b`)
var pemKey = regexp.MustCompile(`(?s)-----BEGIN [^-]*PRIVATE KEY-----.*?-----END [^-]*PRIVATE KEY-----`)

func redactString(v string, limit int) string {
	v = strings.ToValidUTF8(v, "�")
	v = strings.ReplaceAll(v, "\x00", "")
	v = pemKey.ReplaceAllString(v, "[REDACTED]")
	v = cookieHeader.ReplaceAllString(v, "Cookie: [REDACTED]")
	v = urlPassword.ReplaceAllString(v, "${1}[REDACTED]@")
	v = bearer.ReplaceAllString(v, "[REDACTED]")
	v = credentialPair.ReplaceAllString(v, "${1}[REDACTED]")
	v = jwt.ReplaceAllString(v, "[REDACTED]")
	if len(v) > limit {
		v = v[:limit]
		for !utf8.ValidString(v) {
			v = v[:len(v)-1]
		}
	}
	return v
}
func cloneFacts(f Facts) Facts {
	f.Related = append([]Ref(nil), f.Related...)
	if f.Attributes != nil {
		a := make(map[string]string, len(f.Attributes))
		for k, v := range f.Attributes {
			a[k] = v
		}
		f.Attributes = a
	}
	if f.Retryable != nil {
		v := *f.Retryable
		f.Retryable = &v
	}
	if f.UserFixable != nil {
		v := *f.UserFixable
		f.UserFixable = &v
	}
	return f
}
func (s *serviceImpl) redact(f Facts) Facts {
	for _, v := range []*string{&f.Source, &f.Category, &f.Component, &f.Operation, &f.Code, &f.RequestID, &f.TraceID, &f.Subject.Kind, &f.Subject.ID} {
		*v = redactString(*v, 256)
	}
	f.Summary = redactString(f.Summary, 2048)
	f.Detail = redactString(f.Detail, 16384)
	f.Stack = redactString(f.Stack, 16384)
	if len(f.Related) > 16 {
		f.Related = f.Related[:16]
	}
	for i := range f.Related {
		f.Related[i].Kind = redactString(f.Related[i].Kind, 64)
		f.Related[i].ID = redactString(f.Related[i].ID, 256)
	}
	a := map[string]string{}
	for k, v := range f.Attributes {
		if s.allowed[k] && !sensitiveKey.MatchString(k) && len(k) <= 64 {
			a[k] = redactString(v, 512)
		}
	}
	f.Attributes = a
	return f
}
func validFacts(f Facts) bool {
	switch f.Nature {
	case NatureSystem, NatureBusiness, NatureUserInput, NatureExternalDependency, NatureDataConsistency, NatureSecurity, NatureResource, NatureUnknown:
	default:
		return false
	}
	switch f.Severity {
	case SeverityInfo, SeverityWarning, SeverityError, SeverityCritical:
	default:
		return false
	}
	return f.Source != "" && f.Summary != ""
}
func summary(f Facts) Facts { f = cloneFacts(f); f.Detail = ""; f.Stack = ""; return f }
func (s *serviceImpl) prepare(ctx context.Context, in RecordInput) (out RecordInput, suppressed bool, err error) {
	defer func() {
		if recover() != nil {
			err = ErrHookFailed
		}
	}()
	if !safeID(string(in.ID)) || len(in.GroupKey) > 512 {
		return in, false, ErrInvalid
	}
	// Bound all untrusted text before extension code and redact again afterward.
	out = in
	out.Facts = cloneFacts(in.Facts)
	if len(in.Facts.Attributes) > 128 || len(in.Facts.Related) > 128 {
		return in, false, ErrInvalid
	}
	for _, v := range []string{in.Facts.Summary, in.Facts.Detail, in.Facts.Stack} {
		if len(v) > 1024*1024 {
			return in, false, ErrInvalid
		}
	}
	hookctx, cancel := context.WithTimeout(ctx, s.options.Config.SynchronousHookBudget)
	defer cancel()
	for _, h := range s.normalizers {
		out.Facts, err = h.hook.Normalize(hookctx, cloneFacts(out.Facts))
		if err != nil || hookctx.Err() != nil {
			return out, false, ErrHookFailed
		}
	}
	for _, h := range s.classifiers {
		c, e := h.hook.Classify(hookctx, cloneFacts(out.Facts))
		if e != nil || hookctx.Err() != nil {
			return out, false, ErrHookFailed
		}
		if c == nil {
			continue
		}
		if c.Suppress {
			return out, true, nil
		}
		f := &out.Facts
		if f.Nature == "" {
			f.Nature = c.Nature
		}
		if f.Category == "" {
			f.Category = c.Category
		}
		if f.Severity == "" {
			f.Severity = c.Severity
		}
		if f.Retryable == nil {
			f.Retryable = c.Retryable
		}
		if f.UserFixable == nil {
			f.UserFixable = c.UserFixable
		}
		if out.GroupKey == "" {
			out.GroupKey = c.GroupKey
		}
		break
	}
	for _, h := range s.redactors {
		out.Facts, err = h.hook.Redact(hookctx, cloneFacts(out.Facts))
		if err != nil || hookctx.Err() != nil {
			return out, false, ErrHookFailed
		}
	}
	out.Facts = s.redact(cloneFacts(out.Facts))
	out.GroupKey = redactString(out.GroupKey, 512)
	if out.Facts.Nature == "" {
		out.Facts.Nature = NatureUnknown
	}
	if out.Facts.Severity == "" {
		out.Facts.Severity = SeverityError
	}
	if !validFacts(out.Facts) {
		return out, false, ErrInvalid
	}
	if !out.OccurredAt.IsZero() {
		out.OccurredAt = out.OccurredAt.UTC().Truncate(time.Microsecond)
	}
	return out, false, nil
}

type causalKey struct{}

func (s *serviceImpl) Record(ctx context.Context, in RecordInput) (RecordReceipt, error) {
	if ctx == nil {
		return RecordReceipt{}, ErrInvalid
	}
	if err := s.beginOperation(); err != nil {
		return RecordReceipt{}, err
	}
	defer s.work.Done()
	ctx, cancel := context.WithTimeout(ctx, s.options.Config.RecordTimeout)
	defer cancel()
	if err := ctx.Err(); err != nil {
		return RecordReceipt{}, err
	}
	if err := validateInput(in); err != nil {
		return RecordReceipt{}, err
	}

	// Versioned, built-in-redacted producer identity excludes mutable hooks/defaults.
	// Inputs differing only in removed secrets/truncated content are equivalent.
	original := in
	original.Facts = cloneFacts(in.Facts)
	fixed := &serviceImpl{allowed: map[string]bool{}}
	for key := range original.Facts.Attributes {
		fixed.allowed[key] = true
	}
	original.Facts = fixed.redact(original.Facts)
	original.GroupKey = redactString(original.GroupKey, 512)
	if !original.OccurredAt.IsZero() {
		original.OccurredAt = original.OccurredAt.UTC().Truncate(time.Microsecond)
	}
	digest := "v1:" + hash(original)

	var duplicate *RecordReceipt
	if err := s.tx(ctx, nil, func(tx dbsource.Transaction) error {
		var err error
		duplicate, err = s.existingRecord(ctx, tx, in.ID, digest)
		return err
	}); err != nil {
		return RecordReceipt{}, err
	}
	if duplicate != nil {
		return *duplicate, nil
	}
	if s.options.Config.Disabled {
		return RecordReceipt{Disposition: RecordDisabled, RecordID: in.ID}, nil
	}
	in, suppressed, err := s.prepare(ctx, in)
	if err != nil {
		return RecordReceipt{}, err
	}
	if suppressed {
		return RecordReceipt{Disposition: RecordSuppressed, RecordID: in.ID, ReasonCode: "classifier_suppressed"}, nil
	}
	lineage, _ := ctx.Value(causalKey{}).(Causation)
	if lineage.Depth > s.options.Config.MaxCausationDepth {
		return RecordReceipt{}, ErrCausationLimit
	}
	grouping := any(in.GroupKey)
	if in.GroupKey == "" {
		grouping = struct {
			Source                                        string
			Nature                                        Nature
			Category, Component, Operation, Code, Summary string
		}{in.Facts.Source, in.Facts.Nature, in.Facts.Category, in.Facts.Component, in.Facts.Operation, in.Facts.Code, ""}
		if in.Facts.Code == "" {
			grouping = struct {
				Routing any
				Summary string
			}{grouping, in.Facts.Summary}
		}
	}
	groupID := GroupID(hash([]any{s.scopeKey, s.options.Config.GroupingVersion, grouping}))
	receipt := RecordReceipt{RecordID: in.ID, GroupID: groupID}
	err = s.tx(ctx, nil, func(tx dbsource.Transaction) error {
		// Scope serialization gives exact dedup/group counts and backlog admission across instances.
		if err := s.checkRegistry(ctx, tx, "FOR UPDATE"); err != nil {
			return err
		}

		old, e := s.existingRecord(ctx, tx, in.ID, digest)
		if e != nil {
			return e
		}
		if old != nil {
			receipt = *old
			return nil
		}

		var sequence int64
		if _, e := tx.QueryRowContext(ctx, &sequence, `UPDATE `+s.table+`scopes SET record_sequence=record_sequence+1 WHERE scope=$1 RETURNING record_sequence`, s.scopeKey); e != nil {
			return storageError(e)
		}
		var now time.Time
		if _, e := tx.QueryRowContext(ctx, &now, `SELECT clock_timestamp()`); e != nil {
			return storageError(e)
		}
		occurred := in.OccurredAt
		if occurred.IsZero() {
			occurred = now
		}
		receipt.CommittedAt = now
		rep, _ := json.Marshal(summary(in.Facts))
		n, e := tx.ExecContext(ctx, `INSERT INTO `+s.table+`groups(scope,id,key_version,first_seen_at,last_seen_at,entry_count,representative) VALUES($1,$2,$3,$4,$4,0,$5) ON CONFLICT DO NOTHING`, s.scopeKey, string(groupID), s.options.Config.GroupingVersion, occurred, string(rep))
		if e != nil {
			return storageError(e)
		}
		kinds := []EventKind{EventErrorRecorded}
		if n == 1 {
			kinds = append(kinds, EventGroupCreated)
		}
		type pending struct {
			event         Event
			registrations []registeredHandler
		}
		events := []pending{}
		var admissions int
		for _, kind := range kinds {
			event := Event{ID: EventID(hash([]any{s.scopeKey, in.ID, kind})), SchemaVersion: 1, Scope: s.Scope(), Kind: kind, CommittedAt: now, RecordID: in.ID, GroupID: groupID, Facts: ptrFacts(summary(in.Facts)), Causation: lineage}
			if event.Causation.RootEventID == "" {
				event.Causation.RootEventID = event.ID
			}
			p := pending{event: event}
			for _, h := range s.handlers {
				if matches(h.Registration.Filter, event) {
					p.registrations = append(p.registrations, h)
					if !contains(lineage.HandlerPath, h.Registration.ID) {
						admissions++
					}
				}
			}
			events = append(events, p)
		}
		if admissions > 0 {
			var count int
			if _, e := tx.QueryRowContext(ctx, &count, `SELECT count(*) FROM `+s.table+`deliveries WHERE scope=$1 AND state IN ('pending','running')`, s.scopeKey); e != nil {
				return storageError(e)
			}
			if count+admissions > s.options.Config.MaxPendingDeliveries {
				return ErrBackpressure
			}
		}
		payload, _ := json.Marshal(in.Facts)
		if _, e := tx.ExecContext(ctx, `INSERT INTO `+s.table+`records(scope,id,group_id,occurred_at,recorded_at,facts,details_expire_at,details_retained,digest,sequence) VALUES($1,$2,$3,$4,$5,$6,$7,true,$8,$9)`, s.scopeKey, string(in.ID), string(groupID), occurred, now, string(payload), now.Add(s.options.Config.DetailRetention), digest, sequence); e != nil {
			return storageError(e)
		}
		if _, e := tx.ExecContext(ctx, `UPDATE `+s.table+`groups SET entry_count=entry_count+1,first_seen_at=LEAST(first_seen_at,$3),last_seen_at=GREATEST(last_seen_at,$3) WHERE scope=$1 AND id=$2`, s.scopeKey, string(groupID), occurred); e != nil {
			return storageError(e)
		}
		for _, p := range events {
			raw, _ := json.Marshal(p.event)
			if _, e := tx.ExecContext(ctx, `INSERT INTO `+s.table+`events(scope,id,payload) VALUES($1,$2,$3)`, s.scopeKey, string(p.event.ID), string(raw)); e != nil {
				return storageError(e)
			}
			for _, h := range p.registrations {
				state, code := DeliveryPending, ""
				if contains(lineage.HandlerPath, h.Registration.ID) {
					state = DeliveryBlocked
					code = "causation_loop"
				}
				policy, _ := json.Marshal(h.Registration.Delivery)
				id := hash([]any{s.scopeKey, p.event.ID, h.Registration.ID})
				if _, e := tx.ExecContext(ctx, `INSERT INTO `+s.table+`deliveries(scope,id,event_id,handler_id,state,next_attempt_at,created_at,last_error_code,policy) VALUES($1,$2,$3,$4,$5,$6,$6,$7,$8)`, s.scopeKey, id, string(p.event.ID), string(h.Registration.ID), string(state), now, code, string(policy)); e != nil {
					return storageError(e)
				}
			}
		}
		receipt.Disposition = RecordStored
		return nil
	})
	if err != nil {
		return RecordReceipt{}, err
	}
	return receipt, nil
}
func ptrFacts(f Facts) *Facts { return &f }
func contains[T comparable](values []T, v T) bool {
	for _, item := range values {
		if item == v {
			return true
		}
	}
	return false
}
func matches(filter EventFilter, event Event) bool {
	if !filter.AllKinds && !contains(filter.Kinds, event.Kind) {
		return false
	}
	f := event.Facts
	if f == nil {
		return false
	}
	if len(filter.Sources) > 0 && !contains(filter.Sources, f.Source) || len(filter.Natures) > 0 && !contains(filter.Natures, f.Nature) || len(filter.Categories) > 0 && !contains(filter.Categories, f.Category) || len(filter.Codes) > 0 && !contains(filter.Codes, f.Code) || len(filter.Severities) > 0 && !contains(filter.Severities, f.Severity) {
		return false
	}
	return filter.Retryable == nil || f.Retryable != nil && *filter.Retryable == *f.Retryable
}

func (s *serviceImpl) existingRecord(ctx context.Context, tx dbsource.Transaction, id RecordID, digest string) (*RecordReceipt, error) {
	var row struct {
		Digest     string    `db:"digest"`
		GroupID    string    `db:"group_id"`
		RecordedAt time.Time `db:"recorded_at"`
	}
	found, err := tx.QueryRowContext(ctx, &row, `SELECT digest,group_id,recorded_at FROM `+s.table+`records WHERE scope=$1 AND id=$2`, s.scopeKey, string(id))
	if err != nil {
		return nil, storageError(err)
	}
	if !found {
		return nil, nil
	}
	if row.Digest != digest {
		return nil, ErrConflict
	}
	return &RecordReceipt{Disposition: RecordDuplicate, RecordID: id, GroupID: GroupID(row.GroupID), CommittedAt: row.RecordedAt}, nil
}

func validateInput(in RecordInput) error {
	if !safeID(string(in.ID)) || len(in.GroupKey) > 512 || len(in.Facts.Attributes) > 128 || len(in.Facts.Related) > 128 {
		return ErrInvalid
	}
	if !in.OccurredAt.IsZero() && (in.OccurredAt.UTC().Year() < 1 || in.OccurredAt.UTC().Year() > 9999) {
		return ErrInvalid
	}
	f := in.Facts
	total := 0
	add := func(value string, limit int) bool {
		total += len(value)
		return len(value) <= limit && total <= 2*1024*1024
	}
	for _, v := range []string{f.Source, string(f.Nature), f.Category, string(f.Severity), f.Component, f.Operation, f.Code, f.RequestID, f.TraceID, f.Subject.Kind, f.Subject.ID} {
		if !add(v, 4096) {
			return ErrInvalid
		}
	}
	for _, v := range []string{f.Summary, f.Detail, f.Stack} {
		if !add(v, 1024*1024) {
			return ErrInvalid
		}
	}
	for _, ref := range f.Related {
		if !add(ref.Kind, 4096) || !add(ref.ID, 4096) {
			return ErrInvalid
		}
	}
	for key, value := range f.Attributes {
		if !add(key, 256) || !add(value, 16384) {
			return ErrInvalid
		}
	}
	return nil
}

func safeID(id string) bool { return opaqueID.MatchString(id) && redactString(id, len(id)) == id }
