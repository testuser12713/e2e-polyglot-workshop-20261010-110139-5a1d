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
	"testing"
	"time"

	"workshop/backend/internal/model"
	"workshop/backend/internal/order"
	"workshop/backend/internal/store"
)

// pool is the package-wide PostgreSQL pool. It points at a private schema so
// the dashboard counts are not polluted by the rows other test packages write
// into the same database in parallel. It is nil when DATABASE_URL is absent and
// every test then skips (there is no SQLite fallback, AC-20).
var pool store.Pool

func TestMain(m *testing.M) {
	os.Exit(run(m))
}

func run(m *testing.M) int {
	base := os.Getenv("DATABASE_URL")
	if base == "" || strings.Contains(strings.ToLower(base), "sqlite") {
		return m.Run()
	}

	ctx := context.Background()
	admin, err := store.NewPool(ctx, base)
	if err != nil {
		fmt.Fprintf(os.Stderr, "dashboard_test: open postgres: %v\n", err)
		return 1
	}
	defer admin.Close()

	schema := fmt.Sprintf("ws_dashboard_%d", time.Now().UnixNano())
	if _, err := admin.Exec(ctx, "CREATE SCHEMA "+schema); err != nil {
		fmt.Fprintf(os.Stderr, "dashboard_test: create schema: %v\n", err)
		return 1
	}
	defer func() { _, _ = admin.Exec(ctx, "DROP SCHEMA "+schema+" CASCADE") }()

	poolURL, err := withSearchPath(base, schema)
	if err != nil {
		fmt.Fprintf(os.Stderr, "dashboard_test: build pool url: %v\n", err)
		return 1
	}
	p, err := store.NewPool(ctx, poolURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "dashboard_test: open postgres: %v\n", err)
		return 1
	}
	defer p.Close()
	if err := store.Migrate(ctx, p); err != nil {
		fmt.Fprintf(os.Stderr, "dashboard_test: migrate: %v\n", err)
		return 1
	}
	pool = p

	return m.Run()
}

// withSearchPath returns the connection URL with the search_path runtime
// parameter set to schema, so every pooled connection resolves unqualified
// table names inside that schema.
func withSearchPath(rawURL, schema string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}
	q := u.Query()
	q.Set("search_path", schema)
	u.RawQuery = q.Encode()
	return u.String(), nil
}

// requirePool returns the package pool or skips the test when no database is
// configured.
func requirePool(t *testing.T) store.Pool {
	t.Helper()
	if pool == nil {
		t.Skip("DATABASE_URL is not set; the office runs this test against PostgreSQL")
	}
	return pool
}

// resetDB clears the tables this package writes to. The pool lives in its own
// schema, so no other test owns these rows.
func resetDB(t *testing.T, pool store.Pool) {
	t.Helper()
	_, err := pool.Exec(context.Background(),
		`TRUNCATE order_status_log, order_items, invoices, invoice_items, orders, vehicles, customers RESTART IDENTITY CASCADE`)
	if err != nil {
		t.Fatalf("reset tables: %v", err)
	}
}

// seedVehicle inserts a customer and a vehicle and returns the vehicle id.
func seedVehicle(t *testing.T, pool store.Pool) int {
	t.Helper()
	ctx := context.Background()
	suffix := time.Now().UnixNano()
	var customerID int
	if err := pool.QueryRow(ctx,
		`INSERT INTO customers (name, email, phone) VALUES ($1, $2, $3) RETURNING id`,
		"Test Kunde", "kunde@example.test", "0000-000000").Scan(&customerID); err != nil {
		t.Fatalf("insert customer: %v", err)
	}
	plate := fmt.Sprintf("TEST-%d", suffix)
	var vehicleID int
	if err := pool.QueryRow(ctx,
		`INSERT INTO vehicles (customer_id, plate, plate_normalized, brand, model, mileage_km)
		 VALUES ($1, $2, $3, $4, $5, $6) RETURNING id`,
		customerID, plate, plate, "VW", "Golf", 120000).Scan(&vehicleID); err != nil {
		t.Fatalf("insert vehicle: %v", err)
	}
	return vehicleID
}

// createOrder creates an order for the seed vehicle through the shared order
// service, exactly as the product does.
func createOrder(t *testing.T, pool store.Pool, vehicleID int) model.Order {
	t.Helper()
	ord, err := order.CreateOrder(context.Background(), pool, vehicleID, "2026-11-05", "Bremsen")
	if err != nil {
		t.Fatalf("CreateOrder: %v", err)
	}
	return ord
}

// setStatus moves an order's status and appends a status-log entry at the given
// time, mirroring what the workflow writes.
func setStatus(t *testing.T, pool store.Pool, orderID int, status string, changedAt time.Time) {
	t.Helper()
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `UPDATE orders SET status = $1 WHERE id = $2`, status, orderID); err != nil {
		t.Fatalf("update order status: %v", err)
	}
	if _, err := pool.Exec(ctx,
		`INSERT INTO order_status_log (order_id, status, changed_at) VALUES ($1, $2, $3)`,
		orderID, status, changedAt); err != nil {
		t.Fatalf("insert status log: %v", err)
	}
}

// insertInvoice writes an invoice row directly, as the worker does.
func insertInvoice(t *testing.T, pool store.Pool, orderID, grossCents int, createdAt time.Time) {
	t.Helper()
	number := fmt.Sprintf("RE-%d-%d", orderID, createdAt.UnixNano())
	if _, err := pool.Exec(context.Background(),
		`INSERT INTO invoices (invoice_number, order_id, net_cents, tax_rate_percent, tax_cents, gross_cents, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		number, orderID, grossCents, 19, grossCents-grossCents/119, grossCents, createdAt); err != nil {
		t.Fatalf("insert invoice: %v", err)
	}
}

// dashboardServer mounts the dashboard on the route the router registers.
func dashboardServer(t *testing.T, pool store.Pool) *httptest.Server {
	t.Helper()
	h := NewHandler(pool)
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/workshop/dashboard", h.Dashboard)
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func getDashboard(t *testing.T, srv *httptest.Server) DashboardResponse {
	t.Helper()
	resp, err := http.Get(srv.URL + "/api/workshop/dashboard")
	if err != nil {
		t.Fatalf("GET dashboard: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var body DashboardResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode dashboard: %v", err)
	}
	return body
}

func TestDashboardOpenCountsPerStatus(t *testing.T) {
	p := requirePool(t)
	resetDB(t, p)
	vehicleID := seedVehicle(t, p)

	createOrder(t, p, vehicleID)
	createOrder(t, p, vehicleID)

	confirmed := createOrder(t, p, vehicleID)
	setStatus(t, p, confirmed.ID, model.StatusConfirmed, time.Now().UTC())

	inProgress := createOrder(t, p, vehicleID)
	setStatus(t, p, inProgress.ID, model.StatusInProgress, time.Now().UTC())

	closed := createOrder(t, p, vehicleID)
	setStatus(t, p, closed.ID, model.StatusCompleted, time.Now().UTC())

	srv := dashboardServer(t, p)
	got := getDashboard(t, srv)

	if got.OpenCounts.Requested != 2 {
		t.Errorf("requested = %d, want 2", got.OpenCounts.Requested)
	}
	if got.OpenCounts.Confirmed != 1 {
		t.Errorf("confirmed = %d, want 1", got.OpenCounts.Confirmed)
	}
	if got.OpenCounts.InProgress != 1 {
		t.Errorf("in_progress = %d, want 1", got.OpenCounts.InProgress)
	}
}

func TestDashboardCompletedTodayHonoursUTCDay(t *testing.T) {
	p := requirePool(t)
	resetDB(t, p)
	vehicleID := seedVehicle(t, p)

	today := createOrder(t, p, vehicleID)
	setStatus(t, p, today.ID, model.StatusCompleted, time.Now().UTC())

	yesterday := createOrder(t, p, vehicleID)
	setStatus(t, p, yesterday.ID, model.StatusCompleted, time.Now().UTC().Add(-25*time.Hour))

	srv := dashboardServer(t, p)
	got := getDashboard(t, srv)

	if got.CompletedToday != 1 {
		t.Errorf("completedToday = %d, want 1 (only the order completed today)", got.CompletedToday)
	}
}

func TestDashboardCompletedStillCountsAfterPickedUp(t *testing.T) {
	p := requirePool(t)
	resetDB(t, p)
	vehicleID := seedVehicle(t, p)

	ord := createOrder(t, p, vehicleID)
	setStatus(t, p, ord.ID, model.StatusCompleted, time.Now().UTC())
	setStatus(t, p, ord.ID, model.StatusPickedUp, time.Now().UTC())

	srv := dashboardServer(t, p)
	got := getDashboard(t, srv)

	if got.CompletedToday != 1 {
		t.Errorf("completedToday = %d, want 1 (an order that reached completed today)", got.CompletedToday)
	}
}

func TestDashboardMonthRevenueGrossCents(t *testing.T) {
	p := requirePool(t)
	resetDB(t, p)
	vehicleID := seedVehicle(t, p)

	now := time.Now().UTC()
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	lastMonth := monthStart.Add(-time.Hour)

	firstCurrent := createOrder(t, p, vehicleID)
	insertInvoice(t, p, firstCurrent.ID, 11900, now)
	secondCurrent := createOrder(t, p, vehicleID)
	insertInvoice(t, p, secondCurrent.ID, 1000, now)

	old := createOrder(t, p, vehicleID)
	insertInvoice(t, p, old.ID, 5000, lastMonth)

	srv := dashboardServer(t, p)
	got := getDashboard(t, srv)

	if got.MonthRevenueGrossCents != 12900 {
		t.Errorf("monthRevenueGrossCents = %d, want 12900 (only invoices of the current month)", got.MonthRevenueGrossCents)
	}
}

func TestDashboardEmptyDatabaseIsAllZero(t *testing.T) {
	p := requirePool(t)
	resetDB(t, p)

	srv := dashboardServer(t, p)
	got := getDashboard(t, srv)

	if got.OpenCounts.Requested != 0 || got.OpenCounts.Confirmed != 0 || got.OpenCounts.InProgress != 0 {
		t.Errorf("openCounts = %+v, want all zero", got.OpenCounts)
	}
	if got.CompletedToday != 0 {
		t.Errorf("completedToday = %d, want 0", got.CompletedToday)
	}
	if got.MonthRevenueGrossCents != 0 {
		t.Errorf("monthRevenueGrossCents = %d, want 0", got.MonthRevenueGrossCents)
	}
}
