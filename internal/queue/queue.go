package queue

import (
	"context"
	"errors"
	"sync"
)

var ErrClosed = errors.New("queue is closed")

type Queue[T any] struct {
	messages   chan T
	done       chan struct{}
	mu         sync.Mutex
	closed     bool
	publishers sync.WaitGroup
	closeOnce  sync.Once
}

func New[T any](capacity int) *Queue[T] {
	return &Queue[T]{
		messages: make(chan T, capacity),
		done:     make(chan struct{}),
	}
}

func (q *Queue[T]) Publish(ctx context.Context, message T) error {
	if !q.beginPublish() {
		return ErrClosed
	}
	defer q.publishers.Done()

	select {
	case q.messages <- message:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-q.done:
		return ErrClosed
	}
}

func (q *Queue[T]) Messages() <-chan T {
	return q.messages
}

func (q *Queue[T]) Close() {
	q.closeOnce.Do(func() {
		q.mu.Lock()
		q.closed = true
		close(q.done)
		q.mu.Unlock()

		q.publishers.Wait()
		close(q.messages)
	})
}

func (q *Queue[T]) beginPublish() bool {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.closed {
		return false
	}

	q.publishers.Add(1)
	return true
}
