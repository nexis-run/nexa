# gRPC 日志

`kit/micro` 提供请求日志中间件，通过服务选项显式启用：

```go
server := micro.New(
    ":9000",
    registerServices,
    micro.LoggingMiddlewareServerOption(),
)
err := server.Start(micro.NewContext("my-service"))
```

`registerServices` 的类型为 `func(*grpc.Server)`，其中 `grpc` 来自 `github.com/go-kratos/kratos/v2/transport/grpc`。服务关闭时调用 `server.Stop(ctx)`。

中间件记录传输类型 `kind`、方法 `operation`、状态码 `code` 和耗时 `duration`。错误请求额外记录 `reason` 与 `error`，不自动记录请求和响应正文。

## 日志配置

```go
err := logger.Setup(&configure.Logger{
    Name:   "my-service",
    Stdout: true,
    Kafka: &configure.LoggerKafka{
        Brokers: []string{"localhost:9092"},
        Topic:   "logs",
    },
})
if err != nil {
    return err
}
```

控制台输出从 Debug 级别开始，Kafka 输出从 Info 级别开始。Kafka 日志单条上限为 256 KiB，待处理正文合计最多 16 MiB，队列最多 4096 条；容量耗尽时丢弃日志并报告，超大日志直接返回错误。

停机时先停止接收业务请求，然后使用带有超时时间的上下文调用 `logger.Close(ctx)`。关闭超时表示等待结束，后台仍可能继续排空。

## 错误与敏感信息

普通 RPC panic 返回 `Internal` 和通用错误文本；原始 panic 和堆栈仅写入服务端日志。业务代码自行添加的日志字段、错误消息和 panic 内容仍需避免包含密码、令牌等敏感数据。

`micro.New` 默认包含保护完整流式处理过程的 `RecoverStreamInterceptor`。使用自定义 `grpc.StreamInterceptor(...)` 选项时，需要显式将 `micro.RecoverStreamInterceptor()` 放入组合中的第一项，使其覆盖后续拦截器和 handler。
