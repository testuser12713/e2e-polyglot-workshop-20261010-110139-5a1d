// Package auth owns workshop sign-in: issuing and verifying tokens, the
// RequireAuth guard, the first-employee seed and the login rate limiter. This
// ticket ships the signatures only; tickets #9 and #4 fill the bodies.
package auth

import (
	"net/http"

	"workshop/backend/internal/httpx"
)

// Login handles POST /api/auth/login. This ticket ships the stub; ticket #9
// fills the body (credential check, token issue, rate limit wiring).
func Login(w http.ResponseWriter, r *http.Request) {
	httpx.NotImplemented(w, "login")
}
