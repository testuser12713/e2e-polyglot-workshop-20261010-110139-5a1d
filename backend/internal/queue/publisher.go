// Package queue publishes invoice jobs to the Valkey list the worker consumes.
package queue

import (
	"context"

	"github.com/redis/go-redis/v9"

	"workshop/backend/internal/order"
)

// Publisher pushes completed-order messages onto a Valkey list.
type Publisher struct {
	client    *redis.Client
	queueName string
}

// NewPublisher builds a publisher for the given queue.
func NewPublisher(client *redis.Client, queueName string) *Publisher {
	return &Publisher{client: client, queueName: queueName}
}

// NotifyOrderCompleted pushes exactly one JSON message for a completed order.
// This ticket ships the signature only; ticket #19 fills the body.
func (p *Publisher) NotifyOrderCompleted(ctx context.Context, orderNumber string) error {
	return nil
}

// compile-time proof that Publisher satisfies order.Notifier.
var _ order.Notifier = (*Publisher)(nil)
