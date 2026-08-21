package embeddedmqservice

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/coffeehc/commons/asyncservice"
)

func TestQueueConsumerDedupKeys(t *testing.T) {
	consumer := &queueConsumer{
		queuedKeys:     map[string]struct{}{},
		activeMessages: map[string]struct{}{},
	}

	if !consumer.markQueued("job-1") {
		t.Fatalf("首次登记队列去重键应成功")
	}
	if consumer.markQueued("job-1") {
		t.Fatalf("重复登记队列去重键不应成功")
	}
	consumer.unmarkQueued("job-1")
	if !consumer.markQueued("job-1") {
		t.Fatalf("释放后再次登记队列去重键应成功")
	}
}

func TestNormalizeConsumerConfigUsesInternalShardDefault(t *testing.T) {
	config := ConsumerConfig{
		Name:        "default-shards",
		Directory:   t.TempDir(),
		Concurrency: 4,
	}
	normalizeConsumerConfig(&config)
	if config.QueueCount != defaultQueueShardCount {
		t.Fatalf("默认分片数应为内部常量，got=%d", config.QueueCount)
	}
	if config.Concurrency != 4 {
		t.Fatalf("Concurrency 不应反向压低分片数或被覆盖，got=%d", config.Concurrency)
	}
}

func TestQueueConsumerPurgePayloadsFiltersBusinessPayload(t *testing.T) {
	consumer := &queueConsumer{
		config: ConsumerConfig{
			Name: "purge-test", QueueName: "purge-test", Directory: t.TempDir(), QueueCount: 2,
		},
		queuedKeys: map[string]struct{}{},
	}
	normalizeConsumerConfig(&consumer.config)
	if err := consumer.ensureLayout(); err != nil {
		t.Fatalf("初始化队列目录失败: %v", err)
	}
	for index, payload := range [][]byte{
		[]byte(`{"user_id":7,"value":"remove"}`),
		[]byte(`{"user_id":8,"value":"keep"}`),
	} {
		envelope := queueEnvelope{
			MessageID: fmt.Sprintf("message-%d", index),
			Payload:   payload, CreatedAt: 1, UpdatedAt: 1, AvailableAt: 1,
		}
		if err := consumer.writePending(index, envelope); err != nil {
			t.Fatalf("写入测试 payload 失败: %v", err)
		}
	}
	deleted, err := consumer.purgePayloads(func(payload []byte) (bool, error) {
		value := struct {
			UserID int64 `json:"user_id"`
		}{}
		if unmarshalErr := json.Unmarshal(payload, &value); unmarshalErr != nil {
			return false, unmarshalErr
		}
		return value.UserID == 7, nil
	})
	if err != nil {
		t.Fatalf("清理队列 payload 失败: %v", err)
	}
	if deleted != 1 {
		t.Fatalf("应只删除目标用户 payload，got=%d", deleted)
	}
	remaining := countQueueFiles(consumer.pendingDir(0)) + countQueueFiles(consumer.pendingDir(1))
	if remaining != 1 {
		t.Fatalf("其他用户 payload 必须保留，remaining=%d", remaining)
	}
}

func TestServicePurgePayloadsRestartsConsumer(t *testing.T) {
	async := asyncservice.NewService(context.Background(), &asyncservice.Config{PoolSize: 2})
	impl := &serviceImpl{asyncService: async, consumers: map[string]*queueConsumer{}}
	handler := &queueTestHandler{done: make(chan error, 2)}
	config := ConsumerConfig{
		Name: "service-purge", QueueName: "service-purge", Directory: t.TempDir(),
		Concurrency: 1, QueueCount: 1, PollInterval: time.Hour,
	}
	if err := impl.StartConsumer(context.Background(), config, handler); err != nil {
		t.Fatal(err)
	}
	defer impl.StopConsumer(config.Name)
	time.Sleep(20 * time.Millisecond)
	for _, input := range []EnqueueInput{
		{ConsumerName: config.Name, DedupKey: "remove", Payload: []byte(`{"remove":true}`)},
		{ConsumerName: config.Name, DedupKey: "keep", Payload: []byte(`{"remove":false}`)},
	} {
		if err := impl.Enqueue(context.Background(), input); err != nil {
			t.Fatal(err)
		}
	}
	deleted, err := impl.PurgePayloads(context.Background(), config.Name, func(payload []byte) (bool, error) {
		return string(payload) == `{"remove":true}`, nil
	})
	if err != nil || deleted != 1 {
		t.Fatalf("purge payloads: deleted=%d err=%v", deleted, err)
	}
	if impl.getConsumer(config.Name) == nil || impl.Depth(config.Name) != 1 {
		t.Fatalf("consumer was not restarted or retained depth is wrong: consumer=%+v depth=%d", impl.getConsumer(config.Name), impl.Depth(config.Name))
	}
	if err = impl.Enqueue(context.Background(), EnqueueInput{ConsumerName: config.Name, DedupKey: "remove", Payload: []byte("new")}); err != nil {
		t.Fatalf("deleted dedup key should be reusable: %v", err)
	}
}

func TestServicePurgePayloadsValidation(t *testing.T) {
	impl := &serviceImpl{consumers: map[string]*queueConsumer{}}
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := impl.PurgePayloads(canceled, "queue", func([]byte) (bool, error) { return false, nil }); err == nil {
		t.Fatal("canceled context should fail")
	}
	if _, err := impl.PurgePayloads(context.Background(), "", func([]byte) (bool, error) { return false, nil }); err == nil {
		t.Fatal("empty consumer name should fail")
	}
	if _, err := impl.PurgePayloads(context.Background(), "queue", nil); err == nil {
		t.Fatal("nil predicate should fail")
	}
	if _, err := impl.PurgePayloads(context.Background(), "missing", func([]byte) (bool, error) { return false, nil }); err == nil {
		t.Fatal("missing consumer should fail")
	}
}

func TestPurgeDeadLettersKeepsPendingAndInflight(t *testing.T) {
	now := time.Date(2026, 8, 9, 12, 0, 0, 0, time.UTC)
	consumer := &queueConsumer{
		config:     ConsumerConfig{Name: "cleanup", QueueName: "cleanup", Directory: t.TempDir(), QueueCount: 1},
		queuedKeys: map[string]struct{}{}, activeMessages: map[string]struct{}{},
	}
	normalizeConsumerConfig(&consumer.config)
	if err := consumer.ensureLayout(); err != nil {
		t.Fatal(err)
	}
	for name, updatedAt := range map[string]time.Time{
		"old": now.Add(-31 * 24 * time.Hour), "recent-1": now.Add(-time.Hour), "recent-2": now.Add(-2 * time.Hour),
	} {
		if err := writeQueueEnvelope(filepath.Join(consumer.deadDir(0), name+queueFileExt), queueEnvelope{
			MessageID: name, Payload: []byte(name), UpdatedAt: updatedAt.UnixMilli(),
		}); err != nil {
			t.Fatal(err)
		}
	}
	if err := writeQueueEnvelope(filepath.Join(consumer.pendingDir(0), "pending"+queueFileExt), queueEnvelope{MessageID: "pending"}); err != nil {
		t.Fatal(err)
	}
	if err := writeQueueEnvelope(filepath.Join(consumer.inflightDir(0), "inflight"+queueFileExt), queueEnvelope{MessageID: "inflight"}); err != nil {
		t.Fatal(err)
	}
	impl := &serviceImpl{consumers: map[string]*queueConsumer{consumer.config.Name: consumer}}
	result, err := impl.PurgeDeadLetters(context.Background(), DeadLetterCleanupPolicy{
		Cutoff: now.Add(-30 * 24 * time.Hour), MaxEntries: 2, MaxBytes: 1 << 20,
	})
	if err != nil || result.RemovedEntries != 1 {
		t.Fatalf("normal cleanup result=%+v err=%v", result, err)
	}
	if countQueueFiles(consumer.deadDir(0)) != 2 || countQueueFiles(consumer.pendingDir(0)) != 1 || countQueueFiles(consumer.inflightDir(0)) != 1 {
		t.Fatalf("normal cleanup crossed queue states: dead=%d pending=%d inflight=%d",
			countQueueFiles(consumer.deadDir(0)), countQueueFiles(consumer.pendingDir(0)), countQueueFiles(consumer.inflightDir(0)))
	}
	result, err = impl.PurgeDeadLetters(context.Background(), DeadLetterCleanupPolicy{Cutoff: now, MaxEntries: 0, MaxBytes: 0})
	if err != nil || result.RemovedEntries != 2 || countQueueFiles(consumer.deadDir(0)) != 0 {
		t.Fatalf("hard cleanup result=%+v dead=%d err=%v", result, countQueueFiles(consumer.deadDir(0)), err)
	}
	if countQueueFiles(consumer.pendingDir(0)) != 1 || countQueueFiles(consumer.inflightDir(0)) != 1 {
		t.Fatal("hard cleanup must not remove pending or inflight")
	}
}

func TestPurgeDeadLettersValidatesPolicy(t *testing.T) {
	impl := &serviceImpl{consumers: map[string]*queueConsumer{}}
	if _, err := impl.PurgeDeadLetters(context.Background(), DeadLetterCleanupPolicy{}); err == nil {
		t.Fatal("empty policy should fail")
	}
	if _, err := impl.PurgeDeadLetters(context.Background(), DeadLetterCleanupPolicy{Cutoff: time.Now(), MaxEntries: -1}); err == nil {
		t.Fatal("negative cap should fail")
	}
}

type queueTestHandler struct {
	done      chan error
	failures  int
	payloads  [][]byte
	payloadMu sync.Mutex
}

func (handler *queueTestHandler) HandleQueuePayload(ctx context.Context, payload []byte) error {
	handler.payloadMu.Lock()
	handler.payloads = append(handler.payloads, append([]byte(nil), payload...))
	failures := handler.failures
	if handler.failures > 0 {
		handler.failures--
	}
	handler.payloadMu.Unlock()
	if failures > 0 {
		return errors.New("planned failure")
	}
	handler.done <- ctx.Err()
	return nil
}

func TestStartConsumerDoesNotInheritCallerCancel(t *testing.T) {
	async := asyncservice.NewService(context.Background(), &asyncservice.Config{PoolSize: 2})
	impl := &serviceImpl{
		asyncService: async,
		consumers:    map[string]*queueConsumer{},
	}
	requestCtx, cancel := context.WithCancel(context.Background())
	handler := &queueTestHandler{done: make(chan error, 1)}
	if err := impl.StartConsumer(requestCtx, ConsumerConfig{
		Name:        "test",
		QueueName:   "test",
		Directory:   t.TempDir(),
		Concurrency: 1,
		QueueCount:  1,
	}, handler); err != nil {
		t.Fatalf("启动消费者失败: %v", err)
	}
	defer impl.StopConsumer("test")
	cancel()
	if err := impl.Enqueue(context.Background(), EnqueueInput{
		ConsumerName: "test",
		Payload:      []byte("payload"),
	}); err != nil {
		t.Fatalf("入队失败: %v", err)
	}
	select {
	case err := <-handler.done:
		if err != nil {
			t.Fatalf("期望消费者使用自身生命周期 context，got=%v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("等待消费者处理超时")
	}
}

func TestQueueConsumerRetriesFailedPayload(t *testing.T) {
	async := asyncservice.NewService(context.Background(), &asyncservice.Config{PoolSize: 4})
	impl := &serviceImpl{
		asyncService: async,
		consumers:    map[string]*queueConsumer{},
	}
	handler := &queueTestHandler{done: make(chan error, 1), failures: 1}
	if err := impl.StartConsumer(context.Background(), ConsumerConfig{
		Name:         "retry",
		QueueName:    "retry",
		Directory:    t.TempDir(),
		Concurrency:  1,
		QueueCount:   1,
		RetryDelay:   10 * time.Millisecond,
		PollInterval: 10 * time.Millisecond,
		MaxAttempts:  3,
	}, handler); err != nil {
		t.Fatalf("启动消费者失败: %v", err)
	}
	defer impl.StopConsumer("retry")
	if err := impl.Enqueue(context.Background(), EnqueueInput{
		ConsumerName: "retry",
		Payload:      []byte("payload"),
		DedupKey:     "payload",
	}); err != nil {
		t.Fatalf("入队失败: %v", err)
	}
	select {
	case err := <-handler.done:
		if err != nil {
			t.Fatalf("期望重试成功后 context 正常，got=%v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("等待消费者重试超时")
	}
	if depth := impl.Depth("retry"); depth != 0 {
		t.Fatalf("重试成功后队列深度应为 0, got=%d", depth)
	}
	handler.payloadMu.Lock()
	attempts := len(handler.payloads)
	handler.payloadMu.Unlock()
	if attempts != 2 {
		t.Fatalf("期望处理 2 次，got=%d", attempts)
	}
	waitForQueueCondition(t, 3*time.Second, func() bool {
		return impl.Enqueue(context.Background(), EnqueueInput{
			ConsumerName: "retry",
			Payload:      []byte("payload"),
			DedupKey:     "payload",
		}) == nil
	})
}

func TestQueueConsumerMovesFailedPayloadToDeadLetter(t *testing.T) {
	async := asyncservice.NewService(context.Background(), &asyncservice.Config{PoolSize: 4})
	impl := &serviceImpl{
		asyncService: async,
		consumers:    map[string]*queueConsumer{},
	}
	queueDir := t.TempDir()
	handler := &queueTestHandler{done: make(chan error, 1), failures: 10}
	if err := impl.StartConsumer(context.Background(), ConsumerConfig{
		Name:         "dead",
		QueueName:    "dead",
		Directory:    queueDir,
		Concurrency:  1,
		QueueCount:   1,
		RetryDelay:   10 * time.Millisecond,
		PollInterval: 10 * time.Millisecond,
		MaxAttempts:  2,
	}, handler); err != nil {
		t.Fatalf("启动消费者失败: %v", err)
	}
	defer impl.StopConsumer("dead")
	if err := impl.Enqueue(context.Background(), EnqueueInput{
		ConsumerName: "dead",
		Payload:      []byte("payload"),
		DedupKey:     "dead-key",
	}); err != nil {
		t.Fatalf("入队失败: %v", err)
	}
	waitForQueueCondition(t, 3*time.Second, func() bool {
		if countQueueFiles(filepath.Join(queueDir, queueDeadDirName)) != 1 || impl.Depth("dead") != 0 {
			return false
		}
		return impl.Enqueue(context.Background(), EnqueueInput{
			ConsumerName: "dead",
			Payload:      []byte("payload"),
			DedupKey:     "dead-key",
		}) == nil
	})
	if err := impl.Enqueue(context.Background(), EnqueueInput{
		ConsumerName: "dead",
		Payload:      []byte("payload"),
		DedupKey:     "dead-key-2",
	}); err != nil {
		t.Fatalf("dead letter 后去重键应释放，got=%v", err)
	}
}

func TestQueueConsumerRecoversInflightPayloadAfterRestart(t *testing.T) {
	async := asyncservice.NewService(context.Background(), &asyncservice.Config{PoolSize: 4})
	impl := &serviceImpl{
		asyncService: async,
		consumers:    map[string]*queueConsumer{},
	}
	queueDir := t.TempDir()
	blocker := make(chan struct{})
	var unblockOnce sync.Once
	unblock := func() {
		unblockOnce.Do(func() {
			close(blocker)
		})
	}
	handler := queueBlockingHandler{started: make(chan struct{}, 1), unblock: blocker}
	config := ConsumerConfig{
		Name:         "recover",
		QueueName:    "recover",
		Directory:    queueDir,
		Concurrency:  1,
		QueueCount:   1,
		PollInterval: 10 * time.Millisecond,
		LeaseTimeout: 20 * time.Millisecond,
	}
	if err := impl.StartConsumer(context.Background(), config, handler); err != nil {
		t.Fatalf("启动消费者失败: %v", err)
	}
	if err := impl.Enqueue(context.Background(), EnqueueInput{
		ConsumerName: "recover",
		Payload:      []byte("payload"),
		DedupKey:     "recover-key",
	}); err != nil {
		t.Fatalf("入队失败: %v", err)
	}
	select {
	case <-handler.started:
	case <-time.After(3 * time.Second):
		t.Fatal("等待首次消费超时")
	}
	impl.mu.Lock()
	firstConsumer := impl.consumers["recover"]
	delete(impl.consumers, "recover")
	impl.mu.Unlock()
	defer func() {
		unblock()
		firstConsumer.close()
	}()

	secondHandler := &queueTestHandler{done: make(chan error, 1)}
	if err := impl.StartConsumer(context.Background(), config, secondHandler); err != nil {
		t.Fatalf("重启消费者失败: %v", err)
	}
	defer impl.StopConsumer("recover")
	select {
	case err := <-secondHandler.done:
		if err != nil {
			t.Fatalf("期望恢复消息处理成功，got=%v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("等待恢复消息处理超时")
	}
	unblock()
	firstConsumer.close()
}

func TestQueueConsumerDoesNotRedeliverActiveLease(t *testing.T) {
	async := asyncservice.NewService(context.Background(), &asyncservice.Config{PoolSize: 4})
	impl := &serviceImpl{
		asyncService: async,
		consumers:    map[string]*queueConsumer{},
	}
	queueDir := t.TempDir()
	blocker := make(chan struct{})
	handler := queueBlockingHandler{started: make(chan struct{}, 2), unblock: blocker}
	if err := impl.StartConsumer(context.Background(), ConsumerConfig{
		Name:         "active-lease",
		QueueName:    "active-lease",
		Directory:    queueDir,
		Concurrency:  2,
		QueueCount:   2,
		PollInterval: 5 * time.Millisecond,
		LeaseTimeout: 20 * time.Millisecond,
		MaxAttempts:  3,
	}, handler); err != nil {
		t.Fatalf("启动消费者失败: %v", err)
	}
	defer impl.StopConsumer("active-lease")
	if err := impl.Enqueue(context.Background(), EnqueueInput{
		ConsumerName: "active-lease",
		Payload:      []byte("payload"),
		DedupKey:     "active-lease-key",
	}); err != nil {
		t.Fatalf("入队失败: %v", err)
	}
	select {
	case <-handler.started:
	case <-time.After(3 * time.Second):
		t.Fatal("等待首次消费超时")
	}
	select {
	case <-handler.started:
		t.Fatal("租约过期不应重复投递当前进程仍在处理的消息")
	case <-time.After(150 * time.Millisecond):
	}
	close(blocker)
	waitForQueueCondition(t, 3*time.Second, func() bool {
		return impl.Depth("active-lease") == 0
	})
}

func TestQueueConsumerProcessesShardsConcurrently(t *testing.T) {
	async := asyncservice.NewService(context.Background(), &asyncservice.Config{PoolSize: 8})
	impl := &serviceImpl{
		asyncService: async,
		consumers:    map[string]*queueConsumer{},
	}
	handler := &queueConcurrentHandler{
		started: make(chan string, 4),
		unblock: make(chan struct{}),
		done:    make(chan struct{}, 4),
	}
	if err := impl.StartConsumer(context.Background(), ConsumerConfig{
		Name:         "sharded",
		QueueName:    "sharded",
		Directory:    t.TempDir(),
		Concurrency:  4,
		QueueCount:   4,
		PollInterval: 5 * time.Millisecond,
	}, handler); err != nil {
		t.Fatalf("启动消费者失败: %v", err)
	}
	defer impl.StopConsumer("sharded")
	for _, key := range []string{"key-a", "key-b", "key-c", "key-d"} {
		if err := impl.Enqueue(context.Background(), EnqueueInput{
			ConsumerName: "sharded",
			Payload:      []byte(key),
			DedupKey:     key,
		}); err != nil {
			t.Fatalf("入队失败: %v", err)
		}
	}
	seen := map[string]struct{}{}
	deadline := time.After(3 * time.Second)
	for len(seen) < 4 {
		select {
		case key := <-handler.started:
			seen[key] = struct{}{}
		case <-deadline:
			close(handler.unblock)
			t.Fatalf("等待并发消费启动超时，started=%d", len(seen))
		}
	}
	close(handler.unblock)
	waitForQueueCondition(t, 3*time.Second, func() bool {
		return impl.Depth("sharded") == 0
	})
}

func TestQueueConsumerSerializesSameShardKey(t *testing.T) {
	async := asyncservice.NewService(context.Background(), &asyncservice.Config{PoolSize: 8})
	impl := &serviceImpl{
		asyncService: async,
		consumers:    map[string]*queueConsumer{},
	}
	handler := &queueConcurrentHandler{
		started: make(chan string, 2),
		unblock: make(chan struct{}),
		done:    make(chan struct{}, 2),
	}
	if err := impl.StartConsumer(context.Background(), ConsumerConfig{
		Name:         "same-shard",
		QueueName:    "same-shard",
		Directory:    t.TempDir(),
		Concurrency:  2,
		QueueCount:   4,
		PollInterval: 5 * time.Millisecond,
	}, handler); err != nil {
		t.Fatalf("启动消费者失败: %v", err)
	}
	defer impl.StopConsumer("same-shard")
	for _, payload := range []string{"first", "second"} {
		if err := impl.Enqueue(context.Background(), EnqueueInput{
			ConsumerName: "same-shard",
			Payload:      []byte(payload),
			DedupKey:     payload,
			ShardKey:     "session:1001",
		}); err != nil {
			t.Fatalf("入队失败: %v", err)
		}
	}
	select {
	case payload := <-handler.started:
		if payload != "first" {
			t.Fatalf("期望先处理 first，got=%s", payload)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("等待首次消费超时")
	}
	select {
	case payload := <-handler.started:
		t.Fatalf("同一 ShardKey 不应并发处理第二条消息，got=%s", payload)
	case <-time.After(100 * time.Millisecond):
	}
	close(handler.unblock)
	waitForQueueCondition(t, 3*time.Second, func() bool {
		return impl.Depth("same-shard") == 0
	})
}

type queueBlockingHandler struct {
	started chan struct{}
	unblock <-chan struct{}
}

func (handler queueBlockingHandler) HandleQueuePayload(ctx context.Context, _ []byte) error {
	handler.started <- struct{}{}
	<-handler.unblock
	return ctx.Err()
}

type queueConcurrentHandler struct {
	started chan string
	unblock chan struct{}
	done    chan struct{}
}

func (handler *queueConcurrentHandler) HandleQueuePayload(_ context.Context, payload []byte) error {
	handler.started <- string(payload)
	<-handler.unblock
	handler.done <- struct{}{}
	return nil
}

func waitForQueueCondition(t *testing.T, timeout time.Duration, condition func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("等待队列条件超时")
}
