// Copyright (C) micros. 2025-present.
//
// Created at 2025-02-10, by liasica

package micro

import (
	"fmt"
	"io"

	kratoslog "github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	"go.uber.org/zap"
)

func init() {
	// 关闭kratos日志
	kratoslog.SetLogger(kratoslog.NewStdLogger(io.Discard))
}

// 防止静态检查工具误报
var _ = Run

type Handler func(s *grpc.Server)

// Run 启动 gRPC 服务器
func Run(app, address string, h Handler, opts ...grpc.ServerOption) (server *grpc.Server, ch chan error) {
	ctx := NewContext(app)

	opts = append([]grpc.ServerOption{
		grpc.Address(address),
		grpc.Middleware(
			RecoverMiddleware(),
		),
	}, opts...)

	server = grpc.NewServer(opts...)

	h(server)

	// 使用协程启动gRPC服务器
	ch = make(chan error, 1)

	go func() {
		if err := server.Start(ctx); err != nil {
			ch <- fmt.Errorf("gRPC 服务启动失败: %w", err)
		}
	}()

	zap.L().Info(fmt.Sprintf("⇨ gRPC server listening on %s", address))

	return
}
