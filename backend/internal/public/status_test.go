package public

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"workshop/backend/internal/model"
	"workshop/backend/internal/store"
)

var seedSeq int64

func uniqueSuffix() string {
	return fmt.Sprintf("%d%04d", time.Now().UnixNano()%1_000_000_000, atomic.AddInt64(&seedSeq, 1))
}

// testPool connects to the real PostgreSQL database the office provides. There
// is no SQLite fallback (AC-20); without DATABASE_URL the test is skipped
// rather than faking a database.
func testPool(t *testing.T) store.Pool {
	t.Helper()
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("DATABASE_URL is not set; the office runs this test against PostgreSQL")
	}
	if strings.Contains(strings.ToLower(dbURL), "sqlite") {
		t.Fatalf("DATABASE_URL points at SQLite; this product requires PostgreSQL")
	}
	ctx := context.Background()
	pool, err := store.NewPool(ctx, dbURL)
	if err != nil {
		t.Fatalf("open postgres: %v", err)
	}
	t.Cleanup(pool.Close)
	if err := store.Migrate(ctx, pool); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return pool
}

// statusServer serves exactly the public order-status route under test, over a
// real mux so path values are extracted the way the router extracts them.
func statusServer(t *testing.T, pool store.Pool) *httptest.Server {
	t.Helper()
	h := NewHandler(pool)
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/public/orders/{orderNumber}", h.GetOrderStatus)
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

// seed is one customer/vehicle/order fixture plus its identifying PII, so the
// privacy assertions can prove none of it reaches the response.
type seed struct {
	orderID       int
	orderNumber   string
	normalized    string
	rawPlate      string
	customerName  string
	customerEmail string
	customerPhone string
}

// seedOrder inserts a customer, a vehicle, an order, two history rows and —
// when withInvoice is true — an invoice with one position, straight into the
// skeleton's schema. The rows are removed again after the test.
func seedOrder(t *testing.T, ctx context.Context, pool store.Pool, status string, withInvoice bool) seed {
	t.Helper()
	suffix := uniqueSuffix()
	s := seed{
		orderNumber:   "PUB-" + suffix,
		normalized:    "BXX" + suffix,
		rawPlate:      "B-XX " + suffix,
		customerName:  "Test Kunde " + suffix,
		customerEmail: "kunde." + suffix + "@example.com",
		customerPhone: "0170" + suffix,
	}

	var customerID int
	if err := pool.QueryRow(ctx,
		`INSERT INTO customers (name, email, phone) VALUES ($1, $2, $3) RETURNING id`,
		s.customerName, s.customerEmail, s.customerPhone,
	).Scan(&customerID); err != nil {
		t.Fatalf("insert customer: %v", err)
	}

	var vehicleID int
	if err := pool.QueryRow(ctx,
		`INSERT INTO vehicles (customer_id, plate, plate_normalized, brand, model, mileage_km)
		 VALUES ($1, $2, $3, $4, $5, $6) RETURNING id`,
		customerID, s.normalized, s.normalized, "VW", "Golf", 120000,
	).Scan(&vehicleID); err != nil {
		t.Fatalf("insert vehicle: %v", err)
	}

	if err := pool.QueryRow(ctx,
		`INSERT INTO orders (order_number, vehicle_id, requested_date, problem_description, status)
		 VALUES ($1, $2, $3, $4, $5) RETURNING id`,
		s.orderNumber, vehicleID, time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC), "Bremsen quietschen", status,
	).Scan(&s.orderID); err != nil {
		t.Fatalf("insert order: %v", err)
	}

	base := time.Date(2026, 5, 1, 8, 0, 0, 0, time.UTC)
	if _, err := pool.Exec(ctx,
		`INSERT INTO order_status_log (order_id, status, changed_at) VALUES ($1, $2, $3)`,
		s.orderID, model.StatusRequested, base); err != nil {
		t.Fatalf("insert history requested: %v", err)
	}
	if _, err := pool.Exec(ctx,
		`INSERT INTO order_status_log (order_id, status, changed_at) VALUES ($1, $2, $3)`,
		s.orderID, status, base.Add(time.Hour)); err != nil {
		t.Fatalf("insert history %s: %v", status, err)
	}

	if withInvoice {
		var invoiceID int
		if err := pool.QueryRow(ctx,
			`INSERT INTO invoices (invoice_number, order_id, net_cents, tax_rate_percent, tax_cents, gross_cents)
			 VALUES ($1, $2, $3, $4, $5, $6) RETURNING id`,
			"RE-"+suffix, s.orderID, 10000, 19, 1900, 11900,
		).Scan(&invoiceID); err != nil {
			t.Fatalf("insert invoice: %v", err)
		}
		if _, err := pool.Exec(ctx,
			`INSERT INTO invoice_items (invoice_id, kind, description, hours, quantity, unit_price_cents, amount_cents)
			 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
			invoiceID, "labor", "Bremsbeläge erneuern", 2.5, nil, nil, 10000); err != nil {
			t.Fatalf("insert invoice item: %v", err)
		}
	}

	t.Cleanup(func() {
		cleanupCtx := context.Background()
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM invoices WHERE order_id = $1`, s.orderID)
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM orders WHERE id = $1`, s.orderID)
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM vehicles WHERE id = $1`, vehicleID)
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM customers WHERE id = $1`, customerID)
	})

	return s
}

func getStatus(t *testing.T, srv *httptest.Server, orderNumber, plate string) (int, []byte) {
	t.Helper()
	resp, err := http.Get(srv.URL + "/api/public/orders/" + url.PathEscape(orderNumber) + "?plate=" + url.QueryEscape(plate))
	if err != nil {
		t.Fatalf("GET order status: %v", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	return resp.StatusCode, body
}

func TestOrderStatusMatchingPairReturnsStatusAndHistory(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	srv := statusServer(t, pool)
	s := seedOrder(t, ctx, pool, model.StatusConfirmed, false)

	status, body := getStatus(t, srv, s.orderNumber, s.rawPlate)
	if status != http.StatusOK {
		t.Fatalf("status %d, want 200; body=%s", status, body)
	}

	var got struct {
		OrderNumber   string `json:"orderNumber"`
		Plate         string `json:"plate"`
		Status        string `json:"status"`
		RequestedDate string `json:"requestedDate"`
		History       []struct {
			Status    string `json:"status"`
			ChangedAt string `json:"changedAt"`
		} `json:"history"`
		Invoice *model.Invoice `json:"invoice"`
	}
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("decode body: %v (%s)", err, body)
	}

	if got.OrderNumber != s.orderNumber {
		t.Errorf("orderNumber = %q, want %q", got.OrderNumber, s.orderNumber)
	}
	if got.Plate != s.normalized {
		t.Errorf("plate = %q, want %q", got.Plate, s.normalized)
	}
	if got.Status != model.StatusConfirmed {
		t.Errorf("status = %q, want %q", got.Status, model.StatusConfirmed)
	}
	if got.RequestedDate != "2026-05-01" {
		t.Errorf("requestedDate = %q, want 2026-05-01", got.RequestedDate)
	}
	if len(got.History) != 2 {
		t.Fatalf("history len = %d, want 2; body=%s", len(got.History), body)
	}
	if got.History[0].Status != model.StatusRequested || got.History[1].Status != model.StatusConfirmed {
		t.Errorf("history statuses = %q,%q want requested,confirmed", got.History[0].Status, got.History[1].Status)
	}
	if got.History[0].ChangedAt == "" {
		t.Error("history changedAt is empty")
	}
	if _, err := time.Parse(time.RFC3339, got.History[0].ChangedAt); err != nil {
		t.Errorf("changedAt %q is not RFC3339: %v", got.History[0].ChangedAt, err)
	}
	if got.Invoice != nil {
		t.Errorf("invoice = %+v, want null", got.Invoice)
	}
}

func TestOrderStatusResponseHasNoCustomerFields(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	srv := statusServer(t, pool)
	s := seedOrder(t, ctx, pool, model.StatusConfirmed, true)

	status, body := getStatus(t, srv, s.orderNumber, s.rawPlate)
	if status != http.StatusOK {
		t.Fatalf("status %d, want 200; body=%s", status, body)
	}
	raw := string(body)

	if strings.Contains(raw, s.customerName) {
		t.Errorf("response leaks customer name; body=%s", raw)
	}
	if strings.Contains(raw, s.customerEmail) {
		t.Errorf("response leaks customer email; body=%s", raw)
	}
	if strings.Contains(raw, s.customerPhone) {
		t.Errorf("response leaks customer phone; body=%s", raw)
	}
	for _, key := range []string{`"name"`, `"email"`, `"phone"`, `"customerId"`} {
		if strings.Contains(raw, key) {
			t.Errorf("response contains forbidden key %s; body=%s", key, raw)
		}
	}
}

func TestOrderStatusWithInvoiceReturnsInvoice(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	srv := statusServer(t, pool)
	s := seedOrder(t, ctx, pool, model.StatusCompleted, true)

	status, body := getStatus(t, srv, s.orderNumber, s.rawPlate)
	if status != http.StatusOK {
		t.Fatalf("status %d, want 200; body=%s", status, body)
	}

	var got struct {
		Invoice *model.Invoice `json:"invoice"`
	}
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("decode body: %v (%s)", err, body)
	}
	if got.Invoice == nil {
		t.Fatalf("invoice is null, want an invoice; body=%s", body)
	}
	if got.Invoice.InvoiceNumber == "" {
		t.Error("invoiceNumber is empty")
	}
	if got.Invoice.NetCents != 10000 || got.Invoice.TaxRatePercent != 19 || got.Invoice.TaxCents != 1900 || got.Invoice.GrossCents != 11900 {
		t.Errorf("invoice totals = %d/%d/%d/%d, want 10000/19/1900/11900",
			got.Invoice.NetCents, got.Invoice.TaxRatePercent, got.Invoice.TaxCents, got.Invoice.GrossCents)
	}
	if len(got.Invoice.Items) != 1 {
		t.Fatalf("invoice items len = %d, want 1", len(got.Invoice.Items))
	}
	item := got.Invoice.Items[0]
	if item.Kind != model.OrderItemKindLabor || item.AmountCents != 10000 {
		t.Errorf("invoice item = %+v, want labor/10000", item)
	}
}

func TestOrderStatusWithoutInvoiceReturnsNull(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	srv := statusServer(t, pool)
	s := seedOrder(t, ctx, pool, model.StatusRequested, false)

	status, body := getStatus(t, srv, s.orderNumber, s.rawPlate)
	if status != http.StatusOK {
		t.Fatalf("status %d, want 200; body=%s", status, body)
	}
	var got map[string]json.RawMessage
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("decode body: %v (%s)", err, body)
	}
	if string(got["invoice"]) != "null" {
		t.Errorf("invoice = %s, want null", got["invoice"])
	}
}

func TestOrderStatusMismatchedPlateReturns404AndDisclosesNothing(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	srv := statusServer(t, pool)
	s := seedOrder(t, ctx, pool, model.StatusConfirmed, false)

	status, body := getStatus(t, srv, s.orderNumber, "ZZ-9999")
	if status != http.StatusNotFound {
		t.Fatalf("status %d, want 404; body=%s", status, body)
	}
	if strings.Contains(string(body), s.orderNumber) {
		t.Errorf("404 response discloses the order number; body=%s", body)
	}
	if strings.Contains(string(body), s.customerEmail) {
		t.Errorf("404 response discloses customer data; body=%s", body)
	}
	var eb struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &eb); err != nil {
		t.Fatalf("decode error body: %v (%s)", err, body)
	}
	if eb.Error.Code != model.CodeNotFound {
		t.Errorf("code = %q, want %q", eb.Error.Code, model.CodeNotFound)
	}
}

func TestOrderStatusUnknownOrderReturns404(t *testing.T) {
	pool := testPool(t)
	srv := statusServer(t, pool)

	status, body := getStatus(t, srv, "PUB-DOES-NOT-EXIST", "B-XX 1234")
	if status != http.StatusNotFound {
		t.Fatalf("status %d, want 404; body=%s", status, body)
	}
}
