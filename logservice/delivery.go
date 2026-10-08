package logservice

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/coffeehc/commons/dbsource"
	"github.com/google/uuid"
)

type claimRow struct {
	ID        string    `db:"id"`
	EventID   string    `db:"event_id"`
	Attempts  uint32    `db:"attempts"`
	CreatedAt time.Time `db:"created_at"`
	Payload   string    `db:"payload"`
	Policy    string    `db:"policy"`
	Now       time.Time `db:"now"`
}
type claimed struct {
	delivery Delivery
	token    string
	created  time.Time
	policy   DeliveryPolicy
}

var errHandlerPanic = errors.New("logservice: handler panic")

func (s *serviceImpl) claim(ctx context.Context, h registeredHandler) (*claimed, error) {
	var result *claimed
	err := s.tx(ctx, nil, func(tx dbsource.Transaction) error {
		if err := s.checkRegistry(ctx, tx, "FOR SHARE"); err != nil {
			return err
		}
		var row claimRow
		found, err := tx.QueryRowContext(ctx, &row, `SELECT d.id,d.event_id,d.attempts,d.created_at,d.policy::text AS policy,e.payload::text AS payload,clock_timestamp() AS now FROM `+s.table+`deliveries d JOIN `+s.table+`events e ON e.scope=d.scope AND e.id=d.event_id WHERE d.scope=$1 AND d.handler_id=$2 AND ((d.state='pending' AND d.next_attempt_at<=clock_timestamp()) OR (d.state='running' AND d.lease_until<=clock_timestamp())) ORDER BY d.created_at,d.id FOR UPDATE OF d SKIP LOCKED LIMIT 1`, s.scopeKey, string(h.Registration.ID))
		if err != nil {
			return storageError(err)
		}
		if !found {
			return nil
		}
		var p DeliveryPolicy
		if err := json.Unmarshal([]byte(row.Policy), &p); err != nil {
			return storageError(err)
		}
		if row.Attempts >= p.MaxAttempts || !row.Now.Before(row.CreatedAt.Add(p.MaxAge)) {
			code := "attempts_exhausted"
			if !row.Now.Before(row.CreatedAt.Add(p.MaxAge)) {
				code = "delivery_expired"
			}
			_, err = tx.ExecContext(ctx, `UPDATE `+s.table+`deliveries SET state='failed',lease_until=NULL,lease_token='',last_error_code=$3,version=version+1 WHERE scope=$1 AND id=$2`, s.scopeKey, row.ID, code)
			return storageError(err)
		}
		var event Event
		if err := json.Unmarshal([]byte(row.Payload), &event); err != nil {
			return storageError(err)
		}
		token := uuid.NewString()
		_, err = tx.ExecContext(ctx, `UPDATE `+s.table+`deliveries SET state='running',attempts=attempts+1,lease_token=$3,lease_until=$4,version=version+1 WHERE scope=$1 AND id=$2`, s.scopeKey, row.ID, token, row.Now.Add(p.Timeout+time.Second))
		if err != nil {
			return storageError(err)
		}
		result = &claimed{delivery: Delivery{ID: DeliveryID(row.ID), IdempotencyKey: row.ID, Attempt: row.Attempts + 1, Event: event}, token: token, created: row.CreatedAt, policy: p}
		return nil
	})
	return result, err
}
func (s *serviceImpl) worker(h registeredHandler) {
	defer s.work.Done()
	for {
		if s.ctx.Err() != nil {
			return
		}
		ctx, cancel := context.WithTimeout(s.ctx, 5*time.Second)
		job, err := s.claim(ctx, h)
		cancel()
		if errors.Is(err, ErrConflict) {
			return
		}
		if err != nil || job == nil {
			select {
			case <-s.ctx.Done():
				return
			case <-time.After(s.options.Config.PollInterval):
				continue
			}
		}
		p := job.policy
		ctx, cancel = context.WithTimeout(s.ctx, p.Timeout)
		causal := job.delivery.Event.Causation
		causal.ParentEventID = job.delivery.Event.ID
		causal.Depth++
		causal.HandlerPath = append(append([]HandlerID(nil), causal.HandlerPath...), h.Registration.ID)
		ctx = context.WithValue(ctx, causalKey{}, causal)
		err = invoke(ctx, h.handler, job.delivery)
		if ctx.Err() != nil {
			err = ctx.Err()
		}
		cancel()
		finishCtx, finishCancel := context.WithTimeout(context.Background(), 5*time.Second)
		_ = s.finish(finishCtx, h, job, err)
		finishCancel()
	}
}
func invoke(ctx context.Context, handler EventHandler, delivery Delivery) (err error) {
	defer func() {
		if recover() != nil {
			err = errHandlerPanic
		}
	}()
	return handler.Handle(ctx, delivery)
}
func (s *serviceImpl) finish(ctx context.Context, h registeredHandler, job *claimed, handlerErr error) error {
	return s.tx(ctx, nil, func(tx dbsource.Transaction) error {
		var now time.Time
		if _, err := tx.QueryRowContext(ctx, &now, `SELECT clock_timestamp()`); err != nil {
			return storageError(err)
		}
		state, code := DeliverySucceeded, ""
		next := now
		if handlerErr != nil {
			code = handlerErrorCode(handlerErr)
			if errors.Is(handlerErr, errHandlerPanic) {
				code = "handler_panic"
			}
			if errors.Is(handlerErr, ErrPermanentHandler) || job.delivery.Attempt >= job.policy.MaxAttempts || !now.Before(job.created.Add(job.policy.MaxAge)) {
				state = DeliveryFailed
			} else {
				state = DeliveryPending
				backoff := job.policy.InitialBackoff
				for i := uint32(1); i < job.delivery.Attempt; i++ {
					if backoff >= job.policy.MaxBackoff/2 {
						backoff = job.policy.MaxBackoff
						break
					}
					backoff *= 2
				}
				if backoff > job.policy.MaxBackoff {
					backoff = job.policy.MaxBackoff
				}
				next = now.Add(backoff)
			}
		}
		// A completion from an expired/replaced worker must never overwrite a newer attempt.
		_, err := tx.ExecContext(ctx, `UPDATE `+s.table+`deliveries SET state=$4,next_attempt_at=$5,last_error_code=$6,lease_until=NULL,lease_token='',version=version+1 WHERE scope=$1 AND id=$2 AND lease_token=$3 AND state='running' AND lease_until>clock_timestamp()`, s.scopeKey, string(job.delivery.ID), job.token, string(state), next, code)
		return storageError(err)
	})
}
