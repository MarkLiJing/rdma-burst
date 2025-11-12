# RDMA 大文件传输服务构建脚本

# 项目信息
PROJECT_NAME := rdma-burst
VERSION := 1.0.0
BUILD_TIME := $(shell date +%Y%m%d%H%M%S)
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# 构建目录
BUILD_DIR := build
BIN_DIR := bin
DIST_DIR := dist

# 可执行文件
SERVER_BINARY := server
CLIENT_BINARY := client
COMBINED_BINARY := rdma-burst

# Go 构建参数
GO := go
GO_BUILD_FLAGS := -ldflags "-X main.version=$(VERSION) -X main.buildTime=$(BUILD_TIME) -X main.gitCommit=$(GIT_COMMIT)"
GO_TEST_FLAGS := -v -race

# 默认目标
.PHONY: all
all: build

# 构建所有目标
.PHONY: build
build: clean server client combined

# 构建服务端
.PHONY: server
server:
	@echo "构建服务端..."
	@mkdir -p $(BUILD_DIR) $(BIN_DIR)
	$(GO) build $(GO_BUILD_FLAGS) -o $(BUILD_DIR)/$(SERVER_BINARY) ./cmd/server
	@cp $(BUILD_DIR)/$(SERVER_BINARY) $(BIN_DIR)/
	@echo "服务端构建完成: $(BUILD_DIR)/$(SERVER_BINARY)"

# 构建客户端
.PHONY: client
client:
	@echo "构建客户端..."
	@mkdir -p $(BUILD_DIR) $(BIN_DIR)
	$(GO) build $(GO_BUILD_FLAGS) -o $(BUILD_DIR)/$(CLIENT_BINARY) ./cmd/client
	@cp $(BUILD_DIR)/$(CLIENT_BINARY) $(BIN_DIR)/
	@echo "客户端构建完成: $(BUILD_DIR)/$(CLIENT_BINARY)"

# 构建统一的可执行文件（支持服务端和客户端模式）
.PHONY: combined
combined:
	@echo "构建统一可执行文件..."
	@mkdir -p $(BUILD_DIR) $(BIN_DIR) $(DIST_DIR)
	$(GO) build $(GO_BUILD_FLAGS) -o $(BUILD_DIR)/$(COMBINED_BINARY) ./cmd/combined
	@cp $(BUILD_DIR)/$(COMBINED_BINARY) $(BIN_DIR)/
	@echo "统一可执行文件构建完成: $(BUILD_DIR)/$(COMBINED_BINARY)"

# 安装依赖
.PHONY: deps
deps:
	@echo "安装依赖..."
	$(GO) mod download
	$(GO) mod tidy

# 运行测试
.PHONY: test
test:
	@echo "运行测试..."
	$(GO) test $(GO_TEST_FLAGS) ./...

# 运行单元测试
.PHONY: test-unit
test-unit:
	@echo "运行单元测试..."
	$(GO) test $(GO_TEST_FLAGS) ./tests/unit/...

# 运行集成测试
.PHONY: test-integration
test-integration:
	@echo "运行集成测试..."
	$(GO) test $(GO_TEST_FLAGS) ./tests/integration/...

# 运行端到端测试
.PHONY: test-e2e
test-e2e:
	@echo "运行端到端测试..."
	$(GO) test $(GO_TEST_FLAGS) ./tests/e2e/...

# 代码格式化
.PHONY: fmt
fmt:
	@echo "格式化代码..."
	$(GO) fmt ./...

# 代码检查
.PHONY: vet
vet:
	@echo "检查代码..."
	$(GO) vet ./...

# 代码质量检查
.PHONY: lint
lint: fmt vet
	@echo "代码质量检查完成"

# 清理构建文件
.PHONY: clean
clean:
	@echo "清理构建文件..."
	@rm -rf $(BUILD_DIR) $(BIN_DIR) $(DIST_DIR)
	@echo "清理完成"

# 创建发布包
.PHONY: dist
dist: build
	@echo "创建发布包..."
	@mkdir -p $(DIST_DIR)
	@tar -czf $(DIST_DIR)/$(PROJECT_NAME)-$(VERSION)-linux-amd64.tar.gz \
		-C $(BUILD_DIR) \
		$(SERVER_BINARY) $(CLIENT_BINARY) $(COMBINED_BINARY)
	@echo "发布包创建完成: $(DIST_DIR)/$(PROJECT_NAME)-$(VERSION)-linux-amd64.tar.gz"

# 安装到系统
.PHONY: install
install: build
	@echo "安装到系统..."
	@install -m 755 $(BUILD_DIR)/$(SERVER_BINARY) /usr/local/bin/$(SERVER_BINARY)
	@install -m 755 $(BUILD_DIR)/$(CLIENT_BINARY) /usr/local/bin/$(CLIENT_BINARY)
	@install -m 755 $(BUILD_DIR)/$(COMBINED_BINARY) /usr/local/bin/$(COMBINED_BINARY)
	@echo "安装完成"

# 卸载
.PHONY: uninstall
uninstall:
	@echo "卸载..."
	@rm -f /usr/local/bin/$(SERVER_BINARY)
	@rm -f /usr/local/bin/$(CLIENT_BINARY)
	@rm -f /usr/local/bin/$(COMBINED_BINARY)
	@echo "卸载完成"

# 开发模式运行服务端
.PHONY: run-server
run-server: build
	@echo "启动服务端..."
	@$(BUILD_DIR)/$(SERVER_BINARY) --config configs/server.yaml

# 开发模式运行客户端
.PHONY: run-client
run-client: build
	@echo "启动客户端..."
	@$(BUILD_DIR)/$(CLIENT_BINARY) --help

# 显示帮助信息
.PHONY: help
help:
	@echo "RDMA 大文件传输服务构建系统"
	@echo ""
	@echo "可用目标:"
	@echo "  all          构建所有目标（默认）"
	@echo "  build        构建服务端和客户端"
	@echo "  server       仅构建服务端"
	@echo "  client       仅构建客户端"
	@echo "  combined     构建统一可执行文件"
	@echo "  deps         安装依赖"
	@echo "  test         运行所有测试"
	@echo "  test-unit    运行单元测试"
	@echo "  test-integration 运行集成测试"
	@echo "  test-e2e     运行端到端测试"
	@echo "  fmt          格式化代码"
	@echo "  vet          检查代码"
	@echo "  lint         代码质量检查"
	@echo "  clean        清理构建文件"
	@echo "  dist         创建发布包"
	@echo "  install      安装到系统"
	@echo "  uninstall    卸载"
	@echo "  run-server   开发模式运行服务端"
	@echo "  run-client   开发模式运行客户端"
	@echo "  help         显示此帮助信息"
	@echo ""
	@echo "示例:"
	@echo "  make build          # 构建所有目标"
	@echo "  make run-server     # 运行服务端"
	@echo "  make test           # 运行测试"

# 显示版本信息
.PHONY: version
version:
	@echo "$(PROJECT_NAME) version $(VERSION)"
	@echo "Build time: $(BUILD_TIME)"
	@echo "Git commit: $(GIT_COMMIT)"