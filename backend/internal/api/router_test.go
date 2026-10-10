package api

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"workshop/backend/internal/config"
	"workshop/backend/internal/httpx"
	"workshop/backend/internal/model"
	"workshop/backend/internal/store"
)

const testOrigin = "http://localhost:5173"

func newTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(NewRouter(Deps{Config: &config.Config{CORSOrigin: testOrigin}}))
	t.Cleanup(srv.Close)
	return srv
}

func decodeError(t *testing.T, body []byte) httpx.ErrorBody {
	t.Helper()
	var eb httpx.ErrorBody
	if err := json.Unmarshal(body, &eb); err != nil {
		t.Fatalf("error body is not JSON: %v (%s)", err, body)
	}
	return eb
}

func TestHealthRoute(t *testing.T) {
	srv := newTestServer(t)

	resp, err := http.Get(srv.URL + "/api/health")
	if err != nil {
		t.Fatalf("GET /api/health: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /api/health: status %d, want 200", resp.StatusCode)
	}
	var body map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body["status"] != "ok" {
		t.Fatalf("status = %q, want %q", body["status"], "ok")
	}
}

func TestUnknownRouteReturnsUnifiedNotFound(t *testing.T) {
	srv := newTestServer(t)

	resp, err := http.Get(srv.URL + "/api/does-not-exist")
	if err != nil {
		t.Fatalf("GET unknown route: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status %d, want 404", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Fatalf("Content-Type = %q, want application/json", ct)
	}
	raw, _ := io.ReadAll(resp.Body)
	eb := decodeError(t, raw)
	if eb.Error.Code != model.CodeNotFound {
		t.Fatalf("code = %q, want %q", eb.Error.Code, model.CodeNotFound)
	}
	if eb.Error.Message == "" {
		t.Fatal("message is empty")
	}
}

func TestMalformedJSONReturnsValidationError(t *testing.T) {
	srv := newTestServer(t)

	resp, err := http.Post(srv.URL+"/api/auth/login", "application/json", strings.NewReader("{not valid json"))
	if err != nil {
		t.Fatalf("POST malformed body: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("status %d, want 422", resp.StatusCode)
	}
	raw, _ := io.ReadAll(resp.Body)
	eb := decodeError(t, raw)
	if eb.Error.Code != model.CodeValidation {
		t.Fatalf("code = %q, want %q", eb.Error.Code, model.CodeValidation)
	}
}

// TestLoginRouteIsRegistered proves the login route is wired under the agreed
// path and verb: it is reachable and never answers 404/405. It asserts nothing
// about the temporary stub body, so it stays true after ticket #9 lands.
func TestLoginRouteIsRegistered(t *testing.T) {
	srv := newTestServer(t)

	resp, err := http.Post(srv.URL+"/api/auth/login", "application/json", strings.NewReader(`{}`))
	if err != nil {
		t.Fatalf("POST /api/auth/login: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusMethodNotAllowed {
		t.Fatalf("login route is not registered: status %d", resp.StatusCode)
	}
}

func TestCORSSameOriginIsAllowed(t *testing.T) {
	srv := newTestServer(t)

	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/api/health", nil)
	req.Header.Set("Origin", testOrigin)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if got := resp.Header.Get("Access-Control-Allow-Origin"); got != testOrigin {
		t.Fatalf("Access-Control-Allow-Origin = %q, want %q", got, testOrigin)
	}
}

func TestCORSForeignOriginIsRejected(t *testing.T) {
	srv := newTestServer(t)

	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/api/health", nil)
	req.Header.Set("Origin", "http://evil.example")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if got := resp.Header.Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("Access-Control-Allow-Origin = %q, want no header", got)
	}
}

func TestCORSForeignPreflightGetsNoGrant(t *testing.T) {
	srv := newTestServer(t)

	req, _ := http.NewRequest(http.MethodOptions, srv.URL+"/api/workshop/orders", nil)
	req.Header.Set("Origin", "http://evil.example")
	req.Header.Set("Access-Control-Request-Method", "POST")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if got := resp.Header.Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("Access-Control-Allow-Origin = %q, want no header", got)
	}
}

// TestRouterWithRealPool wires httptest to a real PostgreSQL pool: it migrates
// the schema, builds the router on that pool and probes it. This is the
// httptest-over-PostgreSQL path AC-20 asks for, with no SQLite anywhere.
func TestRouterWithRealPool(t *testing.T) {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Skip("DATABASE_URL is not set; the office runs this test against PostgreSQL")
	}
	ctx := context.Background()
	pool, err := store.NewPool(ctx, url)
	if err != nil {
		t.Fatalf("open postgres: %v", err)
	}
	defer pool.Close()
	if err := store.Migrate(ctx, pool); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	srv := httptest.NewServer(NewRouter(Deps{Config: &config.Config{CORSOrigin: testOrigin}, Pool: pool}))
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/health")
	if err != nil {
		t.Fatalf("GET /api/health: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /api/health on a real pool: status %d, want 200", resp.StatusCode)
	}
}

// TestMigrateAgainstPostgres proves the schema is created on a real PostgreSQL
// database — never SQLite (AC-20). The office provides DATABASE_URL; without it
// the test is skipped rather than silently faking a database.
func TestMigrateAgainstPostgres(t *testing.T) {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Skip("DATABASE_URL is not set; the office runs this test against PostgreSQL")
	}
	if strings.Contains(strings.ToLower(url), "sqlite") {
		t.Fatalf("DATABASE_URL points at SQLite; this product requires PostgreSQL")
	}

	ctx := context.Background()
	pool, err := store.NewPool(ctx, url)
	if err != nil {
		t.Fatalf("open postgres: %v", err)
	}
	defer pool.Close()

	if err := store.Migrate(ctx, pool); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	tables := []string{
		"customers", "vehicles", "orders", "order_items", "order_status_log",
		"employees", "invoices", "invoice_items", "outbox",
	}
	for _, table := range tables {
		var exists bool
		if err := pool.QueryRow(ctx,
			`SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema='public' AND table_name=$1)`,
			table).Scan(&exists); err != nil {
			t.Fatalf("check table %s: %v", table, err)
		}
		if !exists {
			t.Errorf("table %s was not created", table)
		}
	}

	uniqueIndexes := []string{
		"vehicles_plate_normalized_key",
		"orders_order_number_key",
		"invoices_invoice_number_key",
		"invoices_order_id_key",
	}
	for _, idx := range uniqueIndexes {
		var exists bool
		if err := pool.QueryRow(ctx,
			`SELECT EXISTS (SELECT 1 FROM pg_indexes WHERE schemaname='public' AND indexname=$1)`,
			idx).Scan(&exists); err != nil {
			t.Fatalf("check index %s: %v", idx, err)
		}
		if !exists {
			t.Errorf("unique constraint %s is missing", idx)
		}
	}
}
