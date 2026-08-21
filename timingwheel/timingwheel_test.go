package timingwheel

import (
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"
)

func TestAfterFuncTriggersOnce(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		wheel := New()
		defer wheel.Stop()
		var count atomic.Int32
		wheel.AfterFunc(time.Hour, func() { count.Add(1) })

		time.Sleep(time.Hour - time.Nanosecond)
		synctest.Wait()
		if got := count.Load(); got != 0 {
			t.Fatalf("callback count before delay = %d, want 0", got)
		}

		time.Sleep(time.Nanosecond)
		synctest.Wait()
		if got := count.Load(); got != 1 {
			t.Fatalf("callback count after delay = %d, want 1", got)
		}
	})
}

func TestTimerStopIsIdempotent(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		wheel := New()
		defer wheel.Stop()
		var called atomic.Bool
		timer := wheel.AfterFunc(time.Hour, func() { called.Store(true) })

		if !timer.Stop() {
			t.Fatal("first Stop returned false, want true")
		}
		if timer.Stop() {
			t.Fatal("second Stop returned true, want false")
		}
		time.Sleep(time.Hour)
		synctest.Wait()
		if called.Load() {
			t.Fatal("stopped timer invoked callback")
		}
	})
}

func TestScheduleStopsFutureTriggers(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		wheel := New()
		defer wheel.Stop()
		var count atomic.Int32
		timer := wheel.Schedule(time.Minute, func() { count.Add(1) })

		time.Sleep(3 * time.Minute)
		synctest.Wait()
		if got := count.Load(); got != 3 {
			t.Fatalf("callback count before Stop = %d, want 3", got)
		}
		if !timer.Stop() {
			t.Fatal("Stop returned false for active schedule")
		}

		time.Sleep(3 * time.Minute)
		synctest.Wait()
		if got := count.Load(); got != 3 {
			t.Fatalf("callback count after Stop = %d, want 3", got)
		}
	})
}

func TestWheelStopPreventsCurrentAndFutureTimers(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		wheel := New()
		var count atomic.Int32
		wheel.Schedule(time.Minute, func() { count.Add(1) })
		wheel.Stop()
		stoppedTimer := wheel.AfterFunc(time.Minute, func() { count.Add(1) })

		time.Sleep(2 * time.Minute)
		synctest.Wait()
		if got := count.Load(); got != 0 {
			t.Fatalf("callback count after Wheel.Stop = %d, want 0", got)
		}
		if stoppedTimer.Stop() {
			t.Fatal("timer registered after Wheel.Stop was active")
		}
	})
}
