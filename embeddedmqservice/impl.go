package embeddedmqservice

import (
	"context"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/coffeehc/base/log"
	bootplugin "github.com/coffeehc/boot/plugin"
	"github.com/coffeehc/commons/asyncservice"
	"go.uber.org/zap"
)

const (
	queuePendingDirName  = "pending"
	queueInflightDirName = "inflight"
	queueDeadDirName     = "dead"
	queueFileExt         = ".json"

	defaultQueuePollInterval = 200 * time.Millisecond
	defaultQueueLeaseTimeout = 30 * time.Minute
	defaultQueueRetryDelay   = time.Second
	defaultQueueMaxDelay     = time.Minute
	defaultQueueMaxAttempts  = 5
	defaultQueueShardCount   = 16
	maxQueueShardCount       = 16
)

// serviceImpl 实现嵌入式持久 MQ 服务。
type serviceImpl struct {
	// asyncService 提供 MQ reader 与 worker 的后台执行能力。
	asyncService asyncservice.Service
	// lifecycleMu 隔离消费者读写操作与启动、停止、清理和服务停止操作。
	lifecycleMu sync.RWMutex
	// mu 保护消费者注册表。
	mu sync.RWMutex
	// consumers 保存已经启动的消费者。
	consumers map[string]*queueConsumer
	// stopped 表示服务已经执行 Stop，不再允许启动消费者。
	stopped bool
}

// queueConsumer 表示一个已经启动的嵌入式 MQ 消费者。
type queueConsumer struct {
	// config 保存归一化后的消费者配置。
	config ConsumerConfig
	// handler 处理已经领取的业务 payload。
	handler QueueHandler
	// stopCh 通知 reader 停止领取消息。
	stopCh chan struct{}
	// doneCh 在 reader 退出后关闭。
	doneCh chan struct{}
	// slots 限制当前消费者未确认消息的并发数。
	slots chan struct{}
	// ctx 是传递给业务 handler 的消费者生命周期上下文。
	ctx context.Context
	// cancel 在消费者完全停止后取消 handler 上下文。
	cancel context.CancelFunc

	// fileMu 串行化消息文件及活跃投递状态变更。
	fileMu sync.Mutex
	// activeMessages 记录当前进程正在处理的消息。
	activeMessages map[string]struct{}
	// activeShards 记录当前正在处理消息的分片。
	activeShards map[int]struct{}
	// mu 保护进程内去重键索引。
	mu sync.Mutex
	// queuedKeys 保存 pending 和 inflight 消息的非空去重键。
	queuedKeys map[string]struct{}
	// stopOnce 保证消费者停止流程只执行一次。
	stopOnce sync.Once
	// wg 等待已经派发给 asyncservice 的 worker 完成。
	wg sync.WaitGroup
	// nextID 生成进程内单调递增的消息 ID 后缀。
	nextID uint64
	// nextShard 轮转无路由键消息和下一次扫描起点。
	nextShard uint64
}

// queueEnvelope 是 embeddedmqservice 写入本地文件队列的内部 envelope。
type queueEnvelope struct {
	// MessageID 是消息文件的稳定标识。
	MessageID string `json:"message_id"`
	// DedupKey 是消息未确认期间使用的可选去重键。
	DedupKey string `json:"dedup_key,omitempty"`
	// Shard 是消息所属的物理分片编号。
	Shard int `json:"shard,omitempty"`
	// Payload 是调用方写入的原始业务负载。
	Payload []byte `json:"payload"`
	// Attempts 是消息已经被领取的次数。
	Attempts int `json:"attempts"`
	// CreatedAt 是消息创建时间，单位为 Unix 毫秒。
	CreatedAt int64 `json:"created_at"`
	// UpdatedAt 是 envelope 最后更新时间，单位为 Unix 毫秒。
	UpdatedAt int64 `json:"updated_at"`
	// AvailableAt 是消息下一次允许领取的时间，单位为 Unix 毫秒。
	AvailableAt int64 `json:"available_at"`
	// LockedAt 是本次领取时间，单位为 Unix 毫秒。
	LockedAt int64 `json:"locked_at,omitempty"`
	// LeaseExpiresAt 是本次消费租约到期时间，单位为 Unix 毫秒。
	LeaseExpiresAt int64 `json:"lease_expires_at,omitempty"`
	// LastError 保存最近一次处理或恢复错误。
	LastError string `json:"last_error,omitempty"`
}

// queueWorkItem 表示已经领取到 inflight 的一条队列消息。
type queueWorkItem struct {
	// shard 是消息所属的分片编号。
	shard int
	// fileName 是消息文件名。
	fileName string
	// path 是当前 inflight 文件路径。
	path string
	// envelope 是本次领取后的消息 envelope。
	envelope queueEnvelope
}

// deadLetterFile 表示一份进入清理候选集的普通死信文件。
type deadLetterFile struct {
	// path 表示死信文件绝对路径。
	path string
	// directory 表示死信所属分片目录，用于删除后同步目录元数据。
	directory string
	// updatedAt 表示 envelope 更新时间；无法解析时回退文件修改时间，单位为 Unix 毫秒。
	updatedAt int64
	// size 表示死信文件物理字节数。
	size int64
}

var _ bootplugin.Plugin = (*serviceImpl)(nil)

// newService 使用全局 asyncservice 创建嵌入式 MQ 服务。
func newService(ctx context.Context) Service {
	asyncservice.EnablePlugin(ctx)
	return NewService(asyncservice.GetService())
}

// NewService 使用指定异步执行服务创建一个独立的嵌入式 MQ 服务。
// 调用方负责保证 asyncService 的生命周期长于返回的 MQ 服务；传入 nil 会 panic。
func NewService(asyncService asyncservice.Service) Service {
	if asyncService == nil {
		panic("embeddedmqservice: async service must not be nil")
	}
	return &serviceImpl{
		asyncService: asyncService,
		consumers:    map[string]*queueConsumer{},
	}
}

// Start 启动嵌入式 MQ 服务。
func (impl *serviceImpl) Start(context.Context) error {
	log.Debug("嵌入式 MQ 服务启动完成")
	return nil
}

// Stop 停止全部嵌入式 MQ 消费者。
func (impl *serviceImpl) Stop(context.Context) error {
	impl.lifecycleMu.Lock()
	defer impl.lifecycleMu.Unlock()
	impl.stopped = true
	impl.mu.Lock()
	consumers := make([]*queueConsumer, 0, len(impl.consumers))
	for _, consumer := range impl.consumers {
		consumers = append(consumers, consumer)
	}
	impl.consumers = map[string]*queueConsumer{}
	impl.mu.Unlock()
	for _, consumer := range consumers {
		consumer.close()
	}
	log.Debug("嵌入式 MQ 服务已停止")
	return nil
}

// StartConsumer 启动或替换一个嵌入式 MQ 消费者。
func (impl *serviceImpl) StartConsumer(ctx context.Context, config ConsumerConfig, handler QueueHandler) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := validateConsumerConfig(config, handler); err != nil {
		return err
	}
	normalizeConsumerConfig(&config)
	impl.lifecycleMu.Lock()
	defer impl.lifecycleMu.Unlock()
	return impl.startConsumer(config, handler)
}

// startConsumer 在持有 lifecycleMu 时替换并启动指定消费者。
func (impl *serviceImpl) startConsumer(config ConsumerConfig, handler QueueHandler) error {
	if impl.stopped {
		return fmt.Errorf("%w: %s", ErrServiceStopped, config.Name)
	}
	if impl.asyncService.PoolStatus().Stopped {
		return fmt.Errorf("异步执行服务已经停止: %s", config.Name)
	}
	if existing := impl.detachConsumer(config.Name); existing != nil {
		existing.close()
	}
	consumerCtx, cancel := context.WithCancel(context.Background())
	consumer := &queueConsumer{
		config:         config,
		handler:        handler,
		stopCh:         make(chan struct{}),
		doneCh:         make(chan struct{}),
		slots:          make(chan struct{}, config.Concurrency),
		ctx:            consumerCtx,
		cancel:         cancel,
		activeMessages: map[string]struct{}{},
		activeShards:   map[int]struct{}{},
		queuedKeys:     map[string]struct{}{},
	}
	if err := consumer.ensureLayout(); err != nil {
		cancel()
		return err
	}
	if err := consumer.recoverInflight(true); err != nil {
		cancel()
		return err
	}
	if err := consumer.rebuildQueuedKeys(); err != nil {
		cancel()
		return err
	}
	impl.mu.Lock()
	impl.consumers[config.Name] = consumer
	impl.mu.Unlock()
	impl.asyncService.Submit(func() {
		impl.runConsumer(consumer)
	})
	log.Debug("嵌入式 MQ 消费者启动完成",
		zap.String("consumer", config.Name),
		zap.String("queue", config.QueueName),
		zap.String("dir", config.Directory),
		zap.Int("concurrency", config.Concurrency),
		zap.Int("queue_count", config.QueueCount),
		zap.Int("max_attempts", config.MaxAttempts),
	)
	return nil
}

// StopConsumer 停止指定消费者。
func (impl *serviceImpl) StopConsumer(name string) {
	impl.lifecycleMu.Lock()
	defer impl.lifecycleMu.Unlock()
	consumer := impl.detachConsumer(name)
	if consumer != nil {
		consumer.close()
	}
}

// Enqueue 向指定消费者写入一条 payload。
func (impl *serviceImpl) Enqueue(ctx context.Context, input EnqueueInput) error {
	impl.lifecycleMu.RLock()
	defer impl.lifecycleMu.RUnlock()
	if err := ctx.Err(); err != nil {
		return err
	}
	consumer := impl.getConsumer(input.ConsumerName)
	if consumer == nil {
		return fmt.Errorf("队列消费者未启动: %s", input.ConsumerName)
	}
	if len(input.Payload) == 0 {
		return fmt.Errorf("队列 payload 不能为空: %s", input.ConsumerName)
	}
	if input.DedupKey != "" && !consumer.markQueued(input.DedupKey) {
		return ErrDuplicate
	}
	now := time.Now().UnixMilli()
	envelope := queueEnvelope{
		MessageID:   consumer.nextMessageID(),
		DedupKey:    input.DedupKey,
		Payload:     input.Payload,
		CreatedAt:   now,
		UpdatedAt:   now,
		AvailableAt: now,
	}
	shard := consumer.resolveShard(resolveEnqueueShardKey(input))
	if err := consumer.writePending(shard, envelope); err != nil {
		consumer.unmarkQueued(input.DedupKey)
		return err
	}
	return nil
}

// PurgePayloads 停止指定消费者，按业务 predicate 清理持久 payload 后恢复消费。
func (impl *serviceImpl) PurgePayloads(ctx context.Context, consumerName string, predicate PayloadPredicate) (int, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	if strings.TrimSpace(consumerName) == "" {
		return 0, fmt.Errorf("队列消费者名称不能为空")
	}
	if predicate == nil {
		return 0, fmt.Errorf("队列 payload predicate 不能为空")
	}
	impl.lifecycleMu.Lock()
	defer impl.lifecycleMu.Unlock()
	consumer := impl.detachConsumer(consumerName)
	if consumer == nil {
		return 0, fmt.Errorf("队列消费者未启动: %s", consumerName)
	}
	consumer.close()
	deleted, purgeErr := consumer.purgePayloads(predicate)
	var restartErr error
	if impl.asyncService.PoolStatus().Stopped {
		restartErr = fmt.Errorf("异步执行服务已经停止: %s", consumerName)
	} else {
		restartErr = impl.startConsumer(consumer.config, consumer.handler)
	}
	if purgeErr != nil {
		if restartErr != nil {
			return deleted, fmt.Errorf("清理队列 payload 失败: %v; 恢复队列消费者失败: %w", purgeErr, restartErr)
		}
		return deleted, purgeErr
	}
	if restartErr != nil {
		return deleted, restartErr
	}
	return deleted, nil
}

// PurgeDeadLetters 按保留期和容量上限清理所有已启动 consumer 的死信。
func (impl *serviceImpl) PurgeDeadLetters(ctx context.Context, policy DeadLetterCleanupPolicy) (DeadLetterCleanupResult, error) {
	if err := ctx.Err(); err != nil {
		return DeadLetterCleanupResult{}, err
	}
	if policy.Cutoff.IsZero() || policy.MaxEntries < 0 || policy.MaxBytes < 0 {
		return DeadLetterCleanupResult{}, fmt.Errorf("死信清理策略不完整")
	}
	impl.lifecycleMu.RLock()
	defer impl.lifecycleMu.RUnlock()
	impl.mu.RLock()
	consumers := make([]*queueConsumer, 0, len(impl.consumers))
	for _, consumer := range impl.consumers {
		consumers = append(consumers, consumer)
	}
	impl.mu.RUnlock()
	sort.Slice(consumers, func(left int, right int) bool {
		return consumers[left].config.Name < consumers[right].config.Name
	})
	result := DeadLetterCleanupResult{}
	for _, consumer := range consumers {
		removed, err := consumer.purgeDeadLetters(ctx, policy)
		result.RemovedEntries += removed.RemovedEntries
		result.RemovedBytes += removed.RemovedBytes
		if err != nil {
			return result, err
		}
	}
	return result, nil
}

// purgeDeadLetters 在 consumer 文件锁内删除过期或超出容量上限的死信。
func (consumer *queueConsumer) purgeDeadLetters(ctx context.Context, policy DeadLetterCleanupPolicy) (DeadLetterCleanupResult, error) {
	consumer.fileMu.Lock()
	defer consumer.fileMu.Unlock()
	files := make([]deadLetterFile, 0)
	for shard := 0; shard < consumer.config.QueueCount; shard++ {
		directory := consumer.deadDir(shard)
		entries, err := queueJSONEntries(directory)
		if err != nil {
			return DeadLetterCleanupResult{}, err
		}
		for _, entry := range entries {
			if err = ctx.Err(); err != nil {
				return DeadLetterCleanupResult{}, err
			}
			if entry.Type()&os.ModeSymlink != 0 {
				continue
			}
			info, infoErr := entry.Info()
			if infoErr != nil {
				return DeadLetterCleanupResult{}, infoErr
			}
			if !info.Mode().IsRegular() {
				continue
			}
			path := filepath.Join(directory, entry.Name())
			updatedAt := info.ModTime().UnixMilli()
			if envelope, readErr := readQueueEnvelope(path); readErr == nil && envelope.UpdatedAt > 0 {
				updatedAt = envelope.UpdatedAt
			}
			files = append(files, deadLetterFile{path: path, directory: directory, updatedAt: updatedAt, size: info.Size()})
		}
	}
	sort.Slice(files, func(left int, right int) bool {
		if files[left].updatedAt == files[right].updatedAt {
			return files[left].path < files[right].path
		}
		return files[left].updatedAt < files[right].updatedAt
	})
	remainingEntries := len(files)
	remainingBytes := int64(0)
	for _, file := range files {
		remainingBytes += file.size
	}
	removePaths := make(map[string]struct{})
	cutoffUnixMilli := policy.Cutoff.UnixMilli()
	for _, file := range files {
		if file.updatedAt > cutoffUnixMilli {
			continue
		}
		removePaths[file.path] = struct{}{}
		remainingEntries--
		remainingBytes -= file.size
	}
	for _, file := range files {
		if remainingEntries <= policy.MaxEntries && remainingBytes <= policy.MaxBytes {
			break
		}
		if _, exists := removePaths[file.path]; exists {
			continue
		}
		removePaths[file.path] = struct{}{}
		remainingEntries--
		remainingBytes -= file.size
	}
	result := DeadLetterCleanupResult{}
	syncedDirectories := make(map[string]struct{})
	for _, file := range files {
		if _, remove := removePaths[file.path]; !remove {
			continue
		}
		if err := os.Remove(file.path); err != nil && !os.IsNotExist(err) {
			return result, err
		}
		result.RemovedEntries++
		result.RemovedBytes += file.size
		syncedDirectories[file.directory] = struct{}{}
	}
	for directory := range syncedDirectories {
		if err := syncDir(directory); err != nil {
			return result, err
		}
	}
	return result, nil
}

// Depth 返回指定消费者当前仍需处理的消息数量。
func (impl *serviceImpl) Depth(name string) int64 {
	impl.lifecycleMu.RLock()
	defer impl.lifecycleMu.RUnlock()
	consumer := impl.getConsumer(name)
	if consumer == nil {
		return 0
	}
	consumer.fileMu.Lock()
	defer consumer.fileMu.Unlock()
	var depth int64
	for shard := 0; shard < consumer.config.QueueCount; shard++ {
		depth += countQueueFiles(consumer.pendingDir(shard)) + countQueueFiles(consumer.inflightDir(shard))
	}
	return depth
}

// runConsumer 持续领取 pending 消息并派发 worker。
func (impl *serviceImpl) runConsumer(consumer *queueConsumer) {
	defer close(consumer.doneCh)
	ticker := time.NewTicker(consumer.config.PollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-consumer.stopCh:
			return
		default:
		}
		select {
		case consumer.slots <- struct{}{}:
		case <-consumer.stopCh:
			return
		}
		item, ok, err := consumer.claimNext()
		if err != nil {
			<-consumer.slots
			log.Error("领取嵌入式 MQ 消息失败",
				zap.String("consumer", consumer.config.Name),
				zap.Error(err),
			)
			if !waitQueueTick(ticker, consumer.stopCh) {
				return
			}
			continue
		}
		if !ok {
			<-consumer.slots
			if err = consumer.recoverInflight(false); err != nil {
				log.Error("恢复过期嵌入式 MQ 消息失败",
					zap.String("consumer", consumer.config.Name),
					zap.Error(err),
				)
			}
			if !waitQueueTick(ticker, consumer.stopCh) {
				return
			}
			continue
		}
		consumer.wg.Add(1)
		if impl.asyncService.PoolStatus().Available == 0 {
			consumer.engine(item)
			continue
		}
		impl.asyncService.Submit(func() {
			consumer.engine(item)
		})
	}
}

// detachConsumer 移除并返回指定消费者。
func (impl *serviceImpl) detachConsumer(name string) *queueConsumer {
	impl.mu.Lock()
	defer impl.mu.Unlock()
	consumer := impl.consumers[name]
	delete(impl.consumers, name)
	return consumer
}

// getConsumer 返回指定消费者当前态。
func (impl *serviceImpl) getConsumer(name string) *queueConsumer {
	impl.mu.RLock()
	defer impl.mu.RUnlock()
	return impl.consumers[name]
}

// engine 调用业务 handler 并根据结果 ack、retry 或 dead-letter。
func (consumer *queueConsumer) engine(item *queueWorkItem) {
	defer func() {
		consumer.releaseActive(item.envelope.MessageID, item.shard)
		<-consumer.slots
		consumer.wg.Done()
	}()
	if err := consumer.handler.HandleQueuePayload(consumer.ctx, item.envelope.Payload); err != nil {
		if consumer.ctx.Err() != nil {
			if requeueErr := consumer.requeueCanceled(item); requeueErr != nil {
				log.Error("停止嵌入式 MQ 消费者时回退消息失败",
					zap.String("consumer", consumer.config.Name),
					zap.String("message_id", item.envelope.MessageID),
					zap.Error(requeueErr),
				)
			}
			return
		}
		log.Error("消费嵌入式 MQ 消息失败：业务处理失败",
			zap.String("consumer", consumer.config.Name),
			zap.String("message_id", item.envelope.MessageID),
			zap.Int("attempts", item.envelope.Attempts),
			zap.Error(err),
		)
		if retryErr := consumer.retryOrDead(item, err); retryErr != nil {
			log.Error("消费嵌入式 MQ 消息失败：回写重试状态失败",
				zap.String("consumer", consumer.config.Name),
				zap.String("message_id", item.envelope.MessageID),
				zap.Error(retryErr),
			)
		}
		return
	}
	if err := consumer.ack(item); err != nil {
		log.Error("消费嵌入式 MQ 消息失败：ack 删除消息失败",
			zap.String("consumer", consumer.config.Name),
			zap.String("message_id", item.envelope.MessageID),
			zap.Error(err),
		)
		return
	}
}

// requeueCanceled 把因消费者停止而中断的消息放回 pending，且不消耗本次 attempt。
func (consumer *queueConsumer) requeueCanceled(item *queueWorkItem) error {
	consumer.fileMu.Lock()
	defer consumer.fileMu.Unlock()
	if _, err := os.Stat(item.path); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	envelope := item.envelope
	if envelope.Attempts > 0 {
		envelope.Attempts--
	}
	now := time.Now().UnixMilli()
	envelope.LockedAt = 0
	envelope.LeaseExpiresAt = 0
	envelope.UpdatedAt = now
	envelope.AvailableAt = now
	if err := writeQueueEnvelope(item.path, envelope); err != nil {
		return err
	}
	return renameQueueFile(item.path, filepath.Join(consumer.pendingDir(item.shard), item.fileName))
}

// close 停止消费者并等待已经派发的 worker 完成。
func (consumer *queueConsumer) close() {
	if consumer == nil {
		return
	}
	consumer.stopOnce.Do(func() {
		close(consumer.stopCh)
		if consumer.cancel != nil {
			consumer.cancel()
		}
		<-consumer.doneCh
		consumer.wg.Wait()
	})
}

// ensureLayout 确保当前队列目录结构存在。
func (consumer *queueConsumer) ensureLayout() error {
	for shard := 0; shard < consumer.config.QueueCount; shard++ {
		for _, dir := range []string{consumer.pendingDir(shard), consumer.inflightDir(shard), consumer.deadDir(shard)} {
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return err
			}
		}
	}
	return nil
}

// recoverInflight 把崩溃或租约过期后遗留的 inflight 消息放回 pending。
func (consumer *queueConsumer) recoverInflight(force bool) error {
	consumer.fileMu.Lock()
	defer consumer.fileMu.Unlock()
	for shard := 0; shard < consumer.config.QueueCount; shard++ {
		entries, err := queueJSONEntries(consumer.inflightDir(shard))
		if err != nil {
			return err
		}
		now := time.Now().UnixMilli()
		for _, entry := range entries {
			inflightPath := filepath.Join(consumer.inflightDir(shard), entry.Name())
			envelope, err := readQueueEnvelope(inflightPath)
			if err != nil {
				if moveErr := consumer.moveCorruptToDead(shard, inflightPath, entry.Name(), err); moveErr != nil {
					return moveErr
				}
				continue
			}
			if !force && envelope.LeaseExpiresAt > now {
				continue
			}
			if _, active := consumer.activeMessages[envelope.MessageID]; !force && active {
				continue
			}
			envelope.LockedAt = 0
			envelope.LeaseExpiresAt = 0
			envelope.UpdatedAt = now
			if envelope.AvailableAt < now {
				envelope.AvailableAt = now
			}
			if envelope.Attempts >= consumer.config.MaxAttempts {
				envelope.LastError = "队列消息租约过期且已达到最大重试次数"
				if err = consumer.moveEnvelopeToDead(shard, inflightPath, entry.Name(), envelope); err != nil {
					return err
				}
				consumer.unmarkQueued(envelope.DedupKey)
				continue
			}
			if err = writeQueueEnvelope(inflightPath, envelope); err != nil {
				return err
			}
			if err = renameQueueFile(inflightPath, filepath.Join(consumer.pendingDir(shard), entry.Name())); err != nil {
				return err
			}
		}
	}
	return nil
}

// rebuildQueuedKeys 从 pending 与 inflight 文件恢复进程内去重索引。
func (consumer *queueConsumer) rebuildQueuedKeys() error {
	consumer.fileMu.Lock()
	defer consumer.fileMu.Unlock()
	consumer.mu.Lock()
	consumer.queuedKeys = map[string]struct{}{}
	consumer.mu.Unlock()
	for shard := 0; shard < consumer.config.QueueCount; shard++ {
		for _, dir := range []string{consumer.pendingDir(shard), consumer.inflightDir(shard)} {
			entries, err := queueJSONEntries(dir)
			if err != nil {
				return err
			}
			for _, entry := range entries {
				envelope, err := readQueueEnvelope(filepath.Join(dir, entry.Name()))
				if err != nil {
					continue
				}
				consumer.markQueued(envelope.DedupKey)
			}
		}
	}
	return nil
}

// purgePayloads 在消费者停止后按业务 payload 清理 pending、inflight 和 dead 文件。
func (consumer *queueConsumer) purgePayloads(predicate PayloadPredicate) (int, error) {
	consumer.fileMu.Lock()
	defer consumer.fileMu.Unlock()
	paths := make([]string, 0)
	for shard := 0; shard < consumer.config.QueueCount; shard++ {
		for _, dir := range []string{consumer.pendingDir(shard), consumer.inflightDir(shard), consumer.deadDir(shard)} {
			entries, err := queueJSONEntries(dir)
			if err != nil {
				return 0, err
			}
			for _, entry := range entries {
				path := filepath.Join(dir, entry.Name())
				envelope, readErr := readQueueEnvelope(path)
				if readErr != nil {
					continue
				}
				matched, matchErr := predicate(append([]byte(nil), envelope.Payload...))
				if matchErr != nil {
					return 0, matchErr
				}
				if matched {
					paths = append(paths, path)
				}
			}
		}
	}
	for _, path := range paths {
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return 0, err
		}
	}
	for _, dir := range uniqueQueueDirectories(paths) {
		if err := syncDir(dir); err != nil {
			return 0, err
		}
	}
	return len(paths), nil
}

// uniqueQueueDirectories 返回文件路径对应的去重目录集合。
func uniqueQueueDirectories(paths []string) []string {
	directories := make([]string, 0)
	seen := make(map[string]struct{}, len(paths))
	for _, path := range paths {
		directory := filepath.Dir(path)
		if _, exists := seen[directory]; exists {
			continue
		}
		seen[directory] = struct{}{}
		directories = append(directories, directory)
	}
	return directories
}

// claimNext 领取下一条已经到期的 pending 消息。
func (consumer *queueConsumer) claimNext() (*queueWorkItem, bool, error) {
	consumer.fileMu.Lock()
	defer consumer.fileMu.Unlock()
	start := int(atomic.AddUint64(&consumer.nextShard, 1) % uint64(consumer.config.QueueCount))
	for offset := 0; offset < consumer.config.QueueCount; offset++ {
		shard := (start + offset) % consumer.config.QueueCount
		if consumer.isShardActiveLocked(shard) {
			continue
		}
		item, ok, err := consumer.claimNextFromShard(shard)
		if err != nil || ok {
			return item, ok, err
		}
	}
	return nil, false, nil
}

func (consumer *queueConsumer) claimNextFromShard(shard int) (*queueWorkItem, bool, error) {
	entries, err := queueJSONEntries(consumer.pendingDir(shard))
	if err != nil {
		return nil, false, err
	}
	now := time.Now().UnixMilli()
	for _, entry := range entries {
		pendingPath := filepath.Join(consumer.pendingDir(shard), entry.Name())
		envelope, err := readQueueEnvelope(pendingPath)
		if err != nil {
			if moveErr := consumer.moveCorruptToDead(shard, pendingPath, entry.Name(), err); moveErr != nil {
				return nil, false, moveErr
			}
			continue
		}
		if len(envelope.Payload) == 0 {
			if err = consumer.moveEnvelopeToDead(shard, pendingPath, entry.Name(), envelope); err != nil {
				return nil, false, err
			}
			consumer.unmarkQueued(envelope.DedupKey)
			continue
		}
		if envelope.AvailableAt > now {
			continue
		}
		if envelope.MessageID == "" {
			envelope.MessageID = strings.TrimSuffix(entry.Name(), queueFileExt)
		}
		envelope.Shard = shard
		envelope.Attempts++
		envelope.LockedAt = now
		envelope.LeaseExpiresAt = time.Now().Add(consumer.config.LeaseTimeout).UnixMilli()
		envelope.UpdatedAt = now
		inflightPath := filepath.Join(consumer.inflightDir(shard), entry.Name())
		if err = renameQueueFile(pendingPath, inflightPath); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, false, err
		}
		if err = writeQueueEnvelope(inflightPath, envelope); err != nil {
			log.Error("领取嵌入式 MQ 消息后更新 envelope 失败",
				zap.String("consumer", consumer.config.Name),
				zap.String("message_id", envelope.MessageID),
				zap.Error(err),
			)
		}
		consumer.activeMessages[envelope.MessageID] = struct{}{}
		consumer.activeShards[shard] = struct{}{}
		return &queueWorkItem{shard: shard, fileName: entry.Name(), path: inflightPath, envelope: envelope}, true, nil
	}
	return nil, false, nil
}

// retryOrDead 根据 attempts 决定消息重试还是进入 dead letter。
func (consumer *queueConsumer) retryOrDead(item *queueWorkItem, handleErr error) error {
	consumer.fileMu.Lock()
	defer consumer.fileMu.Unlock()
	envelope := item.envelope
	now := time.Now().UnixMilli()
	envelope.LockedAt = 0
	envelope.LeaseExpiresAt = 0
	envelope.UpdatedAt = now
	envelope.LastError = handleErr.Error()
	if envelope.Attempts >= consumer.config.MaxAttempts {
		if err := consumer.moveEnvelopeToDead(item.shard, item.path, item.fileName, envelope); err != nil {
			return err
		}
		consumer.unmarkQueued(envelope.DedupKey)
		return nil
	}
	envelope.AvailableAt = time.Now().Add(consumer.retryDelay(envelope.Attempts)).UnixMilli()
	if err := writeQueueEnvelope(item.path, envelope); err != nil {
		return err
	}
	return renameQueueFile(item.path, filepath.Join(consumer.pendingDir(item.shard), item.fileName))
}

// writePending 把一条新消息写入 pending。
func (consumer *queueConsumer) writePending(shard int, envelope queueEnvelope) error {
	consumer.fileMu.Lock()
	defer consumer.fileMu.Unlock()
	envelope.Shard = shard
	return writeQueueEnvelope(filepath.Join(consumer.pendingDir(shard), envelope.MessageID+queueFileExt), envelope)
}

// ack 删除 inflight 消息并释放去重键。
func (consumer *queueConsumer) ack(item *queueWorkItem) error {
	consumer.fileMu.Lock()
	defer consumer.fileMu.Unlock()
	if err := os.Remove(item.path); err != nil && !os.IsNotExist(err) {
		return err
	}
	if err := syncDir(filepath.Dir(item.path)); err != nil {
		return err
	}
	consumer.unmarkQueued(item.envelope.DedupKey)
	return nil
}

// releaseActive 清理当前进程内正在处理的 delivery 与分片占用标记。
func (consumer *queueConsumer) releaseActive(messageID string, shard int) {
	consumer.fileMu.Lock()
	if messageID != "" {
		delete(consumer.activeMessages, messageID)
	}
	delete(consumer.activeShards, shard)
	consumer.fileMu.Unlock()
}

// moveCorruptToDead 把无法解析的消息移动到 dead letter。
func (consumer *queueConsumer) moveCorruptToDead(shard int, sourcePath, fileName string, cause error) error {
	envelope := queueEnvelope{
		MessageID:   strings.TrimSuffix(fileName, queueFileExt),
		Attempts:    consumer.config.MaxAttempts,
		CreatedAt:   time.Now().UnixMilli(),
		UpdatedAt:   time.Now().UnixMilli(),
		LastError:   cause.Error(),
		AvailableAt: time.Now().UnixMilli(),
	}
	envelope.Shard = shard
	return consumer.moveEnvelopeToDead(shard, sourcePath, fileName, envelope)
}

// moveEnvelopeToDead 写入 dead letter 并删除原文件。
func (consumer *queueConsumer) moveEnvelopeToDead(shard int, sourcePath, fileName string, envelope queueEnvelope) error {
	envelope.LockedAt = 0
	envelope.LeaseExpiresAt = 0
	envelope.UpdatedAt = time.Now().UnixMilli()
	envelope.Shard = shard
	deadPath := filepath.Join(consumer.deadDir(shard), fileName)
	if err := writeQueueEnvelope(deadPath, envelope); err != nil {
		return err
	}
	if err := os.Remove(sourcePath); err != nil && !os.IsNotExist(err) {
		return err
	}
	return syncDir(filepath.Dir(sourcePath))
}

// retryDelay 返回当前 attempt 对应的指数退避延迟。
func (consumer *queueConsumer) retryDelay(attempts int) time.Duration {
	delay := consumer.config.RetryDelay
	for i := 1; i < attempts; i++ {
		delay *= 2
		if delay >= consumer.config.MaxRetryDelay {
			return consumer.config.MaxRetryDelay
		}
	}
	if delay > consumer.config.MaxRetryDelay {
		return consumer.config.MaxRetryDelay
	}
	return delay
}

// nextMessageID 生成当前 consumer 内的消息文件名主键。
func (consumer *queueConsumer) nextMessageID() string {
	seq := atomic.AddUint64(&consumer.nextID, 1)
	return fmt.Sprintf("%s-%06d", time.Now().UTC().Format("20060102T150405.000000000Z"), seq)
}

func (consumer *queueConsumer) resolveShard(dedupKey string) int {
	if consumer.config.QueueCount <= 1 {
		return 0
	}
	if dedupKey == "" {
		return int(atomic.AddUint64(&consumer.nextShard, 1) % uint64(consumer.config.QueueCount))
	}
	hash := fnv.New32a()
	_, _ = hash.Write([]byte(dedupKey))
	return int(hash.Sum32() % uint32(consumer.config.QueueCount))
}

// isShardActiveLocked 判断当前分片是否已有 worker 正在处理。
func (consumer *queueConsumer) isShardActiveLocked(shard int) bool {
	if consumer.activeShards == nil {
		consumer.activeShards = map[int]struct{}{}
		return false
	}
	_, active := consumer.activeShards[shard]
	return active
}

func (consumer *queueConsumer) shardDir(shard int) string {
	if consumer.config.QueueCount <= 1 {
		return consumer.config.Directory
	}
	return filepath.Join(consumer.config.Directory, fmt.Sprintf("q%02d", shard))
}

func (consumer *queueConsumer) pendingDir(shard int) string {
	return filepath.Join(consumer.shardDir(shard), queuePendingDirName)
}

func (consumer *queueConsumer) inflightDir(shard int) string {
	return filepath.Join(consumer.shardDir(shard), queueInflightDirName)
}

func (consumer *queueConsumer) deadDir(shard int) string {
	return filepath.Join(consumer.shardDir(shard), queueDeadDirName)
}

// markQueued 尝试登记一个仍在队列中或正在消费的去重键。
func (consumer *queueConsumer) markQueued(key string) bool {
	if key == "" {
		return true
	}
	consumer.mu.Lock()
	defer consumer.mu.Unlock()
	if _, exists := consumer.queuedKeys[key]; exists {
		return false
	}
	consumer.queuedKeys[key] = struct{}{}
	return true
}

// unmarkQueued 释放一个去重键。
func (consumer *queueConsumer) unmarkQueued(key string) {
	if key == "" {
		return
	}
	consumer.mu.Lock()
	delete(consumer.queuedKeys, key)
	consumer.mu.Unlock()
}

func resolveEnqueueShardKey(input EnqueueInput) string {
	if input.ShardKey != "" {
		return input.ShardKey
	}
	return input.DedupKey
}

// validateConsumerConfig 校验消费者启动所需的最小参数。
func validateConsumerConfig(config ConsumerConfig, handler QueueHandler) error {
	if config.Name == "" {
		return fmt.Errorf("队列消费者名称不能为空")
	}
	if config.Directory == "" {
		return fmt.Errorf("队列目录不能为空: %s", config.Name)
	}
	if handler == nil {
		return fmt.Errorf("队列 handler 不能为空: %s", config.Name)
	}
	return nil
}

// normalizeConsumerConfig 填充可靠队列默认参数。
func normalizeConsumerConfig(config *ConsumerConfig) {
	if config.QueueName == "" {
		config.QueueName = config.Name
	}
	if config.QueueCount <= 0 {
		config.QueueCount = defaultQueueShardCount
	}
	if config.QueueCount > maxQueueShardCount {
		config.QueueCount = maxQueueShardCount
	}
	if config.Concurrency <= 0 {
		config.Concurrency = config.QueueCount
	}
	if config.PollInterval <= 0 {
		config.PollInterval = defaultQueuePollInterval
	}
	if config.LeaseTimeout <= 0 {
		config.LeaseTimeout = defaultQueueLeaseTimeout
	}
	if config.RetryDelay <= 0 {
		config.RetryDelay = defaultQueueRetryDelay
	}
	if config.MaxRetryDelay <= 0 {
		config.MaxRetryDelay = defaultQueueMaxDelay
	}
	if config.MaxAttempts <= 0 {
		config.MaxAttempts = defaultQueueMaxAttempts
	}
}

func waitQueueTick(ticker *time.Ticker, stopCh <-chan struct{}) bool {
	select {
	case <-ticker.C:
		return true
	case <-stopCh:
		return false
	}
}

func queueJSONEntries(dir string) ([]os.DirEntry, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	filtered := make([]os.DirEntry, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), queueFileExt) {
			continue
		}
		filtered = append(filtered, entry)
	}
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].Name() < filtered[j].Name()
	})
	return filtered, nil
}

func readQueueEnvelope(path string) (queueEnvelope, error) {
	payload, err := os.ReadFile(path)
	if err != nil {
		return queueEnvelope{}, err
	}
	envelope := queueEnvelope{}
	if err = json.Unmarshal(payload, &envelope); err != nil {
		return queueEnvelope{}, err
	}
	return envelope, nil
}

func writeQueueEnvelope(path string, envelope queueEnvelope) error {
	payload, err := json.MarshalIndent(envelope, "", "  ")
	if err != nil {
		return err
	}
	dir := filepath.Dir(path)
	if err = os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmpPath := filepath.Join(dir, "."+filepath.Base(path)+fmt.Sprintf(".%d.tmp", time.Now().UnixNano()))
	file, err := os.OpenFile(tmpPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	if _, err = file.Write(append(payload, '\n')); err != nil {
		_ = file.Close()
		_ = os.Remove(tmpPath)
		return err
	}
	if err = file.Sync(); err != nil {
		_ = file.Close()
		_ = os.Remove(tmpPath)
		return err
	}
	if err = file.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	if err = renameQueueFile(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	return nil
}

func syncDir(dir string) error {
	file, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer file.Close()
	return file.Sync()
}

func renameQueueFile(sourcePath string, targetPath string) error {
	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		return err
	}
	if err := os.Rename(sourcePath, targetPath); err != nil {
		return err
	}
	if err := syncDir(filepath.Dir(sourcePath)); err != nil {
		return err
	}
	if filepath.Dir(sourcePath) == filepath.Dir(targetPath) {
		return nil
	}
	return syncDir(filepath.Dir(targetPath))
}

func countQueueFiles(dir string) int64 {
	entries, err := queueJSONEntries(dir)
	if err != nil {
		return 0
	}
	return int64(len(entries))
}
