package workshop

import (
	"log"
	"net/http"
	"strings"
	"time"

	"workshop/backend/internal/httpx"
	"workshop/backend/internal/model"
)

// listOrdersSQL is the order-list query. It joins each order with its vehicle to
// produce the compact OrderSummary shape. Both filters are optional and combine
// with AND: an empty value disables its own filter. Every value is bound as a
// pgx parameter ($1..$3), so a search term like `' OR 1=1 --` reaches the
// database as data and can never alter the statement (AC-24).
//
// Plate matching is case-insensitive and partial:
//   - the raw fragment is compared against the human plate (case-folded), and
//   - the normalised fragment (upper case, no spaces/dashes) against the
//     normalised plate, so "B-MW 12" and "bmw12" both find "BMW1234".
const listOrdersSQL = `
SELECT o.id,
       o.order_number,
       v.plate,
       v.brand,
       v.model,
       o.status,
       o.requested_date,
       o.created_at
FROM orders o
JOIN vehicles v ON v.id = o.vehicle_id
WHERE ($1 = '' OR o.status = $1)
  AND ($2 = ''
       OR strpos(lower(v.plate), lower($2)) > 0
       OR ($3 <> '' AND strpos(v.plate_normalized, $3) > 0))
ORDER BY o.created_at DESC, o.id DESC`

// listOrdersResponse is the body of a successful order-list response:
// {"orders":[...]}. Orders is never null, so an empty result serialises as [].
type listOrdersResponse struct {
	Orders []model.OrderSummary `json:"orders"`
}

// ListOrders handles GET /api/workshop/orders.
//
// It reads the summary rows straight from the schema with its own query, never
// through order.LoadOrder, since the list needs a compact shape. The optional
// status and plate query parameters are applied together.
func (h *Handler) ListOrders(w http.ResponseWriter, r *http.Request) {
	status := strings.TrimSpace(r.URL.Query().Get("status"))
	plate := strings.TrimSpace(r.URL.Query().Get("plate"))

	rows, err := h.Pool.Query(r.Context(), listOrdersSQL, status, plate, normalizePlateFragment(plate))
	if err != nil {
		log.Printf("workshop: list orders query failed: %v", err)
		httpx.Error(w, model.CodeInternal, http.StatusInternalServerError, "could not load orders")
		return
	}
	defer rows.Close()

	orders := make([]model.OrderSummary, 0)
	for rows.Next() {
		var (
			o         model.OrderSummary
			requested time.Time
			created   time.Time
		)
		if err := rows.Scan(&o.ID, &o.OrderNumber, &o.Plate, &o.Brand, &o.Model, &o.Status, &requested, &created); err != nil {
			log.Printf("workshop: scan order row failed: %v", err)
			httpx.Error(w, model.CodeInternal, http.StatusInternalServerError, "could not load orders")
			return
		}
		o.RequestedDate = requested.Format("2006-01-02")
		o.CreatedAt = created.UTC().Format(time.RFC3339)
		orders = append(orders, o)
	}
	if err := rows.Err(); err != nil {
		log.Printf("workshop: iterate orders failed: %v", err)
		httpx.Error(w, model.CodeInternal, http.StatusInternalServerError, "could not load orders")
		return
	}

	httpx.JSON(w, http.StatusOK, listOrdersResponse{Orders: orders})
}

// normalizePlateFragment mirrors the plate normalisation used when a vehicle is
// stored: upper case with spaces, tabs and dashes removed. It lets a partly
// formatted search ("b-mw 12") match the normalised plate column.
func normalizePlateFragment(s string) string {
	r := strings.NewReplacer(" ", "", "\t", "", "-", "")
	return strings.ToUpper(strings.TrimSpace(r.Replace(s)))
}
