// Package auth owns workshop sign-in: issuing and verifying tokens, the
// RequireAuth guard, the first-employee seed and the login rate limiter.
package auth

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"

	"workshop/backend/internal/httpx"
	"workshop/backend/internal/model"
)

// loginRequest is the body of POST /api/auth/login.
type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// loginResponse is the successful answer of POST /api/auth/login. It never
// carries the password or the hash.
type loginResponse struct {
	Token     string `json:"token"`
	ExpiresAt string `json:"expiresAt"`
}

// Login handles POST /api/auth/login: it verifies the password against the
// stored hash and answers with a token and its expiry. An unknown e-mail and a
// wrong password both answer 401 with the same message, so the response never
// reveals whether an account exists. The plain password is never echoed and
// never logged (AC-14, AC-29).
func Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.Error(w, model.CodeValidation, http.StatusUnprocessableEntity, "invalid request body")
		return
	}

	email := strings.TrimSpace(req.Email)
	if email == "" || req.Password == "" {
		httpx.Error(w, model.CodeValidation, http.StatusUnprocessableEntity, "email and password are required")
		return
	}

	if loginQuerier == nil {
		httpx.Error(w, model.CodeInternal, http.StatusInternalServerError, "authentication is not configured")
		return
	}

	var id int
	var storedHash string
	err := loginQuerier.QueryRow(r.Context(),
		`SELECT id, password_hash FROM employees WHERE email = $1`, email,
	).Scan(&id, &storedHash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			unauthorized(w)
			return
		}
		httpx.Error(w, model.CodeInternal, http.StatusInternalServerError, "internal server error")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(req.Password)); err != nil {
		unauthorized(w)
		return
	}

	token, err := IssueToken(id, email)
	if err != nil {
		httpx.Error(w, model.CodeInternal, http.StatusInternalServerError, "internal server error")
		return
	}
	claims, err := VerifyToken(token)
	if err != nil {
		httpx.Error(w, model.CodeInternal, http.StatusInternalServerError, "internal server error")
		return
	}

	httpx.JSON(w, http.StatusOK, loginResponse{
		Token:     token,
		ExpiresAt: time.Unix(claims.Exp, 0).UTC().Format(time.RFC3339),
	})
}

// unauthorized answers 401 with the one message shared by an unknown e-mail and
// a wrong password.
func unauthorized(w http.ResponseWriter) {
	httpx.Error(w, model.CodeUnauthorized, http.StatusUnauthorized, "invalid email or password")
}
