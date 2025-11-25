package main

import (
	"context"
	"log"
	"time"

	"github.com/yafeiaa/protoc-gen-cloudevents-go/examples/basic/events"
	"github.com/yafeiaa/protoc-gen-cloudevents-go/transport/memory"
)

func main() {
	ctx := context.Background()

	// Create in-memory event bus (for demonstration)
	bus := memory.NewMemoryBus()

	// Example 1: Publish user registration event
	log.Println("📤 Publishing user registration event...")
	err := events.PublishUserRegistered(ctx, bus,
		&events.UserRegisteredPayload{
			UserId:       "user-123",
			Email:        "user@example.com",
			Username:     "johndoe",
			RegisteredAt: time.Now().Unix(),
		},
		events.WithSource("myapp/api-server"),
		events.WithExtension("trace_id", "trace-abc-123"),
	)
	if err != nil {
		log.Fatalf("Failed to publish event: %v", err)
	}
	log.Println("✅ User registration event published successfully")

	// Example 2: Publish order creation event
	log.Println("\n📤 Publishing order creation event...")
	err = events.PublishOrderCreated(ctx, bus,
		&events.OrderCreatedPayload{
			OrderId:  "order-456",
			UserId:   "user-123",
			Amount:   99.99,
			Currency: "USD",
			Items:    []string{"item-1", "item-2"},
		},
		events.WithSource("myapp/order-service"),
		events.WithExtension("region", "us-west-2"),
	)
	if err != nil {
		log.Fatalf("Failed to publish event: %v", err)
	}
	log.Println("✅ Order creation event published successfully")
}
