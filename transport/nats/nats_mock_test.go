package nats

import (
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/assert"
)

// TestMockConnBasic 测试 MockConn 的基本功能
func TestMockConnBasic(t *testing.T) {
	conn := NewMockConn()
	assert.NotNil(t, conn)
	assert.False(t, conn.IsClosed())
	assert.False(t, conn.IsDrained())

	// 测试关闭
	conn.Close()
	assert.True(t, conn.IsClosed())

	// 测试排空
	conn = NewMockConn()
	err := conn.Drain()
	assert.NoError(t, err)
	
	// 给排空一些时间
	time.Sleep(100 * time.Millisecond)
	assert.True(t, conn.IsClosed())
}

// TestMockConnPublishSubscribe 测试 MockConn 的发布订阅功能
func TestMockConnPublishSubscribe(t *testing.T) {
	conn := NewMockConn()
	defer conn.Close()

	subject := "test.subject"
	messageReceived := false

	handler := func(msg *nats.Msg) {
		messageReceived = true
		assert.Equal(t, subject, msg.Subject)
		assert.Equal(t, []byte("test data"), msg.Data)
	}

	// 订阅
	sub, err := conn.Subscribe(subject, handler)
	assert.NoError(t, err)
	assert.NotNil(t, sub)

	// 发布消息
	err = conn.Publish(subject, []byte("test data"))
	assert.NoError(t, err)

	// 等待消息处理
	time.Sleep(50 * time.Millisecond)
	assert.True(t, messageReceived, "消息应该被接收")

	// 取消订阅
	err = sub.Unsubscribe()
	assert.NoError(t, err)
}

// TestMockConnQueueSubscribe 测试 MockConn 的队列订阅功能
func TestMockConnQueueSubscribe(t *testing.T) {
	conn := NewMockConn()
	defer conn.Close()

	subject := "test.queue"
	queue := "test-group"
	messageReceived := false

	handler := func(msg *nats.Msg) {
		messageReceived = true
		assert.Equal(t, subject, msg.Subject)
		assert.Equal(t, []byte("queue test data"), msg.Data)
	}

	// 队列订阅
	sub, err := conn.QueueSubscribe(subject, queue, handler)
	assert.NoError(t, err)
	assert.NotNil(t, sub)

	// 发布消息
	err = conn.Publish(subject, []byte("queue test data"))
	assert.NoError(t, err)

	// 等待消息处理
	time.Sleep(50 * time.Millisecond)
	assert.True(t, messageReceived, "消息应该被接收")
}

// TestMockConnClosedOperations 测试已关闭连接的操作
func TestMockConnClosedOperations(t *testing.T) {
	conn := NewMockConn()
	
	// 关闭连接
	conn.Close()
	assert.True(t, conn.IsClosed())

	// 尝试发布消息
	err := conn.Publish("test.subject", []byte("test"))
	assert.Error(t, err)

	// 尝试订阅
	_, err = conn.Subscribe("test.subject", func(msg *nats.Msg) {})
	assert.Error(t, err)

	// 尝试排空
	err = conn.Drain()
	assert.Error(t, err)
}

// TestMockNATSBusBasic 测试模拟 NATS 总线的基本功能
func TestMockNATSBusBasic(t *testing.T) {
	bus := NewMockNATSBus()
	assert.NotNil(t, bus)
	assert.False(t, bus.conn.IsClosed())
}

// TestMockNATSBusDrain 测试模拟总线的排空功能
func TestMockNATSBusDrain(t *testing.T) {
	bus := NewMockNATSBus()
	err := bus.Drain(nil)
	assert.NoError(t, err)
	
	// 给排空一些时间
	time.Sleep(100 * time.Millisecond)
	assert.True(t, bus.conn.IsClosed())
}

// TestMockNATSBusClose 测试模拟总线的关闭功能
func TestMockNATSBusClose(t *testing.T) {
	bus := NewMockNATSBus()
	err := bus.Close(nil)
	assert.NoError(t, err)
	assert.True(t, bus.conn.IsClosed())
}