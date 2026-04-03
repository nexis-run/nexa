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

// 防止静态检查工具误报
var _ Gracefully = (*Server)(nil)
var _ = Run

type Gracefully interface {
	Start()
	Stop(ctx context.Context)
}

type Server struct{}

func (s *Server) Start() {}

func (s *Server) Stop(_ context.Context) {}

// Run 启动服务
func Run(s Gracefully, opts ...Option) {
	// 设置默认选项
	o := &option{
		timeout: 30 * time.Second, // 默认超时时间为30秒
	}

	for _, opt := range opts {
		opt.apply(o)
	}

	sig, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	// 启动服务
	go s.Start()

	// 当中断信号发生时，关闭服务器并返回
	<-sig.Done()

	// 如果有设置超时时间，则使用该时间来优雅地关闭服务
	ctx := context.Background()

	if o.timeout > 0 {
		var cancel context.CancelFunc

		ctx, cancel = context.WithTimeout(ctx, o.timeout)
		defer cancel()
	}

	s.Stop(ctx)
}
