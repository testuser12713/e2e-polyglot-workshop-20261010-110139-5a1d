package workshop

import (
	"context"
	"encoding/json"
	"fmt"
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

// These tests exercise the order list against a real PostgreSQL database
// (AC-20). Without DATABASE_URL they skip rather than fake a database; the
// office provides it and runs them.

var fixtureSeq int64

// uniqueToken returns a value unique within this test process, used to keep the
// rows this file inserts apart from anything a parallel test creates.
func uniqueToken() string {
	return fmt.Sprintf("%d-%d", time.Now().UnixNano(), atomic.AddInt64(&fixtureSeq, 1))
}

// normPlate mirrors the product's plate normalisation independently of the
// implementation under test, so a fixture's stored normalised plate is what a
// caller would expect.
func normPlate(plate string) string {
	r := strings.NewReplacer(" ", "", "\t", "", "-", "")
	return strings.ToUpper(strings.TrimSpace(r.Replace(plate)))
}

// newTestPool opens and migrates a real PostgreSQL pool. The pool is closed when
// the test ends.
func newTestPool(t *testing.T) (context.Context, store.Pool) {
	t.Helper()
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL is not set; the office runs these tests against PostgreSQL")
	}
	if strings.Contains(strings.ToLower(databaseURL), "sqlite") {
		t.Fatalf("DATABASE_URL points at SQLite; this product requires PostgreSQL")
	}
	ctx := context.Background()
	pool, err := store.NewPool(ctx, databaseURL)
	if err != nil {
		t.Fatalf("open postgres: %v", err)
	}
	if err := store.Migrate(ctx, pool); err != nil {
		pool.Close()
		t.Fatalf("migrate: %v", err)
	}
	t.Cleanup(pool.Close)
	return ctx, pool
}

// createOrderFixture inserts one customer, one vehicle and one order straight
// into the skeleton's schema and registers their removal by primary key, so it
// never truncates tables another ticket owns. It returns the order's id.
func createOrderFixture(t *testing.T, ctx context.Context, pool store.Pool, plate, brand, model, status, requestedDate string) int {
	t.Helper()
	token := uniqueToken()

	var customerID int
	if err := pool.QueryRow(ctx,
		`INSERT INTO customers (name, email, phone) VALUES ($1, $2, $3) RETURNING id`,
		"Listentest Kunde", "list-"+token+"@example.test", "0000-000000",
	).Scan(&customerID); err != nil {
		t.Fatalf("insert customer: %v", err)
	}

	var vehicleID int
	if err := pool.QueryRow(ctx,
		`INSERT INTO vehicles (customer_id, plate, plate_normalized, brand, model, mileage_km)
		 VALUES ($1, $2, $3, $4, $5, $6) RETURNING id`,
		customerID, plate, normPlate(plate), brand, model, 120000,
	).Scan(&vehicleID); err != nil {
		t.Fatalf("insert vehicle: %v", err)
	}

	requested, err := time.Parse("2006-01-02", requestedDate)
	if err != nil {
		t.Fatalf("parse requested date %q: %v", requestedDate, err)
	}
	createdAt := time.Now().UTC().Truncate(time.Second)

	var orderID int
	if err := pool.QueryRow(ctx,
		`INSERT INTO orders (order_number, vehicle_id, requested_date, problem_description, status, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6) RETURNING id`,
		"TST-"+token, vehicleID, requested, "Listenfixture", status, createdAt,
	).Scan(&orderID); err != nil {
		t.Fatalf("insert order: %v", err)
	}

	t.Cleanup(func() {
		_, _ = pool.Exec(ctx, `DELETE FROM orders WHERE id = $1`, orderID)
		_, _ = pool.Exec(ctx, `DELETE FROM vehicles WHERE id = $1`, vehicleID)
		_, _ = pool.Exec(ctx, `DELETE FROM customers WHERE id = $1`, customerID)
	})
	return orderID
}

// listOrders calls the handler in-process and returns the status and the
// decoded order list.
func listOrders(t *testing.T, pool store.Pool, query string) (int, []model.OrderSummary) {
	t.Helper()
	h := NewHandler(pool)
	req := httptest.NewRequest(http.MethodGet, "/api/workshop/orders"+query, nil)
	rec := httptest.NewRecorder()
	h.ListOrders(rec, req)

	resp := rec.Result()
	defer resp.Body.Close()

	var body listOrdersResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode response (status %d): %v", resp.StatusCode, err)
	}
	return resp.StatusCode, body.Orders
}

// find returns the summary with the given id, or nil.
func find(orders []model.OrderSummary, id int) *model.OrderSummary {
	for i := range orders {
		if orders[i].ID == id {
			return &orders[i]
		}
	}
	return nil
}

func TestListOrdersReturnsEveryOrderWithItsFields(t *testing.T) {
	ctx, pool := newTestPool(t)

	id := createOrderFixture(t, ctx, pool, "B-MW 1234", "BMW", "320d", model.StatusRequested, "2026-11-02")

	status, orders := listOrders(t, pool, "")
	if status != http.StatusOK {
		t.Fatalf("status = %d, want 200", status)
	}
	got := find(orders, id)
	if got == nil {
		t.Fatalf("order %d is missing from the list", id)
	}
	if got.Plate != "B-MW 1234" || got.Brand != "BMW" || got.Model != "320d" {
		t.Errorf("vehicle fields = %q/%q/%q, want B-MW 1234/BMW/320d", got.Plate, got.Brand, got.Model)
	}
	if got.Status != model.StatusRequested {
		t.Errorf("status = %q, want %q", got.Status, model.StatusRequested)
	}
	if got.RequestedDate != "2026-11-02" {
		t.Errorf("requestedDate = %q, want 2026-11-02", got.RequestedDate)
	}
	if got.OrderNumber == "" {
		t.Error("orderNumber is empty")
	}
	created, err := time.Parse(time.RFC3339, got.CreatedAt)
	if err != nil {
		t.Fatalf("createdAt %q is not RFC3339: %v", got.CreatedAt, err)
	}
	if created.Location() != time.UTC {
		t.Errorf("createdAt is not UTC: %q", got.CreatedAt)
	}
}

func TestListOrdersFiltersByStatus(t *testing.T) {
	ctx, pool := newTestPool(t)

	confirmedID := createOrderFixture(t, ctx, pool, "HD-KL 11", "Audi", "A4", model.StatusConfirmed, "2026-11-03")
	createOrderFixture(t, ctx, pool, "HD-KL 12", "Audi", "A4", model.StatusRequested, "2026-11-04")

	status, orders := listOrders(t, pool, "?status="+model.StatusConfirmed)
	if status != http.StatusOK {
		t.Fatalf("status = %d, want 200", status)
	}
	for _, o := range orders {
		if o.Status != model.StatusConfirmed {
			t.Errorf("order %d has status %q, want only confirmed", o.ID, o.Status)
		}
	}
	if find(orders, confirmedID) == nil {
		t.Errorf("confirmed order %d is missing from the filtered list", confirmedID)
	}
}

func TestListOrdersSearchesPlateFragmentCaseInsensitively(t *testing.T) {
	ctx, pool := newTestPool(t)

	id := createOrderFixture(t, ctx, pool, "B-MW 1234", "BMW", "320d", model.StatusRequested, "2026-11-02")

	for _, fragment := range []string{"1234", "b-mw", "BMW12"} {
		status, orders := listOrders(t, pool, "?plate="+url.QueryEscape(fragment))
		if status != http.StatusOK {
			t.Fatalf("plate %q: status = %d, want 200", fragment, status)
		}
		if find(orders, id) == nil {
			t.Errorf("plate fragment %q did not find order %d", fragment, id)
		}
	}

	status, orders := listOrders(t, pool, "?plate="+url.QueryEscape("ZZ-ZZ 99"))
	if status != http.StatusOK {
		t.Fatalf("status = %d, want 200", status)
	}
	if find(orders, id) != nil {
		t.Errorf("a non-matching plate fragment returned order %d", id)
	}
}

func TestListOrdersCombinesStatusAndPlateFilters(t *testing.T) {
	ctx, pool := newTestPool(t)

	confirmedID := createOrderFixture(t, ctx, pool, "HD-KL 99", "Ford", "Focus", model.StatusConfirmed, "2026-11-05")
	requestedID := createOrderFixture(t, ctx, pool, "HD-KL 98", "Ford", "Focus", model.StatusRequested, "2026-11-06")

	status, orders := listOrders(t, pool, "?status="+model.StatusConfirmed+"&plate="+url.QueryEscape("KL"))
	if status != http.StatusOK {
		t.Fatalf("status = %d, want 200", status)
	}
	if find(orders, confirmedID) == nil {
		t.Errorf("confirmed order %d is missing when both filters apply", confirmedID)
	}
	if find(orders, requestedID) != nil {
		t.Errorf("requested order %d leaked through the status filter", requestedID)
	}
	for _, o := range orders {
		if o.Status != model.StatusConfirmed {
			t.Errorf("order %d has status %q, want only confirmed", o.ID, o.Status)
		}
	}
}

// TestListOrdersInjectionAttemptIsSafe proves AC-24 for the search fields: a
// value like `' OR 1=1 --` is bound as a parameter, so it matches no plate and
// the handler answers a clean empty list, never a database error and never a
// foreign row.
func TestListOrdersInjectionAttemptIsSafe(t *testing.T) {
	ctx, pool := newTestPool(t)

	createOrderFixture(t, ctx, pool, "B-MW 1234", "BMW", "320d", model.StatusRequested, "2026-11-02")

	status, orders := listOrders(t, pool, "?plate="+url.QueryEscape("' OR 1=1 --"))
	if status != http.StatusOK {
		t.Fatalf("injection in plate: status = %d, want 200", status)
	}
	if len(orders) != 0 {
		t.Errorf("injection in plate returned %d rows, want 0", len(orders))
	}

	status, orders = listOrders(t, pool, "?status="+url.QueryEscape("' OR 1=1 --"))
	if status != http.StatusOK {
		t.Fatalf("injection in status: status = %d, want 200", status)
	}
	if len(orders) != 0 {
		t.Errorf("injection in status returned %d rows, want 0", len(orders))
	}
}

// TestListOrdersEmptyResultIsAnArray proves the response body carries an empty
// JSON array (not null) when nothing matches.
func TestListOrdersEmptyResultIsAnArray(t *testing.T) {
	_, pool := newTestPool(t)

	h := NewHandler(pool)
	req := httptest.NewRequest(http.MethodGet, "/api/workshop/orders?plate="+url.QueryEscape("NOPE-NOPE 0"), nil)
	rec := httptest.NewRecorder()
	h.ListOrders(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if body := strings.TrimSpace(rec.Body.String()); body != `{"orders":[]}` {
		t.Errorf("body = %s, want {\"orders\":[]}", body)
	}
}
