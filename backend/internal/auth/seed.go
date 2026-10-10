package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"

	"workshop/backend/internal/store"
)

// loginQuerier is the store the login handler reads employees from. main hands
// the pool to this package through SeedFirstEmployee at startup — the one call
// the skeleton already makes into auth — so Login reuses the same connection
// pool instead of opening a second one.
var loginQuerier store.Querier

// SeedFirstEmployee ensures the configured first employee exists exactly once,
// with the password stored only as a hash (bcrypt, never the plain text). It is
// idempotent: a second start finds the employee and changes nothing, and the
// INSERT is guarded by the unique e-mail constraint against a concurrent start.
func SeedFirstEmployee(ctx context.Context, q store.Querier, email, password string) error {
	loginQuerier = q

	email = strings.TrimSpace(email)
	if email == "" || password == "" {
		return nil
	}

	var id int
	err := q.QueryRow(ctx, `SELECT id FROM employees WHERE email = $1`, email).Scan(&id)
	switch {
	case err == nil:
		return nil
	case errors.Is(err, pgx.ErrNoRows):
		// The employee does not exist yet: create it below.
	default:
		return fmt.Errorf("seed employee: look up: %w", err)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("seed employee: hash password: %w", err)
	}

	if _, err := q.Exec(ctx,
		`INSERT INTO employees (email, password_hash) VALUES ($1, $2) ON CONFLICT (email) DO NOTHING`,
		email, string(hash),
	); err != nil {
		return fmt.Errorf("seed employee: insert: %w", err)
	}
	return nil
}
