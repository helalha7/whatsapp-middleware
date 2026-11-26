package worker

import (
	"context"
	"sync"
)

type WorkerPool[T any] struct {
	workerCount int
	jobs        <-chan T
	process     func(T)
	wg          sync.WaitGroup
	startOnce   sync.Once
}

func NewPool[T any](
	workers int,
	jobs <-chan T,
	process func(T),
) *WorkerPool[T] {
	return &WorkerPool[T]{
		workerCount: workers,
		jobs:        jobs,
		process:     process,
	}
}

func (p *WorkerPool[T]) Start(ctx context.Context) {
	p.startOnce.Do(func() {
		for i := 0; i < p.workerCount; i++ {
			p.wg.Go(func() {
				p.run(ctx)
			})
		}
	})
}

func (p *WorkerPool[T]) Wait() {
	p.wg.Wait()
}

func (p *WorkerPool[T]) run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case job, ok := <-p.jobs:
			if !ok {
				return
			}
			p.process(job)
		}
	}
}
