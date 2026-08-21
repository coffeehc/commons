package localqueue

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

const lockFileSuffix = ".diskqueue.lock"

// queueOwnership 保存一个活跃队列实例的进程内注册和操作系统文件锁。
type queueOwnership struct {
	// key 是规范化后的锁文件绝对路径，也是进程内 owner 唯一键。
	key string
	// file 持有跨进程独占锁；锁文件本身会保留以避免删除与重建之间的竞争。
	file *os.File
}

// queueOwnershipRegistry 保存当前进程已经打开的队列。
type queueOwnershipRegistry struct {
	// mutex 串行化 owner 的注册和释放。
	mutex sync.Mutex
	// owners 以规范化锁路径记录活跃 owner。
	owners map[string]struct{}
}

var activeQueueOwnerships = &queueOwnershipRegistry{owners: make(map[string]struct{})}

// acquireQueueOwnership 为一个队列取得进程内和跨进程独占 ownership。
func acquireQueueOwnership(name, queueDir string) (*queueOwnership, error) {
	lockPath := filepath.Join(queueDir, name+lockFileSuffix)
	activeQueueOwnerships.mutex.Lock()
	if _, exists := activeQueueOwnerships.owners[lockPath]; exists {
		activeQueueOwnerships.mutex.Unlock()
		return nil, fmt.Errorf("%w: %s", errQueueInUse, lockPath)
	}
	activeQueueOwnerships.owners[lockPath] = struct{}{}
	activeQueueOwnerships.mutex.Unlock()

	releaseRegistry := true
	defer func() {
		if releaseRegistry {
			activeQueueOwnerships.release(lockPath)
		}
	}()
	file, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, fmt.Errorf("打开队列 ownership 文件失败: %w", err)
	}
	locked, err := tryLockQueueFile(file)
	if err != nil {
		_ = file.Close()
		return nil, fmt.Errorf("锁定队列 ownership 文件失败: %w", err)
	}
	if !locked {
		_ = file.Close()
		return nil, fmt.Errorf("%w: %s", errQueueInUse, lockPath)
	}
	releaseRegistry = false
	return &queueOwnership{key: lockPath, file: file}, nil
}

// release 释放跨进程文件锁，并在文件关闭后释放进程内 owner 注册。
func (ownership *queueOwnership) release() error {
	if ownership == nil {
		return nil
	}
	unlockErr := unlockQueueFile(ownership.file)
	closeErr := ownership.file.Close()
	activeQueueOwnerships.release(ownership.key)
	return errors.Join(unlockErr, closeErr)
}

// release 删除一个进程内 owner 注册。
func (registry *queueOwnershipRegistry) release(key string) {
	registry.mutex.Lock()
	delete(registry.owners, key)
	registry.mutex.Unlock()
}
