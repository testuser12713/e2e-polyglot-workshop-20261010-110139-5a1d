package auth

import (
	"context"

	"workshop/backend/internal/store"
)

// SeedFirstEmployee ensures the configured first employee exists exactly once,
// with the password stored only as a hash. This ticket ships the signature
// only; ticket #9 fills the body.
func SeedFirstEmployee(ctx context.Context, q store.Querier, email, password string) error {
	return nil
}
