package sender

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSendTextMessage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected method POST, got %s", r.Method)
		}
		if r.URL.Path != "/v26.0/phone-number-id/messages" {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		if authorization := r.Header.Get("Authorization"); authorization != "Bearer access-token" {
			t.Errorf("unexpected authorization header %q", authorization)
		}

		var body struct {
			MessagingProduct string `json:"messaging_product"`
			RecipientType    string `json:"recipient_type"`
			To               string `json:"to"`
			Type             string `json:"type"`
			Text             struct {
				Body string `json:"body"`
			} `json:"text"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		if body.MessagingProduct != "whatsapp" || body.RecipientType != "individual" {
			t.Errorf("unexpected request body: %+v", body)
		}
		if body.To != "15551234567" || body.Type != "text" || body.Text.Body != "Hello" {
			t.Errorf("unexpected message body: %+v", body)
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	sender := New(server.Client())
	sender.baseURL = server.URL + "/v26.0"

	err := sender.SendTextMessage(
		context.Background(),
		"phone-number-id",
		"access-token",
		"15551234567",
		"Hello",
	)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestSendTextMessageReturnsAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":{"message":"invalid token"}}`))
	}))
	defer server.Close()

	sender := New(server.Client())
	sender.baseURL = server.URL

	err := sender.SendTextMessage(
		context.Background(),
		"phone-number-id",
		"access-token",
		"15551234567",
		"Hello",
	)
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	if !strings.Contains(err.Error(), "401 Unauthorized") || !strings.Contains(err.Error(), "invalid token") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestSendTextMessageRespectsContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	sender := New(http.DefaultClient)
	err := sender.SendTextMessage(
		ctx,
		"phone-number-id",
		"access-token",
		"15551234567",
		"Hello",
	)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context cancellation, got %v", err)
	}
}
