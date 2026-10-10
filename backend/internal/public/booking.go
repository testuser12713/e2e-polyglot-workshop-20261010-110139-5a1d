// Package public serves the unauthenticated customer area. This ticket ships
// the signatures only; ticket #10 fills the booking body.
package public

import (
	"net/http"

	"workshop/backend/internal/httpx"
	"workshop/backend/internal/store"
)

// Handler serves the public customer routes.
type Handler struct {
	Pool store.Pool
}

// NewHandler builds a public handler on the given pool.
func NewHandler(pool store.Pool) *Handler {
	return &Handler{Pool: pool}
}

// CreateAppointment handles POST /api/public/appointments.
func (h *Handler) CreateAppointment(w http.ResponseWriter, r *http.Request) {
	httpx.NotImplemented(w, "create appointment")
}
