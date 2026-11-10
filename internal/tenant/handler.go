package tenant

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/helalha7/whatsapp-middleware/internal/data"
	"github.com/helalha7/whatsapp-middleware/internal/httpx"
)

type store interface {
	Insert(ctx context.Context, tenant *Tenant) error
	Get(ctx context.Context, id int) (*Tenant, error)
}

type handler struct {
	tenantsStore store
}

func RegisterRoutes(
	mux *http.ServeMux,
	tenantsStore store,
) {
	h := handler{
		tenantsStore: tenantsStore,
	}

	mux.HandleFunc("POST /v1/tenants", h.Create)
	mux.HandleFunc("GET /v1/tenants/{id}", h.Get)
}

func (h *handler) Create(
	w http.ResponseWriter,
	r *http.Request,
) {
	var input struct {
		Name               string `json:"name"`
		WABAID             string `json:"waba_id"`
		PhoneNumberID      string `json:"phone_number_id"`
		DisplayPhoneNumber string `json:"display_phone_number"`
		AccessToken        string `json:"access_token"`
	}

	err := httpx.ReadJSON(w, r, &input)
	if err != nil {
		httpx.ErrorResponse(w, r, http.StatusBadRequest, err.Error())
		return
	}

	tenant := &Tenant{
		Name:               input.Name,
		WABAID:             input.WABAID,
		PhoneNumberID:      input.PhoneNumberID,
		DisplayPhoneNumber: input.DisplayPhoneNumber,
		AccessToken:        input.AccessToken,
	}
	if errors := tenant.Validate(); errors != nil {
		httpx.ValidationFailedResponse(w, r, errors)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	err = h.tenantsStore.Insert(ctx, tenant)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrDuplicateRecord):
			httpx.DuplicateRecordResponse(w, r)
		default:
			httpx.ServerErrorResponse(w, r, err)
		}
		return
	}

	header := make(http.Header)
	header.Set("Location", fmt.Sprintf("/v1/tenants/%d", tenant.ID))

	err = httpx.WriteJSON(
		w,
		http.StatusCreated,
		httpx.Envelope{"tenant": tenant},
		header,
	)
	if err != nil {
		httpx.ServerErrorResponse(w, r, err)
		return
	}
}

func (h *handler) Get(
	w http.ResponseWriter,
	r *http.Request,
) {
	id, err := httpx.ReadIdParam(r)
	if err != nil {
		httpx.ResourceNotFoundResponse(w, r)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	tenant, err := h.tenantsStore.Get(ctx, id)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			httpx.ResourceNotFoundResponse(w, r)
		default:
			httpx.ServerErrorResponse(w, r, err)
		}
		return
	}

	err = httpx.WriteJSON(
		w,
		http.StatusOK,
		httpx.Envelope{"tenant": tenant},
		nil,
	)
	if err != nil {
		httpx.ServerErrorResponse(w, r, err)
		return
	}
}
