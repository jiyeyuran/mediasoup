package rtc

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestSafeTimer(t *testing.T) {
	t.Run("callback is called after duration", func(t *testing.T) {
		var wg sync.WaitGroup
		wg.Add(1)

		callbackCalled := false
		timer := NewSafeTimer(50*time.Millisecond, func() {
			callbackCalled = true
			wg.Done()
		})

		wg.Wait()
		require.True(t, callbackCalled)
		require.False(t, timer.IsActive())
	})

	t.Run("stop prevents callback from being called", func(t *testing.T) {
		callbackCalled := false
		timer := NewSafeTimer(100*time.Millisecond, func() {
			callbackCalled = true
		})

		stopped := timer.Stop()
		require.True(t, stopped)
		require.False(t, callbackCalled)
		require.False(t, timer.IsActive())
	})

	t.Run("reset restarts the timer", func(t *testing.T) {
		var wg sync.WaitGroup
		wg.Add(1)

		callbackCalled := false
		timer := NewSafeTimer(50*time.Millisecond, func() {
			callbackCalled = true
			wg.Done()
		})

		timer.Reset(100 * time.Millisecond)
		wg.Wait()
		require.True(t, callbackCalled)
		require.False(t, timer.IsActive())
	})

	t.Run("reset after stop restarts the timer", func(t *testing.T) {
		var wg sync.WaitGroup
		wg.Add(1)

		callbackCalled := false
		timer := NewSafeTimer(50*time.Millisecond, func() {
			callbackCalled = true
			wg.Done()
		})

		stopped := timer.Stop()
		require.True(t, stopped)
		require.False(t, callbackCalled)
		require.False(t, timer.IsActive())

		timer.Reset(50 * time.Millisecond)
		wg.Wait()
		require.True(t, callbackCalled)
		require.False(t, timer.IsActive())
	})

	t.Run("isActive returns correct state", func(t *testing.T) {
		timer := NewSafeTimer(100*time.Millisecond, func() {})

		require.True(t, timer.IsActive())
		timer.Stop()
		require.False(t, timer.IsActive())

		timer.Reset(100 * time.Millisecond)
		require.True(t, timer.IsActive())
	})

	t.Run("stop returns false if timer already expired", func(t *testing.T) {
		var wg sync.WaitGroup
		wg.Add(1)

		timer := NewSafeTimer(50*time.Millisecond, func() {
			wg.Done()
		})

		wg.Wait()
		stopped := timer.Stop()
		require.False(t, stopped)
		require.False(t, timer.IsActive())
	})
}
