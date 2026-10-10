// Package queue publishes invoice jobs to the Valkey list the worker consumes.
package queue

import (
	"context"
	"encoding/json"
	"log"

	"github.com/redis/go-redis/v9"

	"workshop/backend/internal/order"
)

// defaultQueueName matches QUEUE_NAME's default when the environment omits it.
const defaultQueueName = "workshop:invoices"

// Publisher pushes completed-order messages onto a Valkey list.
type Publisher struct {
	client    *redis.Client
	queueName string
}

// NewPublisher builds a publisher for the given queue. An empty queue name
// falls back to the contract's default.
func NewPublisher(client *redis.Client, queueName string) *Publisher {
	if queueName == "" {
		queueName = defaultQueueName
	}
	return &Publisher{client: client, queueName: queueName}
}

// orderCompletedMessage is the queue contract's payload, shared with the worker
// that BRPOPs from the same list.
type orderCompletedMessage struct {
	OrderNumber string `json:"orderNumber"`
	OrderID     int    `json:"orderId"`
}

// NotifyOrderCompleted pushes exactly one JSON message for a completed order
// onto the Valkey list. It is best effort: the status change has already been
// committed, so a briefly unreachable Valkey is logged without PII and returned
// as an error for the caller to ignore — the request never fails because of the
// queue.
func (p *Publisher) NotifyOrderCompleted(ctx context.Context, orderID int, orderNumber string) error {
	if p == nil || p.client == nil {
		log.Printf("queue: no Valkey client configured; skipping completed-order notification")
		return nil
	}

	payload, err := json.Marshal(orderCompletedMessage{OrderNumber: orderNumber, OrderID: orderID})
	if err != nil {
		// Marshalling a fixed struct of a string and an int cannot realistically
		// fail; log by type only so no user value can leak.
		log.Printf("queue: could not encode completed-order notification: %T", err)
		return err
	}

	if err := p.client.RPush(ctx, p.queueName, payload).Err(); err != nil {
		// Log the error type only: a raw transport error can carry the
		// connection address, and the payload is user data (AC-29).
		log.Printf("queue: could not publish completed-order notification: %T", err)
		return err
	}
	return nil
}

// compile-time proof that Publisher satisfies order.Notifier.
var _ order.Notifier = (*Publisher)(nil)
