// Copyright (C) micros. 2025-present.
//
// Created at 2025-02-11, by liasica

package micro

import (
	"context"
	"runtime"

	"github.com/go-kratos/kratos/v2/middleware"
	"go.uber.org/zap"
)

func RecoverMiddleware() middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req any) (reply any, err error) {
			defer func() {
				if r := recover(); r != nil {
					buf := make([]byte, 64<<10) //nolint:mnd
					n := runtime.Stack(buf, false)

					buf = buf[:n]
					zap.L().Error("捕获gRPC未处理崩溃", zap.Reflect("request", req), zap.Any("panic", r), zap.String("stack", string(buf)))
				}
			}()

			return handler(ctx, req)
		}
	}
}
