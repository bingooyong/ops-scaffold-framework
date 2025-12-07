.PHONY: build run clean test install

# 默认目标
all: build

# 构建
build:
	@echo "Building agent..."
	@go build -o bin/agent ./cmd/agent

# 运行（使用默认配置）
run: build
	@echo "Running agent..."
	@./bin/agent -config ./configs/agent.yaml

# 运行（使用自定义配置）
run-config: build
	@echo "Running agent with custom config..."
	@./bin/agent -config $(CONFIG)

# 测试
test:
	@echo "Running tests..."
	@go test -v -race -cover ./...

# 清理
clean:
	@echo "Cleaning..."
	@rm -rf bin/
	@go clean

# 安装依赖
deps:
	@echo "Installing dependencies..."
	@go mod download
	@go mod tidy

# 格式化代码
fmt:
	@echo "Formatting code..."
	@go fmt ./...

# 静态检查
lint:
	@echo "Running linter..."
	@golangci-lint run

# 安装到系统（需要 root 权限）
install: build
	@echo "Installing agent to /usr/local/bin..."
	@sudo cp bin/agent /usr/local/bin/
	@echo "Creating config directory..."
	@sudo mkdir -p /etc/agent
	@sudo cp configs/agent.yaml /etc/agent/
	@echo "Agent installed successfully"

# 帮助信息
help:
	@echo "Available targets:"
	@echo "  build         - Build the agent binary"
	@echo "  run           - Run the agent with default config"
	@echo "  run-config    - Run the agent with custom config (CONFIG=path/to/config.yaml)"
	@echo "  test          - Run tests"
	@echo "  clean         - Clean build artifacts"
	@echo "  deps          - Install dependencies"
	@echo "  fmt           - Format code"
	@echo "  lint          - Run linter"
	@echo "  install       - Install to system (requires root)"
	@echo "  start-all     - Start all agents using script"
	@echo "  stop-all      - Stop all agents using script"
	@echo "  status        - Check agents status"
	@echo "  test-int      - Run integration tests"
	@echo "  help          - Show this help message"

# 使用脚本启动所有 Agent
start-all: build
	@echo "Starting all agents..."
	@./scripts/agent.sh start

# 停止所有 Agent
stop-all:
	@echo "Stopping all agents..."
	@./scripts/agent.sh stop

# 检查 Agent 状态
status:
	@./scripts/agent.sh status

# 运行集成测试
test-int:
	@echo "Running integration tests..."
	@./scripts/test-integration.sh
