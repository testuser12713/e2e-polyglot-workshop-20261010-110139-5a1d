// Package store owns the PostgreSQL connection pool and the schema migration.
// All feature queries run against the schema created here.
package store

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Pool is the PostgreSQL connection pool.
type Pool = *pgxpool.Pool

// Querier is the subset of pgx used by the feature services. Both *pgxpool.Pool
// and pgx.Tx satisfy it, so a service can run on the pool or inside a
// transaction without changing its signature. It exists so every query takes
// its user values as pgx parameters (AC-24).
type Querier interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// NewPool opens a pgx connection pool against databaseURL.
func NewPool(ctx context.Context, databaseURL string) (Pool, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("open postgres pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("connect to postgres: %w", err)
	}
	return pool, nil
}
