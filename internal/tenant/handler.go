package tenant

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/helalha7/whatsapp-middleware/internal/data"
	"github.com/helalha7/whatsapp-middleware/internal/httpx"
)

type store interface {
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

	mux.HandleFunc("GET /v1/tenants/{id}", h.Get)
}

func (h *handler) Get(
	w http.ResponseWriter,
	r *http.Request,
) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || id < 1 {
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
