// Package vehicle implements the vehicle use cases: creating, reading and
// updating a customer's vehicle. The plate is normalised (upper case, without
// spaces or dashes) before it is stored and compared, so two spellings of the
// same plate collide on the unique plate_normalized column and are reported as
// ErrDuplicatePlate.
package vehicle

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	apimodel "workshop/backend/internal/model"
	"workshop/backend/internal/store"
)

// Sentinel errors of the vehicle use cases. The HTTP layer maps them to the
// shared error codes; other packages call errors.Is against them.
var (
	// ErrValidation marks a missing or unusable field (422 validation_error).
	ErrValidation = errors.New("validation error")
	// ErrDuplicatePlate marks a plate that already exists in any spelling (409 duplicate_plate).
	ErrDuplicatePlate = errors.New("duplicate plate")
	// ErrCustomerNotFound marks a vehicle that names a customer that does not exist (404 not_found).
	ErrCustomerNotFound = errors.New("customer not found")
	// ErrVehicleNotFound marks an unknown vehicle id (404 not_found).
	ErrVehicleNotFound = errors.New("vehicle not found")
)

// NormalizePlate upper-cases and strips spaces and dashes so that two spellings
// of the same plate compare equal.
func NormalizePlate(plate string) string {
	r := strings.NewReplacer(" ", "", "-", "", "\t", "")
	return strings.ToUpper(strings.TrimSpace(r.Replace(plate)))
}

// validate checks the shared fields of Create and Update and returns a wrapped
// ErrValidation naming the offending field (never a user value, AC-29).
func validate(customerID int, plate, brand, model string, mileageKm int) (string, string, string, error) {
	if customerID <= 0 {
		return "", "", "", fmt.Errorf("%w: customerId is required", ErrValidation)
	}
	normalized := NormalizePlate(plate)
	if normalized == "" {
		return "", "", "", fmt.Errorf("%w: plate is required", ErrValidation)
	}
	brand = strings.TrimSpace(brand)
	if brand == "" {
		return "", "", "", fmt.Errorf("%w: brand is required", ErrValidation)
	}
	model = strings.TrimSpace(model)
	if model == "" {
		return "", "", "", fmt.Errorf("%w: model is required", ErrValidation)
	}
	if mileageKm < 0 {
		return "", "", "", fmt.Errorf("%w: mileageKm must not be negative", ErrValidation)
	}
	return normalized, brand, model, nil
}

// customerExists reports whether a customer with the given id exists.
func customerExists(ctx context.Context, q store.Querier, id int) (bool, error) {
	var exists bool
	err := q.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM customers WHERE id = $1)`, id).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check customer %d: %w", id, err)
	}
	return exists, nil
}

// isUniqueViolation reports whether err is a PostgreSQL 23505 unique-constraint
// violation — here always the unique plate_normalized column.
func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

// isForeignKeyViolation reports whether err is a PostgreSQL 23503 foreign-key
// violation — here a customer_id referencing a missing customer.
func isForeignKeyViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23503"
}

// Create stores a new vehicle for a customer and returns it. The plate is
// normalised before it is stored; a plate that already exists in any spelling
// yields ErrDuplicatePlate and an unknown customer yields ErrCustomerNotFound.
func Create(ctx context.Context, q store.Querier, customerID int, plate, brand, model string, mileageKm int) (apimodel.Vehicle, error) {
	normalized, brand, model, err := validate(customerID, plate, brand, model, mileageKm)
	if err != nil {
		return apimodel.Vehicle{}, err
	}

	exists, err := customerExists(ctx, q, customerID)
	if err != nil {
		return apimodel.Vehicle{}, err
	}
	if !exists {
		return apimodel.Vehicle{}, ErrCustomerNotFound
	}

	var v apimodel.Vehicle
	err = q.QueryRow(ctx,
		`INSERT INTO vehicles (customer_id, plate, plate_normalized, brand, model, mileage_km)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id, customer_id, plate, brand, model, mileage_km`,
		customerID, normalized, normalized, brand, model, mileageKm,
	).Scan(&v.ID, &v.CustomerID, &v.Plate, &v.Brand, &v.Model, &v.MileageKm)
	if err != nil {
		if isUniqueViolation(err) {
			return apimodel.Vehicle{}, ErrDuplicatePlate
		}
		if isForeignKeyViolation(err) {
			return apimodel.Vehicle{}, ErrCustomerNotFound
		}
		return apimodel.Vehicle{}, fmt.Errorf("insert vehicle: %w", err)
	}
	return v, nil
}

// Get loads one vehicle by id.
func Get(ctx context.Context, q store.Querier, id int) (apimodel.Vehicle, error) {
	var v apimodel.Vehicle
	err := q.QueryRow(ctx,
		`SELECT id, customer_id, plate, brand, model, mileage_km FROM vehicles WHERE id = $1`,
		id,
	).Scan(&v.ID, &v.CustomerID, &v.Plate, &v.Brand, &v.Model, &v.MileageKm)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return apimodel.Vehicle{}, ErrVehicleNotFound
		}
		return apimodel.Vehicle{}, fmt.Errorf("load vehicle: %w", err)
	}
	return v, nil
}

// Update changes an existing vehicle and returns it. The plate is normalised;
// an unknown vehicle or customer yields the matching not-found error and a
// plate that collides with another vehicle yields ErrDuplicatePlate.
func Update(ctx context.Context, q store.Querier, id, customerID int, plate, brand, model string, mileageKm int) (apimodel.Vehicle, error) {
	normalized, brand, model, err := validate(customerID, plate, brand, model, mileageKm)
	if err != nil {
		return apimodel.Vehicle{}, err
	}

	if _, err := Get(ctx, q, id); err != nil {
		return apimodel.Vehicle{}, err
	}

	exists, err := customerExists(ctx, q, customerID)
	if err != nil {
		return apimodel.Vehicle{}, err
	}
	if !exists {
		return apimodel.Vehicle{}, ErrCustomerNotFound
	}

	var v apimodel.Vehicle
	err = q.QueryRow(ctx,
		`UPDATE vehicles
		    SET customer_id = $2, plate = $3, plate_normalized = $4, brand = $5, model = $6, mileage_km = $7
		  WHERE id = $1
		  RETURNING id, customer_id, plate, brand, model, mileage_km`,
		id, customerID, normalized, normalized, brand, model, mileageKm,
	).Scan(&v.ID, &v.CustomerID, &v.Plate, &v.Brand, &v.Model, &v.MileageKm)
	if err != nil {
		if isUniqueViolation(err) {
			return apimodel.Vehicle{}, ErrDuplicatePlate
		}
		if isForeignKeyViolation(err) {
			return apimodel.Vehicle{}, ErrCustomerNotFound
		}
		return apimodel.Vehicle{}, fmt.Errorf("update vehicle: %w", err)
	}
	return v, nil
}
