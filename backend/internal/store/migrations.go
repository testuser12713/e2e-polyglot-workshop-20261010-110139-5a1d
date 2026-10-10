package store

import (
	"context"
	"fmt"
)

// schemaStatements is the complete schema of the product. It is idempotent and
// runs at startup, so a freshly cloned repository works against an empty
// database without any manual step.
var schemaStatements = []string{
	`CREATE TABLE IF NOT EXISTS customers (
		id         serial PRIMARY KEY,
		name       text NOT NULL,
		email      text NOT NULL,
		phone      text NOT NULL,
		created_at timestamptz NOT NULL DEFAULT now()
	)`,

	`CREATE TABLE IF NOT EXISTS employees (
		id            serial PRIMARY KEY,
		name          text NOT NULL DEFAULT '',
		email         text NOT NULL UNIQUE,
		password_hash text NOT NULL,
		created_at    timestamptz NOT NULL DEFAULT now()
	)`,

	`CREATE TABLE IF NOT EXISTS vehicles (
		id               serial PRIMARY KEY,
		customer_id      integer NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
		plate            text NOT NULL,
		plate_normalized text NOT NULL UNIQUE,
		brand            text NOT NULL,
		model            text NOT NULL,
		mileage_km       integer NOT NULL DEFAULT 0,
		created_at       timestamptz NOT NULL DEFAULT now()
	)`,

	`CREATE TABLE IF NOT EXISTS orders (
		id                  serial PRIMARY KEY,
		order_number        text NOT NULL UNIQUE,
		vehicle_id          integer NOT NULL REFERENCES vehicles(id),
		requested_date      date NOT NULL,
		problem_description text NOT NULL,
		status              text NOT NULL DEFAULT 'requested'
			CHECK (status IN ('requested','confirmed','in_progress','completed','picked_up')),
		created_at          timestamptz NOT NULL DEFAULT now()
	)`,

	`CREATE TABLE IF NOT EXISTS order_items (
		id               serial PRIMARY KEY,
		order_id         integer NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
		kind             text NOT NULL CHECK (kind IN ('labor','part')),
		description      text NOT NULL,
		hours            numeric(10,2),
		quantity         integer,
		unit_price_cents integer,
		amount_cents     integer NOT NULL,
		created_at       timestamptz NOT NULL DEFAULT now()
	)`,

	`CREATE TABLE IF NOT EXISTS order_status_log (
		id          serial PRIMARY KEY,
		order_id    integer NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
		status      text NOT NULL,
		employee_id integer REFERENCES employees(id),
		changed_at  timestamptz NOT NULL DEFAULT now()
	)`,

	`CREATE TABLE IF NOT EXISTS invoices (
		id               serial PRIMARY KEY,
		invoice_number   text NOT NULL UNIQUE,
		order_id         integer NOT NULL UNIQUE REFERENCES orders(id),
		net_cents        integer NOT NULL,
		tax_rate_percent integer NOT NULL,
		tax_cents        integer NOT NULL,
		gross_cents      integer NOT NULL,
		created_at       timestamptz NOT NULL DEFAULT now()
	)`,

	`CREATE TABLE IF NOT EXISTS invoice_items (
		id               serial PRIMARY KEY,
		invoice_id       integer NOT NULL REFERENCES invoices(id) ON DELETE CASCADE,
		kind             text NOT NULL,
		description      text NOT NULL,
		hours            numeric(10,2),
		quantity         integer,
		unit_price_cents integer,
		amount_cents     integer NOT NULL
	)`,

	`CREATE TABLE IF NOT EXISTS outbox (
		id              serial PRIMARY KEY,
		recipient_email text NOT NULL,
		subject         text NOT NULL,
		body            text NOT NULL,
		status          text NOT NULL DEFAULT 'pending',
		created_at      timestamptz NOT NULL DEFAULT now()
	)`,

	`CREATE INDEX IF NOT EXISTS idx_vehicles_customer_id ON vehicles(customer_id)`,
	`CREATE INDEX IF NOT EXISTS idx_orders_vehicle_id ON orders(vehicle_id)`,
	`CREATE INDEX IF NOT EXISTS idx_orders_status ON orders(status)`,
	`CREATE INDEX IF NOT EXISTS idx_order_items_order_id ON order_items(order_id)`,
	`CREATE INDEX IF NOT EXISTS idx_order_status_log_order_id ON order_status_log(order_id)`,
	`CREATE INDEX IF NOT EXISTS idx_invoice_items_invoice_id ON invoice_items(invoice_id)`,
}

// migrateLockKey is the advisory-lock key that serialises concurrent Migrate
// calls. Without it two processes migrating at once can both pass the
// "IF NOT EXISTS" check and then collide creating the same table.
const migrateLockKey int64 = 7362871301

// Migrate creates the complete schema. It is safe to run on every start and
// safe to run concurrently: the whole migration takes a transaction-level
// advisory lock, so a second caller waits and then sees the schema already
// present instead of racing it.
func Migrate(ctx context.Context, pool Pool) error {
	if pool == nil {
		return fmt.Errorf("migrate: nil pool")
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("migrate: begin: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock($1)`, migrateLockKey); err != nil {
		return fmt.Errorf("migrate: acquire lock: %w", err)
	}

	for i, stmt := range schemaStatements {
		if _, err := tx.Exec(ctx, stmt); err != nil {
			return fmt.Errorf("migrate: statement %d: %w", i+1, err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("migrate: commit: %w", err)
	}
	return nil
}
