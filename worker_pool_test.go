// +build unit !integration

package xsync

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func worker(ctx context.Context, messages <-chan interface{},
	numExecuting *int32, numReceived *int32, signalStart *sync.WaitGroup) error {

	atomic.AddInt32(numExecuting, 1)
	signalStart.Done()
	defer atomic.AddInt32(numExecuting, -1)

	// Keep processing incoming messages. If we receive a nil message,
	// request a shutdown by returning an error.
	for {
		select {
		case <-ctx.Done():
			return nil
		case msg := <-messages:
			atomic.AddInt32(numReceived, 1)
			if msg.(int) == 0 {
				return errors.New("Let's exit")
			}
		}
	}
}

func workerProcessOneItem(ctx context.Context, messages <-chan interface{},
	assert *assert.Assertions, numExecuting *int32, numReceived *int32, signalStart *sync.WaitGroup) error {

	atomic.AddInt32(numExecuting, 1)
	signalStart.Done()
	defer atomic.AddInt32(numExecuting, -1)

	for {
		select {
		case <-ctx.Done():
			assert.Fail("Context should not have been cancelled.")
			return nil
		case <-messages:
			// Process the item and quit but do not ask for termination of the
			// pool.
			atomic.AddInt32(numReceived, 1)
			return nil
		}
	}
}

func hash(i interface{}) uint32 {
	return uint32(i.(int))
}

func TestOrdinaryWorkerPool(t *testing.T) {
	assert := assert.New(t)
	numExecuting := int32(0)
	numReceived := int32(0)
	signalStart := sync.WaitGroup{}
	signalStart.Add(5)

	ctx, cancel := context.WithCancel(context.Background())
	pool := WorkerPool{
		Ctx:        ctx,
		Cancel:     cancel,
		NumWorkers: 5,
		BufferSize: 10,
		Worker: func(ctx context.Context, messages <-chan interface{}, _ int) error {
			return worker(ctx, messages, &numExecuting, &numReceived, &signalStart)
		},
	}

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		pool.Run()
		wg.Done()
	}()

	// Wait for the pool to start (only relevant to prevent races in testing).
	signalStart.Wait()

	// Ensure that we have 5 workers running.
	assert.EqualValues(5, numExecuting)

	// Send a few items.
	assert.NoError(pool.Post(1))
	assert.NoError(pool.Post(2))
	assert.NoError(pool.Post(3))
	assert.NoError(pool.Post(0))

	// Wait for everything to settle down.
	wg.Wait()

	// Check that all items were received.
	assert.EqualValues(4, numReceived)

	// Ensure that all workers have quit.
	assert.EqualValues(0, numExecuting)

	// Trying to post now should return an error.
	assert.Error(pool.Post(1))
}

func TestOrdinaryWorkerPoolCancel(t *testing.T) {
	assert := assert.New(t)
	numExecuting := int32(0)
	numReceived := int32(0)
	signalStart := sync.WaitGroup{}
	signalStart.Add(5)

	ctx, cancel := context.WithCancel(context.Background())
	pool := WorkerPool{
		Ctx:        ctx,
		Cancel:     cancel,
		NumWorkers: 5,
		BufferSize: 10,
		Worker: func(ctx context.Context, messages <-chan interface{}, _ int) error {
			return worker(ctx, messages, &numExecuting, &numReceived, &signalStart)
		},
	}

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		pool.Run()
		wg.Done()
	}()

	// Wait for the pool to start (only relevant to prevent races in testing).
	signalStart.Wait()

	// Ensure that we have 5 workers running.
	assert.EqualValues(5, numExecuting)

	// Send a few items.
	assert.NoError(pool.Post(1))
	assert.NoError(pool.Post(2))
	assert.NoError(pool.Post(3))

	// Now cancel.
	cancel()

	// Wait for everything to settle down.
	wg.Wait()

	// Check that at most 3 items were received. Not all may be received, because
	// the shutdown may have happened before the items were processed.
	assert.True(numReceived <= 3)

	// Ensure that all workers have quit.
	assert.EqualValues(0, numExecuting)

	// Trying to post now should return an error.
	assert.Error(pool.Post(1))
}

func TestShardedWorkerPool(t *testing.T) {
	assert := assert.New(t)
	numExecuting := int32(0)
	numReceived := int32(0)
	signalStart := sync.WaitGroup{}
	signalStart.Add(5)

	ctx, cancel := context.WithCancel(context.Background())
	pool := WorkerPool{
		Ctx:        ctx,
		Cancel:     cancel,
		NumWorkers: 5,
		BufferSize: 10,
		Hash:       hash,
		Worker: func(ctx context.Context, messages <-chan interface{}, _ int) error {
			return worker(ctx, messages, &numExecuting, &numReceived, &signalStart)
		},
	}

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		pool.Run()
		wg.Done()
	}()

	// Wait for the pool to start (only relevant to prevent races in testing).
	signalStart.Wait()

	// Ensure that we have 5 workers running.
	assert.EqualValues(5, numExecuting)

	// Send a few items.
	assert.NoError(pool.Post(1))
	assert.NoError(pool.Post(2))
	assert.NoError(pool.Post(3))
	assert.NoError(pool.Post(0))

	// Wait for everything to settle down.
	wg.Wait()

	// Check that all items were received.
	assert.EqualValues(4, numReceived)

	// Ensure that all workers have quit.
	assert.EqualValues(0, numExecuting)

	// Trying to post now should return an error.
	assert.Error(pool.Post(1))
}

func TestShardedWorkerPoolDelivery(t *testing.T) {
	assert := assert.New(t)
	numExecuting := int32(0)
	numReceived := int32(0)
	signalStart := sync.WaitGroup{}
	signalStart.Add(5)

	ctx, cancel := context.WithCancel(context.Background())
	pool := WorkerPool{
		Ctx:        ctx,
		Cancel:     cancel,
		NumWorkers: 5,
		BufferSize: 10,
		Hash:       hash,
		Worker: func(ctx context.Context, messages <-chan interface{}, _ int) error {
			return workerProcessOneItem(ctx, messages, assert, &numExecuting, &numReceived, &signalStart)
		},
	}

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		pool.Run()
		wg.Done()
	}()

	// Wait for the pool to start (only relevant to prevent races in testing).
	signalStart.Wait()

	// Ensure that we have 5 workers running.
	assert.EqualValues(5, numExecuting)

	// Send exactly 5 items, s.t. each one gets delivered to one worker after
	// which the worker exits.
	assert.NoError(pool.Post(1))
	assert.NoError(pool.Post(2))
	assert.NoError(pool.Post(3))
	assert.NoError(pool.Post(4))
	assert.NoError(pool.Post(5))

	// Wait for everything to settle down.
	wg.Wait()

	// Check that all items were received.
	assert.EqualValues(5, numReceived)

	// Ensure that all workers have quit.
	assert.EqualValues(0, numExecuting)

	// Trying to post now should not an error because all workers quit by themselves
	// without the context being canceled.
	assert.NoError(pool.Post(1))
}
