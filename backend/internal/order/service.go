// Package order implements the order use cases: creating an order, loading it
// with positions and history, and adding positions. This ticket ships the
// signatures only; tickets #11 and #19 fill the bodies.
package order

import (
	"context"

	"workshop/backend/internal/model"
	"workshop/backend/internal/store"
)

// CreateOrder creates an order for a vehicle and returns it.
func CreateOrder(ctx context.Context, q store.Querier, vehicleID int, requestedDate, problemDescription string) (model.Order, error) {
	return model.Order{}, nil
}

// LoadOrder loads one order with its positions and status history.
func LoadOrder(ctx context.Context, q store.Querier, id int) (model.Order, error) {
	return model.Order{}, nil
}

// AddItems appends positions to an order and returns the reloaded order.
func AddItems(ctx context.Context, q store.Querier, orderID int, items []model.OrderItemInput) (model.Order, error) {
	return model.Order{}, nil
}
