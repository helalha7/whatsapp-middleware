package worker

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
)

func TestWorkerPoolDrainsClosedJobsChannel(t *testing.T) {
	jobs := make(chan int, 10)
	for job := 1; job <= 10; job++ {
		jobs <- job
	}
	close(jobs)

	var mu sync.Mutex
	processed := make(map[int]int, 10)
	pool := NewPool(3, jobs, func(job int) {
		mu.Lock()
		processed[job]++
		mu.Unlock()
	})

	pool.Start(context.Background())
	pool.Wait()

	if len(processed) != 10 {
		t.Fatalf("expected 10 processed jobs, got %d", len(processed))
	}
	for job := 1; job <= 10; job++ {
		if processed[job] != 1 {
			t.Errorf("expected job %d to be processed once, got %d", job, processed[job])
		}
	}
}

func TestWorkerPoolStartsOnce(t *testing.T) {
	jobs := make(chan int, 1)
	jobs <- 1
	close(jobs)

	var processed atomic.Int64
	pool := NewPool(1, jobs, func(int) {
		processed.Add(1)
	})

	pool.Start(context.Background())
	pool.Start(context.Background())
	pool.Wait()

	if got := processed.Load(); got != 1 {
		t.Fatalf("expected one processed job, got %d", got)
	}
}

func TestWorkerPoolStopsWhenContextIsCanceled(t *testing.T) {
	jobs := make(chan int)
	ctx, cancel := context.WithCancel(context.Background())
	var processed atomic.Int64
	pool := NewPool(2, jobs, func(int) {
		processed.Add(1)
	})

	pool.Start(ctx)
	cancel()
	pool.Wait()
	if got := processed.Load(); got != 0 {
		t.Fatalf("expected no processed jobs, got %d", got)
	}
}
