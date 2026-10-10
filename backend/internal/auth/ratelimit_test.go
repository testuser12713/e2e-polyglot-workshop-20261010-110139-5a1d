package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"

	"workshop/backend/internal/httpx"
)

// newTestValkey connects to the real Valkey the office runs and skips the test
// when the address is not configured or the server is not reachable. There is no
// in-memory stand-in: the limiter's behaviour is defined by real Valkey calls.
func newTestValkey(t *testing.T) *redis.Client {
	t.Helper()
	raw := strings.TrimSpace(os.Getenv("VALKEY_URL"))
	if raw == "" {
		t.Skip("VALKEY_URL is not set; the office runs this test against Valkey")
	}
	opts, err := redis.ParseURL(raw)
	if err != nil {
		t.Fatalf("parse VALKEY_URL: %v", err)
	}
	client := redis.NewClient(opts)
	t.Cleanup(func() { _ = client.Close() })

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		t.Skipf("Valkey unreachable: %v", err)
	}
	return client
}

// cleanRateLimitKey removes any counter left behind for a test client address.
func cleanRateLimitKey(t *testing.T, client *redis.Client, addr string) {
	t.Helper()
	if err := client.Del(context.Background(), rateLimitKeyPrefix+addr).Err(); err != nil {
		t.Fatalf("clean rate limit key: %v", err)
	}
}

// fakeLogin mimics the login handler the router wires next to the limiter: a
// wrong password answers 401 and a correct one answers 200 with a token. The
// limiter's contract (count on failure, reset on success) is what is under test,
// not the login implementation another ticket owns.
func fakeLogin(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	if body.Password == "correct-horse" {
		httpx.JSON(w, http.StatusOK, map[string]string{"token": "fake-token", "expiresAt": "2030-01-01T00:00:00Z"})
		return
	}
	httpx.Error(w, httpx.CodeUnauthorized, http.StatusUnauthorized, "invalid credentials")
}

// loginAttempt drives one request through the middleware in-process, so the
// client address the limiter keys on can be set for real (a served request
// overwrites RemoteAddr with the socket address).
func loginAttempt(t *testing.T, handler http.Handler, addr, password string) (int, []byte) {
	t.Helper()
	payload := `{"email":"employee@example.com","password":"` + password + `"}`
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = addr
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	resp := rec.Result()
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	return resp.StatusCode, raw
}

func TestRateLimitBlocksAfterTenFailedAttempts(t *testing.T) {
	valkey := newTestValkey(t)
	addr := "203.0.113.10:40001"
	cleanRateLimitKey(t, valkey, "203.0.113.10")

	handler := rateLimit(http.HandlerFunc(fakeLogin), valkey)

	for i := 1; i <= 10; i++ {
		status, _ := loginAttempt(t, handler, addr, "wrong")
		if status != http.StatusUnauthorized {
			t.Fatalf("failed attempt %d: status %d, want 401", i, status)
		}
	}

	status, raw := loginAttempt(t, handler, addr, "wrong")
	if status != http.StatusTooManyRequests {
		t.Fatalf("eleventh attempt: status %d, want 429", status)
	}

	var eb httpx.ErrorBody
	if err := json.Unmarshal(raw, &eb); err != nil {
		t.Fatalf("429 body is not the shared error shape: %v (%s)", err, raw)
	}
	if eb.Error.Code != "rate_limited" {
		t.Fatalf("429 code = %q, want %q", eb.Error.Code, "rate_limited")
	}
	if eb.Error.Message == "" {
		t.Fatal("429 message is empty")
	}
	if bytes.Contains(raw, []byte("token")) {
		t.Fatalf("429 body must not contain a token: %s", raw)
	}

	status, raw = loginAttempt(t, handler, addr, "correct-horse")
	if status != http.StatusTooManyRequests {
		t.Fatalf("attempt while limited: status %d, want 429", status)
	}
	if bytes.Contains(raw, []byte("token")) {
		t.Fatalf("limited response must not contain a token: %s", raw)
	}
}

func TestRateLimitResetsAfterSuccessfulLogin(t *testing.T) {
	valkey := newTestValkey(t)
	addr := "203.0.113.11:40002"
	cleanRateLimitKey(t, valkey, "203.0.113.11")

	handler := rateLimit(http.HandlerFunc(fakeLogin), valkey)

	for i := 0; i < 5; i++ {
		if status, _ := loginAttempt(t, handler, addr, "wrong"); status != http.StatusUnauthorized {
			t.Fatalf("pre-success failure %d: status %d, want 401", i+1, status)
		}
	}

	status, raw := loginAttempt(t, handler, addr, "correct-horse")
	if status != http.StatusOK {
		t.Fatalf("successful login: status %d, want 200 (%s)", status, raw)
	}
	if !bytes.Contains(raw, []byte("token")) {
		t.Fatalf("successful login must return a token: %s", raw)
	}

	// The success cleared the counter: a full fresh allowance is available again.
	for i := 1; i <= 10; i++ {
		if status, _ := loginAttempt(t, handler, addr, "wrong"); status != http.StatusUnauthorized {
			t.Fatalf("post-success failure %d: status %d, want 401 (counter was not reset)", i, status)
		}
	}
	if status, _ := loginAttempt(t, handler, addr, "wrong"); status != http.StatusTooManyRequests {
		t.Fatalf("after a fresh ten failures: status %d, want 429", status)
	}
}

func TestRateLimitCounterCarriesTTL(t *testing.T) {
	valkey := newTestValkey(t)
	addr := "203.0.113.15:40006"
	cleanRateLimitKey(t, valkey, "203.0.113.15")

	handler := rateLimit(http.HandlerFunc(fakeLogin), valkey)

	if status, _ := loginAttempt(t, handler, addr, "wrong"); status != http.StatusUnauthorized {
		t.Fatalf("failed attempt: status %d, want 401", status)
	}

	ttl, err := valkey.TTL(context.Background(), rateLimitKeyPrefix+"203.0.113.15").Result()
	if err != nil {
		t.Fatalf("read TTL: %v", err)
	}
	if ttl <= 0 || ttl > LoginWindow {
		t.Fatalf("counter TTL = %s, want >0 and <=%s", ttl, LoginWindow)
	}
}

func TestRateLimitIsPerClient(t *testing.T) {
	valkey := newTestValkey(t)
	blocked := "203.0.113.12:40003"
	other := "203.0.113.13:40004"
	cleanRateLimitKey(t, valkey, "203.0.113.12")
	cleanRateLimitKey(t, valkey, "203.0.113.13")

	handler := rateLimit(http.HandlerFunc(fakeLogin), valkey)

	for i := 0; i < 10; i++ {
		if status, _ := loginAttempt(t, handler, blocked, "wrong"); status != http.StatusUnauthorized {
			t.Fatalf("blocked client failure %d: status %d, want 401", i+1, status)
		}
	}
	if status, _ := loginAttempt(t, handler, blocked, "wrong"); status != http.StatusTooManyRequests {
		t.Fatalf("blocked client: status %d, want 429", status)
	}
	if status, _ := loginAttempt(t, handler, other, "wrong"); status != http.StatusUnauthorized {
		t.Fatalf("other client: status %d, want 401 (limit is per client)", status)
	}
}

func TestRateLimitDegradesWhenValkeyUnreachable(t *testing.T) {
	// Nothing listens on this port, so every Valkey call fails fast.
	unreachable := redis.NewClient(&redis.Options{
		Addr:         "127.0.0.1:1",
		DialTimeout:  50 * time.Millisecond,
		ReadTimeout:  50 * time.Millisecond,
		WriteTimeout: 50 * time.Millisecond,
		MaxRetries:   -1,
	})
	defer func() { _ = unreachable.Close() }()

	handler := rateLimit(http.HandlerFunc(fakeLogin), unreachable)

	for i := 0; i < 3; i++ {
		status, _ := loginAttempt(t, handler, "203.0.113.14:40005", "wrong")
		if status != http.StatusUnauthorized {
			t.Fatalf("attempt %d while Valkey is down: status %d, want 401 (requests must pass through)", i+1, status)
		}
	}
}
