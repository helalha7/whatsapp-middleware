package httpx

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestReadIdParam(t *testing.T) {
	tests := []struct {
		name    string
		id      string
		want    int
		wantErr bool
	}{
		{name: "valid ID", id: "42", want: 42},
		{name: "non-numeric ID", id: "invalid", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/resources/"+tt.id, nil)
			request.SetPathValue("id", tt.id)

			id, err := ReadIdParam(request)
			if tt.wantErr && err == nil {
				t.Fatal("expected an error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if id != tt.want {
				t.Errorf("expected ID %d, got %d", tt.want, id)
			}
		})
	}
}

func TestWriteJSON(t *testing.T) {
	recorder := httptest.NewRecorder()
	header := make(http.Header)
	header.Set("Location", "/resources/42")

	err := WriteJSON(
		recorder,
		http.StatusCreated,
		Envelope{"resource": map[string]int{"id": 42}},
		header,
	)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if recorder.Code != http.StatusCreated {
		t.Errorf("expected status %d, got %d", http.StatusCreated, recorder.Code)
	}
	if got := recorder.Header().Get("Content-Type"); got != "application/json" {
		t.Errorf("expected JSON content type, got %q", got)
	}
	if got := recorder.Header().Get("Location"); got != "/resources/42" {
		t.Errorf("expected Location header, got %q", got)
	}
	if got := recorder.Body.String(); got != `{"resource":{"id":42}}` {
		t.Errorf("unexpected response body %q", got)
	}
}

func TestWriteJSONReturnsMarshalError(t *testing.T) {
	recorder := httptest.NewRecorder()
	err := WriteJSON(recorder, http.StatusOK, make(chan int), nil)
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	if recorder.Code != http.StatusOK || recorder.Body.Len() != 0 {
		t.Fatalf("expected response to remain unwritten, got status=%d body=%q", recorder.Code, recorder.Body.String())
	}
}

func TestReadJSON(t *testing.T) {
	tests := []struct {
		name      string
		body      string
		wantName  string
		wantError string
	}{
		{name: "valid body", body: `{"name":"Ada"}`, wantName: "Ada"},
		{name: "malformed body", body: `{"name":`, wantError: "body contains badly-formed JSON"},
		{name: "empty body", wantError: "body must not be empty"},
		{name: "incorrect type", body: `{"name":42}`, wantError: `body contains incorrect JSON type for field "name"`},
		{name: "unknown field", body: `{"name":"Ada","extra":true}`, wantError: `body contains unknown key "extra"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/resources", strings.NewReader(tt.body))
			recorder := httptest.NewRecorder()
			var dst struct {
				Name string `json:"name"`
			}

			err := ReadJSON(recorder, request, &dst)
			if tt.wantError == "" && err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if tt.wantError != "" && (err == nil || !strings.Contains(err.Error(), tt.wantError)) {
				t.Fatalf("expected error containing %q, got %v", tt.wantError, err)
			}
			if tt.wantError == "" && dst.Name != tt.wantName {
				t.Errorf("expected name %q, got %q", tt.wantName, dst.Name)
			}
		})
	}
}

func TestReadJSONRejectsOversizedBody(t *testing.T) {
	body := `{"name":"` + strings.Repeat("a", maxJSONBodyBytes) + `"}`
	request := httptest.NewRequest(http.MethodPost, "/resources", strings.NewReader(body))
	recorder := httptest.NewRecorder()
	var dst struct {
		Name string `json:"name"`
	}

	err := ReadJSON(recorder, request, &dst)
	if err == nil || !strings.Contains(err.Error(), "body must not be larger than") {
		t.Fatalf("expected body size error, got %v", err)
	}
}
