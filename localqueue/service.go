package localqueue

import "time"

const (
	defaultMaxBytesPerFile = int64(512 * 1024 * 1024)
	defaultMinMessageSize  = int32(4)
	defaultMaxMessageSize  = int32(4 * 1024 * 1024)
	defaultSyncEvery       = int64(100)
	defaultSyncTimeout     = 5 * time.Second
)

// Interface 定义进程内嵌入式磁盘 FIFO 队列。
//
// 实现支持并发生产者和消费者。消息在发送到 ReadChan 后即视为已消费，
// 不提供确认、重试或可见性超时；这些 MQ 语义不属于 localqueue 的职责。
// 同一个 queueDir 和 name 同一时间只能存在一个活跃实例，该约束同时在
// 进程内和支持文件锁的操作系统上生效。
type Interface interface {
	// Put 将一条消息写入队列。消息大小必须在 4 字节到 4 MiB 之间；
	// 队列初始化失败、已关闭或写入失败时返回错误。
	Put([]byte) error
	// ReadChan 返回用于消费消息的无缓冲只读 channel；初始化失败、
	// Close 或 Delete 完成后 channel 会关闭。
	ReadChan() <-chan []byte
	// Close 停止队列并同步尚未持久化的文件与元数据；重复调用返回首次结果。
	Close() error
	// Delete 停止队列并删除该队列拥有的数据文件。Close 之后也可以调用，
	// 但同名队列已经被其他实例打开时返回占用错误且不会删除文件。
	Delete() error
	// Depth 返回尚未发送给消费者的消息数量快照，不读取或修改磁盘状态。
	Depth() int64
	// Empty 删除当前全部待消费消息，队列随后仍可继续使用；初始化失败、
	// 已关闭或文件删除失败时返回错误。
	Empty() error
}

// queueConfig 保存磁盘队列内部运行参数。
type queueConfig struct {
	// maxBytesPerFile 是单个 segment 的最大字节数。
	maxBytesPerFile int64
	// minMessageSize 是允许写入的最小消息字节数。
	minMessageSize int32
	// maxMessageSize 是允许写入的最大消息字节数。
	maxMessageSize int32
	// syncEvery 是触发一次磁盘同步的读写操作数。
	syncEvery int64
	// syncTimeout 是两次定时磁盘同步之间的最长间隔。
	syncTimeout time.Duration
}

var defaultConfig = queueConfig{
	maxBytesPerFile: defaultMaxBytesPerFile,
	minMessageSize:  defaultMinMessageSize,
	maxMessageSize:  defaultMaxMessageSize,
	syncEvery:       defaultSyncEvery,
	syncTimeout:     defaultSyncTimeout,
}

// NewQueue 打开或创建 name 对应的磁盘队列。
//
// name 必须是单个安全文件名，queueDir 是队列文件所在目录。实例会持有独占
// ownership 直到 Close 或 Delete；锁文件会保留以避免解锁与删除之间的竞争。
// 构造阶段的错误由首次 Put、Empty、Close 或 Delete 返回，ReadChan 会直接关闭。
func NewQueue(name, queueDir string) Interface {
	return newDiskQueue(name, queueDir, defaultConfig)
}
