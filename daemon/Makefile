.PHONY: all build clean test lint coverage proto install

# 版本信息
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME = $(shell date -u '+%Y-%m-%d_%H:%M:%S')
GIT_COMMIT = $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# 构建参数
BINARY = daemon
CTL_BINARY = daemonctl
BIN_DIR = bin
PKG = github.com/bingooyong/ops-scaffold-framework/daemon
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

build: build-daemon build-ctl

build-daemon:
	@echo "Building $(BINARY)..."
	@mkdir -p $(BIN_DIR)
	$(GOBUILD) -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(BINARY) ./cmd/daemon

build-ctl:
	@echo "Building $(CTL_BINARY)..."
	@mkdir -p $(BIN_DIR)
	$(GOBUILD) -o $(BIN_DIR)/$(CTL_BINARY) ./cmd/daemonctl

build-linux:
	@echo "Building $(BINARY) for Linux..."
	@mkdir -p $(BIN_DIR)
	GOOS=linux GOARCH=amd64 $(GOBUILD) -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(BINARY)-linux-amd64 ./cmd/daemon

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
	@mkdir -p pkg/proto/manager
	protoc --go_out=. --go_opt=paths=source_relative \
	       --go-grpc_out=. --go-grpc_opt=paths=source_relative \
	       pkg/proto/daemon.proto pkg/proto/manager/manager.proto

deps:
	@echo "Downloading dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy

deps-update:
	@echo "Updating dependencies..."
	$(GOGET) -u ./...
	$(GOMOD) tidy

install: build
	@echo "Installing $(BINARY) and $(CTL_BINARY)..."
	sudo install -m 755 $(BIN_DIR)/$(BINARY) /usr/local/bin/$(BINARY)
	sudo install -m 755 $(BIN_DIR)/$(CTL_BINARY) /usr/local/bin/$(CTL_BINARY)
	sudo mkdir -p /etc/daemon
	sudo cp configs/daemon.yaml /etc/daemon/ 2>/dev/null || true
	@echo "Installation complete"

uninstall:
	@echo "Uninstalling $(BINARY)..."
	sudo rm -f /usr/local/bin/$(BINARY)
	@echo "Uninstall complete"

run: build
	@echo "Running $(BINARY)..."
	$(BIN_DIR)/$(BINARY) -config configs/daemon.yaml

docker-build:
	@echo "Building Docker image..."
	docker build -t ops-daemon:$(VERSION) .

help:
	@echo "Available targets:"
	@echo "  build          - Build daemon and daemonctl binaries"
	@echo "  build-daemon   - Build daemon binary only"
	@echo "  build-ctl      - Build daemonctl binary only"
	@echo "  build-linux    - Build binary for Linux"
	@echo "  clean          - Clean build artifacts"
	@echo "  test           - Run tests"
	@echo "  test-coverage  - Run tests with coverage report"
	@echo "  lint           - Run linter"
	@echo "  fmt            - Format code"
	@echo "  vet            - Run go vet"
	@echo "  proto          - Generate protobuf code"
	@echo "  deps           - Download dependencies"
	@echo "  deps-update    - Update dependencies"
	@echo "  install        - Install binaries"
	@echo "  uninstall      - Uninstall binaries"
	@echo "  run            - Build and run"
	@echo "  docker-build   - Build Docker image"
