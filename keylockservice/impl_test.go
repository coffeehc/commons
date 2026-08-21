package keylockservice

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestTryLockPreservesTokenOwnership(t *testing.T) {
	service := newService(context.Background())
	firstToken, ok := service.TryLock("key")
	if !ok || firstToken == "" {
		t.Fatal("第一次 TryLock 应成功并返回 token")
	}
	failedToken, ok := service.TryLock("key")
	if ok || failedToken == "" {
		t.Fatal("锁已占用时 TryLock 应失败并保持 token 返回格式")
	}
	service.UnLock("key", failedToken)
	if _, ok = service.TryLock("key"); ok {
		t.Fatal("错误 token 不应释放当前锁")
	}
	service.UnLock("key", firstToken)
	secondToken, ok := service.TryLock("key")
	if !ok {
		t.Fatal("正确 token 释放后应能重新获取锁")
	}
	service.UnLock("key", secondToken)
}

func TestLockWakesWaitersInArrivalOrder(t *testing.T) {
	impl := newService(context.Background()).(*serviceImpl)
	ownerToken, err := impl.Lock(context.Background(), "key")
	if err != nil {
		t.Fatalf("获取初始锁失败: %v", err)
	}

	firstResult := make(chan string, 1)
	secondResult := make(chan string, 1)
	go lockForTest(impl, "key", firstResult)
	waitForWaiterCount(t, impl, "key", 1)
	go lockForTest(impl, "key", secondResult)
	waitForWaiterCount(t, impl, "key", 2)

	impl.UnLock("key", ownerToken)
	firstToken := waitForToken(t, firstResult)
	select {
	case <-secondResult:
		t.Fatal("第一位等待者释放前不应向第二位等待者移交锁")
	default:
	}
	impl.UnLock("key", firstToken)
	secondToken := waitForToken(t, secondResult)
	impl.UnLock("key", secondToken)
}

func TestLockCancellationRemovesWaiter(t *testing.T) {
	impl := newService(context.Background()).(*serviceImpl)
	ownerToken, err := impl.Lock(context.Background(), "key")
	if err != nil {
		t.Fatalf("获取初始锁失败: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	result := make(chan error, 1)
	go func() {
		_, lockErr := impl.Lock(ctx, "key")
		result <- lockErr
	}()
	waitForWaiterCount(t, impl, "key", 1)
	cancel()
	if err = waitForError(t, result); !errors.Is(err, context.Canceled) {
		t.Fatalf("Lock 返回错误 = %v, 期望 context.Canceled", err)
	}
	waitForWaiterCount(t, impl, "key", 0)
	impl.UnLock("key", ownerToken)
	token, ok := impl.TryLock("key")
	if !ok {
		t.Fatal("取消的等待者不应阻止后续获取锁")
	}
	impl.UnLock("key", token)
}

func TestCancellationAndHandoverRaceDoesNotLeakLock(t *testing.T) {
	impl := newService(context.Background()).(*serviceImpl)
	type lockResult struct {
		token string
		err   error
	}
	for range 100 {
		ownerToken, err := impl.Lock(context.Background(), "key")
		if err != nil {
			t.Fatalf("获取初始锁失败: %v", err)
		}
		ctx, cancel := context.WithCancel(context.Background())
		result := make(chan lockResult, 1)
		go func() {
			token, lockErr := impl.Lock(ctx, "key")
			result <- lockResult{token: token, err: lockErr}
		}()
		waitForWaiterCount(t, impl, "key", 1)

		var race sync.WaitGroup
		race.Go(cancel)
		race.Go(func() {
			impl.UnLock("key", ownerToken)
		})
		race.Wait()
		select {
		case acquired := <-result:
			if acquired.err == nil {
				impl.UnLock("key", acquired.token)
			} else if !errors.Is(acquired.err, context.Canceled) {
				t.Fatalf("Lock 返回错误 = %v, 期望成功或 context.Canceled", acquired.err)
			}
		case <-time.After(time.Second):
			t.Fatal("取消与移交竞争后 Lock 未返回")
		}

		token, ok := impl.TryLock("key")
		if !ok {
			t.Fatal("取消与移交竞争后锁状态未释放")
		}
		impl.UnLock("key", token)
	}
}

func TestLockWithTimeoutTransfersOwnership(t *testing.T) {
	impl := newService(context.Background()).(*serviceImpl)
	if _, err := impl.LockWithTimeout(context.Background(), "key", 20*time.Millisecond); err != nil {
		t.Fatalf("获取定时锁失败: %v", err)
	}
	result := make(chan string, 1)
	go lockForTest(impl, "key", result)
	waitForWaiterCount(t, impl, "key", 1)
	token := waitForToken(t, result)
	impl.UnLock("key", token)
}

func TestManualUnlockStopsPreviousTimeout(t *testing.T) {
	impl := newService(context.Background()).(*serviceImpl)
	ownerToken, err := impl.LockWithTimeout(context.Background(), "key", 30*time.Millisecond)
	if err != nil {
		t.Fatalf("获取定时锁失败: %v", err)
	}
	result := make(chan string, 1)
	go lockForTest(impl, "key", result)
	waitForWaiterCount(t, impl, "key", 1)
	impl.UnLock("key", ownerToken)
	waiterToken := waitForToken(t, result)
	time.Sleep(60 * time.Millisecond)
	if token, ok := impl.TryLock("key"); ok {
		impl.UnLock("key", token)
		t.Fatal("前一任持锁者的 timer 不应释放当前锁")
	}
	impl.UnLock("key", waiterToken)
}

func TestDifferentKeysCanRunConcurrently(t *testing.T) {
	service := newService(context.Background())
	firstToken, err := service.Lock(context.Background(), "first")
	if err != nil {
		t.Fatalf("获取 first 锁失败: %v", err)
	}
	secondToken, err := service.Lock(context.Background(), "second")
	if err != nil {
		t.Fatalf("获取 second 锁失败: %v", err)
	}
	service.UnLock("second", secondToken)
	service.UnLock("first", firstToken)
}

func TestInvalidKeyPanicDoesNotPoisonService(t *testing.T) {
	impl := newService(context.Background()).(*serviceImpl)
	func() {
		defer func() {
			if recover() == nil {
				t.Fatal("不可比较 key 应触发 panic")
			}
		}()
		_, _ = impl.Lock(context.Background(), []string{"invalid"})
	}()
	token, ok := impl.TryLock("valid")
	if !ok {
		t.Fatal("非法 key panic 后服务仍应能处理其他 key")
	}
	impl.UnLock("valid", token)
}

func TestHighContentionKeepsSingleOwner(t *testing.T) {
	impl := newService(context.Background()).(*serviceImpl)
	var tasks sync.WaitGroup
	var active atomic.Int32
	var overlap atomic.Bool
	for range 100 {
		tasks.Go(func() {
			token, err := impl.Lock(context.Background(), "key")
			if err != nil {
				t.Errorf("获取锁失败: %v", err)
				return
			}
			if active.Add(1) != 1 {
				overlap.Store(true)
			}
			time.Sleep(time.Microsecond)
			active.Add(-1)
			impl.UnLock("key", token)
		})
	}
	tasks.Wait()
	if overlap.Load() {
		t.Fatal("同一 key 出现多个并发持锁者")
	}
	remaining := 0
	impl.locks.Range(func(_, _ interface{}) bool {
		remaining++
		return true
	})
	if remaining != 0 {
		t.Fatalf("全部释放后仍保留 %d 个锁状态", remaining)
	}
}

func lockForTest(service Service, key interface{}, result chan<- string) {
	token, err := service.Lock(context.Background(), key)
	if err != nil {
		result <- ""
		return
	}
	result <- token
}

func waitForWaiterCount(t *testing.T, impl *serviceImpl, key interface{}, expected int) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for {
		count := 0
		if value, exists := impl.locks.Load(key); exists {
			state := value.(*lockState)
			state.mutex.Lock()
			count = state.waiters.Len()
			state.mutex.Unlock()
		}
		if count == expected {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("等待者数量 = %d, 期望 %d", count, expected)
		}
		time.Sleep(time.Millisecond)
	}
}

func waitForToken(t *testing.T, result <-chan string) string {
	t.Helper()
	select {
	case token := <-result:
		if token == "" {
			t.Fatal("获取锁未返回 token")
		}
		return token
	case <-time.After(time.Second):
		t.Fatal("等待获取锁超时")
		return ""
	}
}

func waitForError(t *testing.T, result <-chan error) error {
	t.Helper()
	select {
	case err := <-result:
		return err
	case <-time.After(time.Second):
		t.Fatal("等待 Lock 返回超时")
		return nil
	}
}
