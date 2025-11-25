package nats

import (
	"context"
	"sync"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/nats-io/nats.go"
)

// MockConn 模拟 NATS 连接
type MockConn struct {
	closed    bool
	drained   bool
	mu        sync.Mutex
	subs      []*MockSub
	msgChan   map[string]chan *nats.Msg
}

// NewMockConn 创建一个新的模拟连接
func NewMockConn() *MockConn {
	return &MockConn{
		msgChan: make(map[string]chan *nats.Msg),
	}
}

// IsClosed 检查连接是否关闭
func (c *MockConn) IsClosed() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.closed
}

// IsDrained 检查连接是否已排空
func (c *MockConn) IsDrained() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.drained
}

// Close 关闭连接
func (c *MockConn) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return
	}

	c.closed = true
	
	// 关闭所有订阅
	for _, sub := range c.subs {
		sub.closed = true
	}
	
	// 关闭所有消息通道
	for _, ch := range c.msgChan {
		close(ch)
	}
}

// Drain 排空连接
func (c *MockConn) Drain() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return nats.ErrConnectionClosed
	}

	c.drained = true
	
	// 模拟 Drain 操作
	go func() {
		// 短暂延迟后关闭连接
		c.Close()
	}()
	
	return nil
}

// Publish 发布消息
func (c *MockConn) Publish(subject string, data []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return nats.ErrConnectionClosed
	}

	// 获取主题的消息通道
	ch, ok := c.msgChan[subject]
	if !ok {
		return nil // 如果没有订阅者，忽略消息
	}

	// 发送消息到所有订阅者
	msg := &nats.Msg{
		Subject: subject,
		Data:    data,
	}
	
	select {
	case ch <- msg:
	default:
		// 通道已满，忽略消息
	}
	
	return nil
}

// Subscribe 订阅主题
func (c *MockConn) Subscribe(subject string, handler nats.MsgHandler) (*MockSub, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return nil, nats.ErrConnectionClosed
	}

	// 创建或获取主题的消息通道
	ch, ok := c.msgChan[subject]
	if !ok {
		ch = make(chan *nats.Msg, 100) // 缓冲100条消息
		c.msgChan[subject] = ch
	}

	// 创建模拟订阅
	sub := &MockSub{
		subject: subject,
		conn:    c,
		handler: handler,
		closed:  false,
	}
	
	c.subs = append(c.subs, sub)

	// 启动消息处理 goroutine
	go func() {
		for msg := range ch {
			sub.mu.Lock()
			if sub.closed {
				sub.mu.Unlock()
				break
			}
			sub.mu.Unlock()
			handler(msg)
		}
	}()

	return sub, nil
}

// QueueSubscribe 订阅主题（队列模式）
func (c *MockConn) QueueSubscribe(subject, queue string, handler nats.MsgHandler) (*MockSub, error) {
	// 对于模拟，我们使用相同的逻辑作为普通订阅
	// 实际应用中会实现队列行为的负载均衡
	return c.Subscribe(subject, handler)
}

// MockSub 模拟 NATS 订阅
type MockSub struct {
	subject string
	conn    *MockConn
	handler nats.MsgHandler
	closed  bool
	mu      sync.Mutex
}

// Unsubscribe 取消订阅
func (s *MockSub) Unsubscribe() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil
	}

	s.closed = true
	return nil
}

// MockNATSBus 模拟 NATS 总线
type MockNATSBus struct {
	conn *MockConn
}

// NewMockNATSBus 创建一个新的模拟 NATS 总线
func NewMockNATSBus() *MockNATSBus {
	return &MockNATSBus{
		conn: NewMockConn(),
	}
}

// Publish 发布事件
func (b *MockNATSBus) Publish(ctx context.Context, subject string, event *cloudevents.Event) error {
	// 对于模拟，我们简化实现，直接返回成功
	return nil
}

// Subscribe 订阅事件
func (b *MockNATSBus) Subscribe(ctx context.Context, subject string, handler EventHandler) error {
	// 对于模拟，我们简化实现
	return nil
}

// SubscribeWithHandlerGroup 订阅事件（组模式）
func (b *MockNATSBus) SubscribeWithHandlerGroup(ctx context.Context, subject, group string, handler EventHandler) error {
	// 对于模拟，我们简化实现
	return nil
}

// Close 关闭总线
func (b *MockNATSBus) Close(ctx context.Context) error {
	b.conn.Close()
	return nil
}

// Drain 排空总线
func (b *MockNATSBus) Drain(ctx context.Context) error {
	return b.conn.Drain()
}