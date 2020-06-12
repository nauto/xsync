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
wg := NewLimitedWaitGroup(100)

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