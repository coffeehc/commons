// Package timingwheel provides lifecycle-managed delayed and periodic timers.
package timingwheel

import (
	"sync"
	"time"
)

// Timer represents one delayed or periodic callback registered with a Wheel.
// Timer is safe for concurrent use.
type Timer struct {
	// mutex protects the timer lifecycle fields below.
	mutex sync.Mutex
	// wheel owns this timer and removes it when the timer becomes inactive.
	wheel *Wheel
	// runtimeTimer is the current standard-library timer used for the next trigger.
	runtimeTimer *time.Timer
	// task is invoked whenever the timer fires while active.
	task func()
	// interval is zero for a one-shot timer and positive for a periodic timer.
	interval time.Duration
	// active reports whether another callback may still be triggered.
	active bool
}

// Stop prevents future callbacks. It returns true only when this call changes
// an active timer to stopped. A callback that has already started is not waited.
func (timer *Timer) Stop() bool {
	return timer.stop(true)
}

// stop changes the timer state before touching the runtime timer so a callback
// racing with Stop can observe the cancellation under the same mutex.
func (timer *Timer) stop(removeFromWheel bool) bool {
	timer.mutex.Lock()
	if !timer.active {
		timer.mutex.Unlock()
		return false
	}
	timer.active = false
	runtimeTimer := timer.runtimeTimer
	timer.runtimeTimer = nil
	timer.mutex.Unlock()

	if runtimeTimer != nil {
		runtimeTimer.Stop()
	}
	if removeFromWheel {
		timer.wheel.remove(timer)
	}
	return true
}

// start activates the underlying runtime timer unless Wheel.Stop won the race.
func (timer *Timer) start(delay time.Duration) {
	timer.mutex.Lock()
	if timer.active {
		timer.runtimeTimer = time.AfterFunc(delay, timer.fire)
	}
	timer.mutex.Unlock()
}

// fire registers a periodic timer's next trigger before invoking its callback.
func (timer *Timer) fire() {
	timer.mutex.Lock()
	if !timer.active {
		timer.mutex.Unlock()
		return
	}
	periodic := timer.interval > 0
	if periodic {
		timer.runtimeTimer = time.AfterFunc(timer.interval, timer.fire)
	} else {
		timer.active = false
		timer.runtimeTimer = nil
	}
	timer.mutex.Unlock()

	if !periodic {
		timer.wheel.remove(timer)
	}
	timer.task()
}

// Wheel owns delayed and periodic timers so they can be stopped together.
// Wheel is safe for concurrent use and starts accepting timers immediately.
type Wheel struct {
	// mutex protects timers and stopped.
	mutex sync.Mutex
	// timers contains every timer that may still trigger a callback.
	timers map[*Timer]struct{}
	// stopped prevents new timers from being activated after shutdown.
	stopped bool
}

// New creates an active Wheel without starting any background goroutine.
func New() *Wheel {
	return &Wheel{timers: make(map[*Timer]struct{})}
}

// AfterFunc invokes task once after delay. A non-positive delay invokes task as
// soon as the runtime can schedule it. A nil task causes a panic.
func (wheel *Wheel) AfterFunc(delay time.Duration, task func()) *Timer {
	return wheel.newTimer(delay, 0, task)
}

// Schedule invokes task periodically. The next trigger is registered before
// the current callback runs, so a slow callback does not stop future triggers.
// A non-positive interval or nil task causes a panic.
func (wheel *Wheel) Schedule(interval time.Duration, task func()) *Timer {
	if interval <= 0 {
		panic("timingwheel: interval must be positive")
	}
	return wheel.newTimer(interval, interval, task)
}

// Stop prevents all registered timers and all future registrations from
// triggering callbacks. Callbacks that have already started are not waited.
func (wheel *Wheel) Stop() {
	wheel.mutex.Lock()
	if wheel.stopped {
		wheel.mutex.Unlock()
		return
	}
	wheel.stopped = true
	timers := make([]*Timer, 0, len(wheel.timers))
	for timer := range wheel.timers {
		timers = append(timers, timer)
	}
	clear(wheel.timers)
	wheel.mutex.Unlock()

	for _, timer := range timers {
		timer.stop(false)
	}
}

// newTimer publishes timer ownership before activation so Wheel.Stop can always
// find and cancel a concurrently registered timer.
func (wheel *Wheel) newTimer(delay time.Duration, interval time.Duration, task func()) *Timer {
	if task == nil {
		panic("timingwheel: task must not be nil")
	}
	timer := &Timer{wheel: wheel, task: task, interval: interval, active: true}

	wheel.mutex.Lock()
	if wheel.stopped {
		timer.active = false
		wheel.mutex.Unlock()
		return timer
	}
	wheel.timers[timer] = struct{}{}
	wheel.mutex.Unlock()

	timer.start(delay)
	return timer
}

// remove drops an inactive timer from the wheel ownership set.
func (wheel *Wheel) remove(timer *Timer) {
	wheel.mutex.Lock()
	delete(wheel.timers, timer)
	wheel.mutex.Unlock()
}
