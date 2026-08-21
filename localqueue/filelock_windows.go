//go:build windows

package localqueue

import (
	"errors"
	"os"

	"golang.org/x/sys/windows"
)

// tryLockQueueFile 尝试在 Windows 系统取得非阻塞独占文件锁。
func tryLockQueueFile(file *os.File) (bool, error) {
	overlapped := &windows.Overlapped{}
	err := windows.LockFileEx(
		windows.Handle(file.Fd()),
		windows.LOCKFILE_EXCLUSIVE_LOCK|windows.LOCKFILE_FAIL_IMMEDIATELY,
		0,
		1,
		0,
		overlapped,
	)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, windows.ERROR_LOCK_VIOLATION) {
		return false, nil
	}
	return false, err
}

// unlockQueueFile 释放 Windows 独占文件锁。
func unlockQueueFile(file *os.File) error {
	return windows.UnlockFileEx(windows.Handle(file.Fd()), 0, 1, 0, &windows.Overlapped{})
}
