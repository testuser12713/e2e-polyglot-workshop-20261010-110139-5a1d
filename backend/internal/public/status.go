package public

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"workshop/backend/internal/httpx"
	"workshop/backend/internal/model"
	"workshop/backend/internal/store"
)

// publicStatusEntry is one history entry of the public status response. It
// carries ONLY status and changedAt — never the employee name of the workshop
// shape (AC-30).
type publicStatusEntry struct {
	Status    string `json:"status"`
	ChangedAt string `json:"changedAt"`
}

// statusResponse is the public order-status response. It deliberately declares
// no customer name, e-mail or phone field, so customer PII can never be
// serialised into this response (AC-30).
type statusResponse struct {
	OrderNumber   string              `json:"orderNumber"`
	Plate         string              `json:"plate"`
	Status        string              `json:"status"`
	RequestedDate string              `json:"requestedDate"`
	History       []publicStatusEntry `json:"history"`
	Invoice       *model.Invoice      `json:"invoice"`
}

// GetOrderStatus handles GET /api/public/orders/{orderNumber}?plate=<plate>.
//
// It reads the order together with its vehicle's plate in a single narrow query
// — never through order.LoadOrder, which loads the whole workshop view
// including the customer — then loads only the history and invoice it is
// allowed to return. The plate is matched in that same query with the value
// bound as a pgx parameter. A missing order or a plate that does not match the
// order answers 404 not_found and discloses nothing about the order.
func (h *Handler) GetOrderStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	orderNumber := strings.TrimSpace(r.PathValue("orderNumber"))
	plate := normalizePlate(r.URL.Query().Get("plate"))
	if orderNumber == "" || plate == "" {
		httpx.Error(w, model.CodeNotFound, http.StatusNotFound, "order not found")
		return
	}

	var (
		orderID       int
		storedNumber  string
		storedPlate   string
		status        string
		requestedDate time.Time
	)
	err := h.Pool.QueryRow(ctx,
		`SELECT o.id, o.order_number, v.plate, o.status, o.requested_date
		   FROM orders o
		   JOIN vehicles v ON v.id = o.vehicle_id
		  WHERE o.order_number = $1 AND v.plate_normalized = $2`,
		orderNumber, plate,
	).Scan(&orderID, &storedNumber, &storedPlate, &status, &requestedDate)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			httpx.Error(w, model.CodeNotFound, http.StatusNotFound, "order not found")
			return
		}
		// No user value (order number, plate) is logged (AC-29).
		log.Printf("public: get order status: query failed")
		httpx.Error(w, model.CodeInternal, http.StatusInternalServerError, "internal server error")
		return
	}

	history, err := loadHistory(ctx, h.Pool, orderID)
	if err != nil {
		log.Printf("public: get order status: history load failed")
		httpx.Error(w, model.CodeInternal, http.StatusInternalServerError, "internal server error")
		return
	}

	invoice, err := loadInvoice(ctx, h.Pool, orderID)
	if err != nil {
		log.Printf("public: get order status: invoice load failed")
		httpx.Error(w, model.CodeInternal, http.StatusInternalServerError, "internal server error")
		return
	}

	httpx.JSON(w, http.StatusOK, statusResponse{
		OrderNumber:   storedNumber,
		Plate:         storedPlate,
		Status:        status,
		RequestedDate: requestedDate.Format("2006-01-02"),
		History:       history,
		Invoice:       invoice,
	})
}

// loadHistory reads the status history of one order, newest last. Only status
// and changedAt leave this function (AC-30).
func loadHistory(ctx context.Context, q store.Querier, orderID int) ([]publicStatusEntry, error) {
	rows, err := q.Query(ctx,
		`SELECT status, changed_at
		   FROM order_status_log
		  WHERE order_id = $1
		  ORDER BY changed_at, id`,
		orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	history := make([]publicStatusEntry, 0)
	for rows.Next() {
		var (
			status    string
			changedAt time.Time
		)
		if err := rows.Scan(&status, &changedAt); err != nil {
			return nil, err
		}
		history = append(history, publicStatusEntry{
			Status:    status,
			ChangedAt: changedAt.UTC().Format(time.RFC3339),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return history, nil
}

// loadInvoice returns the invoice of the order with its items, or nil when the
// order has no invoice yet.
func loadInvoice(ctx context.Context, q store.Querier, orderID int) (*model.Invoice, error) {
	var (
		invoiceID int
		inv       model.Invoice
	)
	err := q.QueryRow(ctx,
		`SELECT id, invoice_number, net_cents, tax_rate_percent, tax_cents, gross_cents
		   FROM invoices
		  WHERE order_id = $1`,
		orderID,
	).Scan(&invoiceID, &inv.InvoiceNumber, &inv.NetCents, &inv.TaxRatePercent, &inv.TaxCents, &inv.GrossCents)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	rows, err := q.Query(ctx,
		`SELECT kind, description, hours::float8, quantity, unit_price_cents, amount_cents
		   FROM invoice_items
		  WHERE invoice_id = $1
		  ORDER BY id`,
		invoiceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]model.InvoiceItem, 0)
	for rows.Next() {
		var it model.InvoiceItem
		if err := rows.Scan(&it.Kind, &it.Description, &it.Hours, &it.Quantity, &it.UnitPriceCents, &it.AmountCents); err != nil {
			return nil, err
		}
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	inv.Items = items
	return &inv, nil
}

// normalizePlate matches the vehicle service's normalisation: upper case with
// spaces and dashes removed, so the customer's typed plate finds the stored
// plate_normalized regardless of separators.
func normalizePlate(plate string) string {
	var b strings.Builder
	for _, r := range strings.ToUpper(strings.TrimSpace(plate)) {
		switch r {
		case ' ', '-', '\t':
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}
