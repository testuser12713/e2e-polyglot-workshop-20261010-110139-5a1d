package auth

import (
	"net/http"
)

// RateLimit throttles the login endpoint per client and answers 429 once the
// limit is exceeded. This ticket ships a pass-through; ticket #4 replaces it
// with a counter kept in Valkey.
func RateLimit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r)
	})
}
