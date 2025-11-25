// Package memory provides an in-memory event bus implementation for testing and demonstration purposes
package memory

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
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

// matchSubject checks if a subject matches a pattern with wildcards
// Supports "*" wildcard matching (e.g., "app.*.created" matches "app.user.created")
func matchSubject(pattern, subject string) bool {
	// If pattern doesn't contain wildcards, use exact match
	if !strings.Contains(pattern, "*") {
		return pattern == subject
	}

	// Convert NATS-style wildcards to filepath-style for filepath.Match
	// NATS uses * for single-level matching and > for multi-level
	filePattern := strings.ReplaceAll(pattern, "*", "*")
	
	// Use filepath.Match which supports * wildcards
	matched, err := filepath.Match(filePattern, subject)
	if err != nil {
		return false
	}
	return matched
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
	// Validate inputs
	if subject == "" {
		return fmt.Errorf("subject is required")
	}
	if event == nil {
		return fmt.Errorf("event is required")
	}

	b.mu.RLock()
	defer b.mu.RUnlock()

	// Broadcast to all broadcast-mode subscribers with matching subjects
	for pattern, handlers := range b.handlers {
		if matchSubject(pattern, subject) {
			for _, handler := range handlers {
				// Handle errors but continue processing other handlers
				_ = handler(ctx, event)
			}
		}
	}

	// Handler group mode (load balancing) with wildcard support
	for pattern, groupHandlers := range b.groups {
		if matchSubject(pattern, subject) {
			for group, handlers := range groupHandlers {
				if len(handlers) == 0 {
					continue
				}

				// Round-robin handler selection
				index := b.groupIndex[pattern][group] % len(handlers)
				handler := handlers[index]
				b.groupIndex[pattern][group]++

				// Handle errors but continue processing other handlers
				_ = handler(ctx, event)
			}
		}
	}

	return nil
}

// Subscribe subscribes to events (broadcast mode)
func (b *MemoryBus) Subscribe(ctx context.Context, subject string, handler EventHandler) error {
	if subject == "" {
		return fmt.Errorf("subject is required")
	}
	if handler == nil {
		return fmt.Errorf("handler is required")
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	b.handlers[subject] = append(b.handlers[subject], handler)
	return nil
}

// SubscribeWithHandlerGroup subscribes to events (handler group mode)
func (b *MemoryBus) SubscribeWithHandlerGroup(ctx context.Context, subject, group string, handler EventHandler) error {
	if subject == "" {
		return fmt.Errorf("subject is required")
	}
	if group == "" {
		return fmt.Errorf("group is required")
	}
	if handler == nil {
		return fmt.Errorf("handler is required")
	}

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
