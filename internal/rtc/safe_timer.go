package rtc

import (
	"sync"
	"time"
)

type SafeTimer struct {
	timer    *time.Timer
	active   bool
	mu       sync.Mutex
	callback func() // Define a callback function
}

// NewSafeTimer creates and starts a new SafeTimer with the given duration and callback.
func NewSafeTimer(duration time.Duration, cb func()) *SafeTimer {
	st := &SafeTimer{
		timer:    time.NewTimer(duration),
		active:   true,
		callback: cb,
	}
	go st.waitTimer()
	return st
}

// waitTimer waits for the timer to expire, calls the callback, and sets the active flag to false.
func (st *SafeTimer) waitTimer() {
	<-st.timer.C
	st.mu.Lock()
	if st.active {
		st.active = false
	}
	st.mu.Unlock()
	if st.callback != nil {
		st.callback() // Execute the callback function
	}
}

// Stop stops the timer and returns whether it was stopped before firing.
func (st *SafeTimer) Stop() bool {
	st.mu.Lock()
	defer st.mu.Unlock()
	stopped := st.timer.Stop()
	if !stopped && st.active {
		<-st.timer.C // Ensure that the channel is drained.
	}
	st.active = false
	return stopped
}

// Reset resets the timer to a new duration.
func (st *SafeTimer) Reset(duration time.Duration) {
	st.mu.Lock()
	defer st.mu.Unlock()
	if !st.timer.Stop() && st.active {
		<-st.timer.C // Drain the channel if the timer already expired.
	}
	st.timer.Reset(duration)
	st.active = true
}

// IsActive returns the current activity state of the timer.
func (st *SafeTimer) IsActive() bool {
	st.mu.Lock()
	defer st.mu.Unlock()
	return st.active
}
