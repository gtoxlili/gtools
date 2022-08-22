# GTools


> GTools is a golang library that provides some handy ways to handle many of the typical requirements when developing GO
> applications.

## Contains

### [Goroutine Pool](pool/pool.go)

> Supports setting the minimum number of resident concurrent goroutine, the maximum number of concurrent goroutine
> allowed
> in the pool, the maximum time a non-resident concurrent goroutine can wait for a new task, and the maximum time a task
> can
> afford to wait for a concurrent goroutine to receive it (when no concurrent goroutine is free after that time, another
> concurrent goroutine is opened to work on it). Also, when submitting tasks to the concurrent pool, setting the
> priority
> of
> the task is supported.

#### Usage

```go
package main

import (
	"fmt"
	"time"

	"github.com/gtoxlili/gtools/pool"
)

func main() {

	// Creates a concurrent pool with at least two resident concurrent goroutines and up to 1024 worker concurrent goroutines, 
	// while non-resident concurrent goroutines are automatically destroyed after 1 minute of inactivity, 
	// and new worker concurrent goroutines are created when the task waits 5 seconds without being processed by the current worker concurrent goroutine.
	p := pool.New(time.Minute, time.Second*5, 1024, 2)

	for i := 0; i < 100; i++ {
		tmp := i
		p.Execute(func() {
			fmt.Println(tmp)
		}, tmp)
	}
	// Output: like 99,98 ... 0 
	// Because of the priority, in the above code, the tasks that are pushed later are instead given priority

	// if you want to stop the pool, you can call the following method
	p.Done()
	// The current policy is to not execute all cached tasks when the concurrent pool is closed
	// If you want to close the pool after all the tasks of the current push have been executed
	// You can try to post the close command as a task and give him a small enough priority
	p.Execute(func() {
		p.Done()
	}, -1)

	<-make(chan struct{}, 1)
}
```

##### default construction method

```go
package main

import (
	"github.com/gtoxlili/gtools/pool"
)

func main() {

	// Creates a goroutine pool that creates new goroutines as needed, 
	// but will reuse previously constructed goroutines when they are available.
	// These pools will typically improve the performance of programs that execute many short-lived asynchronous tasks
	p := pool.DefaultCachedPool()

	// Creates a goroutine pool that reuses a fixed number of goroutines operating off a shared unbounded queue. 
	// At any point, at most N goroutines will be active processing tasks. 
	p = pool.DefaultFixedPool(10)

	// Creates an Executor that uses a single worker thread operating off an unbounded queue.
	p = pool.DefaultSinglePool()

}

```

### [Unbounded Chan](unboundedChan/chan.go)

> A byproduct of developing [Goroutine Pool](pool/pool.go)
> , a based linked table based Unbounded Channel with support for setting priorities

#### Usage

```go
package main

import "github.com/gtoxlili/gtools/unboundedChan"

func main() {
	ch := unboundedChan.NewUnboundedChan[string]()
	ch.In <- unboundedChan.ChanItem[string]{
		Value:    "AAA",
		Priority: 1,
	}
	ch.In <- unboundedChan.ChanItem[string]{
		Value:    "BBB",
		Priority: 2,
	}
	ch.In <- unboundedChan.ChanItem[string]{
		Value:    "CCC",
		Priority: 3,
	}
	for i := 0; i < 3; i++ {
		val, ok := <-ch.Out
		if !ok {
			break
		}
		println(val)
	}
	// Output: CCC, BBB, AAA

	// close method
	close(ch.In)
}
```

### [2Q Cache](twoQueues/cache.go)

> thread-safe memory cache based on two-queue (lru-2) algorithm,

> In benchmark tests using [bigcache-bench](https://github.com/allegro/bigcache-bench), the concurrent read capability
> is comparable to FastCache and BigCache, but the write capability is slightly worse, but because of the two-queue
> algorithm used, the hit rate is perhaps higher than theirs in actual scenarios?

#### Usage

```go
package main

import "github.com/gtoxlili/gtools/twoQueues"

func main() {
	// Created a memory cache with two shards with a maximum capacity of 1024 
	// (theoretically, the more shards, the lower the possibility of lock conflicts during reading and writing, 
	// but it will also lead to more memory usage, so please decide the number of shards according to the actual maximum capacity to use)
	cache := twoQueues.New[string, int](1024, 2)
	cache.Set("a", 1)
	cache.Set("b", 2)
	println(cache.Get("a")) // 1 true
	println(cache.Get("b")) // 2 true
	println(cache.Get("c")) //   false
}
```
