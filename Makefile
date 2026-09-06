PLATFORMS = linux-amd64 linux-arm64 darwin-amd64 darwin-arm64 windows-amd64 windows-arm64
BUILD_TARGETS = $(addprefix build-,$(PLATFORMS))

.PHONY: clean all $(BUILD_TARGETS)

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

# 目标名形如 build-<GOOS>-<GOARCH>，Windows 产物追加 .exe 后缀
$(BUILD_TARGETS): build-%:
	$(call check_version)
	@echo "构建 $(subst -,/,$*)（版本：$(VERSION)，提交：$(HASH)）"
	@mkdir -p bin
	GOOS=$(firstword $(subst -, ,$*)) GOARCH=$(lastword $(subst -, ,$*)) $(GO_BUILD) -o bin/nexa-$*$(if $(filter windows-%,$*),.exe) ./cmd/nexa

all: $(BUILD_TARGETS)
	@echo "全部平台构建完成"
