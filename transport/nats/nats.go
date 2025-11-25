// Package nats provides a NATS-based event bus implementation
package nats

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/nats-io/nats.go"
)

// EventHandler is the function signature for event handlers
type EventHandler func(context.Context, *cloudevents.Event) error

// NATSBus implements an event bus using NATS messaging system
type NATSBus struct {
	conn          *nats.Conn
	subscriptions []*nats.Subscription
	mu            sync.Mutex
}

// Config holds the configuration for NATS connection
type Config struct {
	// URL is the NATS server URL (e.g., "nats://localhost:4222")
	URL string

	// Options allows customizing the NATS connection
	Options []nats.Option
}

// NewNATSBus creates a new NATS event bus with the given configuration
func NewNATSBus(cfg Config) (*NATSBus, error) {
	if cfg.URL == "" {
		cfg.URL = nats.DefaultURL
	}

	conn, err := nats.Connect(cfg.URL, cfg.Options...)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}

	return &NATSBus{
		conn:          conn,
		subscriptions: make([]*nats.Subscription, 0),
	}, nil
}

// Publish publishes an event to NATS
func (b *NATSBus) Publish(ctx context.Context, subject string, event *cloudevents.Event) error {
	if b.conn == nil || b.conn.IsClosed() {
		return fmt.Errorf("nats: connection is closed")
	}

	// Serialize CloudEvents to JSON
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("nats: failed to marshal event: %w", err)
	}

	// Publish to NATS subject
	if err := b.conn.Publish(subject, data); err != nil {
		return fmt.Errorf("nats: failed to publish: %w", err)
	}

	return nil
}

// Subscribe subscribes to events on a subject (broadcast mode)
// All subscribers with the same subject will receive all messages
func (b *NATSBus) Subscribe(ctx context.Context, subject string, handler EventHandler) error {
	if b.conn == nil || b.conn.IsClosed() {
		return fmt.Errorf("nats: connection is closed")
	}

	sub, err := b.conn.Subscribe(subject, func(msg *nats.Msg) {
		var event cloudevents.Event
		if err := json.Unmarshal(msg.Data, &event); err != nil {
			// Log error but don't stop processing
			return
		}

		if err := handler(ctx, &event); err != nil {
			// Log error but don't stop processing
			return
		}
	})
	if err != nil {
		return fmt.Errorf("nats: failed to subscribe: %w", err)
	}

	b.mu.Lock()
	b.subscriptions = append(b.subscriptions, sub)
	b.mu.Unlock()

	return nil
}

// SubscribeWithHandlerGroup subscribes to events using a queue group (handler group mode)
// Messages are load-balanced across subscribers in the same group
func (b *NATSBus) SubscribeWithHandlerGroup(ctx context.Context, subject, group string, handler EventHandler) error {
	if b.conn == nil || b.conn.IsClosed() {
		return fmt.Errorf("nats: connection is closed")
	}

	if group == "" {
		return fmt.Errorf("nats: group name is required")
	}

	sub, err := b.conn.QueueSubscribe(subject, group, func(msg *nats.Msg) {
		var event cloudevents.Event
		if err := json.Unmarshal(msg.Data, &event); err != nil {
			// Log error but don't stop processing
			return
		}

		if err := handler(ctx, &event); err != nil {
			// Log error but don't stop processing
			return
		}
	})
	if err != nil {
		return fmt.Errorf("nats: failed to queue subscribe: %w", err)
	}

	b.mu.Lock()
	b.subscriptions = append(b.subscriptions, sub)
	b.mu.Unlock()

	return nil
}

// Close closes all subscriptions and the NATS connection
func (b *NATSBus) Close(ctx context.Context) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Unsubscribe all
	for _, sub := range b.subscriptions {
		if err := sub.Unsubscribe(); err != nil {
			// Log but continue
		}
	}
	b.subscriptions = nil

	// Close connection
	if b.conn != nil && !b.conn.IsClosed() {
		b.conn.Close()
	}

	return nil
}

// Drain gracefully drains all subscriptions and closes the connection
// This ensures all in-flight messages are processed before closing
func (b *NATSBus) Drain(ctx context.Context) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.conn == nil || b.conn.IsClosed() {
		return nil
	}

	// Drain connection and wait for it to close
	if err := b.conn.Drain(); err != nil {
		return err
	}

	// Add subscriptions to be tracked
	b.subscriptions = nil

	return nil
}
