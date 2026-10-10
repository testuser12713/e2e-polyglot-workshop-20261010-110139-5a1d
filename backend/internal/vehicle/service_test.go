package vehicle

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"

	"workshop/backend/internal/store"
)

// testPool opens a real PostgreSQL pool (AC-20: never SQLite), migrates the
// schema and closes the pool when the test ends. It skips — it does not fake the
// database — when DATABASE_URL is not provided by the office.
func testPool(t *testing.T) store.Pool {
	t.Helper()
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Skip("DATABASE_URL is not set; the office runs this test against PostgreSQL")
	}
	if strings.Contains(strings.ToLower(url), "sqlite") {
		t.Fatalf("DATABASE_URL points at SQLite; this product requires PostgreSQL")
	}
	ctx := context.Background()
	pool, err := store.NewPool(ctx, url)
	if err != nil {
		t.Fatalf("open postgres: %v", err)
	}
	t.Cleanup(pool.Close)
	if err := store.Migrate(ctx, pool); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return pool
}

// createCustomer inserts a customer directly (the customer service belongs to
// another ticket and is not part of this slice) and removes it again by id,
// which cascades to its vehicles. It never truncates the customers table.
func createCustomer(t *testing.T, pool store.Pool) int {
	t.Helper()
	email := fmt.Sprintf("vehicle-test-%s@example.test", strings.ReplaceAll(t.Name(), "/", "-"))
	var id int
	err := pool.QueryRow(context.Background(),
		`INSERT INTO customers (name, email, phone) VALUES ($1, $2, $3) RETURNING id`,
		"Test Customer", email, "0000-000000",
	).Scan(&id)
	if err != nil {
		t.Fatalf("insert customer: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM customers WHERE id = $1`, id)
	})
	return id
}

func TestCreateAndGetVehicle(t *testing.T) {
	pool := testPool(t)
	customerID := createCustomer(t, pool)
	ctx := context.Background()

	created, err := Create(ctx, pool, customerID, "b-xy 123", "VW", "Golf", 120000)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if created.ID == 0 {
		t.Fatal("Create returned a vehicle without an id")
	}
	if created.CustomerID != customerID {
		t.Fatalf("CustomerID = %d, want %d", created.CustomerID, customerID)
	}
	if created.Plate != "BXY123" {
		t.Fatalf("Plate = %q, want normalised %q", created.Plate, "BXY123")
	}
	if created.Brand != "VW" || created.Model != "Golf" || created.MileageKm != 120000 {
		t.Fatalf("unexpected vehicle fields: %+v", created)
	}

	got, err := Get(ctx, pool, created.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got != created {
		t.Fatalf("Get = %+v, want %+v", got, created)
	}
}

func TestGetUnknownVehicleIsNotFound(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()

	_, err := Get(ctx, pool, 999999999)
	if !errors.Is(err, ErrVehicleNotFound) {
		t.Fatalf("Get unknown: err = %v, want ErrVehicleNotFound", err)
	}
}

func TestCreateDuplicatePlateInDifferentSpelling(t *testing.T) {
	pool := testPool(t)
	customerID := createCustomer(t, pool)
	ctx := context.Background()

	if _, err := Create(ctx, pool, customerID, "B-XY 123", "VW", "Golf", 1000); err != nil {
		t.Fatalf("first Create: %v", err)
	}

	_, err := Create(ctx, pool, customerID, "bxy123", "Audi", "A4", 2000)
	if !errors.Is(err, ErrDuplicatePlate) {
		t.Fatalf("duplicate Create: err = %v, want ErrDuplicatePlate", err)
	}
}

func TestCreateUnknownCustomer(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()

	_, err := Create(ctx, pool, 999999999, "XY-99 9", "VW", "Golf", 1000)
	if !errors.Is(err, ErrCustomerNotFound) {
		t.Fatalf("Create with unknown customer: err = %v, want ErrCustomerNotFound", err)
	}
}

func TestCreateValidatesRequiredFields(t *testing.T) {
	pool := testPool(t)
	customerID := createCustomer(t, pool)
	ctx := context.Background()

	cases := []struct {
		name       string
		customerID int
		plate      string
		brand      string
		model      string
		mileage    int
	}{
		{"missing customer", 0, "AB-123", "VW", "Golf", 1000},
		{"missing plate", customerID, "   ", "VW", "Golf", 1000},
		{"missing brand", customerID, "AB-123", " ", "Golf", 1000},
		{"missing model", customerID, "AB-123", "VW", " ", 1000},
		{"negative mileage", customerID, "AB-123", "VW", "Golf", -1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Create(ctx, pool, tc.customerID, tc.plate, tc.brand, tc.model, tc.mileage)
			if !errors.Is(err, ErrValidation) {
				t.Fatalf("err = %v, want ErrValidation", err)
			}
		})
	}
}

func TestUpdateVehicle(t *testing.T) {
	pool := testPool(t)
	customerID := createCustomer(t, pool)
	ctx := context.Background()

	created, err := Create(ctx, pool, customerID, "AB-123", "VW", "Golf", 1000)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	updated, err := Update(ctx, pool, created.ID, customerID, "cd 456", "Audi", "A4", 5000)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.ID != created.ID {
		t.Fatalf("Update changed the id: %d -> %d", created.ID, updated.ID)
	}
	if updated.Plate != "CD456" || updated.Brand != "Audi" || updated.Model != "A4" || updated.MileageKm != 5000 {
		t.Fatalf("unexpected updated vehicle: %+v", updated)
	}

	got, err := Get(ctx, pool, created.ID)
	if err != nil {
		t.Fatalf("Get after Update: %v", err)
	}
	if got != updated {
		t.Fatalf("stored vehicle = %+v, want %+v", got, updated)
	}
}

func TestUpdateDuplicatePlateInDifferentSpelling(t *testing.T) {
	pool := testPool(t)
	customerID := createCustomer(t, pool)
	ctx := context.Background()

	if _, err := Create(ctx, pool, customerID, "AB-123", "VW", "Golf", 1000); err != nil {
		t.Fatalf("Create first: %v", err)
	}
	second, err := Create(ctx, pool, customerID, "CD-456", "Audi", "A4", 2000)
	if err != nil {
		t.Fatalf("Create second: %v", err)
	}

	_, err = Update(ctx, pool, second.ID, customerID, "ab123", "Audi", "A4", 2000)
	if !errors.Is(err, ErrDuplicatePlate) {
		t.Fatalf("Update to duplicate: err = %v, want ErrDuplicatePlate", err)
	}

	got, err := Get(ctx, pool, second.ID)
	if err != nil {
		t.Fatalf("Get second: %v", err)
	}
	if got.Plate != "CD456" {
		t.Fatalf("plate changed after rejected update: %q", got.Plate)
	}
}

func TestUpdateUnknownVehicle(t *testing.T) {
	pool := testPool(t)
	customerID := createCustomer(t, pool)
	ctx := context.Background()

	_, err := Update(ctx, pool, 999999999, customerID, "AB-123", "VW", "Golf", 1000)
	if !errors.Is(err, ErrVehicleNotFound) {
		t.Fatalf("err = %v, want ErrVehicleNotFound", err)
	}
}

func TestUpdateUnknownCustomer(t *testing.T) {
	pool := testPool(t)
	customerID := createCustomer(t, pool)
	ctx := context.Background()

	created, err := Create(ctx, pool, customerID, "AB-123", "VW", "Golf", 1000)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	_, err = Update(ctx, pool, created.ID, 999999999, "AB-123", "VW", "Golf", 1000)
	if !errors.Is(err, ErrCustomerNotFound) {
		t.Fatalf("err = %v, want ErrCustomerNotFound", err)
	}
}
