package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"

	"workshop/backend/internal/httpx"
	"workshop/backend/internal/model"
	"workshop/backend/internal/store"
)

const testJWTSecret = "test-secret-not-a-real-key"

// testPool opens the real PostgreSQL database and creates the schema. It skips
// when DATABASE_URL is absent (a workstation without the office's database) and
// refuses SQLite outright — this product runs on PostgreSQL only (AC-20).
func testPool(t *testing.T) store.Pool {
	t.Helper()
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
	t.Cleanup(pool.Close)
	if err := store.Migrate(ctx, pool); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return pool
}

// uniqueEmail returns an address no other test run uses, so the employees table
// is shared without tests colliding.
func uniqueEmail() string {
	return fmt.Sprintf("seed-%d@example.test", time.Now().UnixNano())
}

func seedForTest(t *testing.T, pool store.Pool, email, password string) {
	t.Helper()
	if err := SeedFirstEmployee(context.Background(), pool, email, password); err != nil {
		t.Fatalf("seed employee: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM employees WHERE email = $1`, email)
	})
}

func login(t *testing.T, email, password string) *httptest.ResponseRecorder {
	t.Helper()
	body := fmt.Sprintf(`{"email":%q,"password":%q}`, email, password)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(body))
	rec := httptest.NewRecorder()
	Login(rec, req)
	return rec
}

func decodeErrorBody(t *testing.T, raw string) httpx.ErrorBody {
	t.Helper()
	var eb httpx.ErrorBody
	if err := json.Unmarshal([]byte(raw), &eb); err != nil {
		t.Fatalf("error body is not JSON: %v (%s)", err, raw)
	}
	return eb
}

func TestSeedFirstEmployeeIsIdempotentAndStoresOnlyHash(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	email := uniqueEmail()
	password := "correct-horse-battery"
	seedForTest(t, pool, email, password)

	if err := SeedFirstEmployee(ctx, pool, email, password); err != nil {
		t.Fatalf("second seed: %v", err)
	}

	var count int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM employees WHERE email = $1`, email).Scan(&count); err != nil {
		t.Fatalf("count employees: %v", err)
	}
	if count != 1 {
		t.Fatalf("employee count = %d, want exactly 1 after two starts", count)
	}

	var hash string
	if err := pool.QueryRow(ctx, `SELECT password_hash FROM employees WHERE email = $1`, email).Scan(&hash); err != nil {
		t.Fatalf("read password hash: %v", err)
	}
	if hash == password {
		t.Fatalf("password stored in clear text")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)); err != nil {
		t.Fatalf("stored hash does not match the password: %v", err)
	}
}

func TestLoginSucceedsAndIssuesUsableToken(t *testing.T) {
	t.Setenv("JWT_SECRET", testJWTSecret)
	pool := testPool(t)
	email := uniqueEmail()
	password := "a-good-password"
	seedForTest(t, pool, email, password)

	rec := login(t, email, password)
	if rec.Code != http.StatusOK {
		t.Fatalf("login status = %d, want 200 (body %s)", rec.Code, rec.Body.String())
	}

	var resp struct {
		Token     string `json:"token"`
		ExpiresAt string `json:"expiresAt"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode login response: %v", err)
	}
	if resp.Token == "" {
		t.Fatal("login response has no token")
	}
	if _, err := time.Parse(time.RFC3339, resp.ExpiresAt); err != nil {
		t.Fatalf("expiresAt is not RFC3339: %v (%q)", err, resp.ExpiresAt)
	}

	claims, err := VerifyToken(resp.Token)
	if err != nil {
		t.Fatalf("verify issued token: %v", err)
	}
	if claims.Email != email {
		t.Fatalf("claims email = %q, want %q", claims.Email, email)
	}
	if claims.Sub == "" {
		t.Fatal("claims sub is empty")
	}
	if claims.Exp <= time.Now().Unix() {
		t.Fatalf("token expires in the past: %d", claims.Exp)
	}

	if strings.Contains(rec.Body.String(), password) {
		t.Fatal("login response contains the plain password")
	}
}

func TestLoginWrongPasswordIsUnauthorized(t *testing.T) {
	t.Setenv("JWT_SECRET", testJWTSecret)
	pool := testPool(t)
	email := uniqueEmail()
	seedForTest(t, pool, email, "the-right-password")

	rec := login(t, email, "the-wrong-password")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
	eb := decodeErrorBody(t, rec.Body.String())
	if eb.Error.Code != model.CodeUnauthorized {
		t.Fatalf("error code = %q, want %q", eb.Error.Code, model.CodeUnauthorized)
	}
	if strings.Contains(rec.Body.String(), "token") {
		t.Fatal("unauthorized response must not carry a token")
	}
}

func TestLoginUnknownEmailIsUnauthorized(t *testing.T) {
	t.Setenv("JWT_SECRET", testJWTSecret)
	pool := testPool(t)
	loginQuerier = pool

	rec := login(t, uniqueEmail(), "whatever-password")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
	eb := decodeErrorBody(t, rec.Body.String())
	if eb.Error.Code != model.CodeUnauthorized {
		t.Fatalf("error code = %q, want %q", eb.Error.Code, model.CodeUnauthorized)
	}
}

func TestValidTokenReachesWorkshopRoute(t *testing.T) {
	t.Setenv("JWT_SECRET", testJWTSecret)

	reached := false
	var employeeID int
	guarded := RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reached = true
		employeeID, _ = EmployeeIDFromContext(r.Context())
		httpx.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
	}))

	token, err := IssueToken(42, "boss@example.test")
	if err != nil {
		t.Fatalf("issue token: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/workshop/orders", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	guarded.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %s)", rec.Code, rec.Body.String())
	}
	if !reached {
		t.Fatal("guarded workshop handler was not reached with a valid token")
	}
	if employeeID != 42 {
		t.Fatalf("employee id from context = %d, want 42", employeeID)
	}
}

// assertWorkshopRouteBlocked drives a guarded handler with the given token and
// asserts the caller gets 401 and the handler is never reached.
func assertWorkshopRouteBlocked(t *testing.T, token string) {
	t.Helper()
	reached := false
	guarded := RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reached = true
		httpx.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/workshop/orders", nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	guarded.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
	if reached {
		t.Fatal("workshop handler was reached without a valid token")
	}
	eb := decodeErrorBody(t, rec.Body.String())
	if eb.Error.Code != model.CodeUnauthorized {
		t.Fatalf("error code = %q, want %q", eb.Error.Code, model.CodeUnauthorized)
	}
}

func TestMissingTokenIsRejected(t *testing.T) {
	t.Setenv("JWT_SECRET", testJWTSecret)
	assertWorkshopRouteBlocked(t, "")
}

func TestMalformedTokenIsRejected(t *testing.T) {
	t.Setenv("JWT_SECRET", testJWTSecret)
	assertWorkshopRouteBlocked(t, "not.a.jwt")
}

func TestTamperedTokenIsRejected(t *testing.T) {
	t.Setenv("JWT_SECRET", testJWTSecret)
	token, err := IssueToken(7, "boss@example.test")
	if err != nil {
		t.Fatalf("issue token: %v", err)
	}
	assertWorkshopRouteBlocked(t, token+"x")
}

func TestExpiredTokenIsRejected(t *testing.T) {
	t.Setenv("JWT_SECRET", testJWTSecret)
	expired, err := signToken(model.Claims{
		Sub:   "7",
		Email: "boss@example.test",
		Exp:   time.Now().Add(-time.Minute).Unix(),
	})
	if err != nil {
		t.Fatalf("sign expired token: %v", err)
	}
	assertWorkshopRouteBlocked(t, expired)
}
