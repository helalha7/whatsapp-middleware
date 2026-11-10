package tenant

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/helalha7/whatsapp-middleware/internal/data"
)

type stubStore struct {
	insert func(context.Context, *Tenant) error
	get    func(context.Context, int) (*Tenant, error)
}

func (s *stubStore) Insert(ctx context.Context, tenant *Tenant) error {
	if s.insert == nil {
		return nil
	}
	return s.insert(ctx, tenant)
}

func (s *stubStore) Get(ctx context.Context, id int) (*Tenant, error) {
	if s.get == nil {
		return nil, nil
	}
	return s.get(ctx, id)
}

func TestCreate(t *testing.T) {
	createdAt := time.Date(2026, time.November, 10, 12, 0, 0, 0, time.UTC)
	store := &stubStore{
		insert: func(_ context.Context, tenant *Tenant) error {
			tenant.ID = 3
			tenant.CreatedAt = createdAt
			tenant.Version = 1
			return nil
		},
	}

	recorder := serveRequest(t, store, http.MethodPost, "/v1/tenants", validTenantJSON())

	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, recorder.Code)
	}
	if location := recorder.Header().Get("Location"); location != "/v1/tenants/3" {
		t.Errorf("expected Location /v1/tenants/3, got %q", location)
	}
	if strings.Contains(recorder.Body.String(), "access_token") || strings.Contains(recorder.Body.String(), "secret-token") {
		t.Error("response exposed the access token")
	}

	var response struct {
		Tenant Tenant `json:"tenant"`
	}
	decodeResponse(t, recorder, &response)
	if response.Tenant.ID != 3 || response.Tenant.Name != "Demo Business" {
		t.Errorf("unexpected tenant response: %+v", response.Tenant)
	}
}

func TestCreateRejectsInvalidTenant(t *testing.T) {
	insertCalled := false
	store := &stubStore{
		insert: func(context.Context, *Tenant) error {
			insertCalled = true
			return nil
		},
	}

	recorder := serveRequest(t, store, http.MethodPost, "/v1/tenants", `{}`)

	if recorder.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected status %d, got %d", http.StatusUnprocessableEntity, recorder.Code)
	}
	if insertCalled {
		t.Error("store Insert was called for an invalid tenant")
	}

	var response struct {
		Error map[string]string `json:"error"`
	}
	decodeResponse(t, recorder, &response)
	for _, field := range []string{"name", "waba_id", "phone_number_id", "display_phone_number", "access_token"} {
		if response.Error[field] == "" {
			t.Errorf("expected a validation error for %q", field)
		}
	}
}

func TestCreateRejectsMalformedJSON(t *testing.T) {
	recorder := serveRequest(t, &stubStore{}, http.MethodPost, "/v1/tenants", `{"name":`)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, recorder.Code)
	}
}

func TestCreateHandlesStoreErrors(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
	}{
		{
			name:       "duplicate record",
			err:        data.ErrDuplicateRecord,
			wantStatus: http.StatusConflict,
		},
		{
			name:       "unexpected error",
			err:        errors.New("database unavailable"),
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &stubStore{
				insert: func(context.Context, *Tenant) error {
					return tt.err
				},
			}
			recorder := serveRequest(t, store, http.MethodPost, "/v1/tenants", validTenantJSON())
			if recorder.Code != tt.wantStatus {
				t.Fatalf("expected status %d, got %d", tt.wantStatus, recorder.Code)
			}
		})
	}
}

func TestGet(t *testing.T) {
	tenant := &Tenant{ID: 3, Name: "Demo Business"}

	tests := []struct {
		name       string
		path       string
		store      *stubStore
		wantStatus int
	}{
		{
			name: "returns tenant",
			path: "/v1/tenants/3",
			store: &stubStore{get: func(_ context.Context, id int) (*Tenant, error) {
				if id != 3 {
					t.Errorf("expected ID 3, got %d", id)
				}
				return tenant, nil
			}},
			wantStatus: http.StatusOK,
		},
		{
			name:       "rejects invalid ID",
			path:       "/v1/tenants/not-an-id",
			store:      &stubStore{},
			wantStatus: http.StatusNotFound,
		},
		{
			name: "handles missing tenant",
			path: "/v1/tenants/99",
			store: &stubStore{get: func(context.Context, int) (*Tenant, error) {
				return nil, data.ErrRecordNotFound
			}},
			wantStatus: http.StatusNotFound,
		},
		{
			name: "handles unexpected error",
			path: "/v1/tenants/3",
			store: &stubStore{get: func(context.Context, int) (*Tenant, error) {
				return nil, errors.New("database unavailable")
			}},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := serveRequest(t, tt.store, http.MethodGet, tt.path, "")
			if recorder.Code != tt.wantStatus {
				t.Fatalf("expected status %d, got %d", tt.wantStatus, recorder.Code)
			}
		})
	}
}

func serveRequest(t *testing.T, store store, method string, path string, body string) *httptest.ResponseRecorder {
	t.Helper()

	mux := http.NewServeMux()
	RegisterRoutes(mux, store)

	request := httptest.NewRequest(method, path, strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, request)

	return recorder
}

func decodeResponse(t *testing.T, recorder *httptest.ResponseRecorder, dst any) {
	t.Helper()
	if err := json.Unmarshal(recorder.Body.Bytes(), dst); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
}

func validTenantJSON() string {
	return `{
		"name":"Demo Business",
		"waba_id":"2161887421343402",
		"phone_number_id":"1257434784125832",
		"display_phone_number":"15552050459",
		"access_token":"secret-token"
	}`
}
