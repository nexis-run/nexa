// Copyright (C) micros. 2025-present.
//
// Created at 2025-02-11, by liasica

package micro

import (
	"context"

	"github.com/go-kratos/kratos/v2/middleware"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func RecoverMiddleware() middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req any) (reply any, err error) {
			defer func() {
				if r := recover(); r != nil {
					reply = nil
					err = panicError(r)
				}
			}()

			reply, err = handler(ctx, req)

			return
		}
	}
}

// RecoverStreamInterceptor 保护完整的流处理过程，可与自定义流拦截器组合
func RecoverStreamInterceptor() grpc.StreamServerInterceptor {
	return func(server any, stream grpc.ServerStream, _ *grpc.StreamServerInfo, handler grpc.StreamHandler) (err error) {
		defer func() {
			if r := recover(); r != nil {
				err = panicError(r)
			}
		}()

		err = handler(server, stream)

		return
	}
}

func panicError(value any) error {
	zap.L().Error("捕获 gRPC 未处理崩溃", zap.Any("panic", value), zap.Stack("stack"))

	return status.Error(codes.Internal, "Internal Server Error")
}
