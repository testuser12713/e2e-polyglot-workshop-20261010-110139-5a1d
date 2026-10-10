package order

import (
	"net/http"

	"workshop/backend/internal/httpx"
	"workshop/backend/internal/store"
)

// Handler serves the workshop order routes.
type Handler struct {
	Pool store.Pool
}

// NewHandler builds an order handler on the given pool.
func NewHandler(pool store.Pool) *Handler {
	return &Handler{Pool: pool}
}

// Create handles POST /api/workshop/orders.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	httpx.NotImplemented(w, "create order")
}

// Get handles GET /api/workshop/orders/{id}.
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	httpx.NotImplemented(w, "get order")
}

// AddItems handles POST /api/workshop/orders/{id}/items.
func (h *Handler) AddItems(w http.ResponseWriter, r *http.Request) {
	httpx.NotImplemented(w, "add order items")
}

// ChangeStatus handles POST /api/workshop/orders/{id}/status.
func (h *Handler) ChangeStatus(w http.ResponseWriter, r *http.Request) {
	httpx.NotImplemented(w, "change order status")
}
