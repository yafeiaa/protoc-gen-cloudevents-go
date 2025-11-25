# protoc-gen-cloudevents-go

[![CI](https://github.com/yafeiaa/protoc-gen-cloudevents-go/workflows/CI/badge.svg)](https://github.com/yafeiaa/protoc-gen-cloudevents-go/actions)
[![Go Report Card](https://goreportcard.com/badge/github.com/yafeiaa/protoc-gen-cloudevents-go)](https://goreportcard.com/report/github.com/yafeiaa/protoc-gen-cloudevents-go)
[![GoDoc](https://godoc.org/github.com/yafeiaa/protoc-gen-cloudevents-go?status.svg)](https://godoc.org/github.com/yafeiaa/protoc-gen-cloudevents-go)
[![codecov](https://codecov.io/gh/yafeiaa/protoc-gen-cloudevents-go/branch/main/graph/badge.svg)](https://codecov.io/gh/yafeiaa/protoc-gen-cloudevents-go)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

**Automatically generate type-safe CloudEvents publish/subscribe code from Protobuf definitions**

> A Protobuf code generator for event-driven architectures that automatically generates type-safe event publishing and subscription functions based on the [CloudEvents](https://cloudevents.io/) standard.
>
> 🌏 **[中文文档](README_CN.md)**

## ✨ Features

- 🚀 **Zero Boilerplate** - Auto-generate publish/subscribe functions from Protobuf definitions
- 🔒 **Type Safety** - Compile-time checking to avoid runtime errors
- ☁️ **CloudEvents Standard** - Fully compatible with CloudEvents specification
- 📡 **Multiple Patterns** - Broadcast mode + Handler group (load balancing) mode
- 🎯 **Runtime Flexibility** - Dynamic configuration of source/subject/group parameters
- 🔌 **Transport Agnostic** - NATS, In-Memory, and extensible to Kafka, HTTP, etc.
- 📝 **Single Source of Truth** - Proto files serve as documentation

## 🎬 Quick Start

### Installation

```bash
go install github.com/yafeiaa/protoc-gen-cloudevents-go/cmd/protoc-gen-cloudevents@latest
```

### Define Events (events.proto)

```protobuf
syntax = "proto3";

package myapp.events;
option go_package = "github.com/myapp/pkg/events;events";

import "google/protobuf/descriptor.proto";
import "cloudevents/event_meta.proto";

// User registration event
message UserRegisteredPayload {
  option (cloudevents.event_meta) = {
    event_type: "myapp.user.registered"
    description: "User registered successfully"
  };
  
  string user_id = 1;
  string email = 2;
  int64 registered_at = 3;
}

// Order creation event
message OrderCreatedPayload {
  option (cloudevents.event_meta) = {
    event_type: "myapp.order.created"
    description: "Order created successfully"
  };
  
  string order_id = 1;
  string user_id = 2;
  double amount = 3;
}
```

### Generate Code

```bash
# Method 1: Using protoc plugin
protoc \
  -I ./proto \
  -I ./third_party \
  --go_out=./pkg/events \
  --go_opt=paths=source_relative \
  --cloudevents_out=./pkg/events \
  --cloudevents_opt=paths=source_relative \
  ./proto/events.proto

# Method 2: Using convenience script
./scripts/generate.sh
```

### Use Generated Code

#### Publishing Events with NATS

```go
package main

import (
    "context"
    "github.com/myapp/pkg/events"
    natstransport "github.com/yafeiaa/protoc-gen-cloudevents-go/transport/nats"
)

func main() {
    ctx := context.Background()
    
    // Create NATS event bus
    bus, _ := natstransport.NewNATSBus(natstransport.Config{
        URL: "nats://localhost:4222",
    })
    defer bus.Close(ctx)
    
    // Publish user registration event (source is required)
    events.PublishUserRegistered(ctx, bus,
        &events.UserRegisteredPayload{
            UserId:       "user-123",
            Email:        "user@example.com",
            RegisteredAt: time.Now().Unix(),
        },
        events.WithSource("myapp/api-server"),         // Required
        events.WithExtension("trace_id", traceID),     // Optional
    )
}
```

#### Subscribing to Events

```go
// Broadcast mode - All subscribers receive events
events.SubscribeUserRegistered(ctx, bus,
    func(ctx context.Context, payload *events.UserRegisteredPayload) error {
        log.Printf("User registered: %s (%s)", payload.UserId, payload.Email)
        return nil
    })

// Handler group mode - Load balanced within same group
events.SubscribeUserRegisteredWithGroup(ctx, bus, "email-sender",
    func(ctx context.Context, payload *events.UserRegisteredPayload) error {
        return sendWelcomeEmail(payload.Email)
    })

events.SubscribeUserRegisteredWithGroup(ctx, bus, "analytics",
    func(ctx context.Context, payload *events.UserRegisteredPayload) error {
        return trackUserSignup(payload)
    })
```

## 📖 Examples

### Basic Usage (In-Memory Transport)

See [examples/basic](./examples/basic) for a simple example using in-memory transport:

```bash
# Terminal 1 - Subscriber
cd examples/basic/subscriber
go run main.go

# Terminal 2 - Publisher
cd examples/basic/publisher
go run main.go
```

### NATS Integration (Production)

**Prerequisites**: Start NATS server

```bash
docker run -d -p 4222:4222 --name nats-server nats:latest
```

**Run Examples**:

```bash
# Terminal 1 - Subscriber
cd examples/nats/subscriber
go run main.go

# Terminal 2 - Publisher
cd examples/nats/publisher
go run main.go
```

**Expected Output** (Subscriber):
```
Connected to NATS server
🔔 Subscribing to events...
✅ All subscribers ready

✉️  Received user registration: user_id=user-123
📧 [email-sender] Sending order confirmation: order_id=order-456
📊 [analytics] Recording analytics: order_id=order-456
```

**Key Features Demonstrated**:
- Broadcast mode: All subscribers receive user registration events
- Handler groups: Load balanced order processing (email-sender vs analytics)
- CloudEvents standard compliance

**Cleanup**:
```bash
docker stop nats-server && docker rm nats-server
```

### Coming Soon

- [Kafka Integration](./examples/kafka) - Apache Kafka transport adapter
- [Microservices Example](./examples/microservices) - Complete event-driven architecture

## 🔧 EventMeta Options

```protobuf
message EventMeta {
  // event_type: Globally unique event type (required)
  // Format: {domain}.{resource}.{action}
  // Example: "myapp.user.registered"
  string event_type = 1;
  
  // description: Event description (optional, used for documentation)
  string description = 2;
}
```

### Naming Conventions

#### event_type Naming

Use reverse domain name format:

```
{domain}.{resource}.{action}

✅ Recommended:
  myapp.user.registered
  myapp.order.created
  myapp.payment.completed

❌ Avoid:
  user.registered  (too short, potential conflicts)
  registered       (no context)
```

#### Proto Message Naming

Format: `{Resource}{Action}Payload`

| event_type | Proto Message |
|-----------|---------------|
| `myapp.user.registered` | `UserRegisteredPayload` |
| `myapp.order.created` | `OrderCreatedPayload` |

## 🎯 Publish Options

### WithSource (Required)

Set the event source to identify the publisher:

```go
// Single instance service
events.WithSource("myapp/api-server")

// Multi-instance service (dynamic source)
hostname, _ := os.Hostname()
events.WithSource(fmt.Sprintf("myapp/api-server/%s", hostname))

// Kubernetes Pod
podName := os.Getenv("POD_NAME")
events.WithSource(fmt.Sprintf("myapp/controller/%s", podName))

// Multi-cluster
clusterID := getClusterID()
events.WithSource(fmt.Sprintf("myapp/api-server@%s", clusterID))
```

### WithSubject (Optional)

Override the default NATS subject (defaults to event_type):

```go
// Route by user ID
events.PublishUserRegistered(ctx, bus, payload,
    events.WithSource("api-server"),
    events.WithSubject(fmt.Sprintf("myapp.user.registered.%s", userID)),
)

// Route by region
events.WithSubject(fmt.Sprintf("myapp.order.created.%s", region))
```

### WithExtension (Optional)

Add CloudEvents extension attributes:

```go
events.PublishUserRegistered(ctx, bus, payload,
    events.WithSource("api-server"),
    events.WithExtension("trace_id", traceID),
    events.WithExtension("user_agent", userAgent),
    events.WithExtension("region", "us-west-2"),
)
```

## 📡 Subscription Modes

### Broadcast Mode

All subscribers receive all events:

```go
// Logging service
events.SubscribeUserRegistered(ctx, bus, logHandler)

// Audit service
events.SubscribeUserRegistered(ctx, bus, auditHandler)

// Monitoring service
events.SubscribeUserRegistered(ctx, bus, metricsHandler)
```

**Use cases**: Logging, auditing, monitoring, metrics

### Handler Group Mode

Subscribers in the same group compete for messages (load balancing):

```go
// email-sender group - Load balanced across instances
events.SubscribeUserRegisteredWithGroup(ctx, bus, "email-sender", handler1)  // Instance 1
events.SubscribeUserRegisteredWithGroup(ctx, bus, "email-sender", handler2)  // Instance 2

// analytics group - Independent consumption
events.SubscribeUserRegisteredWithGroup(ctx, bus, "analytics", analyticsHandler)
```

**Use cases**: Async task processing, message queues, worker pools

## 🔌 Transport Adapters

### NATS (Production Ready)

```go
import natstransport "github.com/yafeiaa/protoc-gen-cloudevents-go/transport/nats"

bus, err := natstransport.NewNATSBus(natstransport.Config{
    URL: "nats://localhost:4222",
    Options: []nats.Option{
        nats.Name("my-service"),
        nats.MaxReconnects(10),
    },
})
```

**Features**:
- Production-ready NATS integration
- Broadcast mode via standard subscriptions
- Handler group mode via NATS queue groups
- Graceful shutdown with Drain()
- Full test coverage

### In-Memory (Testing)

```go
import "github.com/yafeiaa/protoc-gen-cloudevents-go/transport/memory"

bus := memory.NewMemoryBus()
```

**Features**:
- Fast in-memory implementation
- Perfect for unit tests
- No external dependencies

### Custom Adapters

Implement the `Publisher` and `Subscriber` interfaces:

```go
type Publisher interface {
    Publish(ctx context.Context, subject string, event *cloudevents.Event) error
}

type Subscriber interface {
    Subscribe(ctx context.Context, subject string, handler EventHandler) error
}

type HandlerGroupSubscriber interface {
    SubscribeWithHandlerGroup(ctx context.Context, subject, group string, handler EventHandler) error
}
```

Coming soon:
- Kafka adapter
- HTTP adapter
- Redis Streams adapter

## 🏗️ Project Structure

```
protoc-gen-cloudevents-go/
├── cmd/
│   └── protoc-gen-cloudevents/    # Code generator binary
│       └── main.go
├── proto/
│   └── cloudevents/               # Proto extension definitions
│       └── event_meta.proto
├── transport/                     # Transport adapters
│   ├── nats/                      # NATS implementation ✅
│   │   ├── nats.go
│   │   └── nats_test.go
│   └── memory/                    # In-memory implementation ✅
│       ├── memory.go
│       └── memory_test.go
├── examples/                      # Example applications
│   ├── basic/                     # Basic usage
│   └── nats/                      # NATS integration
├── scripts/
│   └── generate.sh                # Code generation script
├── Makefile                       # Build automation
├── README.md                      # English documentation
└── README_CN.md                   # Chinese documentation
```

## 🧪 Testing

```bash
# Run all tests
make test

# Run unit tests only
make test-unit

# Run integration tests
make test-integration

# View coverage
make coverage

# Run benchmarks
make bench
```

### NATS Tests

NATS tests require a running NATS server:

```bash
# Start NATS with Docker
docker run -d -p 4222:4222 --name nats-server nats:latest

# Run tests
cd transport/nats
go test -v

# Cleanup
docker stop nats-server && docker rm nats-server
```

## 🤝 Contributing

Contributions are welcome! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for details.

### Development Setup

```bash
# Clone repository
git clone https://github.com/yafeiaa/protoc-gen-cloudevents-go.git
cd protoc-gen-cloudevents-go

# Install dependencies
make deps

# Build
make build

# Run tests
make test

# Generate code for examples
make generate
```

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- [CloudEvents](https://cloudevents.io/) - Event specification standard
- [NATS](https://nats.io/) - Cloud-native messaging system
- [Protocol Buffers](https://developers.google.com/protocol-buffers) - Data serialization format

## 📮 Contact

- GitHub: https://github.com/yafeiaa
- Project Homepage: https://github.com/yafeiaa/protoc-gen-cloudevents-go
- Issues: https://github.com/yafeiaa/protoc-gen-cloudevents-go/issues

---

**Star ⭐ this project if you find it helpful!**
