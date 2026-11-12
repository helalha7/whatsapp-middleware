package queue

import (
	"context"
	"errors"
	"testing"
)

func TestPublish(t *testing.T) {
	queue := New[string](1)

	if err := queue.Publish(context.Background(), "message"); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if message := <-queue.Messages(); message != "message" {
		t.Errorf("expected message, got %q", message)
	}
}

func TestPublishRespectsContextCancellation(t *testing.T) {
	queue := New[string](1)
	if err := queue.Publish(context.Background(), "first message"); err != nil {
		t.Fatalf("fill queue: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := queue.Publish(ctx, "second message")
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context cancellation, got %v", err)
	}
}

func TestClose(t *testing.T) {
	queue := New[string](1)
	queue.Close()
	queue.Close()

	if _, open := <-queue.Messages(); open {
		t.Fatal("expected messages channel to be closed")
	}
}
