package order

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5"

	"workshop/backend/internal/model"
	"workshop/backend/internal/store"
)

// ErrInvalidTransition reports a status change that is not the single forward
// step of the workflow. The HTTP layer maps it to 409 invalid_transition and
// the order is left untouched.
var ErrInvalidTransition = errors.New("order: invalid transition")

// Notifier is the seam the order workflow uses to announce a completed order to
// the invoice queue. The API's queue publisher implements it.
//
// The notifier carries the order id as well as its number because the queue
// contract's message is {"orderNumber":string,"orderId":int} — the worker loads
// the invoice inputs (positions, recipient) by orderId, so the number alone is
// not enough to produce a valid message.
type Notifier interface {
	NotifyOrderCompleted(ctx context.Context, orderID int, orderNumber string) error
}

// notifier is set once at startup via SetNotifier.
var notifier Notifier

// SetNotifier installs the notifier used when an order reaches completed.
func SetNotifier(n Notifier) {
	notifier = n
}

// txBeginner is the subset of a pool that can open a transaction. A Querier
// that is already a transaction does not implement it, which is exactly how
// runInTx tells the two apart.
type txBeginner interface {
	Begin(ctx context.Context) (pgx.Tx, error)
}

// ChangeStatus moves an order along the allowed workflow
// (requested -> confirmed -> in_progress -> completed -> picked_up). Every other
// transition is rejected with ErrInvalidTransition and leaves the order
// unchanged. An unknown status is ErrValidation and an unknown order is
// ErrNotFound.
//
// On a successful change it updates the order and appends exactly one
// order_status_log row — the new status, the acting employee and a UTC
// timestamp — in the same transaction, so the status and its history can never
// disagree. The current status is read with a row lock, so two concurrent
// requests cannot both apply a transition. When the new status is completed it
// then invokes the wired Notifier; the queue is best effort, so a notifier
// failure is logged without PII and never fails the status change.
func ChangeStatus(ctx context.Context, q store.Querier, orderID int, newStatus string, employeeID int) (model.Order, error) {
	if newStatus == "" || !model.ValidStatus(newStatus) {
		return model.Order{}, fmt.Errorf("%w: unknown status %q", ErrValidation, newStatus)
	}
	if orderID <= 0 {
		return model.Order{}, fmt.Errorf("%w: orderId is required", ErrValidation)
	}

	err := runInTx(ctx, q, func(tx store.Querier) error {
		var current string
		err := tx.QueryRow(ctx, `SELECT status FROM orders WHERE id = $1 FOR UPDATE`, orderID).Scan(&current)
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("%w: order %d", ErrNotFound, orderID)
		}
		if err != nil {
			return fmt.Errorf("load order status: %w", err)
		}
		if !model.CanTransition(current, newStatus) {
			return fmt.Errorf("%w: %s -> %s", ErrInvalidTransition, current, newStatus)
		}
		return writeStatusChange(ctx, tx, orderID, newStatus, employeeID)
	})
	if err != nil {
		return model.Order{}, err
	}

	ord, err := LoadOrder(ctx, q, orderID)
	if err != nil {
		return model.Order{}, err
	}

	if newStatus == model.StatusCompleted {
		notifyCompleted(ctx, orderID, ord.OrderNumber)
	}
	return ord, nil
}

// runInTx runs fn in a transaction. When q is already a transaction, fn runs on
// it directly; when it is a pool, a fresh transaction is opened and committed. A
// returned error rolls the transaction back, so a rejected transition changes
// nothing.
func runInTx(ctx context.Context, q store.Querier, fn func(store.Querier) error) error {
	b, ok := q.(txBeginner)
	if !ok {
		return fn(q)
	}

	tx, err := b.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin status change: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := fn(tx); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit status change: %w", err)
	}
	return nil
}

// writeStatusChange performs the two writes that make up one status change: the
// order's new status and the matching history row. They share the caller's
// transaction, so either both land or neither does.
func writeStatusChange(ctx context.Context, q store.Querier, orderID int, newStatus string, employeeID int) error {
	if _, err := q.Exec(ctx, `UPDATE orders SET status = $1 WHERE id = $2`, newStatus, orderID); err != nil {
		return fmt.Errorf("update order status: %w", err)
	}

	// An unattributed change (no authenticated employee) stores NULL rather
	// than 0, which is not a valid employee id.
	var employee any
	if employeeID > 0 {
		employee = employeeID
	}
	if _, err := q.Exec(ctx,
		`INSERT INTO order_status_log (order_id, status, employee_id, changed_at)
		 VALUES ($1, $2, $3, now())`,
		orderID, newStatus, employee); err != nil {
		return fmt.Errorf("insert status log: %w", err)
	}
	return nil
}

// notifyCompleted invokes the wired notifier best effort. The status change is
// already committed, so a queue failure is logged without PII (never the order
// number or any customer value) and never returned to the caller.
func notifyCompleted(ctx context.Context, orderID int, orderNumber string) {
	if notifier == nil {
		return
	}
	if err := notifier.NotifyOrderCompleted(ctx, orderID, orderNumber); err != nil {
		log.Printf("order: completed-order notification failed: %T", err)
	}
}
