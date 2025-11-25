package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

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
	defer bus.Drain(ctx)

	log.Println("Connected to NATS server")

	// Subscribe to user registration events (broadcast mode)
	log.Println("🔔 Subscribing to user registration events (broadcast mode)...")
	err = events.SubscribeUserRegistered(ctx, bus,
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
			log.Printf("📧 [email-sender] Sending order confirmation email: order_id=%s, amount=%.2f %s",
				payload.OrderId, payload.Amount, payload.Currency)
			return nil
		})
	if err != nil {
		log.Fatalf("Failed to subscribe: %v", err)
	}

	log.Println("🔔 Subscribing to order creation events (handler group: analytics)...")
	err = events.SubscribeOrderCreatedWithGroup(ctx, bus, "analytics",
		func(ctx context.Context, payload *events.OrderCreatedPayload) error {
			log.Printf("📊 [analytics] Recording order analytics data: order_id=%s, items=%d",
				payload.OrderId, len(payload.Items))
			return nil
		})
	if err != nil {
		log.Fatalf("Failed to subscribe: %v", err)
	}

	log.Println("\n✅ All subscribers ready, waiting for events...")
	log.Println("Press Ctrl+C to exit")

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	log.Println("\n👋 Shutting down gracefully...")
}
