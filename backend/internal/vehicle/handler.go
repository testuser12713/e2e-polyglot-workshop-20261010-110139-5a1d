package vehicle

import (
	"net/http"

	"workshop/backend/internal/httpx"
	"workshop/backend/internal/store"
)

// Handler serves the workshop vehicle routes.
type Handler struct {
	Pool store.Pool
}

// NewHandler builds a vehicle handler on the given pool.
func NewHandler(pool store.Pool) *Handler {
	return &Handler{Pool: pool}
}

// Create handles POST /api/workshop/vehicles.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	httpx.NotImplemented(w, "create vehicle")
}

// Get handles GET /api/workshop/vehicles/{id}.
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	httpx.NotImplemented(w, "get vehicle")
}

// Update handles PUT /api/workshop/vehicles/{id}.
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	httpx.NotImplemented(w, "update vehicle")
}
