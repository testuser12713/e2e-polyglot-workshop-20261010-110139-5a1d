package queue

import (
	"context"
	"encoding/json"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"

	"workshop/backend/internal/model"
	"workshop/backend/internal/order"
	"workshop/backend/internal/store"
)

// queueTestPool opens a real PostgreSQL pool and creates the schema, skipping
// the test when DATABASE_URL is not configured. There is no SQLite fallback.
func queueTestPool(t *testing.T) store.Pool {
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

// testValkey opens a real Valkey client and skips the test when VALKEY_URL is
// not configured.
func testValkey(t *testing.T) *redis.Client {
	t.Helper()
	url := os.Getenv("VALKEY_URL")
	if url == "" {
		t.Skip("VALKEY_URL is not set; the office runs this test against Valkey")
	}
	opts, err := redis.ParseURL(url)
	if err != nil {
		t.Fatalf("parse VALKEY_URL: %v", err)
	}
	client := redis.NewClient(opts)
	t.Cleanup(func() { _ = client.Close() })
	if err := client.Ping(context.Background()).Err(); err != nil {
		t.Fatalf("ping valkey: %v", err)
	}
	return client
}

// seedOrder creates a customer, a vehicle and an order through the shared
// order.CreateOrder and returns the order, so the publisher test exercises the
// same id and order number the API would publish.
func seedOrder(t *testing.T, pool store.Pool) model.Order {
	t.Helper()
	ctx := context.Background()
	suffix := time.Now().UnixNano()

	var customerID int
	if err := pool.QueryRow(ctx,
		`INSERT INTO customers (name, email, phone) VALUES ($1, $2, $3) RETURNING id`,
		"Queue Kunde", "queue@example.test", "0000-000000").Scan(&customerID); err != nil {
		t.Fatalf("insert customer: %v", err)
	}

	plate := "QUEUE-" + strconv.FormatInt(suffix, 10)
	var vehicleID int
	if err := pool.QueryRow(ctx,
		`INSERT INTO vehicles (customer_id, plate, plate_normalized, brand, model, mileage_km)
		 VALUES ($1, $2, $3, $4, $5, $6) RETURNING id`,
		customerID, plate, plate, "VW", "Golf", 1000).Scan(&vehicleID); err != nil {
		t.Fatalf("insert vehicle: %v", err)
	}

	ord, err := order.CreateOrder(ctx, pool, vehicleID, "2026-11-05", "Rechnung")
	if err != nil {
		t.Fatalf("CreateOrder: %v", err)
	}
	return ord
}

// uniqueQueue returns a queue name no other test shares and clears it after the
// test, so it never touches the product's real list.
func uniqueQueue(t *testing.T, client *redis.Client) string {
	t.Helper()
	name := "workshop:test:" + strconv.FormatInt(time.Now().UnixNano(), 10)
	t.Cleanup(func() { _ = client.Del(context.Background(), name).Err() })
	return name
}

func TestNotifyOrderCompletedPushesOneMessage(t *testing.T) {
	pool := queueTestPool(t)
	client := testValkey(t)
	ctx := context.Background()
	ord := seedOrder(t, pool)
	queueName := uniqueQueue(t, client)

	pub := NewPublisher(client, queueName)
	if err := pub.NotifyOrderCompleted(ctx, ord.ID, ord.OrderNumber); err != nil {
		t.Fatalf("NotifyOrderCompleted: %v", err)
	}

	msgs, err := client.LRange(ctx, queueName, 0, -1).Result()
	if err != nil {
		t.Fatalf("LRange: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("messages = %d, want exactly 1", len(msgs))
	}

	var payload struct {
		OrderNumber string `json:"orderNumber"`
		OrderID     int    `json:"orderId"`
	}
	if err := json.Unmarshal([]byte(msgs[0]), &payload); err != nil {
		t.Fatalf("decode message %q: %v", msgs[0], err)
	}
	if payload.OrderNumber != ord.OrderNumber {
		t.Fatalf("orderNumber = %q, want %q", payload.OrderNumber, ord.OrderNumber)
	}
	if payload.OrderID != ord.ID {
		t.Fatalf("orderId = %d, want %d", payload.OrderID, ord.ID)
	}
}

func TestNotifyOrderCompletedPushesOneMessagePerCall(t *testing.T) {
	pool := queueTestPool(t)
	client := testValkey(t)
	ctx := context.Background()
	first := seedOrder(t, pool)
	second := seedOrder(t, pool)
	queueName := uniqueQueue(t, client)

	pub := NewPublisher(client, queueName)
	for _, ord := range []model.Order{first, second} {
		if err := pub.NotifyOrderCompleted(ctx, ord.ID, ord.OrderNumber); err != nil {
			t.Fatalf("NotifyOrderCompleted: %v", err)
		}
	}

	length, err := client.LLen(ctx, queueName).Result()
	if err != nil {
		t.Fatalf("LLen: %v", err)
	}
	if length != 2 {
		t.Fatalf("queue length = %d, want 2 (one per call)", length)
	}
}

func TestNotifyOrderCompletedIsBestEffortWhenValkeyIsDown(t *testing.T) {
	// A client pointed at a closed port: RPUSH fails fast, and the publisher
	// must report the failure instead of panicking.
	client := redis.NewClient(&redis.Options{
		Addr:        "127.0.0.1:1",
		DialTimeout: 200 * time.Millisecond,
		MaxRetries:  -1,
	})
	t.Cleanup(func() { _ = client.Close() })

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	pub := NewPublisher(client, "workshop:test")
	if err := pub.NotifyOrderCompleted(ctx, 1, "AU-2026-00001"); err == nil {
		t.Fatal("want an error when Valkey is unreachable, got nil")
	}
}

func TestNewPublisherDefaultsQueueName(t *testing.T) {
	if got := NewPublisher(nil, "").queueName; got != defaultQueueName {
		t.Fatalf("queueName = %q, want %q", got, defaultQueueName)
	}
	if got := NewPublisher(nil, "custom").queueName; got != "custom" {
		t.Fatalf("queueName = %q, want custom", got)
	}
}
