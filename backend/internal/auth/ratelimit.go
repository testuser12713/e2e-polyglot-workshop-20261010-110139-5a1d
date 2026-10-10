package auth

import (
	"context"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"

	"workshop/backend/internal/httpx"
)

// Rate limiting parameters for the login endpoint (AC-22: a client gets at most
// ten failed attempts inside a window of fifteen minutes).
const (
	// LoginMaxAttempts is the number of failed attempts a client may make inside
	// LoginWindow before the endpoint answers 429.
	LoginMaxAttempts = 10
	// LoginWindow is the length of the sliding window the failed attempts are
	// counted in.
	LoginWindow = 15 * time.Minute

	// rateLimitKeyPrefix namespaces the per-client counter in Valkey. Only the
	// login route uses it, so the prefix is a constant.
	rateLimitKeyPrefix = "auth:login:ratelimit:"
)

// checkScript atomically drops the attempts that have aged out of the window and
// reports whether the client has reached the limit. Returning 1 means "limited".
// The sliding window is the ZSET's score range: every attempt is a member scored
// with its unix-nanosecond timestamp.
var checkScript = redis.NewScript(`
redis.call('ZREMRANGEBYSCORE', KEYS[1], 0, ARGV[1])
local count = redis.call('ZCARD', KEYS[1])
if count >= tonumber(ARGV[2]) then
	return 1
end
return 0
`)

// recordScript appends one failed attempt and refreshes the TTL on every write,
// so the key never outlives the window after the last failure.
var recordScript = redis.NewScript(`
redis.call('ZADD', KEYS[1], ARGV[1], ARGV[2])
redis.call('PEXPIRE', KEYS[1], ARGV[3])
return redis.call('ZCARD', KEYS[1])
`)

var (
	valkeyOnce   sync.Once
	valkeyClient *redis.Client
)

// sharedValkeyClient lazily builds the login rate limiter's Valkey client from
// VALKEY_URL. Reading it lazily keeps a missing variable from killing the
// process at import time; when it is absent or unparsable the limiter degrades
// to allowing requests instead of locking everyone out.
func sharedValkeyClient() *redis.Client {
	valkeyOnce.Do(func() {
		raw := strings.TrimSpace(os.Getenv("VALKEY_URL"))
		if raw == "" {
			log.Printf("auth: VALKEY_URL is not set; login rate limiting is disabled")
			return
		}
		opts, err := redis.ParseURL(raw)
		if err != nil {
			log.Printf("auth: VALKEY_URL is invalid; login rate limiting is disabled: %v", err)
			return
		}
		opts.DialTimeout = 2 * time.Second
		opts.MaxRetries = 1
		valkeyClient = redis.NewClient(opts)
	})
	return valkeyClient
}

// RateLimit throttles the login endpoint per client: it counts failed attempts
// per client address in Valkey inside a sliding window and answers 429
// rate_limited once the limit is reached, without ever producing a token. A
// successful login clears the counter. When Valkey is unreachable the request
// is allowed through rather than locking every client out.
func RateLimit(next http.Handler) http.Handler {
	return rateLimit(next, sharedValkeyClient())
}

// rateLimit is the middleware body with the Valkey client injected, so tests can
// drive it against a real server (or an unreachable one) directly.
func rateLimit(next http.Handler, client *redis.Client) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if client == nil {
			next.ServeHTTP(w, r)
			return
		}

		ctx := r.Context()
		key := rateLimitKeyPrefix + clientAddress(r)
		now := time.Now()

		limited, err := isLimited(ctx, client, key, now)
		if err != nil {
			// Degrade: a Valkey outage must not lock every client out of login.
			next.ServeHTTP(w, r)
			return
		}
		if limited {
			httpx.Error(w, httpx.CodeRateLimited, http.StatusTooManyRequests,
				"too many failed login attempts; try again later")
			return
		}

		rec := &statusRecorder{ResponseWriter: w}
		next.ServeHTTP(rec, r)

		switch {
		case rec.status >= http.StatusOK && rec.status < http.StatusMultipleChoices:
			// A successful login clears the client's counter.
			_ = client.Del(ctx, key).Err()
		case rec.status == http.StatusUnauthorized:
			_ = recordFailure(ctx, client, key, now)
		}
	})
}

// isLimited prunes aged-out attempts and reports whether the client has used up
// its allowance. The operations run in one Lua script so the count and the
// decision cannot be interleaved by a concurrent request.
func isLimited(ctx context.Context, client *redis.Client, key string, now time.Time) (bool, error) {
	cutoff := strconv.FormatInt(now.Add(-LoginWindow).UnixNano(), 10)
	limited, err := checkScript.Run(ctx, client, []string{key}, cutoff, int64(LoginMaxAttempts)).Int()
	if err != nil {
		return false, err
	}
	return limited == 1, nil
}

// recordFailure appends one failed attempt and sets the TTL on every write.
func recordFailure(ctx context.Context, client *redis.Client, key string, now time.Time) error {
	score := strconv.FormatInt(now.UnixNano(), 10)
	ttlMillis := strconv.FormatInt(LoginWindow.Milliseconds(), 10)
	return recordScript.Run(ctx, client, []string{key}, score, score, ttlMillis).Err()
}

// clientAddress extracts the client IP from the request, dropping the ephemeral
// port so every attempt from one host shares a counter.
func clientAddress(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// statusRecorder captures the status a downstream handler wrote so the limiter
// can tell a successful login from a failed one. It is local to this file to
// keep the middleware self-contained.
type statusRecorder struct {
	http.ResponseWriter
	status int
	wrote  bool
}

func (s *statusRecorder) WriteHeader(status int) {
	if !s.wrote {
		s.status = status
		s.wrote = true
	}
	s.ResponseWriter.WriteHeader(status)
}

func (s *statusRecorder) Write(b []byte) (int, error) {
	if !s.wrote {
		s.status = http.StatusOK
		s.wrote = true
	}
	return s.ResponseWriter.Write(b)
}
