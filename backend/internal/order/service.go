// Package order implements the order use cases: creating an order, loading it
// with positions and status history, and adding positions. The status workflow
// lives in status.go (ticket #19) and is wired to the same handler.
package order

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"workshop/backend/internal/model"
	"workshop/backend/internal/store"
)

// Sentinel errors the HTTP layer maps to the contract's error codes. They are
// exported so the status workflow (status.go) and the public handlers can reuse
// the exact same spelling.
var (
	// ErrNotFound reports an unknown order or an unknown referenced vehicle.
	ErrNotFound = errors.New("order: not found")
	// ErrValidation reports a request that is structurally unusable.
	ErrValidation = errors.New("order: validation error")
)

const dateLayout = "2006-01-02"

// createOrderSQL inserts an order and derives a unique, human-readable order
// number from the row id in the same statement. It reserves the id with
// nextval and formats it as AU-<year>-<5-digit id>, so the number can never
// collide with an existing order.
const createOrderSQL = `
WITH new_id AS (
	SELECT nextval(pg_get_serial_sequence('orders', 'id')) AS id
)
INSERT INTO orders (id, order_number, vehicle_id, requested_date, problem_description, status)
SELECT new_id.id,
       'AU-' || to_char(now() AT TIME ZONE 'UTC', 'YYYY') || '-' || lpad(new_id.id::text, 5, '0'),
       $1, $2, $3, $4
FROM new_id
RETURNING id`

const loadOrderSQL = `
SELECT o.id, o.order_number, o.vehicle_id, v.customer_id,
       v.plate, v.brand, v.model,
       o.status, o.requested_date, o.problem_description, o.created_at
FROM orders o
JOIN vehicles v ON v.id = o.vehicle_id
WHERE o.id = $1`

const insertItemSQL = `
INSERT INTO order_items (order_id, kind, description, hours, quantity, unit_price_cents, amount_cents)
VALUES ($1, $2, $3, $4, $5, $6, $7)`

// CreateOrder creates an order for an existing vehicle and returns it in the
// initial status requested. An unknown vehicle is ErrNotFound; a missing or
// unusable vehicleId/requestedDate/problemDescription is ErrValidation.
func CreateOrder(ctx context.Context, q store.Querier, vehicleID int, requestedDate, problemDescription string) (model.Order, error) {
	if vehicleID <= 0 {
		return model.Order{}, fmt.Errorf("%w: vehicleId is required", ErrValidation)
	}
	problemDescription = strings.TrimSpace(problemDescription)
	if problemDescription == "" {
		return model.Order{}, fmt.Errorf("%w: problemDescription is required", ErrValidation)
	}
	date, err := parseDate(requestedDate)
	if err != nil {
		return model.Order{}, err
	}

	var vehicleExists bool
	if err := q.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM vehicles WHERE id = $1)`, vehicleID).Scan(&vehicleExists); err != nil {
		return model.Order{}, fmt.Errorf("check vehicle: %w", err)
	}
	if !vehicleExists {
		return model.Order{}, fmt.Errorf("%w: vehicle %d", ErrNotFound, vehicleID)
	}

	var id int
	if err := q.QueryRow(ctx, createOrderSQL, vehicleID, date, problemDescription, model.StatusRequested).Scan(&id); err != nil {
		return model.Order{}, fmt.Errorf("insert order: %w", err)
	}

	if _, err := q.Exec(ctx,
		`INSERT INTO order_status_log (order_id, status) VALUES ($1, $2)`,
		id, model.StatusRequested); err != nil {
		return model.Order{}, fmt.Errorf("insert initial status: %w", err)
	}

	return LoadOrder(ctx, q, id)
}

// LoadOrder loads one order with its vehicle, customer, positions and status
// history. An unknown order is ErrNotFound.
func LoadOrder(ctx context.Context, q store.Querier, id int) (model.Order, error) {
	var (
		o             model.Order
		requestedDate time.Time
		createdAt     time.Time
	)
	err := q.QueryRow(ctx, loadOrderSQL, id).Scan(
		&o.ID, &o.OrderNumber, &o.VehicleID, &o.CustomerID,
		&o.Plate, &o.Brand, &o.Model,
		&o.Status, &requestedDate, &o.ProblemDescription, &createdAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return model.Order{}, fmt.Errorf("%w: order %d", ErrNotFound, id)
	}
	if err != nil {
		return model.Order{}, fmt.Errorf("load order: %w", err)
	}
	o.RequestedDate = requestedDate.Format(dateLayout)
	o.CreatedAt = createdAt.UTC().Format(time.RFC3339)

	items, err := loadItems(ctx, q, id)
	if err != nil {
		return model.Order{}, err
	}
	history, err := loadHistory(ctx, q, id)
	if err != nil {
		return model.Order{}, err
	}
	o.Items = items
	o.History = history
	return o, nil
}

// AddItems appends positions to an order and returns the reloaded order. An
// unknown order is ErrNotFound; a malformed item is ErrValidation.
func AddItems(ctx context.Context, q store.Querier, orderID int, items []model.OrderItemInput) (model.Order, error) {
	if orderID <= 0 {
		return model.Order{}, fmt.Errorf("%w: orderId is required", ErrValidation)
	}
	if len(items) == 0 {
		return model.Order{}, fmt.Errorf("%w: at least one item is required", ErrValidation)
	}

	var orderExists bool
	if err := q.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM orders WHERE id = $1)`, orderID).Scan(&orderExists); err != nil {
		return model.Order{}, fmt.Errorf("check order: %w", err)
	}
	if !orderExists {
		return model.Order{}, fmt.Errorf("%w: order %d", ErrNotFound, orderID)
	}

	normalized := make([]normalizedItem, 0, len(items))
	for i, in := range items {
		ni, err := normalizeItem(in)
		if err != nil {
			return model.Order{}, fmt.Errorf("%w (item %d): %v", ErrValidation, i, err)
		}
		normalized = append(normalized, ni)
	}

	for _, ni := range normalized {
		if _, err := q.Exec(ctx, insertItemSQL,
			orderID, ni.kind, ni.description, ni.hours, ni.quantity, ni.unitPriceCents, ni.amountCents); err != nil {
			return model.Order{}, fmt.Errorf("insert order item: %w", err)
		}
	}

	return LoadOrder(ctx, q, orderID)
}

// normalizedItem is a validated input item with its amount computed.
type normalizedItem struct {
	kind           string
	description    string
	hours          *float64
	quantity       *int
	unitPriceCents *int
	amountCents    int
}

// normalizeItem validates one input position and computes its amount in whole
// cents. A labor position carries hours (and an optional hourly rate), a part
// carries a positive quantity and its unit price.
func normalizeItem(in model.OrderItemInput) (normalizedItem, error) {
	description := strings.TrimSpace(in.Description)
	if description == "" {
		return normalizedItem{}, errors.New("description is required")
	}

	switch in.Kind {
	case model.OrderItemKindLabor:
		if in.Hours == nil || *in.Hours <= 0 {
			return normalizedItem{}, errors.New("labor item requires positive hours")
		}
		amount := 0
		if in.UnitPriceCents != nil {
			if *in.UnitPriceCents < 0 {
				return normalizedItem{}, errors.New("unitPriceCents must not be negative")
			}
			amount = int(math.Round(*in.Hours * float64(*in.UnitPriceCents)))
		}
		return normalizedItem{
			kind:           in.Kind,
			description:    description,
			hours:          in.Hours,
			unitPriceCents: in.UnitPriceCents,
			amountCents:    amount,
		}, nil
	case model.OrderItemKindPart:
		if in.Quantity == nil || *in.Quantity <= 0 {
			return normalizedItem{}, errors.New("part item requires a positive quantity")
		}
		if in.UnitPriceCents == nil || *in.UnitPriceCents < 0 {
			return normalizedItem{}, errors.New("part item requires a non-negative unitPriceCents")
		}
		return normalizedItem{
			kind:           in.Kind,
			description:    description,
			quantity:       in.Quantity,
			unitPriceCents: in.UnitPriceCents,
			amountCents:    *in.Quantity * *in.UnitPriceCents,
		}, nil
	default:
		return normalizedItem{}, fmt.Errorf("unknown item kind %q", in.Kind)
	}
}

// parseDate accepts only the canonical YYYY-MM-DD form.
func parseDate(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	d, err := time.Parse(dateLayout, s)
	if err != nil || d.Format(dateLayout) != s {
		return time.Time{}, fmt.Errorf("%w: requestedDate must be YYYY-MM-DD", ErrValidation)
	}
	return d, nil
}

// loadItems reads every position of an order, oldest first.
func loadItems(ctx context.Context, q store.Querier, orderID int) ([]model.OrderItem, error) {
	rows, err := q.Query(ctx,
		`SELECT id, kind, description, hours, quantity, unit_price_cents, amount_cents
		 FROM order_items WHERE order_id = $1 ORDER BY id`, orderID)
	if err != nil {
		return nil, fmt.Errorf("load order items: %w", err)
	}
	defer rows.Close()

	items := make([]model.OrderItem, 0)
	for rows.Next() {
		var it model.OrderItem
		if err := rows.Scan(&it.ID, &it.Kind, &it.Description, &it.Hours, &it.Quantity, &it.UnitPriceCents, &it.AmountCents); err != nil {
			return nil, fmt.Errorf("scan order item: %w", err)
		}
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate order items: %w", err)
	}
	return items, nil
}

// loadHistory reads the status log of an order, oldest first, with the name of
// the employee who changed it (empty when the entry has no employee).
func loadHistory(ctx context.Context, q store.Querier, orderID int) ([]model.StatusEntry, error) {
	rows, err := q.Query(ctx,
		`SELECT l.status, l.changed_at, COALESCE(e.name, '')
		 FROM order_status_log l
		 LEFT JOIN employees e ON e.id = l.employee_id
		 WHERE l.order_id = $1
		 ORDER BY l.id`, orderID)
	if err != nil {
		return nil, fmt.Errorf("load order history: %w", err)
	}
	defer rows.Close()

	history := make([]model.StatusEntry, 0)
	for rows.Next() {
		var (
			entry     model.StatusEntry
			changedAt time.Time
		)
		if err := rows.Scan(&entry.Status, &changedAt, &entry.EmployeeName); err != nil {
			return nil, fmt.Errorf("scan order history: %w", err)
		}
		entry.ChangedAt = changedAt.UTC().Format(time.RFC3339)
		history = append(history, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate order history: %w", err)
	}
	return history, nil
}
