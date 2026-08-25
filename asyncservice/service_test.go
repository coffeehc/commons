package asyncservice

import (
	"bytes"
	"context"
	"errors"
	"runtime"
	"runtime/pprof"
	"strings"
	"testing"
	"time"
)

func TestPoolLimitAndDynamicResize(t *testing.T) {
	impl := newTestService(t, 1)
	firstStarted := make(chan struct{})
	secondStarted := make(chan struct{})
	release := make(chan struct{})

	impl.Submit(func() {
		close(firstStarted)
		<-release
	})
	waitSignal(t, firstStarted)
	impl.Submit(func() {
		close(secondStarted)
		<-release
	})
	waitPoolStatus(t, impl, func(status PoolStatus) bool {
		return status.Running == 1 && status.Waiting == 1 && status.Available == 0
	})

	impl.ChangePoolSize(2)
	waitSignal(t, secondStarted)
	status := impl.PoolStatus()
	if status.Limit != 2 || status.Running != 2 || status.Waiting != 0 || status.Available != 0 {
		t.Fatalf("status after resize = %+v", status)
	}
	close(release)
}

func TestPoolShrinkAppliesAfterRunningTasksFinish(t *testing.T) {
	impl := newTestService(t, 2)
	firstStarted := make(chan struct{})
	secondStarted := make(chan struct{})
	thirdStarted := make(chan struct{})
	releaseFirst := make(chan struct{})
	releaseSecond := make(chan struct{})

	impl.Submit(func() {
		close(firstStarted)
		<-releaseFirst
	})
	impl.Submit(func() {
		close(secondStarted)
		<-releaseSecond
	})
	waitSignal(t, firstStarted)
	waitSignal(t, secondStarted)
	impl.ChangePoolSize(1)
	impl.Submit(func() { close(thirdStarted) })
	close(releaseFirst)
	waitPoolStatus(t, impl, func(status PoolStatus) bool {
		return status.Running == 1 && status.Waiting == 1
	})
	select {
	case <-thirdStarted:
		t.Fatal("queued task started while running count still matched reduced limit")
	default:
	}
	close(releaseSecond)
	waitSignal(t, thirdStarted)
}

func TestSubmitAllowsNestedTaskAtPoolLimit(t *testing.T) {
	impl := newTestService(t, 1)
	outerDone := make(chan struct{})
	nestedDone := make(chan struct{})

	impl.Submit(func() {
		impl.Submit(func() { close(nestedDone) })
		close(outerDone)
	})

	waitSignal(t, outerDone)
	waitSignal(t, nestedDone)
}

func TestStopDrainsAcceptedTasksAndRejectsNewTasks(t *testing.T) {
	impl := newTestService(t, 1)
	started := make(chan struct{})
	release := make(chan struct{})
	accepted := make(chan struct{})
	rejected := make(chan struct{})
	impl.Submit(func() {
		close(started)
		<-release
	})
	waitSignal(t, started)
	impl.Submit(func() { close(accepted) })

	stopResult := make(chan error, 1)
	go func() {
		stopResult <- impl.Stop(context.Background())
	}()
	waitPoolStatus(t, impl, func(status PoolStatus) bool { return status.Stopped })
	impl.Submit(func() { close(rejected) })
	close(release)

	if err := waitResult(t, stopResult); err != nil {
		t.Fatalf("Stop returned error: %v", err)
	}
	waitSignal(t, accepted)
	select {
	case <-rejected:
		t.Fatal("task submitted after Stop was executed")
	default:
	}
	status := impl.PoolStatus()
	if status.Running != 0 || status.Waiting != 0 || !status.Stopped {
		t.Fatalf("status after Stop = %+v", status)
	}
}

func TestStopHonorsContextWhileTaskContinuesDraining(t *testing.T) {
	impl := newTestService(t, 1)
	started := make(chan struct{})
	release := make(chan struct{})
	impl.Submit(func() {
		close(started)
		<-release
	})
	waitSignal(t, started)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := impl.Stop(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("Stop error = %v, want context.Canceled", err)
	}
	close(release)
	if err := impl.Stop(context.Background()); err != nil {
		t.Fatalf("second Stop returned error: %v", err)
	}
}

func TestTaskPanicDoesNotStopPool(t *testing.T) {
	impl := newTestService(t, 1)
	afterPanic := make(chan struct{})
	impl.Submit(func() { panic("test panic") })
	impl.Submit(func() { close(afterPanic) })
	waitSignal(t, afterPanic)
}

func TestChangePoolSizeRejectsNonPositiveLimit(t *testing.T) {
	impl := newTestService(t, 2)
	impl.ChangePoolSize(0)
	if status := impl.PoolStatus(); status.Limit != 2 {
		t.Fatalf("limit after invalid resize = %d, want 2", status.Limit)
	}
}

func TestSuspendDrainsAcceptedTasksAndHoldsLaterSubmissions(t *testing.T) {
	impl := newTestService(t, 1)
	controller := Controller(impl)
	firstStarted := make(chan struct{})
	releaseFirst := make(chan struct{})
	acceptedBeforeSuspend := make(chan struct{})
	heldAfterSuspend := make(chan struct{})
	impl.Submit(func() {
		close(firstStarted)
		<-releaseFirst
	})
	impl.Submit(func() { close(acceptedBeforeSuspend) })
	waitSignal(t, firstStarted)

	controller.Suspend()
	impl.Submit(func() { close(heldAfterSuspend) })
	close(releaseFirst)
	waitSignal(t, acceptedBeforeSuspend)
	if err := controller.WaitIdle(t.Context()); err != nil {
		t.Fatalf("WaitIdle returned error: %v", err)
	}
	select {
	case <-heldAfterSuspend:
		t.Fatal("task submitted after suspension started before Resume")
	default:
	}
	status := impl.PoolStatus()
	if !status.Paused || status.Running != 0 || status.Waiting != 1 {
		t.Fatalf("status while suspended = %+v", status)
	}

	controller.Resume()
	waitSignal(t, heldAfterSuspend)
}

func TestControlTaskBypassesSuspendedPool(t *testing.T) {
	impl := newTestService(t, 1)
	controller := Controller(impl)
	controller.Suspend()
	controlDone := make(chan struct{})
	if !controller.SubmitControl(func() { close(controlDone) }) {
		t.Fatal("control task was rejected before shutdown")
	}
	waitSignal(t, controlDone)
	if controller.SubmitIfRunning(func() {}) {
		t.Fatal("SubmitIfRunning accepted a task while suspended")
	}
	controller.Resume()
}

func TestStopLeavesNoLeakedDispatcher(t *testing.T) {
	impl := newTestService(t, 1)
	if err := impl.Stop(context.Background()); err != nil {
		t.Fatalf("Stop returned error: %v", err)
	}
	runtime.GC()
	profile := pprof.Lookup("goroutineleak")
	if profile == nil {
		t.Fatal("Go 1.27 goroutineleak profile is unavailable")
	}
	var output bytes.Buffer
	if err := profile.WriteTo(&output, 1); err != nil {
		t.Fatalf("write goroutineleak profile: %v", err)
	}
	if strings.Contains(output.String(), "asyncservice.(*taskPool).dispatch") {
		t.Fatalf("dispatcher remained in goroutineleak profile:\n%s", output.String())
	}
}

func newTestService(t *testing.T, poolSize int) *serviceImpl {
	t.Helper()
	impl := NewService(t.Context(), &Config{PoolSize: poolSize}).(*serviceImpl)
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		if err := impl.Stop(ctx); err != nil {
			t.Errorf("cleanup Stop returned error: %v", err)
		}
	})
	return impl
}

func waitSignal(t *testing.T, signal <-chan struct{}) {
	t.Helper()
	select {
	case <-signal:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for signal")
	}
}

func waitResult(t *testing.T, result <-chan error) error {
	t.Helper()
	select {
	case err := <-result:
		return err
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for result")
		return nil
	}
}

func waitPoolStatus(t *testing.T, impl *serviceImpl, matches func(PoolStatus) bool) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if matches(impl.PoolStatus()) {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("timed out waiting for pool status, last status = %+v", impl.PoolStatus())
}
