package queue

import (
	"context"
	"sync"
)

type Queue[T any] struct {
	messages chan T
	close    sync.Once
}

func New[T any](capacity int) *Queue[T] {
	return &Queue[T]{
		messages: make(chan T, capacity),
	}
}

func (q *Queue[T]) Publish(ctx context.Context, message T) error {
	select {
	case q.messages <- message:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (q *Queue[T]) Messages() <-chan T {
	return q.messages
}

func (q *Queue[T]) Close() {
	q.close.Do(func() {
		close(q.messages)
	})
}
