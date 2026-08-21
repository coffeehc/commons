package embeddedmqservice

import (
	"context"
	"errors"
	"time"
)

var (
	// ErrDuplicate 表示具备相同 DedupKey 的 payload 已在 MQ 中或正在消费。
	ErrDuplicate = errors.New("queue payload duplicate")
	// ErrServiceStopped 表示嵌入式 MQ 服务已经进入终止状态。
	ErrServiceStopped = errors.New("embedded mq service stopped")
)

// QueueHandler 处理嵌入式 MQ 中已经领取的一条业务 payload。
//
// 职责边界：
// 1. Handler 只接收调用方入队时传入的原始 payload，不感知底层 queue envelope；
// 2. Handler 自己负责解析 payload、执行业务幂等校验与记录业务日志；
// 3. 返回 error 表示本次消费失败，embeddedmqservice 会按 consumer 配置自动重试或写入 dead letter。
type QueueHandler interface {
	// HandleQueuePayload 处理一条已经领取的 payload。
	HandleQueuePayload(ctx context.Context, payload []byte) error
}

// ConsumerConfig 表示一个嵌入式 MQ 消费者配置。
type ConsumerConfig struct {
	// Name 表示 embeddedmqservice 内部消费者名称，必须在当前 Service 中唯一。
	Name string
	// QueueName 表示当前 consumer 的稳定队列名称。
	QueueName string
	// Directory 表示底层消息文件根目录。
	Directory string
	// Concurrency 表示该消费者最多同时处理的 payload 数量；<=0 时使用 QueueCount。
	Concurrency int
	// QueueCount 表示该 consumer 下的持久化分片数量；<=0 时使用默认 16，最大为 16。
	QueueCount int
	// MaxAttempts 表示同一消息最多领取次数；<=0 时使用默认值 5。
	MaxAttempts int
	// RetryDelay 表示首次失败后的重试延迟；<=0 时使用默认值 1 秒。
	RetryDelay time.Duration
	// MaxRetryDelay 表示指数退避最大延迟；<=0 时使用默认值 1 分钟。
	MaxRetryDelay time.Duration
	// LeaseTimeout 表示 worker 领取消息后的租约时长；<=0 时使用默认值 30 分钟。
	LeaseTimeout time.Duration
	// PollInterval 表示无可用消息时的轮询间隔；<=0 时使用默认值 200 毫秒。
	PollInterval time.Duration
}

// EnqueueInput 表示一次嵌入式 MQ 入队请求。
type EnqueueInput struct {
	// ConsumerName 表示目标消费者名称。
	ConsumerName string
	// Payload 表示业务 owner 自己编码后的原始负载。
	Payload []byte
	// DedupKey 表示进程内去重键，可空。
	DedupKey string
	// ShardKey 表示分片路由键，可空；为空时回退 DedupKey。
	ShardKey string
}

// DeadLetterCleanupPolicy 表示一轮死信清理的保留边界。
type DeadLetterCleanupPolicy struct {
	// Cutoff 表示死信更新时间上限；早于或等于该时间的死信允许删除。
	Cutoff time.Time
	// MaxEntries 表示单个 consumer 最多保留的死信数量；0 表示不保留，不能小于 0。
	MaxEntries int
	// MaxBytes 表示单个 consumer 最多保留的死信物理字节数；0 表示不保留，不能小于 0。
	MaxBytes int64
}

// DeadLetterCleanupResult 表示一轮死信清理的物理删除统计。
type DeadLetterCleanupResult struct {
	// RemovedEntries 表示已经删除的死信文件数量。
	RemovedEntries int
	// RemovedBytes 表示已经删除的死信文件字节数。
	RemovedBytes int64
}

// PayloadPredicate 判断一条业务 payload 是否需要从持久队列中清理。
//
// 返回 true 表示删除；返回错误时本轮不删除任何 payload。
type PayloadPredicate func(payload []byte) (bool, error)

// Service 定义嵌入式持久 MQ 能力。
//
// 职责边界：
// 1. 负责持久消息的入队、领取、ack、重试、死信和停止生命周期；
// 2. 不理解 summary、trigger、memory 等业务语义；
// 3. 保证 at-least-once 投递，业务 owner 必须通过业务事实或 DedupKey 保证幂等。
//
// 错误语义：
// 1. 消费者配置非法或异步执行服务已停止时直接返回错误；
// 2. 入队失败返回底层错误，调用方决定是否改变业务状态；
// 3. Handler 返回错误会触发自动重试，超过 MaxAttempts 后进入 dead letter。
//
// 线程安全说明：
// 1. Service 内部串行化消费者索引与去重键访问；
// 2. 同一个 consumer 的本地分片数由 QueueCount 控制，总未 ack 并发由 ConsumerConfig.Concurrency 控制；
// 3. 同一分片同一时刻只会派发一条消息，调用方可通过 ShardKey 保证同一业务锚点串行。
// 4. StopConsumer 会等待该 consumer 已经派发的 worker 完成。
type Service interface {
	// StartConsumer 启动一个嵌入式 MQ 消费者；同名消费者已存在时会先停止并替换。
	StartConsumer(ctx context.Context, config ConsumerConfig, handler QueueHandler) error
	// StopConsumer 停止指定消费者。
	StopConsumer(name string)
	// Enqueue 向指定消费者写入一条 payload。
	Enqueue(ctx context.Context, input EnqueueInput) error
	// PurgePayloads 停止指定消费者，按业务 predicate 清理持久 payload 后恢复消费。
	//
	// 约束说明：该方法会等待已经派发的 worker 退出；predicate 只解释业务 payload，不接触 queue envelope。
	PurgePayloads(ctx context.Context, consumerName string, predicate PayloadPredicate) (int, error)
	// PurgeDeadLetters 按保留期、数量和字节上限清理所有已启动 consumer 的死信，不触碰 pending 或 inflight。
	PurgeDeadLetters(ctx context.Context, policy DeadLetterCleanupPolicy) (DeadLetterCleanupResult, error)
	// Depth 返回指定消费者底层队列当前深度。
	Depth(name string) int64
}
