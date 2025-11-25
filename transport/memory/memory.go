// Package memory provides an in-memory event bus implementation for testing and demonstration purposes
package memory

import (
	"context"
	"fmt"
	"sync"

	cloudevents "github.com/cloudevents/sdk-go/v2"
)

// EventHandler 事件处理函数
type EventHandler func(context.Context, *cloudevents.Event) error

// MemoryBus is an in-memory event bus implementation
type MemoryBus struct {
	mu         sync.RWMutex
	handlers   map[string][]EventHandler
	groups     map[string]map[string][]EventHandler // subject -> group -> handlers
	groupIndex map[string]map[string]int            // subject -> group -> current index
}

// NewMemoryBus creates a new in-memory event bus
func NewMemoryBus() *MemoryBus {
	return &MemoryBus{
		handlers:   make(map[string][]EventHandler),
		groups:     make(map[string]map[string][]EventHandler),
		groupIndex: make(map[string]map[string]int),
	}
}

// Publish publishes an event to the bus
func (b *MemoryBus) Publish(ctx context.Context, subject string, event *cloudevents.Event) error {
	b.mu.RLock()
	defer b.mu.RUnlock()

	// Broadcast to all broadcast-mode subscribers
	for _, handler := range b.handlers[subject] {
		if err := handler(ctx, event); err != nil {
			return fmt.Errorf("handler error: %w", err)
		}
	}

	// Handler group mode (load balancing)
	if groupHandlers, ok := b.groups[subject]; ok {
		for group, handlers := range groupHandlers {
			if len(handlers) == 0 {
				continue
			}

			// Round-robin handler selection
			index := b.groupIndex[subject][group] % len(handlers)
			handler := handlers[index]
			b.groupIndex[subject][group]++

			if err := handler(ctx, event); err != nil {
				return fmt.Errorf("group handler error: %w", err)
			}
		}
	}

	return nil
}

// Subscribe subscribes to events (broadcast mode)
func (b *MemoryBus) Subscribe(ctx context.Context, subject string, handler EventHandler) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.handlers[subject] = append(b.handlers[subject], handler)
	return nil
}

// SubscribeWithHandlerGroup subscribes to events (handler group mode)
func (b *MemoryBus) SubscribeWithHandlerGroup(ctx context.Context, subject, group string, handler EventHandler) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.groups[subject] == nil {
		b.groups[subject] = make(map[string][]EventHandler)
	}
	if b.groupIndex[subject] == nil {
		b.groupIndex[subject] = make(map[string]int)
	}

	b.groups[subject][group] = append(b.groups[subject][group], handler)
	return nil
}

// Close closes the event bus
func (b *MemoryBus) Close(ctx context.Context) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.handlers = make(map[string][]EventHandler)
	b.groups = make(map[string]map[string][]EventHandler)
	b.groupIndex = make(map[string]map[string]int)
	return nil
}
