//go:build darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris

package localqueue

import (
	"errors"
	"os"

	"golang.org/x/sys/unix"
)

// tryLockQueueFile 尝试在 Unix 系统取得非阻塞独占文件锁。
func tryLockQueueFile(file *os.File) (bool, error) {
	err := unix.Flock(int(file.Fd()), unix.LOCK_EX|unix.LOCK_NB)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, unix.EWOULDBLOCK) || errors.Is(err, unix.EAGAIN) {
		return false, nil
	}
	return false, err
}

// unlockQueueFile 释放 Unix 独占文件锁。
func unlockQueueFile(file *os.File) error {
	return unix.Flock(int(file.Fd()), unix.LOCK_UN)
}
