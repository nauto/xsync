# Concurrency utilities

This go library contains a mix of simple wrappers and utilities that we have
found useful. We will keep adding more conccurrency primitives and utilities here.

### AtMost every

Often when we are logging information, printing debugging information, or gathering
some metrics, we may want to only do so every so often so that we do not spam
the receivers. Ideally, a logging library should provide such a primitive,
but several don't.

As an example, if we want to log at most once per minute, you can write the following:
```go
import "github.com/nauto/xsync"

...

atMostEveryMinute := xsync.AtMost{Every: time.Minute}
...
for i := 0; true; i++ {
    ...
    // The following line will print at most once per minute
    atMostEveryMinute.Run(func(){ log.Infof("Iteration %v\n", i)})
    ...
}
...
```

### LimitedWaitGroup

Often there are times when we want to run several goroutines in parallel while
also limiting the number launched and running concurrently. You might want to use
Go's sync.WaitGroup. However, its semantics do not provide a way to limit the
number of goroutines launched. This is what a Semaphore helps with. LimitedWaitGroup
is a very thin wrapper around Go's Semaphore library but provides WaitGroup like
semantics and can function as a drop-in replacement for WaitGroup. The only difference
is that the Add method blocks till there are resources available.

Example:
```go
import "github.com/nauto/xsync"

...

wg := xsync.NewLimitedWaitGroup(100)

for i := 0; i < 1000; i++ {
    wg.Add(1)

    // There will only be at most 100 of the goroutines below launched and
    // running at the same time.
    go func() {
        defer wg.Done()
        // your code
    }()
}
wg.Wait()
```

### Worker Pool

When there is a stream of possibly expensive jobs coming in such that a single
thread cannot keep up, we have no choice but to process them in parallel. One way
is to have many instances of the service running in parallel processing these jobs.
Often this may result in waste especially with larger machines and when each instance
needs to load a lot of resources in order to process these jobs. In Go, one might
want to use Goroutines to process this incoming stream of jobs. Essentially, we
launch a new goroutine for each new job and can use semaphores or LimitedWaitGroup
to keep the number of simultaneously running goroutines manageable.
 
Given that goroutines are cheap to create and destroy, this is almost always a good
solution. In other languages that do not have the equivalent of goroutines and only threads,
one would typically create a pool of threads to process these jobs that once created
are never destroyed because threads are expensive to switch between, create, and destroy. 
 
Even though thread pools have their downsides, there may sometimes be reasons
where we are required to keep some data between jobs and perhaps process all related
jobs in the same thread, which may help avoid the need for some locks or transactions.

Here, we provide a simple interface to create worker pools in Go for those situations
where other solutions would not work. There are two kinds of worker pools:

#### Whichever is free
Whichever worker gets free first picks the next job. The following example shows
how you can create such a worker pool:
```go
import "github.com/nauto/xsync"

...

ctx, cancel := context.WithCancel(context.Background())
pool := xsync.WorkerPool{
    Ctx:        ctx,
    Cancel:     cancel,
    NumWorkers: 5,
    BufferSize: 10,  // Maximum backlog before Posting blocks.
    Worker: func(ctx context.Context, messages <-chan interface{}, worker int) error {
        for {
            select {
            case <-ctx.Done():  // The context was cancelled.
                return nil  
            case msg := <-messages:
                // Process message
            }
        }  
        return nil
    },
}
// Start the Worker pool. This does not return till the pool shuts down. 
pool.Run()
...
...
// In another goroutine/thread, post messages for processing by this worker pool.
pool.Post(msg1)
pool.Post(msg2)
...
```

#### Sharded
Say if there are several related jobs coming in, e.g., account updates that are
persisted in an in-memory store. If these were going to different workers, each
worker would have to lock the entry corresponding to the user before making any
updates. On the other hand, if the updates for a particular id are all done from
the same thread, there would be no need for locking greatly improving performance.

Creating a sharded worker pool is exactly the same as before except that you need
to provide a hash function:

```go
...
pool := xsync.WorkerPool{
    ...
    Hash: func hash(i interface{}) uint32 {
        return uint32(/* hash of i */)
    }
    ...
}
```