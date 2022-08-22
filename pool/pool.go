package pool

import (
	"context"
	"github.com/gtoxlili/gtools/unboundedChan"
	"time"
)

type Pool struct {
	workQueue     *unboundedChan.UnboundedChan[func()]
	keepAliveTime time.Duration
	awaitTime     time.Duration
	corePoolSize  int
	semaphore     chan struct{}
	queue         chan func()

	Done func()
}

func (p *Pool) Execute(run func(), priority int) {
	p.workQueue.In <- unboundedChan.ChanItem[func()]{
		Value:    run,
		Priority: priority,
	}
}

func DefaultCachedPool() *Pool {
	return New(time.Minute, 0, 2147483647, 0)
}

func DefaultFixedPool(nCoroutines int) *Pool {
	return New(time.Hour, 0, nCoroutines, nCoroutines)
}

func DefaultSinglePool() *Pool {
	return New(time.Hour, 0, 1, 1)
}

func New(keepAliveTime time.Duration, awaitTime time.Duration, maximumPoolSize int, corePoolSize int) *Pool {
	if corePoolSize > maximumPoolSize {
		panic("corePoolSize must be less than maximumPoolSize")
	}

	ctx, cancel := context.WithCancel(context.Background())
	p := &Pool{
		keepAliveTime: keepAliveTime,
		awaitTime:     awaitTime,
		queue:         make(chan func()),
		workQueue:     unboundedChan.NewUnboundedChan[func()](),
		semaphore:     make(chan struct{}, maximumPoolSize),
		corePoolSize:  corePoolSize,
		Done:          cancel,
	}
	go p.init(ctx)

	return p
}

func (p *Pool) init(ctx context.Context) {
	for i := 0; i < p.corePoolSize; i++ {
		p.semaphore <- struct{}{}
		go p.coroutine(ctx)
	}

	var run func()

	for {
		// To solve the problem of not receiving the shutdown signal when the thread pool is closed but there is no subsequent task
		select {
		case <-ctx.Done():
		case run = <-p.workQueue.Out:
		}

	pushTask:
		select {
		case <-ctx.Done():
			close(p.workQueue.In)
			// The current policy is to not execute all cached tasks when the concurrent pool is closed
			// So here all the tasks accumulated by OUT are taken out at once
			for p.workQueue.Len() > 0 {
				<-p.workQueue.Out
			}
			return
		case p.queue <- run:
		case <-time.After(p.awaitTime):
			select {
			case p.semaphore <- struct{}{}:
				go p.coroutine(ctx)
				p.queue <- run
			default:
				goto pushTask
			}
		}
	}
}

func (p *Pool) coroutine(ctx context.Context) {
	defer func() {
		<-p.semaphore
	}()
	for {
		select {
		case <-ctx.Done():
			return
		case run := <-p.queue:
			run()
		case <-time.After(p.keepAliveTime):
			if len(p.semaphore) > p.corePoolSize {
				return
			}
		}
	}
}
