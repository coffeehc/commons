//go:build !darwin && !dragonfly && !freebsd && !linux && !netbsd && !openbsd && !solaris && !windows

package localqueue

import "os"

// tryLockQueueFile 依靠进程内 registry 保护不具备文件锁实现的平台。
func tryLockQueueFile(_ *os.File) (bool, error) {
	return true, nil
}

// unlockQueueFile 在不具备文件锁实现的平台无需额外操作。
func unlockQueueFile(_ *os.File) error {
	return nil
}
