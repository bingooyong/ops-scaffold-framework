.PHONY: all build clean test lint coverage run docker-build help

# 版本信息
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME = $(shell date -u '+%Y-%m-%d_%H:%M:%S')
GIT_COMMIT = $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# 构建参数
BINARY = manager
BIN_DIR = bin
PKG = github.com/bingooyong/ops-scaffold-framework/manager
LDFLAGS = -X '$(PKG)/internal/version.Version=$(VERSION)' \
          -X '$(PKG)/internal/version.BuildTime=$(BUILD_TIME)' \
          -X '$(PKG)/internal/version.GitCommit=$(GIT_COMMIT)'

# Go参数
GOCMD = go
GOBUILD = $(GOCMD) build
GOCLEAN = $(GOCMD) clean
GOTEST = $(GOCMD) test
GOGET = $(GOCMD) get
GOMOD = $(GOCMD) mod

all: lint test build

build:
	@echo "Building $(BINARY)..."
	@mkdir -p $(BIN_DIR)
	$(GOBUILD) -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(BINARY) ./cmd/manager

build-linux:
	@echo "Building $(BINARY) for Linux..."
	@mkdir -p $(BIN_DIR)
	GOOS=linux GOARCH=amd64 $(GOBUILD) -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(BINARY)-linux-amd64 ./cmd/manager

clean:
	@echo "Cleaning..."
	$(GOCLEAN)
	@rm -rf $(BIN_DIR)
	@rm -f coverage.out coverage.html

test:
	@echo "Running tests..."
	$(GOTEST) -v -race -timeout 300s ./...

test-coverage:
	@echo "Running tests with coverage..."
	$(GOTEST) -v -race -timeout 300s -coverprofile=coverage.out -covermode=atomic ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

lint:
	@echo "Running linter..."
	@which golangci-lint > /dev/null || (echo "golangci-lint not installed" && exit 1)
	golangci-lint run ./...

fmt:
	@echo "Formatting code..."
	$(GOCMD) fmt ./...
	gofmt -s -w .

vet:
	@echo "Running go vet..."
	$(GOCMD) vet ./...

proto:
	@echo "Generating protobuf code..."
	@which protoc > /dev/null || (echo "protoc not installed" && exit 1)
	@mkdir -p pkg/proto/daemon
	# 使用 daemon/daemon.proto 路径避免与 daemon 模块的 pkg/proto/daemon.proto 冲突
	protoc --go_out=. --go_opt=paths=source_relative \
	       --go-grpc_out=. --go-grpc_opt=paths=source_relative \
	       pkg/proto/daemon/daemon.proto

deps:
	@echo "Downloading dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy

deps-update:
	@echo "Updating dependencies..."
	$(GOGET) -u ./...
	$(GOMOD) tidy

run: build
	@echo "Running $(BINARY)..."
	$(BIN_DIR)/$(BINARY) -config configs/manager.yaml

run-dev: build
	@echo "Running $(BINARY) in development mode..."
	$(BIN_DIR)/$(BINARY) -config configs/manager.dev.yaml

docker-build:
	@echo "Building Docker image..."
	docker build -t ops-manager:$(VERSION) .

# 数据库迁移
migrate-up:
	@echo "Running database migrations..."
	@go run cmd/migrate/main.go up

migrate-down:
	@echo "Rolling back database migrations..."
	@go run cmd/migrate/main.go down

migrate-create:
	@echo "Creating new migration..."
	@go run cmd/migrate/main.go create $(name)

help:
	@echo "Available targets:"
	@echo "  build          - Build the binary"
	@echo "  build-linux    - Build binary for Linux"
	@echo "  clean          - Clean build artifacts"
	@echo "  test           - Run tests"
	@echo "  test-coverage  - Run tests with coverage report"
	@echo "  lint           - Run linter"
	@echo "  fmt            - Format code"
	@echo "  vet            - Run go vet"
	@echo "  deps           - Download dependencies"
	@echo "  deps-update    - Update dependencies"
	@echo "  run            - Build and run"
	@echo "  run-dev        - Build and run in development mode"
	@echo "  docker-build   - Build Docker image"
	@echo "  migrate-up     - Run database migrations"
	@echo "  migrate-down   - Rollback database migrations"
	@echo "  migrate-create - Create new migration (use name=<migration_name>)"
