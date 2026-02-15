# Mujibot Makefile
# 支持多架构编译和UPX压缩

# 应用信息
APP_NAME := mujibot
VERSION := 1.0.0
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S')
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# 构建标志
LDFLAGS := -ldflags "-s -w -X main.version=$(VERSION) -X main.buildTime=$(BUILD_TIME) -X main.gitCommit=$(GIT_COMMIT)"
BUILD_FLAGS := -trimpath $(LDFLAGS)

# 目标目录
BUILD_DIR := ./build
DIST_DIR := ./dist

# 默认目标
.DEFAULT_GOAL := build

# 清理
.PHONY: clean
clean:
	@echo "🧹 Cleaning build artifacts..."
	@rm -rf $(BUILD_DIR) $(DIST_DIR)
	@go clean -cache

# 依赖管理
.PHONY: deps
deps:
	@echo "📦 Downloading dependencies..."
	@go mod download
	@go mod tidy

# 本地构建（当前架构）
.PHONY: build
build: deps
	@echo "🔨 Building $(APP_NAME) for current architecture..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 go build $(BUILD_FLAGS) -o $(BUILD_DIR)/$(APP_NAME) ./cmd/mujibot
	@echo "✅ Build complete: $(BUILD_DIR)/$(APP_NAME)"
	@ls -lh $(BUILD_DIR)/$(APP_NAME)

# 开发模式（带调试信息）
.PHONY: dev
dev:
	@echo "🔧 Building $(APP_NAME) in development mode..."
	@mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(APP_NAME) ./cmd/mujibot
	@echo "✅ Dev build complete: $(BUILD_DIR)/$(APP_NAME)"

# 运行
.PHONY: run
run: build
	@echo "🚀 Running $(APP_NAME)..."
	@$(BUILD_DIR)/$(APP_NAME) --config ./config.json5

# 测试
.PHONY: test
test:
	@echo "🧪 Running tests..."
	@go test -v -race -cover ./...

# 代码检查
.PHONY: lint
lint:
	@echo "🔍 Running linter..."
	@golangci-lint run ./... 2>/dev/null || echo "⚠️  golangci-lint not installed, skipping"
	@go vet ./...

# 格式化代码
.PHONY: fmt
fmt:
	@echo "📝 Formatting code..."
	@go fmt ./...

# ARMv7 构建（玩客云）
.PHONY: build-armv7
build-armv7: deps
	@echo "🔨 Building for ARMv7 (玩客云)..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 GOOS=linux GOARCH=arm GOARM=7 go build $(BUILD_FLAGS) -o $(BUILD_DIR)/$(APP_NAME)-armv7 ./cmd/mujibot
	@echo "✅ ARMv7 build complete"
	@ls -lh $(BUILD_DIR)/$(APP_NAME)-armv7

# ARM64 构建（树莓派4等）
.PHONY: build-arm64
build-arm64: deps
	@echo "🔨 Building for ARM64..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build $(BUILD_FLAGS) -o $(BUILD_DIR)/$(APP_NAME)-arm64 ./cmd/mujibot
	@echo "✅ ARM64 build complete"
	@ls -lh $(BUILD_DIR)/$(APP_NAME)-arm64

# x86_64 构建
.PHONY: build-amd64
build-amd64: deps
	@echo "🔨 Building for x86_64..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build $(BUILD_FLAGS) -o $(BUILD_DIR)/$(APP_NAME)-amd64 ./cmd/mujibot
	@echo "✅ AMD64 build complete"
	@ls -lh $(BUILD_DIR)/$(APP_NAME)-amd64

# 全平台构建
.PHONY: build-all
build-all: build-armv7 build-arm64 build-amd64
	@echo "✅ All builds complete"
	@echo "📊 Build sizes:"
	@ls -lh $(BUILD_DIR)/$(APP_NAME)-*

# UPX压缩（需要安装UPX）
.PHONY: compress
compress: build-all
	@echo "🗜️  Compressing binaries with UPX..."
	@which upx >/dev/null 2>&1 && \
		upx --best --lzma $(BUILD_DIR)/$(APP_NAME)-* 2>/dev/null || \
		echo "⚠️  UPX not installed, skipping compression"
	@echo "📊 Compressed sizes:"
	@ls -lh $(BUILD_DIR)/$(APP_NAME)-* 2>/dev/null || true

# 创建发布包
.PHONY: release
release: clean build-all compress
	@echo "📦 Creating release packages..."
	@mkdir -p $(DIST_DIR)
	
	# ARMv7 包
	@mkdir -p $(DIST_DIR)/$(APP_NAME)-$(VERSION)-linux-armv7
	@cp $(BUILD_DIR)/$(APP_NAME)-armv7 $(DIST_DIR)/$(APP_NAME)-$(VERSION)-linux-armv7/$(APP_NAME)
	@cp config.json5.example $(DIST_DIR)/$(APP_NAME)-$(VERSION)-linux-armv7/config.json5
	@cp README.md $(DIST_DIR)/$(APP_NAME)-$(VERSION)-linux-armv7/
	@tar -czf $(DIST_DIR)/$(APP_NAME)-$(VERSION)-linux-armv7.tar.gz -C $(DIST_DIR) $(APP_NAME)-$(VERSION)-linux-armv7
	
	# ARM64 包
	@mkdir -p $(DIST_DIR)/$(APP_NAME)-$(VERSION)-linux-arm64
	@cp $(BUILD_DIR)/$(APP_NAME)-arm64 $(DIST_DIR)/$(APP_NAME)-$(VERSION)-linux-arm64/$(APP_NAME)
	@cp config.json5.example $(DIST_DIR)/$(APP_NAME)-$(VERSION)-linux-arm64/config.json5
	@cp README.md $(DIST_DIR)/$(APP_NAME)-$(VERSION)-linux-arm64/
	@tar -czf $(DIST_DIR)/$(APP_NAME)-$(VERSION)-linux-arm64.tar.gz -C $(DIST_DIR) $(APP_NAME)-$(VERSION)-linux-arm64
	
	# AMD64 包
	@mkdir -p $(DIST_DIR)/$(APP_NAME)-$(VERSION)-linux-amd64
	@cp $(BUILD_DIR)/$(APP_NAME)-amd64 $(DIST_DIR)/$(APP_NAME)-$(VERSION)-linux-amd64/$(APP_NAME)
	@cp config.json5.example $(DIST_DIR)/$(APP_NAME)-$(VERSION)-linux-amd64/config.json5
	@cp README.md $(DIST_DIR)/$(APP_NAME)-$(VERSION)-linux-amd64/
	@tar -czf $(DIST_DIR)/$(APP_NAME)-$(VERSION)-linux-amd64.tar.gz -C $(DIST_DIR) $(APP_NAME)-$(VERSION)-linux-amd64
	
	@echo "✅ Release packages created in $(DIST_DIR)/"
	@ls -lh $(DIST_DIR)/*.tar.gz

# Docker 构建
.PHONY: docker
docker:
	@echo "🐳 Building Docker image..."
	@docker build -t $(APP_NAME):$(VERSION) -t $(APP_NAME):latest .

# 安装到系统
.PHONY: install
install: build
	@echo "📥 Installing $(APP_NAME) to /usr/local/bin..."
	@sudo cp $(BUILD_DIR)/$(APP_NAME) /usr/local/bin/
	@sudo chmod +x /usr/local/bin/$(APP_NAME)
	@echo "✅ Installed to /usr/local/bin/$(APP_NAME)"

# 卸载
.PHONY: uninstall
uninstall:
	@echo "🗑️  Uninstalling $(APP_NAME)..."
	@sudo rm -f /usr/local/bin/$(APP_NAME)
	@echo "✅ Uninstalled"

# 安装systemd服务
.PHONY: install-service
install-service:
	@echo "🔧 Installing systemd service..."
	@sudo cp scripts/mujibot.service /etc/systemd/system/
	@sudo systemctl daemon-reload
	@echo "✅ Service installed. Use 'sudo systemctl enable --now mujibot' to start"

# 显示帮助
.PHONY: help
help:
	@echo "$(APP_NAME) Makefile"
	@echo ""
	@echo "Targets:"
	@echo "  make build         - Build for current architecture"
	@echo "  make dev           - Build in development mode (with debug info)"
	@echo "  make run           - Build and run"
	@echo "  make test          - Run tests"
	@echo "  make lint          - Run linter"
	@echo "  make fmt           - Format code"
	@echo "  make clean         - Clean build artifacts"
	@echo ""
	@echo "Cross-compilation:"
	@echo "  make build-armv7   - Build for ARMv7 (玩客云)"
	@echo "  make build-arm64   - Build for ARM64 (树莓派4)"
	@echo "  make build-amd64   - Build for x86_64"
	@echo "  make build-all     - Build for all platforms"
	@echo "  make compress      - Compress binaries with UPX"
	@echo ""
	@echo "Release:"
	@echo "  make release       - Create release packages"
	@echo "  make docker        - Build Docker image"
	@echo ""
	@echo "Installation:"
	@echo "  make install       - Install binary to /usr/local/bin"
	@echo "  make uninstall     - Remove binary"
	@echo "  make install-service - Install systemd service"
