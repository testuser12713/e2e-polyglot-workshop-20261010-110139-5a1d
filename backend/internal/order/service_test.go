package order

import (
	"context"
	"errors"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"workshop/backend/internal/model"
	"workshop/backend/internal/store"
)

// testPool opens a real PostgreSQL pool, creates the schema and skips the test
// when DATABASE_URL is not configured. There is no SQLite fallback (AC-20).
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

// seedVehicle inserts a customer and a vehicle and returns the vehicle id. It
// only writes to tables the order flow depends on; it resets nothing, so it
// never touches rows another ticket owns.
func seedVehicle(t *testing.T, pool store.Pool) int {
	t.Helper()
	ctx := context.Background()
	suffix := time.Now().UnixNano()

	var customerID int
	if err := pool.QueryRow(ctx,
		`INSERT INTO customers (name, email, phone) VALUES ($1, $2, $3) RETURNING id`,
		"Test Kunde", "kunde@example.test", "0000-000000").Scan(&customerID); err != nil {
		t.Fatalf("insert customer: %v", err)
	}

	plate := "TEST-" + strconv.FormatInt(suffix, 10)
	var vehicleID int
	if err := pool.QueryRow(ctx,
		`INSERT INTO vehicles (customer_id, plate, plate_normalized, brand, model, mileage_km)
		 VALUES ($1, $2, $3, $4, $5, $6) RETURNING id`,
		customerID, plate, plate, "VW", "Golf", 120000).Scan(&vehicleID); err != nil {
		t.Fatalf("insert vehicle: %v", err)
	}
	return vehicleID
}

func TestCreateOrderSetsNumberAndRequestedStatus(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	vehicleID := seedVehicle(t, pool)

	ord, err := CreateOrder(ctx, pool, vehicleID, "2026-11-05", "Bremsen quietschen")
	if err != nil {
		t.Fatalf("CreateOrder: %v", err)
	}
	if ord.OrderNumber == "" {
		t.Fatal("orderNumber is empty")
	}
	if !strings.HasPrefix(ord.OrderNumber, "AU-") {
		t.Fatalf("orderNumber = %q, want an AU- prefix", ord.OrderNumber)
	}
	if ord.Status != model.StatusRequested {
		t.Fatalf("status = %q, want %q", ord.Status, model.StatusRequested)
	}
	if ord.VehicleID != vehicleID {
		t.Fatalf("vehicleId = %d, want %d", ord.VehicleID, vehicleID)
	}
	if ord.RequestedDate != "2026-11-05" {
		t.Fatalf("requestedDate = %q, want 2026-11-05", ord.RequestedDate)
	}
	if ord.ProblemDescription != "Bremsen quietschen" {
		t.Fatalf("problemDescription = %q", ord.ProblemDescription)
	}
	if len(ord.History) != 1 || ord.History[0].Status != model.StatusRequested {
		t.Fatalf("history = %+v, want one requested entry", ord.History)
	}
	if ord.Items == nil {
		t.Fatal("items is nil, want an empty array")
	}
}

func TestCreateOrderDistinctNumbers(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	vehicleID := seedVehicle(t, pool)

	first, err := CreateOrder(ctx, pool, vehicleID, "2026-11-05", "Erster Auftrag")
	if err != nil {
		t.Fatalf("CreateOrder first: %v", err)
	}
	second, err := CreateOrder(ctx, pool, vehicleID, "2026-11-06", "Zweiter Auftrag")
	if err != nil {
		t.Fatalf("CreateOrder second: %v", err)
	}
	if first.OrderNumber == second.OrderNumber {
		t.Fatalf("order numbers are not unique: %q", first.OrderNumber)
	}
}

func TestCreateOrderUnknownVehicle(t *testing.T) {
	pool := testPool(t)
	_, err := CreateOrder(context.Background(), pool, 999999999, "2026-11-05", "Unbekannt")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}

func TestCreateOrderValidation(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	vehicleID := seedVehicle(t, pool)

	cases := []struct {
		name    string
		vehicle int
		date    string
		problem string
	}{
		{"missing vehicle", 0, "2026-11-05", "Problem"},
		{"bad date", vehicleID, "05.11.2026", "Problem"},
		{"empty problem", vehicleID, "2026-11-05", "   "},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := CreateOrder(ctx, pool, tc.vehicle, tc.date, tc.problem); !errors.Is(err, ErrValidation) {
				t.Fatalf("err = %v, want ErrValidation", err)
			}
		})
	}
}

func TestAddItemsAndReload(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	vehicleID := seedVehicle(t, pool)

	ord, err := CreateOrder(ctx, pool, vehicleID, "2026-11-05", "Inspektion")
	if err != nil {
		t.Fatalf("CreateOrder: %v", err)
	}

	hours := 2.5
	rate := 8000
	quantity := 3
	partPrice := 1250
	items := []model.OrderItemInput{
		{Kind: model.OrderItemKindLabor, Description: "Arbeitszeit", Hours: &hours, UnitPriceCents: &rate},
		{Kind: model.OrderItemKindPart, Description: "Ölfilter", Quantity: &quantity, UnitPriceCents: &partPrice},
	}

	updated, err := AddItems(ctx, pool, ord.ID, items)
	if err != nil {
		t.Fatalf("AddItems: %v", err)
	}
	if len(updated.Items) != 2 {
		t.Fatalf("items = %d, want 2", len(updated.Items))
	}
	labor, part := updated.Items[0], updated.Items[1]
	if labor.Kind != model.OrderItemKindLabor || labor.Hours == nil || *labor.Hours != hours {
		t.Fatalf("labor item = %+v", labor)
	}
	if labor.AmountCents != 20000 {
		t.Fatalf("labor amountCents = %d, want 20000", labor.AmountCents)
	}
	if part.Kind != model.OrderItemKindPart || part.Quantity == nil || *part.Quantity != quantity {
		t.Fatalf("part item = %+v", part)
	}
	if part.AmountCents != 3750 {
		t.Fatalf("part amountCents = %d, want 3750", part.AmountCents)
	}

	reloaded, err := LoadOrder(ctx, pool, ord.ID)
	if err != nil {
		t.Fatalf("LoadOrder: %v", err)
	}
	if len(reloaded.Items) != 2 {
		t.Fatalf("reloaded items = %d, want 2", len(reloaded.Items))
	}
	if reloaded.Items[1].AmountCents != 3750 {
		t.Fatalf("reloaded part amountCents = %d, want 3750", reloaded.Items[1].AmountCents)
	}
}

func TestAddItemsUnknownOrder(t *testing.T) {
	pool := testPool(t)
	hours := 1.0
	_, err := AddItems(context.Background(), pool, 999999999,
		[]model.OrderItemInput{{Kind: model.OrderItemKindLabor, Description: "Arbeit", Hours: &hours}})
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}

func TestAddItemsValidation(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	vehicleID := seedVehicle(t, pool)
	ord, err := CreateOrder(ctx, pool, vehicleID, "2026-11-05", "Inspektion")
	if err != nil {
		t.Fatalf("CreateOrder: %v", err)
	}

	zero := 0
	hours := 2.0
	cases := []struct {
		name  string
		items []model.OrderItemInput
	}{
		{"empty list", nil},
		{"unknown kind", []model.OrderItemInput{{Kind: "other", Description: "x", Hours: &hours}}},
		{"labor without hours", []model.OrderItemInput{{Kind: model.OrderItemKindLabor, Description: "Arbeit"}}},
		{"part without price", []model.OrderItemInput{{Kind: model.OrderItemKindPart, Description: "Teil", Quantity: &zero}}},
		{"blank description", []model.OrderItemInput{{Kind: model.OrderItemKindLabor, Description: "  ", Hours: &hours}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := AddItems(ctx, pool, ord.ID, tc.items); !errors.Is(err, ErrValidation) {
				t.Fatalf("err = %v, want ErrValidation", err)
			}
		})
	}
}

func TestLoadOrderUnknown(t *testing.T) {
	pool := testPool(t)
	if _, err := LoadOrder(context.Background(), pool, 999999999); !errors.Is(err, ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}
