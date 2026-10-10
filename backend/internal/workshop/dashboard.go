// Package workshop serves the authenticated workshop area: the order list and
// the dashboard. This ticket ships the signatures only; tickets #8 and #1 fill
// the bodies.
package workshop

import (
	"net/http"

	"workshop/backend/internal/httpx"
	"workshop/backend/internal/store"
)

// Handler serves the workshop overview routes.
type Handler struct {
	Pool store.Pool
}

// NewHandler builds a workshop handler on the given pool.
func NewHandler(pool store.Pool) *Handler {
	return &Handler{Pool: pool}
}

// Dashboard handles GET /api/workshop/dashboard. This ticket ships the
// signature only; ticket #1 fills the body.
func (h *Handler) Dashboard(w http.ResponseWriter, r *http.Request) {
	httpx.NotImplemented(w, "dashboard")
}
