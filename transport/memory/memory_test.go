package memory

import (
	"context"
	"sync"
	"testing"
	"time"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewMemoryBus 测试创建内存总线
func TestNewMemoryBus(t *testing.T) {
	bus := NewMemoryBus()
	assert.NotNil(t, bus)
	assert.NotNil(t, bus.subscribers)
	assert.NotNil(t, bus.mu)
}

// TestPublish_SimpleEvent 测试发布简单事件
func TestPublish_SimpleEvent(t *testing.T) {
	bus := NewMemoryBus()
	ctx := context.Background()

	event := cloudevents.NewEvent()
	event.SetID("test-123")
	event.SetType("test.event")
	event.SetSource("test/source")
	event.SetData(cloudevents.ApplicationJSON, map[string]string{"key": "value"})

	err := bus.Publish(ctx, "test.subject", &event)
	assert.NoError(t, err)
}

// TestPublish_NilEvent 测试发布空事件
func TestPublish_NilEvent(t *testing.T) {
	bus := NewMemoryBus()
	ctx := context.Background()

	err := bus.Publish(ctx, "test.subject", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "event is required")
}

// TestPublish_EmptySubject 测试发布空主题
func TestPublish_EmptySubject(t *testing.T) {
	bus := NewMemoryBus()
	ctx := context.Background()

	event := cloudevents.NewEvent()
	err := bus.Publish(ctx, "", &event)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "subject is required")
}

// TestSubscribe_SimpleHandler 测试简单订阅
func TestSubscribe_SimpleHandler(t *testing.T) {
	bus := NewMemoryBus()
	ctx := context.Background()

	received := make(chan *cloudevents.Event, 1)
	handler := func(ctx context.Context, event *cloudevents.Event) error {
		received <- event
		return nil
	}

	err := bus.Subscribe(ctx, "test.subject", handler)
	assert.NoError(t, err)

	// 发布事件
	event := cloudevents.NewEvent()
	event.SetID("test-456")
	event.SetType("test.event")
	event.SetSource("test/source")

	err = bus.Publish(ctx, "test.subject", &event)
	assert.NoError(t, err)

	// 验证接收
	select {
	case evt := <-received:
		assert.Equal(t, "test-456", evt.ID())
		assert.Equal(t, "test.event", evt.Type())
	case <-time.After(1 * time.Second):
		t.Fatal("timeout waiting for event")
	}
}

// TestSubscribe_MultipleHandlers 测试多个订阅者
func TestSubscribe_MultipleHandlers(t *testing.T) {
	bus := NewMemoryBus()
	ctx := context.Background()

	received1 := make(chan *cloudevents.Event, 1)
	received2 := make(chan *cloudevents.Event, 1)

	handler1 := func(ctx context.Context, event *cloudevents.Event) error {
		received1 <- event
		return nil
	}

	handler2 := func(ctx context.Context, event *cloudevents.Event) error {
		received2 <- event
		return nil
	}

	// 订阅同一主题
	err := bus.Subscribe(ctx, "test.multi", handler1)
	assert.NoError(t, err)
	err = bus.Subscribe(ctx, "test.multi", handler2)
	assert.NoError(t, err)

	// 发布事件
	event := cloudevents.NewEvent()
	event.SetID("multi-test")
	event.SetType("test.multi.event")

	err = bus.Publish(ctx, "test.multi", &event)
	assert.NoError(t, err)

	// 验证两个处理器都收到
	timeout := time.After(1 * time.Second)
	var count int
	for count < 2 {
		select {
		case evt := <-received1:
			assert.Equal(t, "multi-test", evt.ID())
			count++
		case evt := <-received2:
			assert.Equal(t, "multi-test", evt.ID())
			count++
		case <-timeout:
			t.Fatalf("timeout, only received %d/2 events", count)
		}
	}
}

// TestSubscribe_WildcardPattern 测试通配符订阅
func TestSubscribe_WildcardPattern(t *testing.T) {
	bus := NewMemoryBus()
	ctx := context.Background()

	received := make(chan *cloudevents.Event, 10)
	handler := func(ctx context.Context, event *cloudevents.Event) error {
		received <- event
		return nil
	}

	// 订阅通配符主题
	err := bus.Subscribe(ctx, "app.*.created", handler)
	assert.NoError(t, err)

	// 发布匹配的事件
	tests := []struct {
		subject     string
		shouldMatch bool
	}{
		{"app.user.created", true},
		{"app.order.created", true},
		{"app.product.created", true},
		{"app.user.deleted", false},
		{"user.created", false},
		{"app.user.created.v2", false},
	}

	for _, tt := range tests {
		event := cloudevents.NewEvent()
		event.SetID(tt.subject)
		event.SetType(tt.subject)

		err = bus.Publish(ctx, tt.subject, &event)
		assert.NoError(t, err)
	}

	// 验证只收到匹配的事件
	time.Sleep(100 * time.Millisecond) // 等待异步处理

	matchedCount := 0
	close(received)
	for evt := range received {
		matchedCount++
		assert.Contains(t, evt.ID(), ".created")
	}

	assert.Equal(t, 3, matchedCount, "should receive 3 matched events")
}

// TestSubscribe_NilHandler 测试空处理器
func TestSubscribe_NilHandler(t *testing.T) {
	bus := NewMemoryBus()
	ctx := context.Background()

	err := bus.Subscribe(ctx, "test.subject", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "handler is required")
}

// TestClose_StopsDelivery 测试关闭总线
func TestClose_StopsDelivery(t *testing.T) {
	bus := NewMemoryBus()
	ctx := context.Background()

	received := make(chan *cloudevents.Event, 1)
	handler := func(ctx context.Context, event *cloudevents.Event) error {
		received <- event
		return nil
	}

	err := bus.Subscribe(ctx, "test.close", handler)
	require.NoError(t, err)

	// 关闭总线
	err = bus.Close(ctx)
	assert.NoError(t, err)

	// 发布事件（应该不会被接收）
	event := cloudevents.NewEvent()
	event.SetID("after-close")

	err = bus.Publish(ctx, "test.close", &event)
	assert.NoError(t, err) // Publish 不会报错

	// 验证没有接收到事件
	select {
	case <-received:
		t.Fatal("should not receive event after close")
	case <-time.After(200 * time.Millisecond):
		// 正常，没有接收到
	}
}

// TestConcurrentPublishSubscribe 测试并发发布和订阅
func TestConcurrentPublishSubscribe(t *testing.T) {
	bus := NewMemoryBus()
	ctx := context.Background()

	const (
		numPublishers      = 10
		numSubscribers     = 5
		eventsPerPublisher = 100
	)

	// 订阅者计数
	var receivedCount int64
	var mu sync.Mutex
	received := make(chan *cloudevents.Event, numPublishers*eventsPerPublisher)

	// 启动多个订阅者
	for i := 0; i < numSubscribers; i++ {
		handler := func(ctx context.Context, event *cloudevents.Event) error {
			mu.Lock()
			receivedCount++
			mu.Unlock()
			received <- event
			return nil
		}
		err := bus.Subscribe(ctx, "concurrent.test", handler)
		require.NoError(t, err)
	}

	// 启动多个发布者
	var wg sync.WaitGroup
	for i := 0; i < numPublishers; i++ {
		wg.Add(1)
		go func(publisherID int) {
			defer wg.Done()
			for j := 0; j < eventsPerPublisher; j++ {
				event := cloudevents.NewEvent()
				event.SetID(time.Now().Format(time.RFC3339Nano))
				event.SetType("concurrent.test")
				event.SetSource("publisher/" + string(rune(publisherID)))

				err := bus.Publish(ctx, "concurrent.test", &event)
				assert.NoError(t, err)
			}
		}(i)
	}

	wg.Wait()

	// 等待所有事件处理完成
	time.Sleep(500 * time.Millisecond)

	// 验证总接收数
	expectedTotal := int64(numPublishers * eventsPerPublisher * numSubscribers)
	mu.Lock()
	actualCount := receivedCount
	mu.Unlock()

	assert.Equal(t, expectedTotal, actualCount,
		"每个订阅者应该收到所有事件")
}

// TestHandlerError_DoesNotAffectOthers 测试处理器错误不影响其他订阅者
func TestHandlerError_DoesNotAffectOthers(t *testing.T) {
	bus := NewMemoryBus()
	ctx := context.Background()

	received1 := make(chan *cloudevents.Event, 1)
	received2 := make(chan *cloudevents.Event, 1)

	// 错误处理器
	handler1 := func(ctx context.Context, event *cloudevents.Event) error {
		return assert.AnError // 返回错误
	}

	// 正常处理器
	handler2 := func(ctx context.Context, event *cloudevents.Event) error {
		received2 <- event
		return nil
	}

	err := bus.Subscribe(ctx, "test.error", handler1)
	require.NoError(t, err)
	err = bus.Subscribe(ctx, "test.error", handler2)
	require.NoError(t, err)

	// 发布事件
	event := cloudevents.NewEvent()
	event.SetID("error-test")

	err = bus.Publish(ctx, "test.error", &event)
	assert.NoError(t, err)

	// 验证正常处理器仍然收到事件
	select {
	case evt := <-received2:
		assert.Equal(t, "error-test", evt.ID())
	case <-time.After(1 * time.Second):
		t.Fatal("handler2 should still receive event despite handler1 error")
	}
}

// BenchmarkPublish 基准测试：发布性能
func BenchmarkPublish(b *testing.B) {
	bus := NewMemoryBus()
	ctx := context.Background()

	event := cloudevents.NewEvent()
	event.SetID("bench-test")
	event.SetType("bench.event")
	event.SetSource("bench/source")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = bus.Publish(ctx, "bench.subject", &event)
	}
}

// BenchmarkSubscribeAndPublish 基准测试：订阅和发布
func BenchmarkSubscribeAndPublish(b *testing.B) {
	bus := NewMemoryBus()
	ctx := context.Background()

	handler := func(ctx context.Context, event *cloudevents.Event) error {
		return nil
	}

	_ = bus.Subscribe(ctx, "bench.subject", handler)

	event := cloudevents.NewEvent()
	event.SetID("bench-test")
	event.SetType("bench.event")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = bus.Publish(ctx, "bench.subject", &event)
	}
}
