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
	if err := queue.Publish(context.Background(), "message"); err != nil {
		t.Fatalf("publish message: %v", err)
	}
	queue.Close()
	queue.Close()

	if message, open := <-queue.Messages(); !open || message != "message" {
		t.Fatalf("expected queued message before closure, got message=%q open=%t", message, open)
	}
	if _, open := <-queue.Messages(); open {
		t.Fatal("expected messages channel to be closed")
	}
}

func TestPublishAfterClose(t *testing.T) {
	queue := New[string](1)
	queue.Close()

	err := queue.Publish(context.Background(), "message")
	if !errors.Is(err, ErrClosed) {
		t.Fatalf("expected error %v, got %v", ErrClosed, err)
	}
}

func TestCloseUnblocksPublisher(t *testing.T) {
	queue := New[string](1)
	if err := queue.Publish(context.Background(), "first message"); err != nil {
		t.Fatalf("fill queue: %v", err)
	}

	result := make(chan error, 1)
	go func() {
		result <- queue.Publish(context.Background(), "second message")
	}()

	queue.Close()
	if err := <-result; !errors.Is(err, ErrClosed) {
		t.Fatalf("expected blocked publisher to return %v, got %v", ErrClosed, err)
	}

	messages := make([]string, 0, 1)
	for message := range queue.Messages() {
		messages = append(messages, message)
	}
	if len(messages) != 1 || messages[0] != "first message" {
		t.Fatalf("unexpected drained messages: %v", messages)
	}
}
