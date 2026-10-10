package public

import (
	"net/http"

	"workshop/backend/internal/httpx"
)

// GetOrderStatus handles GET /api/public/orders/{orderNumber}.
//
// It deliberately reads the order, its history and its invoice straight from
// the schema — never through order.LoadOrder — because the public response is a
// narrower shape that must never leak customer name, e-mail or phone (AC-30).
// This ticket ships the signature only; ticket #3 fills the body.
func (h *Handler) GetOrderStatus(w http.ResponseWriter, r *http.Request) {
	httpx.NotImplemented(w, "get order status")
}
