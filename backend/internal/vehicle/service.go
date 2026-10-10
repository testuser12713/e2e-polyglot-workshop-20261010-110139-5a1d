// Package vehicle implements the vehicle use cases, including the normalised
// plate used for the unique constraint. This ticket ships the signatures only;
// ticket #6 fills the bodies.
package vehicle

import (
	"context"
	"strings"

	apimodel "workshop/backend/internal/model"
	"workshop/backend/internal/store"
)

// NormalizePlate upper-cases and strips spaces and dashes so that two spellings
// of the same plate compare equal.
func NormalizePlate(plate string) string {
	r := strings.NewReplacer(" ", "", "-", "", "\t", "")
	return strings.ToUpper(strings.TrimSpace(r.Replace(plate)))
}

// Create stores a new vehicle for a customer and returns it.
func Create(ctx context.Context, q store.Querier, customerID int, plate, brand, model string, mileageKm int) (apimodel.Vehicle, error) {
	return apimodel.Vehicle{}, nil
}

// Get loads one vehicle by id.
func Get(ctx context.Context, q store.Querier, id int) (apimodel.Vehicle, error) {
	return apimodel.Vehicle{}, nil
}

// Update changes an existing vehicle and returns it.
func Update(ctx context.Context, q store.Querier, id, customerID int, plate, brand, model string, mileageKm int) (apimodel.Vehicle, error) {
	return apimodel.Vehicle{}, nil
}
