package embeddedmqservice

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func newQueueStateCoverageConsumer(t *testing.T) *queueConsumer {
	t.Helper()
	consumer := &queueConsumer{
		config: ConsumerConfig{
			Name: "state-coverage", QueueName: "state-coverage", Directory: t.TempDir(),
			QueueCount: 1, Concurrency: 1, LeaseTimeout: time.Minute,
			RetryDelay: time.Second, MaxRetryDelay: 4 * time.Second, MaxAttempts: 3,
		},
		activeMessages: map[string]struct{}{},
		activeShards:   map[int]struct{}{},
		queuedKeys:     map[string]struct{}{},
	}
	if err := consumer.ensureLayout(); err != nil {
		t.Fatal(err)
	}
	return consumer
}

func TestQueueClaimSkipsAndDeadLettersInvalidMessages(t *testing.T) {
	consumer := newQueueStateCoverageConsumer(t)
	pending := consumer.pendingDir(0)
	if err := os.WriteFile(filepath.Join(pending, "01-corrupt.json"), []byte("{"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := writeQueueEnvelope(filepath.Join(pending, "02-empty.json"), queueEnvelope{MessageID: "empty", DedupKey: "empty"}); err != nil {
		t.Fatal(err)
	}
	future := queueEnvelope{MessageID: "future", Payload: []byte("future"), AvailableAt: time.Now().Add(time.Hour).UnixMilli()}
	if err := writeQueueEnvelope(filepath.Join(pending, "03-future.json"), future); err != nil {
		t.Fatal(err)
	}
	ready := queueEnvelope{Payload: []byte("ready"), AvailableAt: 1}
	if err := writeQueueEnvelope(filepath.Join(pending, "04-ready.json"), ready); err != nil {
		t.Fatal(err)
	}

	item, ok, err := consumer.claimNextFromShard(0)
	if err != nil || !ok || item == nil || item.envelope.MessageID != "04-ready" || item.envelope.Attempts != 1 {
		t.Fatalf("claimed=%+v ok=%v err=%v", item, ok, err)
	}
	if countQueueFiles(consumer.deadDir(0)) != 2 || countQueueFiles(consumer.pendingDir(0)) != 1 {
		t.Fatalf("dead=%d pending=%d", countQueueFiles(consumer.deadDir(0)), countQueueFiles(consumer.pendingDir(0)))
	}
	consumer.releaseActive(item.envelope.MessageID, item.shard)
	if err = consumer.ack(item); err != nil {
		t.Fatal(err)
	}
	if item, ok, err = consumer.claimNextFromShard(0); err != nil || ok || item != nil {
		t.Fatalf("future claim=%+v ok=%v err=%v", item, ok, err)
	}
}

func TestQueueRecoverInflightLeaseAndAttemptBoundaries(t *testing.T) {
	consumer := newQueueStateCoverageConsumer(t)
	inflight := consumer.inflightDir(0)
	now := time.Now().UnixMilli()
	write := func(name string, envelope queueEnvelope) {
		t.Helper()
		if err := writeQueueEnvelope(filepath.Join(inflight, name+queueFileExt), envelope); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(inflight, "corrupt.json"), []byte("not-json"), 0o600); err != nil {
		t.Fatal(err)
	}
	write("leased", queueEnvelope{MessageID: "leased", Payload: []byte("leased"), LeaseExpiresAt: now + 60_000})
	write("active", queueEnvelope{MessageID: "active", Payload: []byte("active"), LeaseExpiresAt: now - 1})
	consumer.activeMessages["active"] = struct{}{}
	write("exhausted", queueEnvelope{MessageID: "exhausted", DedupKey: "exhausted", Payload: []byte("failed"), Attempts: consumer.config.MaxAttempts, LeaseExpiresAt: now - 1})
	consumer.markQueued("exhausted")
	write("recover", queueEnvelope{MessageID: "recover", Payload: []byte("recover"), Attempts: 1, AvailableAt: 1, LeaseExpiresAt: now - 1})

	if err := consumer.recoverInflight(false); err != nil {
		t.Fatal(err)
	}
	if countQueueFiles(consumer.deadDir(0)) != 2 || countQueueFiles(consumer.pendingDir(0)) != 1 || countQueueFiles(inflight) != 2 {
		t.Fatalf("dead=%d pending=%d inflight=%d", countQueueFiles(consumer.deadDir(0)), countQueueFiles(consumer.pendingDir(0)), countQueueFiles(inflight))
	}
	if !consumer.markQueued("exhausted") {
		t.Fatal("dead-letter dedup key was not released")
	}
	if err := consumer.recoverInflight(true); err != nil {
		t.Fatal(err)
	}
	if countQueueFiles(inflight) != 0 || countQueueFiles(consumer.pendingDir(0)) != 3 {
		t.Fatalf("forced recovery inflight=%d pending=%d", countQueueFiles(inflight), countQueueFiles(consumer.pendingDir(0)))
	}
}

func TestQueueStateHelperAndFilesystemErrorBoundaries(t *testing.T) {
	consumer := newQueueStateCoverageConsumer(t)
	if got := consumer.retryDelay(1); got != time.Second {
		t.Fatalf("retry delay=%s", got)
	}
	if got := consumer.retryDelay(3); got != 4*time.Second {
		t.Fatalf("capped retry delay=%s", got)
	}
	consumer.config.RetryDelay = 10 * time.Second
	if got := consumer.retryDelay(1); got != consumer.config.MaxRetryDelay {
		t.Fatalf("direct capped retry delay=%s", got)
	}
	if consumer.isShardActiveLocked(0) {
		t.Fatal("new active shard map should be empty")
	}
	consumer.activeShards[0] = struct{}{}
	if !consumer.isShardActiveLocked(0) {
		t.Fatal("active shard was not detected")
	}

	handler := &queueTestHandler{}
	for _, input := range []struct {
		config  ConsumerConfig
		handler QueueHandler
	}{
		{config: ConsumerConfig{Directory: t.TempDir()}, handler: handler},
		{config: ConsumerConfig{Name: "missing-directory"}, handler: handler},
		{config: ConsumerConfig{Name: "missing-handler", Directory: t.TempDir()}},
	} {
		if err := validateConsumerConfig(input.config, input.handler); err == nil {
			t.Fatal("invalid consumer config accepted")
		}
	}

	missing := filepath.Join(t.TempDir(), "missing.json")
	if err := consumer.moveCorruptToDead(0, missing, "missing.json", errors.New("corrupt")); err != nil {
		t.Fatalf("missing corrupt source should still create dead letter: %v", err)
	}
	directoryTarget := t.TempDir()
	if err := writeQueueEnvelope(directoryTarget, queueEnvelope{MessageID: "rename-failure"}); err == nil {
		t.Fatal("renaming an envelope over a directory must fail")
	}
	if err := renameQueueFile(filepath.Join(t.TempDir(), "missing"), filepath.Join(t.TempDir(), "target")); err == nil {
		t.Fatal("missing rename source must fail")
	}

	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	impl := &serviceImpl{consumers: map[string]*queueConsumer{}}
	if err := impl.Enqueue(canceled, EnqueueInput{ConsumerName: "missing", Payload: []byte("payload")}); err == nil {
		t.Fatal("missing consumer must fail")
	}
}

func TestQueueServiceLifecycleAndEnqueueFailureBoundaries(t *testing.T) {
	consumer := newQueueStateCoverageConsumer(t)
	consumer.stopCh = make(chan struct{})
	consumer.doneCh = make(chan struct{})
	close(consumer.doneCh)
	impl := &serviceImpl{consumers: map[string]*queueConsumer{"state": consumer}}
	if err := impl.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := impl.Enqueue(context.Background(), EnqueueInput{ConsumerName: "state"}); err == nil {
		t.Fatal("empty payload must fail")
	}
	if err := impl.Enqueue(context.Background(), EnqueueInput{ConsumerName: "state", DedupKey: "duplicate", Payload: []byte("one")}); err != nil {
		t.Fatal(err)
	}
	if err := impl.Enqueue(context.Background(), EnqueueInput{ConsumerName: "state", DedupKey: "duplicate", Payload: []byte("two")}); !errors.Is(err, ErrDuplicate) {
		t.Fatalf("duplicate error=%v", err)
	}
	if err := impl.Stop(context.Background()); err != nil || len(impl.consumers) != 0 {
		t.Fatalf("stop error=%v consumers=%d", err, len(impl.consumers))
	}
	if err := impl.Stop(context.Background()); err != nil {
		t.Fatal(err)
	}

	blocked := filepath.Join(t.TempDir(), "blocked")
	if err := os.WriteFile(blocked, []byte("file"), 0o600); err != nil {
		t.Fatal(err)
	}
	badConsumer := newQueueStateCoverageConsumer(t)
	badConsumer.config.Directory = blocked
	badImpl := &serviceImpl{consumers: map[string]*queueConsumer{"bad": badConsumer}}
	if err := badImpl.Enqueue(context.Background(), EnqueueInput{ConsumerName: "bad", DedupKey: "retryable", Payload: []byte("payload")}); err == nil {
		t.Fatal("unwritable queue directory must fail")
	}
	if !badConsumer.markQueued("retryable") {
		t.Fatal("failed enqueue must release the dedup key")
	}
}
