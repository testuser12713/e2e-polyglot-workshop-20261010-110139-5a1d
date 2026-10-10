package httpx

import (
	"log"
	"net/http"
	"time"
)

// Middleware is the shape every middleware in this package follows.
type Middleware func(http.Handler) http.Handler

// statusRecorder remembers the status code written so it can be logged.
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	if r.status == 0 {
		r.status = status
	}
	r.ResponseWriter.WriteHeader(status)
}

func (r *statusRecorder) Write(b []byte) (int, error) {
	if r.status == 0 {
		r.status = http.StatusOK
	}
	return r.ResponseWriter.Write(b)
}

// Logging returns middleware that logs one line per request. It logs ONLY the
// method, the path, the status and the duration — never a query string, a
// header, a body or an error argument, because any of those can carry a name,
// an e-mail, a phone number, a plate or a password (AC-29).
func Logging() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rec := &statusRecorder{ResponseWriter: w}
			next.ServeHTTP(rec, r)
			status := rec.status
			if status == 0 {
				status = http.StatusOK
			}
			log.Printf("%s %s -> %d (%s)", r.Method, r.URL.Path, status, time.Since(start))
		})
	}
}

// Recover returns middleware that turns a panic into a 500 with the unified
// error body instead of crashing the connection. It logs only the request line
// and a generic marker — never the panic value, which can contain user input.
func Recover() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					log.Printf("httpx: recovered from panic on %s %s", r.Method, r.URL.Path)
					Error(w, CodeInternal, http.StatusInternalServerError, "internal server error")
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

// CORS returns middleware that allows exactly one origin: the one from the
// configuration (CORS_ORIGIN). A request from any other origin gets no
// Access-Control-Allow-Origin header at all — never "*" (AC-25).
func CORS(origin string) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if origin != "" && r.Header.Get("Origin") == origin {
				h := w.Header()
				h.Set("Access-Control-Allow-Origin", origin)
				h.Set("Access-Control-Allow-Credentials", "true")
				h.Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
				h.Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
				h.Set("Access-Control-Max-Age", "600")
				h.Add("Vary", "Origin")
			}
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
