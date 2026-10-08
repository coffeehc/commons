package logservice

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/coffeehc/commons/dbsource"
)

type cursor struct {
	Version       int
	Binding, Last string
	AsOf          time.Time
	Sequence      int64
}

func (s *serviceImpl) page(ctx context.Context, tx dbsource.Transaction, req PageRequest, kind string, filter any) (cursor, int, error) {
	size := req.Size
	if size == 0 {
		size = 50
	}
	if size < 1 || size > 200 || len(req.Cursor) > 4096 {
		return cursor{}, 0, ErrInvalid
	}
	binding := hash([]any{s.scopeKey, kind, filter})
	if binding == "" {
		return cursor{}, 0, ErrInvalid
	}
	c := cursor{Version: 1, Binding: binding}
	if req.Cursor != "" {
		raw, err := base64.RawURLEncoding.DecodeString(req.Cursor)
		if err != nil || json.Unmarshal(raw, &c) != nil || c.Version != 1 || c.Binding != binding || c.AsOf.IsZero() || c.Sequence < 0 || len(c.Last) > 256 {
			return c, 0, ErrInvalid
		}
	} else {
		var snapshot struct {
			AsOf     time.Time `db:"as_of"`
			Sequence int64     `db:"sequence"`
		}
		if _, err := tx.QueryRowContext(ctx, &snapshot, `SELECT clock_timestamp() AS as_of,record_sequence AS sequence FROM `+s.table+`scopes WHERE scope=$1`, s.scopeKey); err != nil {
			return c, 0, storageError(err)
		}
		c.AsOf = snapshot.AsOf
		c.Sequence = snapshot.Sequence
	}
	return c, size, nil
}
func nextPage(c cursor, last string, more bool) PageInfo {
	p := PageInfo{AsOf: c.AsOf}
	if more {
		c.Last = last
		raw, _ := json.Marshal(c)
		p.NextCursor = base64.RawURLEncoding.EncodeToString(raw)
	}
	return p
}

type where struct {
	sql  string
	args []any
}

func (w *where) add(clause string, v any) {
	w.args = append(w.args, v)
	w.sql += " AND " + fmt.Sprintf(clause, len(w.args))
}
func addValues[T ~string](w *where, field string, values []T) {
	if len(values) == 0 {
		return
	}
	parts := make([]string, 0, len(values))
	for _, v := range values {
		w.args = append(w.args, string(v))
		parts = append(parts, fmt.Sprintf("$%d", len(w.args)))
	}
	w.sql += " AND " + field + " IN (" + strings.Join(parts, ",") + ")"
}
func (s *serviceImpl) recordWhere(filter RecordFilter, asof time.Time, sequence int64) (where, error) {
	if len(filter.Text) > 512 || len(filter.Sources) > 50 || len(filter.Natures) > 50 || len(filter.Categories) > 50 || len(filter.Severities) > 50 || !filter.Since.IsZero() && !filter.Until.IsZero() && !filter.Since.Before(filter.Until) {
		return where{}, ErrInvalid
	}
	w := where{sql: "scope=$1 AND recorded_at<=$2 AND sequence<=$3", args: []any{s.scopeKey, asof, sequence}}
	if !filter.Since.IsZero() {
		w.add("occurred_at>=$%d", filter.Since)
	}
	if !filter.Until.IsZero() {
		w.add("occurred_at<$%d", filter.Until)
	}
	addValues(&w, `facts->>'Source'`, filter.Sources)
	addValues(&w, `facts->>'Nature'`, filter.Natures)
	addValues(&w, `facts->>'Category'`, filter.Categories)
	addValues(&w, `facts->>'Severity'`, filter.Severities)
	if filter.Component != "" {
		w.add(`facts->>'Component'=$%d`, filter.Component)
	}
	if filter.Code != "" {
		w.add(`facts->>'Code'=$%d`, filter.Code)
	}
	if filter.Subject != nil {
		w.add(`facts->'Subject'->>'Kind'=$%d`, filter.Subject.Kind)
		w.add(`facts->'Subject'->>'ID'=$%d`, filter.Subject.ID)
	}
	if filter.Text != "" {
		w.add(`strpos(lower(concat_ws(' ',facts->>'Summary',facts->>'Source',facts->>'Category',facts->>'Component',facts->>'Code')),lower($%d))>0`, filter.Text)
	}
	return w, nil
}

type recordRow struct {
	ID              string    `db:"id"`
	GroupID         string    `db:"group_id"`
	OccurredAt      time.Time `db:"occurred_at"`
	RecordedAt      time.Time `db:"recorded_at"`
	Facts           string    `db:"facts"`
	DetailsRetained bool      `db:"details_retained"`
}

func decodeRecord(row recordRow) (ErrorRecord, error) {
	r := ErrorRecord{ID: RecordID(row.ID), GroupID: GroupID(row.GroupID), OccurredAt: row.OccurredAt, RecordedAt: row.RecordedAt, DetailsRetained: row.DetailsRetained}
	err := json.Unmarshal([]byte(row.Facts), &r.Facts)
	return r, err
}

const recordColumns = `id,group_id,occurred_at,recorded_at,(CASE WHEN details_retained AND details_expire_at>clock_timestamp() THEN facts ELSE facts - 'Detail' - 'Stack' END)::text AS facts,(details_retained AND details_expire_at>clock_timestamp()) AS details_retained`

func (s *serviceImpl) GetRecord(ctx context.Context, id RecordID) (ErrorRecord, error) {
	var result ErrorRecord
	if ctx == nil || id == "" {
		return result, ErrInvalid
	}
	if err := s.beginOperation(); err != nil {
		return result, err
	}
	defer s.work.Done()
	err := s.tx(ctx, &sql.TxOptions{ReadOnly: true}, func(tx dbsource.Transaction) error {
		var row recordRow
		found, err := tx.QueryRowContext(ctx, &row, `SELECT `+recordColumns+` FROM `+s.table+`records WHERE scope=$1 AND id=$2`, s.scopeKey, string(id))
		if err != nil {
			return storageError(err)
		}
		if !found {
			return ErrNotFound
		}
		result, err = decodeRecord(row)
		return storageError(err)
	})
	return result, err
}
func (s *serviceImpl) QueryRecords(ctx context.Context, req QueryRequest) (RecordPage, error) {
	var result RecordPage
	if ctx == nil {
		return result, ErrInvalid
	}
	if err := s.beginOperation(); err != nil {
		return result, err
	}
	defer s.work.Done()
	err := s.tx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true}, func(tx dbsource.Transaction) error {
		c, size, err := s.page(ctx, tx, req.Page, "records", req.Filter)
		if err != nil {
			return err
		}
		result.Items, result.Page, err = s.records(ctx, tx, req.Filter, c, size, "")
		return err
	})
	return result, err
}
func (s *serviceImpl) records(ctx context.Context, tx dbsource.Transaction, filter RecordFilter, c cursor, size int, groupID GroupID) ([]ErrorRecord, PageInfo, error) {
	w, err := s.recordWhere(filter, c.AsOf, c.Sequence)
	if err != nil {
		return nil, PageInfo{}, err
	}
	if groupID != "" {
		w.add("group_id=$%d", string(groupID))
	}
	if c.Last != "" {
		w.add("id>$%d", c.Last)
	}
	w.args = append(w.args, size+1)
	var rows []recordRow
	if err := tx.QueryContext(ctx, &rows, `SELECT `+recordColumns+` FROM `+s.table+`records WHERE `+w.sql+fmt.Sprintf(" ORDER BY id LIMIT $%d", len(w.args)), w.args...); err != nil {
		return nil, PageInfo{}, storageError(err)
	}
	more := len(rows) > size
	if more {
		rows = rows[:size]
	}
	items := make([]ErrorRecord, 0, len(rows))
	last := ""
	for _, row := range rows {
		r, err := decodeRecord(row)
		if err != nil {
			return nil, PageInfo{}, storageError(err)
		}
		items = append(items, r)
		last = row.ID
	}
	return items, nextPage(c, last, more), nil
}

type groupRow struct {
	ID                string    `db:"id"`
	KeyVersion        string    `db:"key_version"`
	FirstSeenAt       time.Time `db:"first_seen_at"`
	LastSeenAt        time.Time `db:"last_seen_at"`
	EntryCount        uint64    `db:"entry_count"`
	Representative    string    `db:"representative"`
	MatchedEntryCount uint64    `db:"matched_entry_count"`
}

func decodeGroup(row groupRow) (ErrorGroup, error) {
	g := ErrorGroup{ID: GroupID(row.ID), KeyVersion: row.KeyVersion, FirstSeenAt: row.FirstSeenAt, LastSeenAt: row.LastSeenAt, EntryCount: row.EntryCount}
	err := json.Unmarshal([]byte(row.Representative), &g.Representative)
	return g, err
}
func (s *serviceImpl) Query(ctx context.Context, req QueryRequest) (QueryResult, error) {
	var result QueryResult
	if ctx == nil {
		return result, ErrInvalid
	}
	if err := s.beginOperation(); err != nil {
		return result, err
	}
	defer s.work.Done()
	err := s.tx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true}, func(tx dbsource.Transaction) error {
		c, size, err := s.page(ctx, tx, req.Page, "groups", req.Filter)
		if err != nil {
			return err
		}
		w, err := s.recordWhere(req.Filter, c.AsOf, c.Sequence)
		if err != nil {
			return err
		}
		var counts struct {
			Groups  uint64 `db:"groups"`
			Entries uint64 `db:"entries"`
		}
		if _, err := tx.QueryRowContext(ctx, &counts, `SELECT count(DISTINCT group_id) AS groups,count(*) AS entries FROM `+s.table+`records WHERE `+w.sql, w.args...); err != nil {
			return storageError(err)
		}
		result.MatchedGroups = counts.Groups
		result.MatchedEntries = counts.Entries
		w.args = append(w.args, c.Last, size+1)
		query := `WITH matched AS (SELECT group_id,count(*) AS matched_entry_count FROM ` + s.table + `records WHERE ` + w.sql + ` GROUP BY group_id), totals AS (SELECT group_id,min(occurred_at) AS first_seen_at,max(occurred_at) AS last_seen_at,count(*) AS entry_count FROM ` + s.table + `records WHERE scope=$1 AND recorded_at<=$2 AND sequence<=$3 GROUP BY group_id) SELECT g.id,g.key_version,t.first_seen_at,t.last_seen_at,t.entry_count,g.representative::text AS representative,m.matched_entry_count FROM matched m JOIN ` + s.table + `groups g ON g.scope=$1 AND g.id=m.group_id JOIN totals t ON t.group_id=g.id WHERE g.id>$` + fmt.Sprint(len(w.args)-1) + ` ORDER BY g.id LIMIT $` + fmt.Sprint(len(w.args))
		var rows []groupRow
		if err := tx.QueryContext(ctx, &rows, query, w.args...); err != nil {
			return storageError(err)
		}
		more := len(rows) > size
		if more {
			rows = rows[:size]
		}
		result.Items = make([]GroupRow, 0, len(rows))
		last := ""
		for _, row := range rows {
			g, err := decodeGroup(row)
			if err != nil {
				return storageError(err)
			}
			result.Items = append(result.Items, GroupRow{Group: g, MatchedEntryCount: row.MatchedEntryCount})
			last = row.ID
		}
		result.Page = nextPage(c, last, more)
		return nil
	})
	return result, err
}
func (s *serviceImpl) Detail(ctx context.Context, req DetailRequest) (DetailResult, error) {
	var result DetailResult
	if ctx == nil || req.GroupID == "" {
		return result, ErrInvalid
	}
	if err := s.beginOperation(); err != nil {
		return result, err
	}
	defer s.work.Done()
	err := s.tx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true}, func(tx dbsource.Transaction) error {
		c, size, err := s.page(ctx, tx, req.Page, "detail", []any{req.GroupID, req.Filter})
		if err != nil {
			return err
		}
		var row groupRow
		found, err := tx.QueryRowContext(ctx, &row, `SELECT g.id,g.key_version,min(r.occurred_at) AS first_seen_at,max(r.occurred_at) AS last_seen_at,count(*) AS entry_count,g.representative::text AS representative,0 AS matched_entry_count FROM `+s.table+`groups g JOIN `+s.table+`records r ON r.scope=g.scope AND r.group_id=g.id WHERE g.scope=$1 AND g.id=$2 AND r.recorded_at<=$3 AND r.sequence<=$4 GROUP BY g.id,g.key_version,g.representative`, s.scopeKey, string(req.GroupID), c.AsOf, c.Sequence)
		if err != nil {
			return storageError(err)
		}
		if !found {
			return ErrNotFound
		}
		result.Group, err = decodeGroup(row)
		if err != nil {
			return storageError(err)
		}
		result.Records, result.Page, err = s.records(ctx, tx, req.Filter, c, size, req.GroupID)
		return err
	})
	return result, err
}

type deliveryRow struct {
	ID            string    `db:"id"`
	EventID       string    `db:"event_id"`
	HandlerID     string    `db:"handler_id"`
	State         string    `db:"state"`
	Attempts      uint32    `db:"attempts"`
	NextAttemptAt time.Time `db:"next_attempt_at"`
	LastErrorCode string    `db:"last_error_code"`
	Version       uint64    `db:"version"`
}

func (s *serviceImpl) QueryDeliveries(ctx context.Context, req DeliveryQuery) (DeliveryPage, error) {
	var result DeliveryPage
	if ctx == nil {
		return result, ErrInvalid
	}
	if err := s.beginOperation(); err != nil {
		return result, err
	}
	defer s.work.Done()
	err := s.tx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true}, func(tx dbsource.Transaction) error {
		c, size, err := s.page(ctx, tx, req.Page, "deliveries", struct {
			Handler HandlerID
			Event   EventID
			States  []DeliveryState
		}{req.HandlerID, req.EventID, req.States})
		if err != nil {
			return err
		}
		w := where{sql: "scope=$1 AND created_at<=$2", args: []any{s.scopeKey, c.AsOf}}
		if req.HandlerID != "" {
			w.add("handler_id=$%d", string(req.HandlerID))
		}
		if req.EventID != "" {
			w.add("event_id=$%d", string(req.EventID))
		}
		if len(req.States) > 5 {
			return ErrInvalid
		}
		for _, state := range req.States {
			switch state {
			case DeliveryPending, DeliveryRunning, DeliverySucceeded, DeliveryFailed, DeliveryBlocked:
			default:
				return ErrInvalid
			}
		}
		addValues(&w, "state", req.States)
		w.add("id>$%d", c.Last)
		w.args = append(w.args, size+1)
		var rows []deliveryRow
		if err := tx.QueryContext(ctx, &rows, `SELECT id,event_id,handler_id,state,attempts,next_attempt_at,last_error_code,version FROM `+s.table+`deliveries WHERE `+w.sql+fmt.Sprintf(" ORDER BY id LIMIT $%d", len(w.args)), w.args...); err != nil {
			return storageError(err)
		}
		more := len(rows) > size
		if more {
			rows = rows[:size]
		}
		result.Items = make([]DeliveryRecord, 0, len(rows))
		last := ""
		for _, row := range rows {
			result.Items = append(result.Items, DeliveryRecord{ID: DeliveryID(row.ID), EventID: EventID(row.EventID), HandlerID: HandlerID(row.HandlerID), State: DeliveryState(row.State), Attempts: row.Attempts, NextAttemptAt: row.NextAttemptAt, LastErrorCode: row.LastErrorCode, Version: row.Version})
			last = row.ID
		}
		result.Page = nextPage(c, last, more)
		return nil
	})
	return result, err
}
