package localqueue

import (
	"errors"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/coffeehc/base/log"
	"go.uber.org/zap"
)

var errQueueClosed = errors.New("exiting")
var errQueueInUse = errors.New("队列已被其他实例占用")

// putRequest 表示一次串行化写入请求。
type putRequest struct {
	// message 是调用方提交的消息。
	message []byte
	// response 返回本次写入结果。
	response chan error
}

// emptyRequest 表示一次串行化清空请求。
type emptyRequest struct {
	// response 返回本次清空结果。
	response chan error
}

// stopRequest 表示一次串行化关闭或删除请求。
type stopRequest struct {
	// delete 表示关闭后是否同时删除队列文件。
	delete bool
	// response 返回本次关闭或删除结果。
	response chan error
}

// pendingMessage 保存已经读取但尚未交付的消息及其下一读取位置。
type pendingMessage struct {
	// payload 是等待交付给消费者的消息内容。
	payload []byte
	// nextFileNum 是交付完成后的读取 segment 编号。
	nextFileNum int64
	// nextPosition 是交付完成后的读取偏移。
	nextPosition int64
}

// diskQueue 通过单一 IO owner 串行化磁盘状态，同时允许 public 方法并发调用。
type diskQueue struct {
	// name 是队列名，也是磁盘文件名前缀。
	name string
	// queueDir 是队列文件所在目录。
	queueDir string
	// config 是队列运行参数。
	config queueConfig
	// ownership 保证相同目录和名称同一时间只有一个活跃 IO owner。
	ownership *queueOwnership

	// lifecycleMutex 阻止关闭过程与新的写入、清空操作并发进入。
	lifecycleMutex sync.RWMutex
	// closed 表示队列已经停止接收新操作。
	closed bool
	// deleted 表示队列文件已经删除。
	deleted bool
	// startErr 保存构造阶段发生的错误。
	startErr error
	// closeErr 保存首次关闭或删除的结果。
	closeErr error

	// depth 是 IO owner 发布给并发读取方的消息数量快照。
	depth atomic.Int64

	// readChannel 向消费者交付消息。
	readChannel chan []byte
	// putRequests 将并发写入请求串行交给 IO owner。
	putRequests chan putRequest
	// emptyRequests 将清空请求串行交给 IO owner。
	emptyRequests chan emptyRequest
	// stopRequests 将生命周期结束请求串行交给 IO owner。
	stopRequests chan stopRequest

	// 以下字段只允许 IO owner goroutine 访问。
	// readFileNum 是当前读取 segment 编号。
	readFileNum int64
	// readPosition 是当前读取偏移。
	readPosition int64
	// writeFileNum 是当前写入 segment 编号。
	writeFileNum int64
	// writePosition 是当前写入偏移。
	writePosition int64
	// readFile 是当前打开的读取文件。
	readFile *os.File
	// writeFile 是当前打开的写入文件。
	writeFile *os.File
	// writeBuffer 复用消息帧编码内存。
	writeBuffer []byte
	// dirty 表示存在尚未同步的状态。
	dirty bool
	// operationCount 是上次同步后的读写操作数。
	operationCount int64
}

var _ Interface = (*diskQueue)(nil)

func newDiskQueue(name, queueDir string, config queueConfig) Interface {
	queue := &diskQueue{
		name:          name,
		queueDir:      queueDir,
		config:        config,
		readChannel:   make(chan []byte),
		putRequests:   make(chan putRequest),
		emptyRequests: make(chan emptyRequest),
		stopRequests:  make(chan stopRequest),
	}
	if err := queue.initializeStorage(); err != nil {
		queue.startErr = errors.Join(err, queue.releaseOwnership())
		queue.closed = true
		close(queue.readChannel)
		return queue
	}
	go queue.run()
	return queue
}

// Put 将一条消息写入队列。
func (queue *diskQueue) Put(message []byte) error {
	queue.lifecycleMutex.RLock()
	defer queue.lifecycleMutex.RUnlock()
	if queue.startErr != nil {
		return queue.startErr
	}
	if queue.closed {
		return errQueueClosed
	}
	request := putRequest{message: message, response: make(chan error, 1)}
	queue.putRequests <- request
	return <-request.response
}

// ReadChan 返回用于消费消息的只读 channel。
func (queue *diskQueue) ReadChan() <-chan []byte {
	return queue.readChannel
}

// Close 停止队列并同步磁盘状态。
func (queue *diskQueue) Close() error {
	return queue.stop(false)
}

// Delete 停止队列并删除其磁盘文件。
func (queue *diskQueue) Delete() error {
	return queue.stop(true)
}

// Depth 返回尚未交付给消费者的消息数量快照。
func (queue *diskQueue) Depth() int64 {
	return queue.depth.Load()
}

// Empty 清空待消费消息并保持队列可用。
func (queue *diskQueue) Empty() error {
	queue.lifecycleMutex.RLock()
	defer queue.lifecycleMutex.RUnlock()
	if queue.startErr != nil {
		return queue.startErr
	}
	if queue.closed {
		return errQueueClosed
	}
	request := emptyRequest{response: make(chan error, 1)}
	queue.emptyRequests <- request
	return <-request.response
}

// stop 串行关闭或删除队列；Delete 在 Close 之后调用时仍会删除磁盘文件。
func (queue *diskQueue) stop(deleteQueue bool) error {
	queue.lifecycleMutex.Lock()
	defer queue.lifecycleMutex.Unlock()
	if queue.startErr != nil {
		return queue.startErr
	}
	if queue.closed {
		if deleteQueue && !queue.deleted {
			deleteErr := queue.deleteClosedQueue()
			queue.deleted = deleteErr == nil
			return errors.Join(queue.closeErr, deleteErr)
		}
		return queue.closeErr
	}
	request := stopRequest{delete: deleteQueue, response: make(chan error, 1)}
	queue.stopRequests <- request
	queue.closeErr = <-request.response
	queue.closed = true
	queue.deleted = deleteQueue && queue.closeErr == nil
	return queue.closeErr
}

// run 是队列唯一的 IO owner，负责读写、位置推进、同步和生命周期收口。
func (queue *diskQueue) run() {
	syncTicker := time.NewTicker(queue.config.syncTimeout)
	defer syncTicker.Stop()
	var pending *pendingMessage
	for {
		if queue.dirty && queue.operationCount >= queue.config.syncEvery {
			queue.syncAndLog()
		}
		if pending == nil && queue.depth.Load() > 0 {
			message, err := queue.readNextMessage()
			if err != nil {
				queue.handleReadFailure(err)
				continue
			}
			pending = message
		}
		var output chan []byte
		var payload []byte
		if pending != nil {
			output = queue.readChannel
			payload = pending.payload
		}

		select {
		case output <- payload:
			queue.commitRead(pending)
			pending = nil
		case request := <-queue.putRequests:
			err := queue.writeMessage(request.message)
			request.response <- err
		case request := <-queue.emptyRequests:
			pending = nil
			request.response <- queue.emptyStorage()
		case <-syncTicker.C:
			if queue.dirty {
				queue.syncAndLog()
			}
		case request := <-queue.stopRequests:
			closeErr := errors.Join(queue.shutdownStorage(request.delete), queue.releaseOwnership())
			close(queue.readChannel)
			request.response <- closeErr
			return
		}
	}
}

// deleteClosedQueue 重新取得独占 ownership 后删除已关闭队列的数据。
//
// Close 已经释放原 ownership；重新加锁可防止旧实例删除后来打开的新实例。
func (queue *diskQueue) deleteClosedQueue() error {
	ownership, err := acquireQueueOwnership(queue.name, queue.queueDir)
	if err != nil {
		return err
	}
	queue.ownership = ownership
	return errors.Join(queue.removeQueueFiles(), queue.releaseOwnership())
}

// releaseOwnership 释放当前队列持有的进程内注册和操作系统文件锁。
func (queue *diskQueue) releaseOwnership() error {
	if queue.ownership == nil {
		return nil
	}
	err := queue.ownership.release()
	queue.ownership = nil
	return err
}

func (queue *diskQueue) syncAndLog() {
	if err := queue.syncStorage(); err != nil {
		log.Error("同步本地磁盘队列失败",
			zap.String("queue", queue.name),
			zap.Error(err),
		)
	}
}

func (queue *diskQueue) validateMessage(message []byte) error {
	messageSize := len(message)
	if messageSize < int(queue.config.minMessageSize) || messageSize > int(queue.config.maxMessageSize) {
		return fmt.Errorf("消息大小 %d 超出允许范围 [%d, %d]", messageSize, queue.config.minMessageSize, queue.config.maxMessageSize)
	}
	return nil
}
