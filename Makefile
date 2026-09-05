.PHONY: clean all build-linux-amd64 build-linux-arm64 build-darwin-amd64 build-darwin-arm64 build-windows-amd64 build-windows-arm64

check_version = \
	$(if $(VERSION),,$(error 请通过 VERSION=xxx 指定版本号))

# 获取当前时间和 git hash（如果未提供）
# 使用 RFC3339 格式，明确显示 UTC 时区偏移
BUILD_TIME ?= $(shell date -u '+%Y-%m-%dT%H:%M:%S+00:00')
HASH ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

LDFLAGS = -X main.Version=$(VERSION) -X main.Hash=$(HASH) -X main.BuildTime=$(BUILD_TIME)
GO_BUILD = CGO_ENABLED=0 go build -trimpath -tags=sonic,poll_opt -ldflags "-s -w $(LDFLAGS)"

clean:
	@echo "正在清理构建文件..."
	rm -rf bin/

build-linux-amd64:
	$(call check_version)
	@echo "构建 linux/amd64（版本：$(VERSION)，提交：$(HASH)）"
	@mkdir -p bin
	GOOS=linux GOARCH=amd64 $(GO_BUILD) -o bin/nexa-linux-amd64 ./cmd/nexa

build-linux-arm64:
	$(call check_version)
	@echo "构建 linux/arm64（版本：$(VERSION)，提交：$(HASH)）"
	@mkdir -p bin
	GOOS=linux GOARCH=arm64 $(GO_BUILD) -o bin/nexa-linux-arm64 ./cmd/nexa

build-darwin-amd64:
	$(call check_version)
	@echo "构建 darwin/amd64（版本：$(VERSION)，提交：$(HASH)）"
	@mkdir -p bin
	GOOS=darwin GOARCH=amd64 $(GO_BUILD) -o bin/nexa-darwin-amd64 ./cmd/nexa

build-darwin-arm64:
	$(call check_version)
	@echo "构建 darwin/arm64（版本：$(VERSION)，提交：$(HASH)）"
	@mkdir -p bin
	GOOS=darwin GOARCH=arm64 $(GO_BUILD) -o bin/nexa-darwin-arm64 ./cmd/nexa

build-windows-amd64:
	$(call check_version)
	@echo "构建 windows/amd64（版本：$(VERSION)，提交：$(HASH)）"
	@mkdir -p bin
	GOOS=windows GOARCH=amd64 $(GO_BUILD) -o bin/nexa-windows-amd64.exe ./cmd/nexa

build-windows-arm64:
	$(call check_version)
	@echo "构建 windows/arm64（版本：$(VERSION)，提交：$(HASH)）"
	@mkdir -p bin
	GOOS=windows GOARCH=arm64 $(GO_BUILD) -o bin/nexa-windows-arm64.exe ./cmd/nexa

all: build-linux-amd64 build-linux-arm64 build-darwin-amd64 build-darwin-arm64 build-windows-amd64 build-windows-arm64
	@echo "全部平台构建完成"
