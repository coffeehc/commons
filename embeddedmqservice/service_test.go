package embeddedmqservice

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/coffeehc/commons/asyncservice"
)

// cancelAwareHandler blocks until the consumer lifecycle context is canceled.
type cancelAwareHandler struct {
	// started reports that the handler has received a payload.
	started chan struct{}
}

// HandleQueuePayload waits for consumer shutdown and returns its cancellation error.
func (handler cancelAwareHandler) HandleQueuePayload(ctx context.Context, _ []byte) error {
	handler.started <- struct{}{}
	<-ctx.Done()
	return ctx.Err()
}

func TestNewServiceRejectsNilAsyncService(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("nil async service should panic")
		}
	}()
	NewService(nil)
}

func TestServiceRejectsCanceledOperations(t *testing.T) {
	async := asyncservice.NewService(t.Context(), &asyncservice.Config{PoolSize: 2})
	service := NewService(async)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	config := ConsumerConfig{Name: "canceled", Directory: t.TempDir(), QueueCount: 1}
	handler := &queueTestHandler{done: make(chan error, 1)}
	if err := service.StartConsumer(ctx, config, handler); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled start should fail with context.Canceled, got=%v", err)
	}
	if err := service.StartConsumer(t.Context(), config, handler); err != nil {
		t.Fatal(err)
	}
	defer service.StopConsumer(config.Name)
	if err := service.Enqueue(ctx, EnqueueInput{ConsumerName: config.Name, Payload: []byte("payload")}); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled enqueue should fail with context.Canceled, got=%v", err)
	}
	if depth := service.Depth(config.Name); depth != 0 {
		t.Fatalf("canceled enqueue must not persist a message, depth=%d", depth)
	}
}

func TestStopConsumerCancelsHandlerContext(t *testing.T) {
	async := asyncservice.NewService(t.Context(), &asyncservice.Config{PoolSize: 2})
	service := NewService(async)
	handler := cancelAwareHandler{started: make(chan struct{}, 1)}
	queueDir := t.TempDir()
	config := ConsumerConfig{
		Name: "cancel-handler", Directory: queueDir, QueueCount: 1, MaxAttempts: 1,
	}
	if err := service.StartConsumer(t.Context(), config, handler); err != nil {
		t.Fatal(err)
	}
	if err := service.Enqueue(t.Context(), EnqueueInput{ConsumerName: config.Name, Payload: []byte("payload")}); err != nil {
		t.Fatal(err)
	}
	select {
	case <-handler.started:
	case <-time.After(3 * time.Second):
		t.Fatal("handler did not start")
	}
	service.StopConsumer(config.Name)
	if countQueueFiles(filepath.Join(queueDir, queueDeadDirName)) != 0 {
		t.Fatal("shutdown cancellation must not move the message to dead letter")
	}
	entries, err := queueJSONEntries(filepath.Join(queueDir, queuePendingDirName))
	if err != nil || len(entries) != 1 {
		t.Fatalf("shutdown cancellation should return the message to pending: entries=%d err=%v", len(entries), err)
	}
	envelope, err := readQueueEnvelope(filepath.Join(queueDir, queuePendingDirName, entries[0].Name()))
	if err != nil {
		t.Fatal(err)
	}
	if envelope.Attempts != 0 {
		t.Fatalf("shutdown cancellation must not consume an attempt, got=%d", envelope.Attempts)
	}
}

func TestConsumerProgressesWithSingleAsyncPoolSlot(t *testing.T) {
	async := asyncservice.NewService(t.Context(), &asyncservice.Config{PoolSize: 1})
	service := NewService(async)
	handler := &queueTestHandler{done: make(chan error, 1)}
	config := ConsumerConfig{
		Name: "single-slot", Directory: t.TempDir(), QueueCount: 1,
		PollInterval: 5 * time.Millisecond,
	}
	if err := service.StartConsumer(t.Context(), config, handler); err != nil {
		t.Fatal(err)
	}
	defer service.StopConsumer(config.Name)
	if err := service.Enqueue(t.Context(), EnqueueInput{
		ConsumerName: config.Name, Payload: []byte("payload"),
	}); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-handler.done:
		if err != nil {
			t.Fatalf("single-slot handler context should be active, got=%v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("single-slot async pool did not make progress")
	}
}

func TestStartConsumerRejectsStoppedAsyncService(t *testing.T) {
	async := asyncservice.NewService(t.Context(), &asyncservice.Config{PoolSize: 1})
	stopper := async.(interface {
		Stop(context.Context) error
	})
	if err := stopper.Stop(t.Context()); err != nil {
		t.Fatal(err)
	}
	service := NewService(async)
	err := service.StartConsumer(t.Context(), ConsumerConfig{
		Name: "stopped-async", Directory: t.TempDir(), QueueCount: 1,
	}, &queueTestHandler{done: make(chan error, 1)})
	if err == nil {
		t.Fatal("stopped async service should reject consumer start")
	}
}

func TestStoppedServiceCannotRestartConsumer(t *testing.T) {
	async := asyncservice.NewService(t.Context(), &asyncservice.Config{PoolSize: 2})
	service := NewService(async)
	stopper := service.(interface {
		Stop(context.Context) error
	})
	if err := stopper.Stop(t.Context()); err != nil {
		t.Fatal(err)
	}
	err := service.StartConsumer(t.Context(), ConsumerConfig{
		Name: "stopped-mq", Directory: t.TempDir(), QueueCount: 1,
	}, &queueTestHandler{done: make(chan error, 1)})
	if !errors.Is(err, ErrServiceStopped) {
		t.Fatalf("stopped service should reject consumer start, got=%v", err)
	}
}
