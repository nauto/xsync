package xsync

// Author: Mykyta Nikitenko.

import (
	"context"

	"golang.org/x/sync/semaphore"
)

// LimitedWaitGroup is similar to sync.WaitGroup, but allows for a limit for
// the number of concurrently running routines. Essentially it provides an
// interface that combines Semaphore and WaitGroup semantics.
//
// Its usage is similar, but the Add method now blocks. The number of routines
// executing is controlled by the Add method.
//
// Example:
// 	wg := NewLimitedWaitGroup(100)
//
//	for i := 0; i < 1000; i++ {
//		wg.Add(1)
//
//		go func() {
//			defer wg.Done()
//
//			// your code
//		}()
//	}
//
//	wg.Wait()
type LimitedWaitGroup struct {
	semaphore *semaphore.Weighted
	limit     int64
}

func NewLimitedWaitGroup(limit int) *LimitedWaitGroup {
	return &LimitedWaitGroup{
		semaphore: semaphore.NewWeighted(int64(limit)),
		limit:     int64(limit),
	}
}

// Done is similar to sync.WaitGroup.Done. It releases a resource back to the
// pool.
func (wg *LimitedWaitGroup) Done() {
	wg.semaphore.Release(1)
}

// Wait is similar to sync.WaitGroup.Wait. It waits till all acquired resources
// have been released.
func (wg *LimitedWaitGroup) Wait() {
	_ = wg.semaphore.Acquire(ctx, wg.limit)
}

// Add uses a resource. It blocks if there are no resources are available.
// Resources can be released by calls to Done().
func (wg *LimitedWaitGroup) Add(n int64) {
	_ = wg.semaphore.Acquire(ctx, n)
}

var ctx = context.Background()
