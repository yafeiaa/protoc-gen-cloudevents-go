.PHONY: all build install test clean generate examples examples-nats

# Variables
BINARY_NAME=protoc-gen-cloudevents
BUILD_DIR=bin
CMD_DIR=cmd/$(BINARY_NAME)

all: build

# Build code generator
build:
	@echo "🔨 Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	@go build -o $(BUILD_DIR)/$(BINARY_NAME) ./$(CMD_DIR)
	@echo "✅ Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

# Install to GOPATH/bin
install:
	@echo "📦 Installing $(BINARY_NAME)..."
	@go install ./$(CMD_DIR)
	@echo "✅ Installed to $(shell go env GOPATH)/bin/$(BINARY_NAME)"

# Run all tests
test:
	@echo "🧪 Running all tests..."
	@go test -v -race -coverprofile=coverage.out ./...
	@echo "✅ Tests complete"

# Run unit tests
test-unit:
	@echo "🧪 Running unit tests..."
	@go test -v -race -short ./transport/...
	@echo "✅ Unit tests complete"

# Run integration tests
test-integration:
	@echo "🧪 Running integration tests..."
	@go test -v -race ./examples/basic/...
	@echo "✅ Integration tests complete"

# Run benchmarks
bench:
	@echo "⚡ Running benchmarks..."
	@go test -bench=. -benchmem ./transport/...
	@echo "✅ Benchmarks complete"

# View test coverage
coverage: test
	@go tool cover -html=coverage.out

# Clean build artifacts
clean:
	@echo "🧹 Cleaning..."
	@rm -rf $(BUILD_DIR)
	@rm -f coverage.out
	@find . -name "*.pb.go" -type f -delete
	@find . -name "*_events.pb.go" -type f -delete
	@echo "✅ Clean complete"

# Generate proto files
generate: build
	@echo "⚙️  Generating code..."
	@PATH=$(PWD)/$(BUILD_DIR):$$PATH ./scripts/generate.sh
	@echo "✅ Code generation complete"

# Generate and run basic examples
examples: generate
	@echo ""
	@echo "📝 Running basic examples..."
	@cd examples/basic/publisher && go run main.go
	@echo ""
	@cd examples/basic/subscriber && go run main.go

# Run NATS examples (requires NATS server)
examples-nats: generate
	@echo ""
	@echo "📝 Running NATS examples..."
	@echo "⚠️  Make sure NATS server is running: docker run -d -p 4222:4222 nats:latest"
	@echo ""
	@echo "Starting subscriber (Ctrl+C to stop)..."
	@cd examples/nats/subscriber && go run main.go

# 格式化代码
fmt:
	@echo "🎨 格式化代码..."
	@go fmt ./...
	@echo "✅ 格式化完成"

# 代码检查
lint:
	@echo "🔍 代码检查..."
	@golint ./...
	@go vet ./...
	@echo "✅ 检查完成"

# 下载依赖
deps:
	@echo "📥 下载依赖..."
	@go mod download
	@go mod tidy
	@echo "✅ 依赖下载完成"

# Help information
help:
	@echo "Available commands:"
	@echo "  make build            - Build code generator"
	@echo "  make install          - Install to GOPATH/bin"
	@echo "  make test             - Run all tests"
	@echo "  make test-unit        - Run unit tests"
	@echo "  make test-integration - Run integration tests"
	@echo "  make bench            - Run benchmarks"
	@echo "  make coverage         - View test coverage"
	@echo "  make clean            - Clean build artifacts"
	@echo "  make generate         - Generate example code"
	@echo "  make examples         - Generate and run basic examples"
	@echo "  make examples-nats    - Run NATS examples"
	@echo "  make fmt              - Format code"
	@echo "  make lint             - Code linting"
	@echo "  make deps             - Download dependencies"
