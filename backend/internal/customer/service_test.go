package customer

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"workshop/backend/internal/store"
)

// testPool opens a pool against the real PostgreSQL database the office runs
// beside the product and creates the schema. Without DATABASE_URL the test is
// skipped rather than faked with another database.
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
	if err := store.Migrate(ctx, pool); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

// uniqueEmail builds an e-mail address distinct per run so parallel tests never
// collide on a row.
func uniqueEmail(prefix string) string {
	return fmt.Sprintf("%s-%d@example.com", prefix, time.Now().UnixNano())
}

func TestCreateGetUpdateRoundTrip(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	email := uniqueEmail("roundtrip")

	created, err := Create(ctx, pool, "Anna Muster", email, "+49 170 1234567")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM customers WHERE id = $1`, created.ID)
	})
	if created.ID == 0 {
		t.Fatal("Create returned a zero id")
	}
	if created.Name != "Anna Muster" || created.Email != email || created.Phone != "+49 170 1234567" {
		t.Fatalf("Create returned %+v, want the values just stored", created)
	}

	got, err := Get(ctx, pool, created.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got != created {
		t.Fatalf("Get returned %+v, want %+v", got, created)
	}

	updated, err := Update(ctx, pool, created.ID, "Anna Beispiel", email, "0151 999888")
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.ID != created.ID || updated.Name != "Anna Beispiel" || updated.Phone != "0151 999888" {
		t.Fatalf("Update returned %+v, want the changed values", updated)
	}

	reloaded, err := Get(ctx, pool, created.ID)
	if err != nil {
		t.Fatalf("Get after Update: %v", err)
	}
	if reloaded != updated {
		t.Fatalf("Get after Update returned %+v, want %+v", reloaded, updated)
	}
}

func TestCreateRejectsInvalidInput(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()

	cases := []struct {
		name      string
		custName  string
		custEmail string
		wantField string
	}{
		{"missing name", "", "valid@example.com", "name is required"},
		{"blank name", "   ", "valid@example.com", "name is required"},
		{"missing email", "Anna", "", "email is required"},
		{"unusable email", "Anna", "not-an-email", "email is invalid"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Create(ctx, pool, tc.custName, tc.custEmail, "0170")
			var ve ValidationError
			if !errors.As(err, &ve) {
				t.Fatalf("Create(%q,%q) error = %v, want a ValidationError", tc.custName, tc.custEmail, err)
			}
			if ve.Message != tc.wantField {
				t.Fatalf("message = %q, want %q", ve.Message, tc.wantField)
			}
		})
	}
}

func TestGetUnknownReturnsNotFound(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()

	_, err := Get(ctx, pool, 2147480000)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("Get(unknown) error = %v, want ErrNotFound", err)
	}
}

func TestUpdateUnknownReturnsNotFound(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()

	_, err := Update(ctx, pool, 2147480000, "Anna", "valid@example.com", "0170")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("Update(unknown) error = %v, want ErrNotFound", err)
	}
}

func TestUpdateRejectsInvalidInput(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()

	created, err := Create(ctx, pool, "Anna", uniqueEmail("update-validation"), "0170")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM customers WHERE id = $1`, created.ID)
	})

	_, err = Update(ctx, pool, created.ID, "", "valid@example.com", "0170")
	var ve ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("Update(blank name) error = %v, want a ValidationError", err)
	}
}
