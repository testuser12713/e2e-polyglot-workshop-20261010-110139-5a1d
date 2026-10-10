package order

import (
	"context"

	"workshop/backend/internal/model"
	"workshop/backend/internal/store"
)

// Notifier is the seam the order workflow uses to announce a completed order to
// the invoice queue. The API's queue publisher implements it.
type Notifier interface {
	NotifyOrderCompleted(ctx context.Context, orderNumber string) error
}

// notifier is set once at startup via SetNotifier.
var notifier Notifier

// SetNotifier installs the notifier used when an order reaches completed.
func SetNotifier(n Notifier) {
	notifier = n
}

// ChangeStatus moves an order along the allowed workflow, appends a history
// entry and — on completed — notifies the invoice queue. This ticket ships the
// signature only; ticket #19 fills the body.
func ChangeStatus(ctx context.Context, q store.Querier, orderID int, newStatus string, employeeID int) (model.Order, error) {
	return model.Order{}, nil
}
