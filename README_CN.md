# protoc-gen-cloudevents-go

[English](./README.md) | 简体中文

**从 Protobuf 定义自动生成类型安全的 CloudEvents 发布/订阅代码**

一个用于事件驱动架构的 Protobuf 代码生成器,帮你告别样板代码,拥抱类型安全!

## ✨ 特性

- 🚀 **零样板代码** - 从 Protobuf 定义自动生成发布/订阅函数
- 🔒 **类型安全** - 编译时检查,避免运行时错误
- ☁️ **CloudEvents 标准** - 完全兼容 CloudEvents 规范
- 📡 **支持多种模式** - 广播模式 + 处理器组(负载均衡)模式
- 🎯 **运行时灵活** - source/subject/group 等参数可动态指定
- 🔌 **传输无关** - 支持 NATS、Kafka、HTTP 等多种传输协议

## 🎬 快速开始

### 安装

```bash
go install github.com/yafeiaa/protoc-gen-cloudevents-go/cmd/protoc-gen-cloudevents@latest
```

### 定义事件

在 `events.proto` 中定义事件:

```protobuf
syntax = "proto3";
package myapp.events;

import "google/protobuf/descriptor.proto";
import "cloudevents/event_meta.proto";

// 用户注册事件
message UserRegisteredPayload {
  option (cloudevents.event_meta) = {
    event_type: "myapp.user.registered"
    description: "用户注册成功"
  };
  
  string user_id = 1;
  string email = 2;
}
```

### 生成代码

```bash
protoc \
  -I . \
  -I ./third_party \
  --go_out=. \
  --cloudevents_out=. \
  events.proto
```

### 发布事件

```go
import "your-module/events"

// 发布用户注册事件
events.PublishUserRegistered(ctx, bus,
    &events.UserRegisteredPayload{
        UserId: "user-123",
        Email:  "user@example.com",
    },
    events.WithSource("myapp/api-server"),  // 必填
)
```

### 订阅事件

```go
// 广播模式 - 所有订阅者都收到事件
events.SubscribeUserRegistered(ctx, bus,
    func(ctx context.Context, payload *events.UserRegisteredPayload) error {
        log.Printf("用户注册: %s", payload.Email)
        return nil
    })

// 处理器组模式 - 同组内竞争消费(负载均衡)
events.SubscribeUserRegisteredWithGroup(ctx, bus, "email-sender",
    func(ctx context.Context, payload *events.UserRegisteredPayload) error {
        return sendWelcomeEmail(payload.Email)
    })
```

## 📖 文档

- [完整英文文档](./README.md) - 详细使用说明和 API 文档
- [示例代码](./examples) - 完整的使用示例
- [贡献指南](./CONTRIBUTING.md) - 如何参与开发

## 🏗️ 为什么选择 protoc-gen-cloudevents?

### 传统方式的痛点

```go
// ❌ 字符串常量,容易拼写错误
bus.Publish("user.registerd", data)  // 注意拼写错误!

// ❌ 类型不安全
payload := map[string]interface{}{
    "user_id": 123,  // 应该是 string
}

// ❌ 重复代码
func PublishUserRegistered(...) { /* 手写 */ }
func PublishOrderCreated(...) { /* 复制粘贴 */ }
func PublishPaymentCompleted(...) { /* 再复制粘贴 */ }
```

### 使用 protoc-gen-cloudevents

```go
// ✅ 类型安全,编译时检查
events.PublishUserRegistered(ctx, bus,
    &events.UserRegisteredPayload{
        UserId: "user-123",  // 正确的类型
        Email:  "user@example.com",
    },
    events.WithSource("api-server"),
)

// ✅ 零样板代码,自动生成
// 只需定义 proto,代码自动生成!
```

## 🎯 使用场景

### 微服务异步通信

```
API Server ──> User Registered Event ──┬──> Analytics Service
                                       ├──> Email Service  
                                       └──> Audit Service
```

### 事件溯源

```
Commands ──> Aggregate ──> Events ──> Event Store
                                   ──> Projections
```

### CQRS 架构

```
Write Side: Commands ──> Event Store ──> Events
                                           ↓
Read Side:                          Read Models
```

## 🔌 传输适配器

### 内存 (测试用)

```go
import "github.com/yafeiaa/protoc-gen-cloudevents/transport/memory"

bus := memory.NewMemoryBus()
```

### NATS (计划中)

```go
import "github.com/yafeiaa/protoc-gen-cloudevents/transport/nats"

bus, err := nats.NewNATSBus("nats://localhost:4222")
```

### Kafka (计划中)

```go
import "github.com/yafeiaa/protoc-gen-cloudevents/transport/kafka"

bus, err := kafka.NewKafkaBus([]string{"localhost:9092"})
```

## 💡 核心概念

### EventMeta 选项

```protobuf
message EventMeta {
  string event_type = 1;    // 全局唯一事件类型 (必填)
  string description = 2;   // 事件描述 (可选)
}
```

### 发布选项

```go
// source - 事件来源 (必填)
events.WithSource("myapp/api-server")

// subject - 自定义 NATS subject (可选)
events.WithSubject("custom.subject")

// extension - 扩展字段 (可选)
events.WithExtension("trace_id", traceID)
```

### 订阅模式

**广播模式**: 所有订阅者都收到事件

```go
events.SubscribeUserRegistered(ctx, bus, handler)
```

**处理器组模式**: 同组内竞争消费,实现负载均衡

```go
events.SubscribeUserRegisteredWithGroup(ctx, bus, "worker-group", handler)
```

## 🤝 贡献

欢迎贡献! 请查看 [贡献指南](./CONTRIBUTING.md)。

## 📄 许可证

MIT License - 详见 [LICENSE](LICENSE)

## 🙏 致谢

- [CloudEvents](https://cloudevents.io/) - 事件规范标准
- [NATS](https://nats.io/) - 云原生消息系统
- [Protocol Buffers](https://developers.google.com/protocol-buffers) - 数据序列化

## 📮 联系方式

- GitHub Issues: https://github.com/yafeiaa/protoc-gen-cloudevents/issues
- GitHub Discussions: https://github.com/yafeiaa/protoc-gen-cloudevents/discussions

---

如果这个项目对你有帮助,请给个 Star ⭐
