// Copyright (C) micros. 2025-present.
//
// Created at 2025-02-10, by liasica

package micro

import (
	"context"
	"time"

	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/transport"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func LoggingMiddlewareServerOption() grpc.ServerOption {
	return grpc.Middleware(
		LoggingMiddleware(),
	)
}

// LoggingMiddleware 创建一个日志中间件，用于记录gRPC请求详情
func LoggingMiddleware() middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			var (
				code      int32
				reason    string
				kind      string
				operation string
			)

			startTime := time.Now()

			// 从transport中获取元信息
			if info, ok := transport.FromServerContext(ctx); ok {
				kind = info.Kind().String()
				operation = info.Operation()
			}

			// 执行实际的处理器
			reply, err := handler(ctx, req)

			// 计算耗时
			duration := time.Since(startTime)

			// 处理错误信息
			if err != nil {
				se := status.Convert(err)

				code = int32(se.Code())
				reason = se.Message()
			} else {
				code = int32(codes.OK)
			}

			// 记录日志
			logger := zap.L()

			fields := []zap.Field{
				zap.String("kind", kind),
				zap.String("operation", operation),
				zap.Int32("code", code),
				zap.Duration("duration", duration),
			}

			if err != nil {
				fields = append(fields, zap.String("reason", reason))
				fields = append(fields, zap.Error(err))
				logger.Error("gRPC request failed", fields...)
			} else {
				logger.Info("gRPC request completed", fields...)
			}

			return reply, err
		}
	}
}
