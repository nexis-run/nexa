# Nexa

Nexa 提供 Go 服务基础组件和代码生成 CLI。

- `cmd/nexa`：项目诊断、配置管理、Ent、DAO、DI 和 Echo Context 生成。
- `kit`：REST、gRPC、权限、日志、应用配置和 Ent 扩展。
- `pkg`：Kafka、Pulsar 和通用工具。

## 安装 CLI

```bash
go install nexis.run/nexa/cmd/nexa@latest
```

也可从带有 `checksums.txt` 的稳定 Release 安装预编译文件：

```bash
curl -fsSL https://nexis.run/nexa | bash
nexa version
```

安装目录依次取 `NEXA_INSTALL_DIR`、`go env GOBIN`、`go env GOPATH` 下的 `bin`，未安装 Go 时使用 `$HOME/.local/bin`。安装器校验 SHA-256 和程序版本，默认拒绝降级或覆盖无法识别版本的程序。

指定安装目录和版本：

```bash
curl -fsSL https://nexis.run/nexa | NEXA_INSTALL_DIR="$HOME/.local/bin" bash -s -- --version v0.1.1
```

版本号须替换为 Releases 中实际存在的稳定版本。重新安装或降级时显式传入 `--force`。

## 代码生成

在已有 Go 模块中运行：

```bash
nexa config init
nexa ent new User Order
go mod tidy
nexa ent generate
nexa new dao User Order
nexa new echoctx Rider
go mod tidy
nexa doctor
```

项目需能下载 `nexis.run/nexa` 及其依赖。`config init` 可以在尚未创建 `go.mod` 时执行；代码生成命令需要 Go 模块。默认配置为 `.nexa.yaml`，CLI 在当前模块内向上查找，没有配置文件时使用默认值。`-c/--config` 精确指定配置文件，不回退到其他文件。输出目录相对配置所属模块根目录解析。

```bash
nexa -c config/.nexa.yaml ent generate
nexa ent list --json
nexa new dao User Order --dry-run
nexa ent generate --check --json
nexa config validate
```

名称必须是大写字母开头的 Go 标识符，且不能生成 `_test.go` 或平台专属文件。DAO 与 Context 生成会检查整个包内的顶层声明冲突。生成命令支持批量预检、`--dry-run` 预览和 `--check` 差异检查。内容相同的文件不重写；初始化、schema、DAO 和 Echo Context 模板内容不同时，需要 `--force` 才能覆盖。`--check` 有差异返回状态码 `2`，其他错误返回 `1`。

`new dao` 根据已有 Ent 实体生成代码，默认构造器接收 `*ent.Client`。DAO 提供 `Client()`、`Query()` 和 `Tx(tx)`，可使用当前实体客户端的完整类型化 API。默认配置的 DI 文件不存在时自动创建，已有 DI 保留自定义字段和 provider，并补齐请求实体的 Wire 字段名单。只生成 DAO 时使用 `--di=false`；需要生成 Wire 注入代码时，在应用的注入函数中提供 `*ent.Client`。显式设置 `ormclient` 时，DAO 使用该表达式创建无参数构造器。

`ent generate` 在临时目录运行 Ent，成功后应用生成文件变更。手写文件不自动覆盖；带有 Ent 生成标记的过期文件会删除。该命令及其预览、检查模式都会执行项目 schema 代码，且依赖需事先准备好，生成过程不自动修改 `go.mod` 或 `go.sum`。

`doctor` 只读检查 Go、配置、模块、schema 和 DI。`version --json` 输出构建信息，`completion` 支持 Bash、Zsh、Fish 和 PowerShell。完整命令和配置说明见 [CLI 快速入门](docs/CLI_QUICK_START.md)。

生成的 Ent 包支持 `go generate`。直接使用 Ent 客户端的应用需要在启动包导入对应的 runtime：

```go
import _ "example.com/app/internal/infrastructure/ent/runtime"
```

生成的 DAO 已包含此导入。使用 `SoftDeleteMixin` 的实体默认过滤已删除记录，并拒绝直接硬删除：

```go
var archived []*ent.User
err := client.User.SoftDeleteOneID(id).Exec(ctx)
archived, err = client.User.QueryDeleted(ctx)
err = client.User.RestoreOneID(id).Exec(ctx)
err = client.User.HardDeleteOneID(ctx, id)
```

需要组合归档查询或定向硬删除时，使用 `entx.SkipSoftDelete(ctx)` 执行带有明确筛选条件的 Ent builder。无条件 `HardDelete(ctx)` 会永久删除该实体的全部记录。

## 服务基础组件

### HTTP 与 gRPC

`rest.New(app, routes)` 和 `micro.New(address, register, options...)` 只构造服务，由调用方管理启动与关闭；`Run` 提供异步启动入口，返回的错误通道在服务停止后关闭。

`graceful.Run` 等待系统信号，`graceful.RunContext` 接受调用方的取消上下文。服务的 `Start` 必须完成初始化、启动后台服务后返回；`Stop` 与初始化串行，使用独立的关闭期限。完整约定见 [服务启动与关闭](docs/graceful-shutdown.md)。

REST 响应的 `code` 同时作为 HTTP 状态码。`SendResponse(result, err)` 与返回错误使用一致的状态映射，支持 Nexa、Echo 和 gRPC／Kratos 错误。未知错误返回 `500`，服务端错误使用通用消息，出错时不输出部分业务数据。业务错误通过 `rest.NewError` 或 `rest.WrapError` 指定状态码。`Validator.Translate(err)` 提供中文与自定义校验消息，`BindAndValidate` 使用该翻译，并保留可通过 `errors.As` 访问的原始错误。普通与流式 RPC 的默认恢复逻辑将 panic 转为 `Internal`，详细错误与堆栈写入服务端日志。自定义流式拦截器时的组合方式见 [gRPC 日志](docs/grpc-logging.md)。

CORS 白名单显式覆盖默认值：

```go
server.Use(rest.CORSMiddleware(
    rest.CORSWithAllowOrigins("https://app.example.com"),
    rest.CORSWithAllowMethods(http.MethodGet, http.MethodPost),
    rest.CORSWithAllowHeaders("Content-Type", "Authorization"),
))
```

未指定来源规则时允许所有来源。配置空白名单时不允许任何来源。代理部署应按网络边界配置 Echo 的 `IPExtractor`。

`GetRequestURL` 支持原始 URL、URI 和转发前缀；相关代理请求头应由可信网关覆盖，不能直接将客户端传入的值用于安全判断。

请求日志默认不记录正文。通过 `DumpConfig.RequestBody`、`ResponseBody` 显式启用，每份正文默认最多记录 64 KiB，可使用 `BodyMaxBytes` 和 skipper 控制。

### 权限客户端

`authz.New` 创建独立客户端，调用方负责 `Close`。REST 通过 `WithRBACClient(client)` 使用指定实例；`authz.Setup` 提供进程级默认客户端并返回初始化错误，`authz.Close` 释放默认连接。

路由所需权限通过 `WithRBACPermissionKey(key)` 绑定，项目通过 `WithRBACProjectCode(code)` 指定；非空配置优先于请求头。未配置时读取对应请求头，调用方负责在可信入口绑定权限与路由。

非本地连接应通过 `WithTransportCredentials` 配置 TLS。认证信息会写入 outgoing metadata，同时保留请求追踪等其他字段。

### 应用配置与 ID 生成

`configure.Load[T]` 读取单个 YAML 文档，支持 `koanf` 字段标签、嵌入配置、时间间隔和逗号分隔列表。应用名称不能是空白，启用的 Kafka 日志配置必须提供非空 topic 和 brokers。整数转换拒绝溢出、符号丢失与小数截断。

使用 `Configure.Sonyflake()` 时必须配置实例编号：

```yaml
app: example
environment: production
sonyflake_machine_id: 1
logger:
  stdout: true
```

编号范围为 `0~65535`，使用同一 ID 空间的每个进程都必须具有唯一编号，包括同机进程和容器。应用部署负责编号分配；同一进程内相同编号共享生成器。库不自动分配或验证跨进程编号。实例编号的唯一性要求见 [Sonyflake 官方说明](https://github.com/sony/sonyflake/tree/v2.2.0)。

### 日志和消息生命周期

`logger.Setup` 校验配置并返回错误。停机时先停止业务写入，再调用 `logger.Close(ctx)` 等待日志排空。

Kafka 日志单条上限为 256 KiB，待处理正文合计上限为 16 MiB，队列最多 4096 条。超大日志直接返回错误，容量耗尽时丢弃并向标准错误输出报告，可通过 `Dropped()` 获取累计丢弃数。该队列不适合作为可靠业务事件存储。独立 `KafkaWriter` 支持 `SyncContext` 和 `CloseContext`；无参数入口最多等待 10 秒。等待超时后，后台发送和资源回收仍可能继续。

`clara.Writer.SendMessages` 将分区级重试交给 Kafka 客户端。超时或取消并不证明消息未送达，业务消费者需要幂等处理。Reader 在回调成功后提交 offset；回调失败即结束本次监听。恢复消费时关闭旧 Reader，再创建相同消费组的新 Reader，从已提交位置继续消费，不要复用失败的 Reader。

Pulsar 消费随上下文取消或客户端关闭退出，正在执行的用户 handler 需要自行返回。`CloseContext` 限制调用方等待时间，开始关闭后拒绝创建新资源。

`pulbus.WithClientOptions` 配置 Pulsar 认证、TLS 与连接超时，连接地址由 `pulbus.New` 的第一个参数指定。`WithAdmin` 的初始化错误由 `New` 返回。`ParseTopicName` 提供严格解析，保留持久化类型与分区号；同一 topic 的并发首次发送共享一次 producer 创建。取消发送会取消等待，不强制中断 SDK 的后台创建；订阅创建的等待时间由 SDK 超时配置控制。

### 通用工具

`pool.Pool[T]` 只接受 `T`，零值可用；为空且没有工厂时返回 `T` 的零值，首次使用后不能复制。`convert.Reverse` 返回副本并保留 nil 与空切片的区别。

`channel.SafeSend` 是阻塞发送，nil 或已关闭的通道返回 `true`。调用方必须协调发送和关闭，panic 恢复不能替代同步；关闭通道前应先停止发送者。

## 开发与发布

需要 `go.mod` 指定的 Go 版本。公开 CLI 可以独立构建和检查：

```bash
go test ./cmd/nexa/...
go vet ./cmd/nexa/...
go build -o bin/nexa ./cmd/nexa
```

全包检查需要私有 RBAC 模块的只读访问权限：

```bash
go mod tidy -diff
go test -race ./...
go vet ./...
```

Pulsar 集成检查需要实际服务，配置方式见对应测试文件。CI 在 PR、`master` 分支推送和手动触发时检查公开 CLI。

发布由 `vX.Y.Z` 稳定版本标签触发，生成 Linux、macOS、Windows 的 amd64／arm64 六个平台文件和 `checksums.txt`。Linux CLI 不依赖动态 glibc。详细构建和发布要求见 [CLI 快速入门](docs/CLI_QUICK_START.md)。

## License

[MIT](LICENSE)。
