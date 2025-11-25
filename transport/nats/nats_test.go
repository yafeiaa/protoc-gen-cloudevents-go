package nats

import (
	"context"
	"testing"
	"time"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Note: These tests require a running NATS server
// To run tests: docker run -d -p 4222:4222 nats:latest

const testNATSURL = "nats://localhost:4222"

func TestNewNATSBus(t *testing.T) {
	bus, err := NewNATSBus(Config{URL: testNATSURL})
	if err != nil {
		t.Skipf("NATS server not available: %v", err)
		return
	}
	defer bus.Close(context.Background())

	assert.NotNil(t, bus)
	assert.NotNil(t, bus.conn)
	assert.False(t, bus.conn.IsClosed())
}

func TestNewNATSBus_DefaultURL(t *testing.T) {
	bus, err := NewNATSBus(Config{})
	if err != nil {
		t.Skipf("NATS server not available: %v", err)
		return
	}
	defer bus.Close(context.Background())

	assert.NotNil(t, bus)
}

func TestPublishSubscribe_BroadcastMode(t *testing.T) {
	ctx := context.Background()
	bus, err := NewNATSBus(Config{URL: testNATSURL})
	if err != nil {
		t.Skipf("NATS server not available: %v", err)
		return
	}
	defer bus.Close(ctx)

	subject := "test.event." + uuid.New().String()
	receivedCh := make(chan *cloudevents.Event, 2)

	// Subscribe two handlers (broadcast mode)
	handler := func(ctx context.Context, event *cloudevents.Event) error {
		receivedCh <- event
		return nil
	}

	require.NoError(t, bus.Subscribe(ctx, subject, handler))
	require.NoError(t, bus.Subscribe(ctx, subject, handler))

	// Give subscriptions time to be ready
	time.Sleep(100 * time.Millisecond)

	// Publish event
	event := cloudevents.NewEvent()
	event.SetID(uuid.New().String())
	event.SetType("test.event")
	event.SetSource("test")
	event.SetData(cloudevents.ApplicationJSON, map[string]string{"key": "value"})

	require.NoError(t, bus.Publish(ctx, subject, &event))

	// Both handlers should receive the event
	select {
	case e := <-receivedCh:
		assert.Equal(t, event.ID(), e.ID())
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for first event")
	}

	select {
	case e := <-receivedCh:
		assert.Equal(t, event.ID(), e.ID())
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for second event")
	}
}

func TestPublishSubscribe_HandlerGroupMode(t *testing.T) {
	ctx := context.Background()
	bus, err := NewNATSBus(Config{URL: testNATSURL})
	if err != nil {
		t.Skipf("NATS server not available: %v", err)
		return
	}
	defer bus.Close(ctx)

	subject := "test.group." + uuid.New().String()
	group := "test-group"
	receivedCount := make(chan int, 10)

	// Subscribe two handlers in the same group
	handler := func(ctx context.Context, event *cloudevents.Event) error {
		receivedCount <- 1
		return nil
	}

	require.NoError(t, bus.SubscribeWithHandlerGroup(ctx, subject, group, handler))
	require.NoError(t, bus.SubscribeWithHandlerGroup(ctx, subject, group, handler))

	time.Sleep(100 * time.Millisecond)

	// Publish multiple events
	eventCount := 10
	for i := 0; i < eventCount; i++ {
		event := cloudevents.NewEvent()
		event.SetID(uuid.New().String())
		event.SetType("test.event")
		event.SetSource("test")
		require.NoError(t, bus.Publish(ctx, subject, &event))
	}

	// Collect received events
	total := 0
	timeout := time.After(3 * time.Second)
	for total < eventCount {
		select {
		case <-receivedCount:
			total++
		case <-timeout:
			t.Fatalf("timeout: received %d out of %d events", total, eventCount)
		}
	}

	assert.Equal(t, eventCount, total)
}

func TestPublishSubscribe_MultipleGroups(t *testing.T) {
	ctx := context.Background()
	bus, err := NewNATSBus(Config{URL: testNATSURL})
	if err != nil {
		t.Skipf("NATS server not available: %v", err)
		return
	}
	defer bus.Close(ctx)

	subject := "test.multigroup." + uuid.New().String()
	group1Ch := make(chan *cloudevents.Event, 1)
	group2Ch := make(chan *cloudevents.Event, 1)

	// Subscribe to different groups
	handler1 := func(ctx context.Context, event *cloudevents.Event) error {
		group1Ch <- event
		return nil
	}
	handler2 := func(ctx context.Context, event *cloudevents.Event) error {
		group2Ch <- event
		return nil
	}

	require.NoError(t, bus.SubscribeWithHandlerGroup(ctx, subject, "group1", handler1))
	require.NoError(t, bus.SubscribeWithHandlerGroup(ctx, subject, "group2", handler2))

	time.Sleep(100 * time.Millisecond)

	// Publish one event
	event := cloudevents.NewEvent()
	event.SetID(uuid.New().String())
	event.SetType("test.event")
	event.SetSource("test")

	require.NoError(t, bus.Publish(ctx, subject, &event))

	// Both groups should receive the event
	select {
	case e := <-group1Ch:
		assert.Equal(t, event.ID(), e.ID())
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for group1")
	}

	select {
	case e := <-group2Ch:
		assert.Equal(t, event.ID(), e.ID())
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for group2")
	}
}

func TestClose(t *testing.T) {
	ctx := context.Background()
	bus, err := NewNATSBus(Config{URL: testNATSURL})
	if err != nil {
		t.Skipf("NATS server not available: %v", err)
		return
	}

	subject := "test.close." + uuid.New().String()
	receivedCh := make(chan *cloudevents.Event, 1)

	handler := func(ctx context.Context, event *cloudevents.Event) error {
		receivedCh <- event
		return nil
	}

	require.NoError(t, bus.Subscribe(ctx, subject, handler))
	time.Sleep(100 * time.Millisecond)

	// Close the bus
	require.NoError(t, bus.Close(ctx))

	// Publishing should fail
	event := cloudevents.NewEvent()
	event.SetID(uuid.New().String())
	err = bus.Publish(ctx, subject, &event)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "connection is closed")
}

func TestSubscribe_EmptyGroup(t *testing.T) {
	ctx := context.Background()
	bus, err := NewNATSBus(Config{URL: testNATSURL})
	if err != nil {
		t.Skipf("NATS server not available: %v", err)
		return
	}
	defer bus.Close(ctx)

	handler := func(ctx context.Context, event *cloudevents.Event) error {
		return nil
	}

	err = bus.SubscribeWithHandlerGroup(ctx, "test.subject", "", handler)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "group name is required")
}

func TestDrain(t *testing.T) {
	ctx := context.Background()
	bus, err := NewNATSBus(Config{URL: testNATSURL})
	if err != nil {
		t.Skipf("NATS server not available: %v", err)
		return
	}

	err = bus.Drain(ctx)
	assert.NoError(t, err)
	
	// Give some time for the connection to close
	time.Sleep(100 * time.Millisecond)
	assert.True(t, bus.conn.IsClosed())
}

// Benchmark tests
func BenchmarkPublish(b *testing.B) {
	ctx := context.Background()
	bus, err := NewNATSBus(Config{URL: testNATSURL})
	if err != nil {
		b.Skipf("NATS server not available: %v", err)
		return
	}
	defer bus.Close(ctx)

	event := cloudevents.NewEvent()
	event.SetID(uuid.New().String())
	event.SetType("test.event")
	event.SetSource("test")
	event.SetData(cloudevents.ApplicationJSON, map[string]string{"key": "value"})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = bus.Publish(ctx, "bench.subject", &event)
	}
}

func BenchmarkSubscribe(b *testing.B) {
	ctx := context.Background()
	bus, err := NewNATSBus(Config{URL: testNATSURL})
	if err != nil {
		b.Skipf("NATS server not available: %v", err)
		return
	}
	defer bus.Close(ctx)

	received := 0
	handler := func(ctx context.Context, event *cloudevents.Event) error {
		received++
		return nil
	}

	subject := "bench.subscribe"
	bus.Subscribe(ctx, subject, handler)
	time.Sleep(100 * time.Millisecond)

	event := cloudevents.NewEvent()
	event.SetID(uuid.New().String())
	event.SetType("test.event")
	event.SetSource("test")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = bus.Publish(ctx, subject, &event)
	}
	time.Sleep(500 * time.Millisecond)
}
