package customer

import (
	"net/http"

	"workshop/backend/internal/httpx"
	"workshop/backend/internal/store"
)

// Handler serves the workshop customer routes.
type Handler struct {
	Pool store.Pool
}

// NewHandler builds a customer handler on the given pool.
func NewHandler(pool store.Pool) *Handler {
	return &Handler{Pool: pool}
}

// Create handles POST /api/workshop/customers.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	httpx.NotImplemented(w, "create customer")
}

// Get handles GET /api/workshop/customers/{id}.
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	httpx.NotImplemented(w, "get customer")
}

// Update handles PUT /api/workshop/customers/{id}.
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	httpx.NotImplemented(w, "update customer")
}
