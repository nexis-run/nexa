// Copyright (C) micros. 2025-present.
//
// Created at 2025-02-10, by liasica

package graceful

import (
	"context"
	"os/signal"
	"syscall"
	"time"
)

var _ Gracefully = (*Server)(nil)

// Gracefully 的 Start 完成初始化并启动后台服务后返回
// Stop 在 Start 返回后调用，应响应传入上下文的截止时间
type Gracefully interface {
	Start()
	Stop(ctx context.Context)
}

type Server struct{}

func (s *Server) Start() {}

func (s *Server) Stop(_ context.Context) {}

// Run 启动服务并等待 SIGINT 或 SIGTERM
func Run(server Gracefully, opts ...Option) {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	// 首次取消后恢复系统信号处理，后续信号可中断关闭过程
	stopHandler := context.AfterFunc(ctx, stop)
	defer stopHandler()

	RunContext(ctx, server, opts...)
}

// RunContext 完成初始化后等待上下文取消，再使用独立截止时间关闭服务
func RunContext(ctx context.Context, server Gracefully, opts ...Option) {
	if ctx.Err() != nil {
		return
	}

	settings := &option{timeout: 30 * time.Second}

	for _, opt := range opts {
		if opt != nil {
			opt(settings)
		}
	}

	// 初始化与关闭串行，Stop 可以读取 Start 完成初始化的资源
	server.Start()
	<-ctx.Done()
	shutdown := context.WithoutCancel(ctx)

	if settings.timeout > 0 {
		var cancel context.CancelFunc

		shutdown, cancel = context.WithTimeout(shutdown, settings.timeout)
		defer cancel()
	}

	server.Stop(shutdown)
}
