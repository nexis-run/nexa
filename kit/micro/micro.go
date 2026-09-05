// Copyright (C) micros. 2025-present.
//
// Created at 2025-02-10, by liasica

package micro

import (
	"fmt"

	"github.com/go-kratos/kratos/v2/transport/grpc"
)

type Handler func(s *grpc.Server)

// New 创建 gRPC 服务器，调用方负责启动和关闭
func New(address string, register Handler, opts ...grpc.ServerOption) (server *grpc.Server) {
	opts = append([]grpc.ServerOption{
		grpc.Address(address),
		grpc.Middleware(
			RecoverMiddleware(),
		),
		grpc.StreamInterceptor(RecoverStreamInterceptor()),
		grpc.UnaryInterceptor(RecoverUnaryInterceptor()),
	}, opts...)

	server = grpc.NewServer(opts...)

	if register != nil {
		register(server)
	}

	return
}

// Run 启动 gRPC 服务器，服务停止后关闭错误通道
func Run(app, address string, register Handler, opts ...grpc.ServerOption) (server *grpc.Server, ch chan error) {
	server = New(address, register, opts...)
	ctx := NewContext(app)

	// 在后台启动 gRPC 服务器
	ch = make(chan error, 1)

	go func() {
		defer close(ch)

		if err := server.Start(ctx); err != nil {
			ch <- fmt.Errorf("gRPC 服务启动失败：%w", err)
		}
	}()

	return
}
