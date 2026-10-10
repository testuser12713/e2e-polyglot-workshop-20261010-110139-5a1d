// Package customer implements the workshop customer use cases. This ticket
// ships the signatures only; ticket #15 fills the bodies.
package customer

import (
	"context"

	"workshop/backend/internal/model"
	"workshop/backend/internal/store"
)

// Create stores a new customer and returns it.
func Create(ctx context.Context, q store.Querier, name, email, phone string) (model.Customer, error) {
	return model.Customer{}, nil
}

// Get loads one customer by id.
func Get(ctx context.Context, q store.Querier, id int) (model.Customer, error) {
	return model.Customer{}, nil
}

// Update changes an existing customer and returns it.
func Update(ctx context.Context, q store.Querier, id int, name, email, phone string) (model.Customer, error) {
	return model.Customer{}, nil
}
