package httpx

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestReadStringQuery(t *testing.T) {
	tests := []struct {
		name         string
		query        string
		defaultValue string
		want         string
	}{
		{name: "provided value", query: "?sort=-created_at", defaultValue: "name", want: "-created_at"},
		{name: "default value", defaultValue: "name", want: "name"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/resources"+tt.query, nil)
			if got := ReadStringQuery(request, "sort", tt.defaultValue); got != tt.want {
				t.Errorf("expected %q, got %q", tt.want, got)
			}
		})
	}
}

func TestReadIntQuery(t *testing.T) {
	tests := []struct {
		name         string
		query        string
		defaultValue int
		want         int
		wantError    bool
	}{
		{name: "provided value", query: "?page=3", defaultValue: 1, want: 3},
		{name: "default value", defaultValue: 1, want: 1},
		{name: "invalid value", query: "?page=invalid", defaultValue: 1, wantError: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/resources"+tt.query, nil)
			got, err := ReadIntQuery(request, "page", tt.defaultValue)
			if tt.wantError && err == nil {
				t.Fatal("expected an error, got nil")
			}
			if !tt.wantError && err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if got != tt.want {
				t.Errorf("expected %d, got %d", tt.want, got)
			}
		})
	}
}
