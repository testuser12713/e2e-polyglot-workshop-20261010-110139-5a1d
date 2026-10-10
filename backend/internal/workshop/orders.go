package workshop

import (
	"net/http"

	"workshop/backend/internal/httpx"
)

// ListOrders handles GET /api/workshop/orders.
//
// It reads the summary rows straight from the schema — never through
// order.LoadOrder — since the list needs a compact shape. This ticket ships the
// signature only; ticket #8 fills the body.
func (h *Handler) ListOrders(w http.ResponseWriter, r *http.Request) {
	httpx.NotImplemented(w, "list orders")
}
