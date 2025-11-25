package main

import (
	"context"
	"log"
	"time"

	"github.com/yafeiaa/protoc-gen-cloudevents-go/examples/basic/events"
	natstransport "github.com/yafeiaa/protoc-gen-cloudevents-go/transport/nats"
)

func main() {
	ctx := context.Background()

	// Create NATS event bus
	// Make sure NATS server is running: docker run -d -p 4222:4222 nats:latest
	bus, err := natstransport.NewNATSBus(natstransport.Config{
		URL: "nats://localhost:4222",
	})
	if err != nil {
		log.Fatalf("Failed to connect to NATS: %v", err)
	}
	defer bus.Close(ctx)

	log.Println("Connected to NATS server")

	// Example 1: Publish user registration event
	log.Println("\n📤 Publishing user registration event...")
	err = events.PublishUserRegistered(ctx, bus,
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

	// Give time for messages to be delivered
	time.Sleep(1 * time.Second)
	log.Println("\n🎉 All events published!")
}
