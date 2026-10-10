// Package customer implements the workshop customer use cases: creating,
// reading and updating a customer. Every query binds its user-supplied values
// as pgx parameters (AC-24) and every stored value is trimmed before it is
// persisted, so the shared entry point Create validates its input in one place
// for every later caller.
package customer

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"strings"

	"github.com/jackc/pgx/v5"

	"workshop/backend/internal/model"
	"workshop/backend/internal/store"
)

// ErrNotFound is returned when no customer carries the requested id.
var ErrNotFound = errors.New("customer not found")

// ValidationError reports that a request carried a missing or unusable field.
// Its message is a static, PII-free sentence safe to send in the error body.
type ValidationError struct {
	Message string
}

// Error implements the error interface.
func (e ValidationError) Error() string { return e.Message }

// validate checks the required fields of a customer. Name and a usable e-mail
// are required; the phone number is optional. It returns a ValidationError for
// the first missing or unusable field, or nil when the input is acceptable.
func validate(name, email string) error {
	if strings.TrimSpace(name) == "" {
		return ValidationError{Message: "name is required"}
	}
	email = strings.TrimSpace(email)
	if email == "" {
		return ValidationError{Message: "email is required"}
	}
	addr, err := mail.ParseAddress(email)
	if err != nil || addr.Address != email {
		return ValidationError{Message: "email is invalid"}
	}
	return nil
}

// Create stores a new customer and returns it.
func Create(ctx context.Context, q store.Querier, name, email, phone string) (model.Customer, error) {
	name = strings.TrimSpace(name)
	email = strings.TrimSpace(email)
	phone = strings.TrimSpace(phone)
	if err := validate(name, email); err != nil {
		return model.Customer{}, err
	}

	var c model.Customer
	err := q.QueryRow(ctx,
		`INSERT INTO customers (name, email, phone)
		 VALUES ($1, $2, $3)
		 RETURNING id, name, email, phone`,
		name, email, phone,
	).Scan(&c.ID, &c.Name, &c.Email, &c.Phone)
	if err != nil {
		return model.Customer{}, fmt.Errorf("create customer: %w", err)
	}
	return c, nil
}

// Get loads one customer by id. An unknown id yields ErrNotFound.
func Get(ctx context.Context, q store.Querier, id int) (model.Customer, error) {
	var c model.Customer
	err := q.QueryRow(ctx,
		`SELECT id, name, email, phone FROM customers WHERE id = $1`,
		id,
	).Scan(&c.ID, &c.Name, &c.Email, &c.Phone)
	if errors.Is(err, pgx.ErrNoRows) {
		return model.Customer{}, ErrNotFound
	}
	if err != nil {
		return model.Customer{}, fmt.Errorf("get customer: %w", err)
	}
	return c, nil
}

// Update changes an existing customer and returns it. An unknown id yields
// ErrNotFound; invalid or missing fields yield a ValidationError.
func Update(ctx context.Context, q store.Querier, id int, name, email, phone string) (model.Customer, error) {
	name = strings.TrimSpace(name)
	email = strings.TrimSpace(email)
	phone = strings.TrimSpace(phone)
	if err := validate(name, email); err != nil {
		return model.Customer{}, err
	}

	var c model.Customer
	err := q.QueryRow(ctx,
		`UPDATE customers
		 SET name = $2, email = $3, phone = $4
		 WHERE id = $1
		 RETURNING id, name, email, phone`,
		id, name, email, phone,
	).Scan(&c.ID, &c.Name, &c.Email, &c.Phone)
	if errors.Is(err, pgx.ErrNoRows) {
		return model.Customer{}, ErrNotFound
	}
	if err != nil {
		return model.Customer{}, fmt.Errorf("update customer: %w", err)
	}
	return c, nil
}
