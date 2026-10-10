package auth

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"workshop/backend/internal/httpx"
	"workshop/backend/internal/model"
)

// ctxKey is the private type of the context keys this package sets, so no
// other package can collide with them.
type ctxKey int

const claimsCtxKey ctxKey = 0

// RequireAuth guards every workshop route. A request without a bearer token,
// with an invalid token or with an expired token is answered 401 with the
// unified error body and never reaches the wrapped handler, so no workshop data
// leaks (AC-15, AC-23).
func RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, ok := bearerToken(r.Header.Get("Authorization"))
		if !ok {
			httpx.Error(w, model.CodeUnauthorized, http.StatusUnauthorized, "authentication required")
			return
		}
		claims, err := VerifyToken(token)
		if err != nil {
			httpx.Error(w, model.CodeUnauthorized, http.StatusUnauthorized, "invalid or expired token")
			return
		}
		ctx := context.WithValue(r.Context(), claimsCtxKey, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// bearerToken extracts the token of an "Authorization: Bearer <token>" header.
func bearerToken(header string) (string, bool) {
	const prefix = "Bearer "
	if len(header) <= len(prefix) || !strings.EqualFold(header[:len(prefix)], prefix) {
		return "", false
	}
	token := strings.TrimSpace(header[len(prefix):])
	if token == "" {
		return "", false
	}
	return token, true
}

// ClaimsFromContext returns the verified claims RequireAuth stored for the
// request. The boolean is false outside an authenticated workshop route.
func ClaimsFromContext(ctx context.Context) (model.Claims, bool) {
	claims, ok := ctx.Value(claimsCtxKey).(model.Claims)
	return claims, ok
}

// EmployeeIDFromContext returns the employee id of the verified token, so a
// workshop handler can attribute an action to the signed-in employee. The
// boolean is false when there is no authenticated employee on the request.
func EmployeeIDFromContext(ctx context.Context) (int, bool) {
	claims, ok := ClaimsFromContext(ctx)
	if !ok {
		return 0, false
	}
	id, err := strconv.Atoi(claims.Sub)
	if err != nil {
		return 0, false
	}
	return id, true
}
