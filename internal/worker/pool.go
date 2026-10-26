package worker

import (
	"context"
	"sync"
)

type WorkerPool[T any] struct {
	workers int
	jobs    <-chan T
	wg      sync.WaitGroup
	so      sync.Once
	process func(T)
}

func NewPool[T any](
	workers int,
	jobs <-chan T,
	process func(T),
) *WorkerPool[T] {
	return &WorkerPool[T]{
		workers: workers,
		jobs:    jobs,
		process: process,
	}
}

func (p *WorkerPool[T]) Start(
	ctx context.Context,
) {
	p.so.Do(
		func() {
			for i := 0; i < p.workers; i++ {
				p.wg.Go(
					func() {
						for {
							select {
							case <-ctx.Done():
								return
							case event, ok := <-p.jobs:
								if !ok {
									return
								}

								p.process(event)
							}
						}
					},
				)
			}
		},
	)
}

func (p *WorkerPool[T]) Wait() {
	p.wg.Wait()
}
