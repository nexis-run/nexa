# 服务启动与关闭

`kit/graceful` 管理初始化、取消与关闭的执行顺序，不负责创建 HTTP、RPC 或消息客户端。

## 生命周期

服务实现以下接口：

```go
type Gracefully interface {
    Start()
    Stop(ctx context.Context)
}
```

`Start` 完成资源初始化并启动后台服务后返回，不直接执行阻塞的 `Serve` 循环。`Stop` 只在 `Start` 返回后调用，可以读取已初始化的资源。启动错误由应用处理；后台服务退出时，应用可取消传给 `RunContext` 的上下文。

等待系统信号：

```go
graceful.Run(server, graceful.WithTimeout(30*time.Second))
```

由调用方控制取消：

```go
graceful.RunContext(ctx, server, graceful.WithTimeout(30*time.Second))
```

`RunContext` 在调用前已被取消时不启动服务。启动过程中收到取消，会等 `Start` 返回后再关闭，不会并发执行初始化与关闭。

`Run` 监听 `SIGINT` 和 `SIGTERM`，首次取消后恢复系统信号处理。其他代码也注册了这些信号时，实际处理行为取决于全部监听者，参见 [Go 信号文档](https://pkg.go.dev/os/signal#NotifyContext)。

## 关闭上下文

关闭上下文保留调用方的上下文值，但不继承已经触发的取消和截止时间。默认关闭期限为 30 秒；`WithTimeout` 小于等于 0 时不设置截止时间。

`Stop` 必须自行响应上下文取消。关闭期限不强制终止 goroutine，也不保证不响应上下文的依赖会返回。

## 资源顺序

应用按依赖关系安排关闭：

1. 停止接收新请求和新任务。
2. 等待已有处理结束，关闭 HTTP、RPC 与消息消费者。
3. 关闭处理过程中使用的数据库和其他客户端。
4. 调用 `logger.Close(ctx)`，等待日志完成发送尝试。

独立资源可以并行关闭，共享资源应在使用者退出后关闭。需要为各阶段预留时间时，由应用创建各自的子上下文。
