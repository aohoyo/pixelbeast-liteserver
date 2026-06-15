# ========================================
# PixelBeast LiteServer - Makefile
# ========================================
# 跨平台构建脚本（需 GNU Make：Linux/macOS 原生，Windows: choco/scoop install make）
#
# 常用命令：
#   make help       查看所有命令
#   make setup      首次设置（安装前端依赖 + 构建）
#   make dev        启动后端（开发模式）
#   make dev-fe     启动前端 Vite dev server（HMR）
#   make build      构建前端 + 编译后端二进制
#   make run        运行编译后的二进制
#   make test       运行测试

# ---------- 变量 ----------
BINARY     := pixelbeast
BACKEND    := ./backend/cmd
FRONTEND   := frontend/vue
GOPROXY    ?= https://goproxy.cn,direct
GOFLAGS    := -ldflags "-s -w"

# ---------- 默认目标 ----------
.DEFAULT_GOAL := help

.PHONY: help
help: ## 显示帮助
	@printf "像素兽 PixelBeast Makefile\n\n"
	@printf "开发:\n"
	@printf "  make setup       首次设置（安装前端依赖 + 构建）\n"
	@printf "  make dev         启动后端开发模式（go run）\n"
	@printf "  make dev-air     后端热重载（需 air）\n"
	@printf "  make dev-fe      前端 Vite dev server（HMR → :5173）\n"
	@printf "\n构建:\n"
	@printf "  make build       完整构建（前端 + 后端二进制）\n"
	@printf "  make build-be    仅编译后端\n"
	@printf "  make build-fe    仅构建前端\n"
	@printf "  make run         运行编译后的二进制\n"
	@printf "  make build-linux    交叉编译 Linux amd64\n"
	@printf "  make build-linux-arm 交叉编译 Linux arm64\n"
	@printf "  make build-windows 交叉编译 Windows\n"
	@printf "  make build-mac   交叉编译 macOS\n"
	@printf "\n测试与检查:\n"
	@printf "  make test        Go 测试\n"
	@printf "  make vet         静态检查\n"
	@printf "  make fmt         格式化 Go\n"
	@printf "  make typecheck-fe 前端类型检查\n"
	@printf "  make check-all   全面检查（vet+test+类型）\n"
	@printf "\n清理:\n"
	@printf "  make clean       清理二进制 + 前端 dist\n"
	@printf "  make clean-all   深度清理（含 node_modules）\n"

# ========================================
# 开发
# ========================================

.PHONY: dev
dev: ## 启动后端开发模式
	@printf "→ 启动后端（开发模式）\n"
	GOPROXY=$(GOPROXY) go run $(BACKEND)

.PHONY: dev-air
dev-air: ## 后端热重载（需 air）
	air

.PHONY: dev-fe
dev-fe: ## 前端 Vite dev server（HMR）
	@printf "→ 前端 Vite dev server → http://localhost:5173\n"
	cd $(FRONTEND) && npm run dev

# ========================================
# 前端
# ========================================

.PHONY: install-fe
install-fe: ## 安装前端依赖
	@printf "→ 安装前端依赖\n"
	cd $(FRONTEND) && npm install

.PHONY: build-fe
build-fe: ## 构建前端
	@printf "→ 构建前端\n"
	cd $(FRONTEND) && npm run build

.PHONY: typecheck-fe
typecheck-fe: ## 前端类型检查
	cd $(FRONTEND) && npx vue-tsc --noEmit

# ========================================
# 后端构建
# ========================================

.PHONY: build
build: build-fe build-be ## 完整构建
	@printf "✓ 构建完成 → ./$(BINARY)\n"

.PHONY: build-be
build-be: ## 编译后端二进制
	@printf "→ 编译后端\n"
	GOPROXY=$(GOPROXY) go build $(GOFLAGS) -o $(BINARY) $(BACKEND)

.PHONY: run
run: ## 运行二进制
	./$(BINARY)

# 交叉编译
.PHONY: build-linux
build-linux: build-fe ## 交叉编译 Linux amd64
	GOOS=linux GOARCH=amd64 GOPROXY=$(GOPROXY) go build $(GOFLAGS) -o $(BINARY)-linux-amd64 $(BACKEND)

.PHONY: build-linux-arm
build-linux-arm: build-fe ## 交叉编译 Linux arm64
	GOOS=linux GOARCH=arm64 GOPROXY=$(GOPROXY) go build $(GOFLAGS) -o $(BINARY)-linux-arm64 $(BACKEND)

.PHONY: build-windows
build-windows: build-fe ## 交叉编译 Windows
	GOOS=windows GOARCH=amd64 GOPROXY=$(GOPROXY) go build $(GOFLAGS) -o $(BINARY).exe $(BACKEND)

.PHONY: build-mac
build-mac: build-fe ## 交叉编译 macOS
	GOOS=darwin GOARCH=arm64 GOPROXY=$(GOPROXY) go build $(GOFLAGS) -o $(BINARY)-darwin-arm64 $(BACKEND)

# ========================================
# 测试与检查
# ========================================

.PHONY: test
test: ## Go 测试
	go test ./backend/...

.PHONY: vet
vet: ## 静态检查
	go vet ./backend/... ./frontend/...

.PHONY: fmt
fmt: ## 格式化 Go
	go fmt ./backend/...

.PHONY: check-all
check-all: vet test typecheck-fe ## 全面检查

# ========================================
# 清理
# ========================================

.PHONY: clean
clean: ## 清理二进制 + 前端 dist
	@printf "→ 清理构建产物\n"
	$(RM) $(BINARY) $(BINARY).exe $(BINARY)-*
	$(RM) -r $(FRONTEND)/dist

.PHONY: clean-all
clean-all: clean ## 深度清理（含 node_modules）
	$(RM) -r $(FRONTEND)/node_modules
	@printf "✓ 深度清理完成（需重新 npm install）\n"

# ========================================
# 首次设置
# ========================================

.PHONY: setup
setup: install-fe build ## 首次设置
	@printf "✓ 设置完成\n"
	@printf "  开发: make dev + make dev-fe\n"
	@printf "  生产: make run\n"
