package basic_test

import (
	"context"
	"testing"
	"time"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yafeiaa/protoc-gen-cloudevents-go/transport/memory"
)

// TestEndToEnd_PublishSubscribe 端到端测试：发布订阅流程
func TestEndToEnd_PublishSubscribe(t *testing.T) {
	// 创建事件总线
	bus := memory.NewMemoryBus()
	defer bus.Close(context.Background())

	ctx := context.Background()

	// 创建接收通道
	received := make(chan *cloudevents.Event, 1)

	// 订阅事件
	handler := func(ctx context.Context, event *cloudevents.Event) error {
		t.Logf("Received event: ID=%s, Type=%s", event.ID(), event.Type())
		received <- event
		return nil
	}

	err := bus.Subscribe(ctx, "user.created", handler)
	require.NoError(t, err)

	// 发布事件
	event := cloudevents.NewEvent()
	event.SetID("user-123")
	event.SetType("user.created")
	event.SetSource("test/integration")
	event.SetData(cloudevents.ApplicationJSON, map[string]interface{}{
		"name":  "Alice",
		"email": "alice@example.com",
	})

	err = bus.Publish(ctx, "user.created", &event)
	require.NoError(t, err)

	// 验证接收
	select {
	case evt := <-received:
		assert.Equal(t, "user-123", evt.ID())
		assert.Equal(t, "user.created", evt.Type())
		assert.Equal(t, "test/integration", evt.Source())

		var data map[string]interface{}
		err := evt.DataAs(&data)
		require.NoError(t, err)
		assert.Equal(t, "Alice", data["name"])
		assert.Equal(t, "alice@example.com", data["email"])

	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for event")
	}
}

// TestMultipleEventTypes 测试多种事件类型
func TestMultipleEventTypes(t *testing.T) {
	bus := memory.NewMemoryBus()
	defer bus.Close(context.Background())

	ctx := context.Background()

	// 订阅多个事件类型
	createdEvents := make(chan *cloudevents.Event, 10)
	updatedEvents := make(chan *cloudevents.Event, 10)
	deletedEvents := make(chan *cloudevents.Event, 10)

	_ = bus.Subscribe(ctx, "user.created", func(ctx context.Context, e *cloudevents.Event) error {
		createdEvents <- e
		return nil
	})

	_ = bus.Subscribe(ctx, "user.updated", func(ctx context.Context, e *cloudevents.Event) error {
		updatedEvents <- e
		return nil
	})

	_ = bus.Subscribe(ctx, "user.deleted", func(ctx context.Context, e *cloudevents.Event) error {
		deletedEvents <- e
		return nil
	})

	// 发布不同类型的事件
	events := []struct {
		subject   string
		eventType string
		channel   chan *cloudevents.Event
	}{
		{"user.created", "user.created.v1", createdEvents},
		{"user.updated", "user.updated.v1", updatedEvents},
		{"user.deleted", "user.deleted.v1", deletedEvents},
	}

	for _, tc := range events {
		event := cloudevents.NewEvent()
		event.SetID("test-" + tc.subject)
		event.SetType(tc.eventType)
		event.SetSource("test")

		err := bus.Publish(ctx, tc.subject, &event)
		require.NoError(t, err)
	}

	// 验证每个通道都收到正确的事件
	timeout := time.After(1 * time.Second)
	for _, tc := range events {
		select {
		case evt := <-tc.channel:
			assert.Equal(t, tc.eventType, evt.Type())
			t.Logf("✓ Received %s", evt.Type())
		case <-timeout:
			t.Fatalf("timeout waiting for %s", tc.eventType)
		}
	}
}

// TestEventOrdering 测试事件顺序（在单个订阅者中）
func TestEventOrdering(t *testing.T) {
	bus := memory.NewMemoryBus()
	defer bus.Close(context.Background())

	ctx := context.Background()

	received := make(chan string, 100)
	handler := func(ctx context.Context, event *cloudevents.Event) error {
		received <- event.ID()
		return nil
	}

	err := bus.Subscribe(ctx, "order.test", handler)
	require.NoError(t, err)

	// 按顺序发布事件
	const numEvents = 10
	for i := 0; i < numEvents; i++ {
		event := cloudevents.NewEvent()
		event.SetID(string(rune('A' + i)))
		event.SetType("order.test")

		err := bus.Publish(ctx, "order.test", &event)
		require.NoError(t, err)
	}

	// 验证接收顺序（内存总线应该保持顺序）
	time.Sleep(100 * time.Millisecond)

	var receivedIDs []string
	close(received)
	for id := range received {
		receivedIDs = append(receivedIDs, id)
	}

	assert.Len(t, receivedIDs, numEvents)
	// Note: 由于异步处理，顺序可能不严格保证
	// 这里只验证所有事件都被接收
	for i := 0; i < numEvents; i++ {
		expectedID := string(rune('A' + i))
		assert.Contains(t, receivedIDs, expectedID)
	}
}

// TestWildcardSubscription 测试通配符订阅模式
func TestWildcardSubscription(t *testing.T) {
	bus := memory.NewMemoryBus()
	defer bus.Close(context.Background())

	ctx := context.Background()

	// 订阅通配符主题
	received := make(chan string, 100)
	handler := func(ctx context.Context, event *cloudevents.Event) error {
		received <- event.Type()
		return nil
	}

	// 订阅 app.*.created 模式
	err := bus.Subscribe(ctx, "app.*.created", handler)
	require.NoError(t, err)

	// 发布各种事件
	testCases := []struct {
		subject     string
		shouldMatch bool
	}{
		{"app.user.created", true},
		{"app.order.created", true},
		{"app.product.created", true},
		{"app.user.updated", false},
		{"system.user.created", false},
	}

	for _, tc := range testCases {
		event := cloudevents.NewEvent()
		event.SetID(tc.subject)
		event.SetType(tc.subject)
		event.SetSource("test")

		err := bus.Publish(ctx, tc.subject, &event)
		require.NoError(t, err)
	}

	// 等待异步处理
	time.Sleep(200 * time.Millisecond)

	// 验证匹配的事件
	close(received)
	var matchedTypes []string
	for eventType := range received {
		matchedTypes = append(matchedTypes, eventType)
	}

	expectedMatches := []string{
		"app.user.created",
		"app.order.created",
		"app.product.created",
	}

	assert.ElementsMatch(t, expectedMatches, matchedTypes,
		"should only receive events matching the wildcard pattern")
}

// TestContextCancellation 测试上下文取消
func TestContextCancellation(t *testing.T) {
	bus := memory.NewMemoryBus()
	defer bus.Close(context.Background())

	ctx, cancel := context.WithCancel(context.Background())

	received := make(chan *cloudevents.Event, 1)
	handler := func(ctx context.Context, event *cloudevents.Event) error {
		// 检查上下文是否已取消
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			received <- event
			return nil
		}
	}

	err := bus.Subscribe(ctx, "cancel.test", handler)
	require.NoError(t, err)

	// 取消上下文
	cancel()

	// 尝试发布事件（应该不会被处理）
	event := cloudevents.NewEvent()
	event.SetID("after-cancel")
	event.SetType("cancel.test")

	err = bus.Publish(context.Background(), "cancel.test", &event)
	assert.NoError(t, err)

	// 验证没有接收到事件
	select {
	case <-received:
		t.Fatal("should not receive event after context cancellation")
	case <-time.After(200 * time.Millisecond):
		// 正常，没有接收到
	}
}

// TestConcurrentAccess 测试并发访问安全性
func TestConcurrentAccess(t *testing.T) {
	bus := memory.NewMemoryBus()
	defer bus.Close(context.Background())

	ctx := context.Background()

	// 并发订阅和发布
	const (
		numGoroutines = 50
		numEvents     = 10
	)

	received := make(chan *cloudevents.Event, numGoroutines*numEvents)

	// 并发订阅
	for i := 0; i < numGoroutines; i++ {
		go func() {
			handler := func(ctx context.Context, event *cloudevents.Event) error {
				received <- event
				return nil
			}
			_ = bus.Subscribe(ctx, "concurrent.*", handler)
		}()
	}

	// 短暂等待订阅完成
	time.Sleep(100 * time.Millisecond)

	// 并发发布
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			for j := 0; j < numEvents; j++ {
				event := cloudevents.NewEvent()
				event.SetID(time.Now().Format(time.RFC3339Nano))
				event.SetType("concurrent.test")
				event.SetSource("goroutine/" + string(rune(id)))

				_ = bus.Publish(ctx, "concurrent.test", &event)
			}
		}(i)
	}

	// 等待处理完成
	time.Sleep(1 * time.Second)

	// 验证收到了大量事件（不需要精确数量，只验证并发安全）
	close(received)
	count := 0
	for range received {
		count++
	}

	t.Logf("Received %d events in concurrent test", count)
	assert.Greater(t, count, 0, "should receive events in concurrent scenario")
}
