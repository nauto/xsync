package xsync

import (
	"context"
	"sync"
)

// HashFunc is the signature of a method to use as a hash function for WorkerPool.
type HashFunc func(interface{}) uint32

// WorkerFunc is the signature of a method to use for WorkerPool.
type WorkerFunc func(ctx context.Context, messages <-chan interface{}, shard int) error

// WorkerPool manages a pool of workers where incoming posted items are
// optionally sharded and sent to the appropriate worker or the next available
// worker. The sharding is useful when, for example, one needs to have a pool of
// workers, but have messages from the same id (used for hashing) go to the same
// worker each time.
//
// See tests for sample uses.
type WorkerPool struct {
	channels []chan interface{}

	// Set these fields before calling Run.
	Ctx        context.Context
	Cancel     context.CancelFunc
	NumWorkers int
	BufferSize int

	// "Override" these function pointers.
	Hash   HashFunc
	Worker WorkerFunc
}

// Post routes the given datum to the appropriate shard. If the channel buffer
// for this shard is full, this method blocks. In the case of an ordinary worker
// pool, it's posted on a common channel, which is shared between all workers.
//
// If the running context has been cancelled (or timed out), it returns the
// corresponding error and does not post to the channels.
func (p *WorkerPool) Post(datum interface{}) error {
	// If the context was cancelled, return an error.
	if err := p.Ctx.Err(); err != nil {
		return err
	}
	shard := p.Hash(datum) % uint32(len(p.channels))
	p.channels[shard] <- datum
	return nil
}

// Run initializes and starts a new worker pool using the set worker function
// with the specified number of workers/shards and buffer size. It also takes a
// context with the context's cancel function.
//
// If a worker finishes with a non-nil error, the context, which is the same as
// the passed-in context and is shared with all the workers, is cancelled. Any
// workers checking for the context cancellation can then choose to gracefully
// finish whatever they are doing. This mechanism can be used if a single worker
// wants to request a graceful shutdown for the whole system including other
// worker pools.
//
// If the set hash function is nil, an ordinary worker pool is created; if it is
// not nil, a sharded worker pool with one shard for each worker is created.
//
// Run does not return till all the workers have finished.
func (p *WorkerPool) Run() {
	numShards := p.NumWorkers
	if p.Hash == nil {
		// Only one shard and several workers on the same shard.
		p.Hash = func(interface{}) uint32 { return 0 }
		numShards = 1
	}
	p.channels = make([]chan interface{}, numShards)

	for i := 0; i != numShards; i++ {
		p.channels[i] = make(chan interface{}, p.BufferSize)
	}

	// Run the workers.
	w := sync.WaitGroup{}
	w.Add(p.NumWorkers)
	for i := 0; i != p.NumWorkers; i++ {
		i := i
		go func() {
			defer w.Done()
			var err error
			if numShards == 1 {
				err = p.Worker(p.Ctx, p.channels[0], i)
			} else {
				err = p.Worker(p.Ctx, p.channels[i], i)
			}
			if err != nil {
				p.Cancel()
			}
		}()
	}
	w.Wait()
}
