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

	// Create in-memory event bus
	bus := memory.NewMemoryBus()

	// Subscribe to user registration events (broadcast mode)
	log.Println("🔔 Subscribing to user registration events (broadcast mode)...")
	err := events.SubscribeUserRegistered(ctx, bus,
		func(ctx context.Context, payload *events.UserRegisteredPayload) error {
			log.Printf("✉️  Received user registration event: user_id=%s, email=%s",
				payload.UserId, payload.Email)
			return nil
		})
	if err != nil {
		log.Fatalf("Failed to subscribe: %v", err)
	}

	// Subscribe to order creation events (handler group mode)
	log.Println("🔔 Subscribing to order creation events (handler group: email-sender)...")
	err = events.SubscribeOrderCreatedWithGroup(ctx, bus, "email-sender",
		func(ctx context.Context, payload *events.OrderCreatedPayload) error {
			log.Printf("📧 Sending order confirmation email: order_id=%s, amount=%.2f %s",
				payload.OrderId, payload.Amount, payload.Currency)
			return nil
		})
	if err != nil {
		log.Fatalf("Failed to subscribe: %v", err)
	}

	log.Println("🔔 Subscribing to order creation events (handler group: analytics)...")
	err = events.SubscribeOrderCreatedWithGroup(ctx, bus, "analytics",
		func(ctx context.Context, payload *events.OrderCreatedPayload) error {
			log.Printf("📊 Recording order analytics data: order_id=%s, items=%d",
				payload.OrderId, len(payload.Items))
			return nil
		})
	if err != nil {
		log.Fatalf("Failed to subscribe: %v", err)
	}

	log.Println("\n✅ All subscribers ready, waiting for events...")

	// Keep running
	time.Sleep(10 * time.Second)
}
