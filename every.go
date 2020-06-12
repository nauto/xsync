package xsync

import (
	"sync"
	"time"
)

// AtMost can be used to execute a function at most once in the specified duration.
// It's thread-safe to call this from multiple go-routines.
//
// AtMost tries to keep the critical region as small as possible. As a result,
// if the specified function takes too long to run compared to the duration,
// another copy of f may be started while the first one is running.
type AtMost struct {
	mu      sync.RWMutex
	lastRun time.Time
	Every   time.Duration
}

// Run will execute the given function at most once in the specified duration.
func (a *AtMost) Run(f func()) {
	now := time.Now()

	// Get a copy of lastRun and check if it has been enough time since then.
	a.mu.RLock()
	lastRun := a.lastRun
	a.mu.RUnlock()

	if now.Sub(lastRun) < a.Every {
		return
	}

	// Acquire a write lock, and check the condition again (Another thread may
	// have gotten in between)
	a.mu.Lock()
	if now.Sub(a.lastRun) < a.Every {
		a.mu.Unlock()
		return
	}
	// Reset lastRun and unlock as soon as possible before calling f (in order
	// to keep the critical region small).
	a.lastRun = now
	a.mu.Unlock()

	f()
}
