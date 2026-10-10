package auth

import (
	"net/http"
)

// RequireAuth guards a workshop route: a request without a valid bearer token
// must be rejected with 401 and no workshop data. This ticket ships the
// signature only; ticket #9 fills the body.
func RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r)
	})
}
