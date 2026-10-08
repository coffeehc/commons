package logservice_test

import (
	"context"
	"time"

	"github.com/coffeehc/commons/dbsource"
	"github.com/coffeehc/commons/logservice"
)

// This compile-checked example leaves data source lifecycle and business effects
// in the host. It is not executed without an explicitly supplied PostgreSQL DB.
func ExampleNew() {
	start := func(ctx context.Context, db dbsource.Service) (logservice.Service, error) {
		b, err := logservice.New(logservice.Options{DataSource: db, Config: logservice.Config{Scope: logservice.Scope{Application: "app", Environment: "test", Tenant: "default"}}})
		if err != nil {
			return nil, err
		}
		err = b.RegisterHandler(logservice.HandlerRegistration{ID: "observer-v1", Filter: logservice.EventFilter{Kinds: []logservice.EventKind{logservice.EventErrorRecorded}}, Delivery: logservice.DeliveryPolicy{Timeout: time.Second}}, logservice.HandlerFunc(func(ctx context.Context, d logservice.Delivery) error {
			// Send only to an explicitly configured target. Use d.IdempotencyKey there.
			return nil
		}))
		if err != nil {
			return nil, err
		}
		return b.Start(ctx)
	}
	_ = start
}
