// +build unit !integration

package xsync

import (
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWait(t *testing.T) {
	wg := NewLimitedWaitGroup(10)
	const routines = 10000
	var executed uint32

	for i := 0; i < routines; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()

			atomic.AddUint32(&executed, 1)
		}()
	}

	wg.Wait()

	assert.Equal(t, uint32(routines), executed, "not all routines executed")
}

func TestConcurrencyLimit(t *testing.T) {
	const limit = 4
	var executing int32

	wg := NewLimitedWaitGroup(limit)

	for i := 0; i < 10000; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()
			defer atomic.AddInt32(&executing, -1)

			atomic.AddInt32(&executing, 1)

			if atomic.LoadInt32(&executing) > limit {
				t.Fatalf("more routines executing than specified limit")
				return
			}
		}()
	}

	wg.Wait()
	assert.Equal(t, int32(0), executing)
}
